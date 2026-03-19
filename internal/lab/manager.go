package lab

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/claudeup/claudeup-lab/internal/docker"
	"github.com/google/uuid"
)

// Manager orchestrates lab lifecycle operations.
type Manager struct {
	baseDir   string
	store     *StateStore
	presets   *PresetStore
	worktrees *WorktreeManager
	profiles  *ProfileManager
	docker    *docker.Client
	images    *docker.ImageManager
}

func NewManager(baseDir string) *Manager {
	return &Manager{
		baseDir:   baseDir,
		store:     NewStateStore(filepath.Join(baseDir, "state")),
		presets:   NewPresetStore(filepath.Join(baseDir, "presets")),
		worktrees: NewWorktreeManager(filepath.Join(baseDir, "repos")),
		profiles:  NewProfileManager(filepath.Join(ClaudeupHome(), "profiles")),
		docker:    docker.NewClient(),
		images:    docker.NewImageManager(),
	}
}

func (m *Manager) Store() *StateStore          { return m.store }
func (m *Manager) Presets() *PresetStore       { return m.presets }
func (m *Manager) Docker() *docker.Client      { return m.docker }
func (m *Manager) Worktrees() *WorktreeManager { return m.worktrees }

// StartOptions configures a new lab.
type StartOptions struct {
	Project     string
	Profile     string
	Branch      string
	Name        string
	Features    []string
	BaseProfile string
	Firewall    bool
	InitScript  string
	EnvVars     map[string]string
	EnvFile     string
}

// Start creates and launches a new lab environment.
func (m *Manager) Start(opts *StartOptions) (*Metadata, error) {
	if err := m.checkPrerequisites(); err != nil {
		return nil, err
	}

	projectPath, err := filepath.Abs(opts.Project)
	if err != nil {
		return nil, fmt.Errorf("resolve project path: %w", err)
	}
	projectName := filepath.Base(projectPath)

	// Verify it's a git repo
	cmd := exec.Command("git", "-C", projectPath, "rev-parse", "--git-dir")
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("%s is not a git repository", projectPath)
	}

	// Validate env vars early to fail fast before expensive operations
	extraEnv, err := mergeEnvVars(opts.EnvFile, opts.EnvVars)
	if err != nil {
		return nil, fmt.Errorf("process environment variables: %w", err)
	}

	// Handle profile snapshotting
	profile := opts.Profile
	var snapshotName string
	if profile == "" {
		id := uuid.New().String()[:8]
		snapshotName, err = m.profiles.Snapshot(id)
		if err != nil {
			return nil, fmt.Errorf("snapshot current config: %w", err)
		}
		profile = snapshotName
	}

	labID := uuid.New().String()

	// Ensure base image
	image, err := m.images.EnsureImage(docker.ImageTag())
	if err != nil {
		return nil, fmt.Errorf("ensure base image: %w", err)
	}

	// Ensure bare clone
	barePath, err := m.worktrees.EnsureBareRepo(projectPath, projectName)
	if err != nil {
		return nil, fmt.Errorf("ensure bare repo: %w", err)
	}

	// Compute display name
	displayName := ComputeDisplayName(projectName, profile, opts.Name)
	if err := ValidateDisplayName(displayName); err != nil {
		return nil, err
	}

	existingNames := make(map[string]bool)
	if labs, _ := m.store.List(); labs != nil {
		for _, l := range labs {
			existingNames[l.DisplayName] = true
		}
	}
	displayName = DisambiguateDisplayName(displayName, labID[:8], existingNames)

	// Compute branch
	branch := opts.Branch
	if branch == "" {
		branch = "lab/" + profile
	}

	// Create worktree
	worktreePath := filepath.Join(m.baseDir, "workspaces", displayName)
	branch, err = m.worktrees.CreateWorktree(barePath, worktreePath, branch)
	if err != nil {
		return nil, fmt.Errorf("create worktree: %w", err)
	}

	// Render devcontainer.json
	dcConfig := &DevcontainerConfig{
		ProjectName:  projectName,
		Profile:      profile,
		ID:           labID,
		DisplayName:  displayName,
		Image:        image,
		BareRepoPath: barePath,
		HomeDir:      os.Getenv("HOME"),
		ClaudeupHome: ClaudeupHome(),
		GitUserName:  gitConfig("user.name"),
		GitUserEmail: gitConfig("user.email"),
		GitHubToken:  os.Getenv("GITHUB_TOKEN"),
		Context7Key:  os.Getenv("CONTEXT7_API_KEY"),
		ConfigRepo:   os.Getenv("CLAUDE_CONFIG_REPO"),
		ConfigBranch: envOrDefault("CLAUDE_CONFIG_BRANCH", "main"),
		BaseProfile:  opts.BaseProfile,
		Features:     opts.Features,
		Firewall:     opts.Firewall,
		InitScript:   opts.InitScript,
		ExtraEnv:     extraEnv,
	}
	if err := RenderDevcontainer(dcConfig, worktreePath); err != nil {
		m.worktrees.RemoveWorktree(barePath, worktreePath)
		return nil, fmt.Errorf("render devcontainer: %w", err)
	}

	// Launch container
	fmt.Println("Starting devcontainer...")
	devCmd := exec.Command("devcontainer", "up", "--workspace-folder", worktreePath)
	var stdoutBuf bytes.Buffer
	devCmd.Stdout = io.MultiWriter(os.Stdout, &stdoutBuf)
	devCmd.Stderr = os.Stderr
	runErr := devCmd.Run()
	result, parseErr := parseDevcontainerResult(stdoutBuf.String())

	// If devcontainer up failed, check whether the container actually started
	// (postCreateCommand failures exit non-zero but the container is running)
	if runErr != nil {
		if result == nil || result.ContainerID == "" {
			m.worktrees.RemoveWorktree(barePath, worktreePath)
			return nil, fmt.Errorf("devcontainer up: %w", runErr)
		}
		// Container started but postCreateCommand failed -- fall through to save metadata
		parseErr = &PostCreateError{Detail: fmt.Sprintf("%v", runErr)}
	}

	var postCreateErr error
	if parseErr != nil {
		postCreateErr = parseErr
	}

	// Save metadata (container is running even if postCreateCommand failed)
	meta := &Metadata{
		ID:          labID,
		DisplayName: displayName,
		Project:     projectPath,
		ProjectName: projectName,
		Profile:     profile,
		BareRepo:    barePath,
		Worktree:    worktreePath,
		Branch:      branch,
		Created:     time.Now().UTC(),
		Snapshot:    snapshotName,
		StartConfig: StartConfigFromResolved(projectPath, profile, branch, opts),
	}
	if err := m.store.Save(meta); err != nil {
		return nil, fmt.Errorf("save metadata: %w", err)
	}

	return meta, postCreateErr
}

// StartConfigFromResolved builds a StartConfig from the resolved local
// variables in Manager.Start (project, profile, branch) and copies the
// remaining fields from opts. The resolved params are passed explicitly
// because opts may still hold raw user input.
func StartConfigFromResolved(project, profile, branch string, opts *StartOptions) *StartConfig {
	return &StartConfig{
		Project:     project,
		Profile:     profile,
		Branch:      branch,
		LabName:     opts.Name,
		Features:    copySlice(opts.Features),
		BaseProfile: opts.BaseProfile,
		Firewall:    opts.Firewall,
		InitScript:  opts.InitScript,
		EnvVars:     copyMap(opts.EnvVars),
		EnvFile:     opts.EnvFile,
	}
}

// LabStatus returns the running status of a lab.
func (m *Manager) LabStatus(meta *Metadata) string {
	id, _ := m.docker.FindContainer(meta.Worktree)
	if id != "" {
		return "running"
	}
	if _, err := os.Stat(meta.Worktree); err == nil {
		return "stopped"
	}
	return "orphaned"
}

// Remove performs a full teardown of a lab.
func (m *Manager) Remove(meta *Metadata, confirmed bool) error {
	if !confirmed {
		return fmt.Errorf("removal not confirmed")
	}

	var errs []string

	// Stop and remove container
	containerID, _ := m.docker.FindContainerIncludingStopped(meta.Worktree)
	if containerID != "" {
		fmt.Println("Removing container...")
		if err := m.docker.RemoveContainer(containerID); err != nil {
			errs = append(errs, fmt.Sprintf("remove container: %v", err))
		}
	}

	// Remove volumes
	fmt.Println("Removing Docker volumes...")
	volumes, _ := m.docker.ListVolumes(meta.ID)
	if len(volumes) > 0 {
		if err := m.docker.RemoveVolumes(volumes); err != nil {
			errs = append(errs, fmt.Sprintf("remove volumes: %v", err))
		}
	}

	// Remove worktree
	fmt.Println("Removing worktree...")
	if err := m.worktrees.RemoveWorktree(meta.BareRepo, meta.Worktree); err != nil {
		errs = append(errs, fmt.Sprintf("remove worktree: %v", err))
	}

	// Remove metadata
	fmt.Println("Removing metadata...")
	if err := m.store.Delete(meta.ID); err != nil {
		errs = append(errs, fmt.Sprintf("remove metadata: %v", err))
	}

	// Clean up snapshot profile if applicable
	if meta.Snapshot != "" {
		m.profiles.CleanupSnapshot(meta.Snapshot)
	}

	if len(errs) > 0 {
		fmt.Fprintf(os.Stderr, "Warning: partial cleanup errors: %s\n", strings.Join(errs, "; "))
	}

	fmt.Printf("Removed lab: %s\n", meta.DisplayName)

	// Check if bare repo has remaining worktrees
	count, err := m.worktrees.WorktreeCount(meta.BareRepo)
	if err == nil && count <= 1 {
		return &BareRepoCleanupPrompt{BareRepo: meta.BareRepo}
	}

	return nil
}

// BareRepoCleanupPrompt is returned when a bare repo has no remaining worktrees.
type BareRepoCleanupPrompt struct {
	BareRepo string
}

func (e *BareRepoCleanupPrompt) Error() string {
	return fmt.Sprintf("bare repo %s has no remaining worktrees", e.BareRepo)
}

func (m *Manager) checkPrerequisites() error {
	if !m.docker.IsRunning() {
		return fmt.Errorf("Docker is not running (start Docker Desktop or the docker daemon)")
	}
	if _, err := exec.LookPath("devcontainer"); err != nil {
		return fmt.Errorf("devcontainer CLI not found (install: npm install -g @devcontainers/cli)")
	}
	if _, err := exec.LookPath("git"); err != nil {
		return fmt.Errorf("git not found on PATH")
	}
	return nil
}

func gitConfig(key string) string {
	cmd := exec.Command("git", "config", key)
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return string(out[:len(out)-1]) // trim newline
}

func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// mergeEnvVars reads an env file (if path is non-empty), then overlays
// flagVars on top. Flag values win over file values for the same key.
func mergeEnvVars(envFile string, flagVars map[string]string) (map[string]string, error) {
	merged := make(map[string]string)

	if envFile != "" {
		fileVars, err := ParseEnvFile(envFile)
		if err != nil {
			return nil, err
		}
		for k, v := range fileVars {
			merged[k] = v
		}
	}

	for k, v := range flagVars {
		merged[k] = v
	}

	return merged, nil
}
