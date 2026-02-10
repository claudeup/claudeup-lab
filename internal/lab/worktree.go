package lab

import (
	"crypto/rand"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// WorktreeManager handles bare clone creation/refresh and git worktree operations.
type WorktreeManager struct {
	reposDir string
}

func NewWorktreeManager(reposDir string) *WorktreeManager {
	return &WorktreeManager{reposDir: reposDir}
}

// EnsureBareRepo creates or refreshes a bare clone of the source project.
// Returns the path to the bare repo.
func (w *WorktreeManager) EnsureBareRepo(sourceProject, projectName string) (string, error) {
	barePath := filepath.Join(w.reposDir, projectName+".git")
	markerFile := "lab-source-project"

	if err := os.MkdirAll(w.reposDir, 0o755); err != nil {
		return "", fmt.Errorf("create repos directory: %w", err)
	}

	// Check for source mismatch if bare repo exists
	if info, err := os.Stat(barePath); err == nil && info.IsDir() {
		stored, _ := os.ReadFile(filepath.Join(barePath, markerFile))
		if string(stored) != sourceProject {
			// Different project with same name -- use hash suffix
			hash := hashPrefix(sourceProject)
			barePath = filepath.Join(w.reposDir, fmt.Sprintf("%s-%s.git", projectName, hash))
		}
	}

	if info, err := os.Stat(barePath); err == nil && info.IsDir() {
		// Refresh existing bare clone
		w.refreshBareRepo(barePath, sourceProject)
		return barePath, nil
	}

	// Create new bare clone
	if err := w.createBareRepo(barePath, sourceProject, markerFile); err != nil {
		return "", err
	}

	return barePath, nil
}

func (w *WorktreeManager) refreshBareRepo(barePath, sourceProject string) {
	exec.Command("git", "-C", barePath, "fetch", "--all", "--prune").Run()
	exec.Command("git", "-C", barePath, "fetch", sourceProject,
		"+refs/heads/*:refs/heads/*").Run()
}

func (w *WorktreeManager) createBareRepo(barePath, sourceProject, markerFile string) error {
	// Try upstream first
	upstreamCmd := exec.Command("git", "-C", sourceProject, "remote", "get-url", "origin")
	upstreamOut, err := upstreamCmd.Output()
	upstream := strings.TrimSpace(string(upstreamOut))

	if err == nil && upstream != "" {
		cmd := exec.Command("git", "clone", "--bare", upstream, barePath)
		if cmd.Run() == nil {
			// Fetch local branches not yet pushed
			exec.Command("git", "-C", barePath, "fetch", sourceProject,
				"+refs/heads/*:refs/heads/*").Run()
		} else {
			// Upstream clone failed, fall back to local
			os.RemoveAll(barePath)
			cmd = exec.Command("git", "clone", "--bare", sourceProject, barePath)
			if err := cmd.Run(); err != nil {
				return fmt.Errorf("clone bare repo from %s: %w", sourceProject, err)
			}
		}
	} else {
		cmd := exec.Command("git", "clone", "--bare", sourceProject, barePath)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("clone bare repo from %s: %w", sourceProject, err)
		}
	}

	os.WriteFile(filepath.Join(barePath, markerFile), []byte(sourceProject), 0o644)
	return nil
}

// CreateWorktree creates a git worktree from the bare repo. If the branch
// is already checked out in another worktree, a random suffix is appended.
func (w *WorktreeManager) CreateWorktree(barePath, worktreePath, branch string) (string, error) {
	if err := os.MkdirAll(filepath.Dir(worktreePath), 0o755); err != nil {
		return "", fmt.Errorf("create workspace parent: %w", err)
	}

	// Check if branch is already checked out in another worktree
	if w.branchInUse(barePath, branch) {
		branch = branch + "-" + randomSuffix()
	}

	// Check if branch already exists in the bare repo
	checkCmd := exec.Command("git", "-C", barePath, "show-ref", "--verify",
		"--quiet", "refs/heads/"+branch)
	if checkCmd.Run() == nil {
		cmd := exec.Command("git", "-C", barePath, "worktree", "add",
			worktreePath, branch)
		if out, err := cmd.CombinedOutput(); err != nil {
			return "", fmt.Errorf("create worktree (existing branch): %w\n%s", err, out)
		}
	} else {
		cmd := exec.Command("git", "-C", barePath, "worktree", "add",
			worktreePath, "-b", branch)
		if out, err := cmd.CombinedOutput(); err != nil {
			return "", fmt.Errorf("create worktree (new branch): %w\n%s", err, out)
		}
	}

	// Exclude .devcontainer/ from git tracking
	gitDir := w.worktreeGitDir(worktreePath)
	excludeFile := filepath.Join(gitDir, "info", "exclude")
	if err := os.MkdirAll(filepath.Dir(excludeFile), 0o755); err != nil {
		return "", fmt.Errorf("create git info directory: %w", err)
	}
	content, _ := os.ReadFile(excludeFile)
	if !strings.Contains(string(content), ".devcontainer/") {
		f, err := os.OpenFile(excludeFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
		if err != nil {
			return "", fmt.Errorf("open git exclude file: %w", err)
		}
		_, writeErr := f.WriteString(".devcontainer/\n")
		closeErr := f.Close()
		if writeErr != nil {
			return "", fmt.Errorf("write git exclude: %w", writeErr)
		}
		if closeErr != nil {
			return "", fmt.Errorf("close git exclude: %w", closeErr)
		}
	}

	return branch, nil
}

func (w *WorktreeManager) RemoveWorktree(barePath, worktreePath string) error {
	cmd := exec.Command("git", "-C", barePath, "worktree", "remove",
		worktreePath, "--force")
	if err := cmd.Run(); err != nil {
		// Fall back to removing the directory directly
		if removeErr := os.RemoveAll(worktreePath); removeErr != nil {
			return fmt.Errorf("remove worktree directory %s: %w (git worktree remove also failed: %v)", worktreePath, removeErr, err)
		}
	}
	return nil
}

// WorktreeCount returns the number of worktrees associated with the bare repo.
func (w *WorktreeManager) WorktreeCount(barePath string) (int, error) {
	cmd := exec.Command("git", "-C", barePath, "worktree", "list", "--porcelain")
	out, err := cmd.Output()
	if err != nil {
		return 0, fmt.Errorf("list worktrees: %w", err)
	}

	count := 0
	for _, line := range strings.Split(string(out), "\n") {
		if strings.HasPrefix(line, "worktree ") {
			count++
		}
	}
	return count, nil
}

func (w *WorktreeManager) branchInUse(barePath, branch string) bool {
	cmd := exec.Command("git", "-C", barePath, "worktree", "list", "--porcelain")
	out, _ := cmd.Output()
	return strings.Contains(string(out), "branch refs/heads/"+branch)
}

func (w *WorktreeManager) worktreeGitDir(worktreePath string) string {
	cmd := exec.Command("git", "-C", worktreePath, "rev-parse", "--git-dir")
	out, err := cmd.Output()
	if err != nil {
		return filepath.Join(worktreePath, ".git")
	}
	result := strings.TrimSpace(string(out))
	if filepath.IsAbs(result) {
		return result
	}
	return filepath.Join(worktreePath, result)
}

func hashPrefix(s string) string {
	var h uint64
	for _, c := range s {
		h = h*31 + uint64(c)
	}
	return fmt.Sprintf("%08x", h)
}

func randomSuffix() string {
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		// Fall back to timestamp-based suffix
		return fmt.Sprintf("%x", time.Now().UnixNano()&0xFFFFFFFF)
	}
	return fmt.Sprintf("%x", b)
}
