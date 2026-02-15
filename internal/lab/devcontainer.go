package lab

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	assets "github.com/claudeup/claudeup-lab/embed"
)

// DevcontainerConfig holds all parameters needed to render a devcontainer.json.
type DevcontainerConfig struct {
	ProjectName  string
	Profile      string
	ID           string
	DisplayName  string
	Image        string
	BareRepoPath string
	HomeDir      string
	ClaudeupHome string // Override for ~/.claudeup; empty falls back to HomeDir/.claudeup
	GitUserName  string
	GitUserEmail string
	GitHubToken  string
	Context7Key  string
	ConfigRepo   string
	ConfigBranch string
	BaseProfile  string
	Features     []string
}

type featureEntry struct {
	Feature        string `json:"feature"`
	DefaultVersion string `json:"default_version"`
}

// RenderDevcontainer writes a devcontainer.json into the .devcontainer/
// directory under worktreePath.
func RenderDevcontainer(config *DevcontainerConfig, worktreePath string) error {
	dcDir := filepath.Join(worktreePath, ".devcontainer")
	if err := os.MkdirAll(dcDir, 0o755); err != nil {
		return fmt.Errorf("create .devcontainer: %w", err)
	}

	dc := buildDevcontainerJSON(config)

	data, err := json.MarshalIndent(dc, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal devcontainer.json: %w", err)
	}

	outPath := filepath.Join(dcDir, "devcontainer.json")
	if err := os.WriteFile(outPath, data, 0o644); err != nil {
		return fmt.Errorf("write devcontainer.json: %w", err)
	}

	return nil
}

func buildDevcontainerJSON(config *DevcontainerConfig) map[string]interface{} {
	mounts := buildMounts(config)
	features := buildFeatures(config.Features)

	env := map[string]string{
		"CLAUDE_CONFIG_DIR":    "/home/node/.claude",
		"CLAUDE_PROFILE":       config.Profile,
		"NODE_OPTIONS":         "--max-old-space-size=4096",
		"GIT_USER_NAME":        config.GitUserName,
		"GIT_USER_EMAIL":       config.GitUserEmail,
		"GITHUB_TOKEN":         config.GitHubToken,
		"CONTEXT7_API_KEY":     config.Context7Key,
		"CLAUDE_CONFIG_REPO":   config.ConfigRepo,
		"CLAUDE_CONFIG_BRANCH": config.ConfigBranch,
		"CLAUDE_BASE_PROFILE":  config.BaseProfile,
	}

	dc := map[string]interface{}{
		"name":              fmt.Sprintf("claudeup-lab - %s (%s)", config.ProjectName, config.Profile),
		"image":             config.Image,
		"features":          features,
		"remoteUser":        "node",
		"mounts":            mounts,
		"containerEnv":      env,
		"workspaceFolder":   fmt.Sprintf("/workspaces/%s", config.DisplayName),
		"postCreateCommand": "claude upgrade && /usr/local/bin/init-claude-config.sh && /usr/local/bin/init-config-repo.sh && /usr/local/bin/init-claudeup.sh",
		"waitFor":           "postCreateCommand",
	}

	return dc
}

// ClaudeupHome returns the claudeup home directory. It checks CLAUDEUP_HOME
// first and falls back to $HOME/.claudeup.
func ClaudeupHome() string {
	if v := os.Getenv("CLAUDEUP_HOME"); v != "" {
		return v
	}
	return filepath.Join(os.Getenv("HOME"), ".claudeup")
}

// claudeupHomeFor returns the claudeup home to use for a config. If the config
// has an explicit ClaudeupHome, that is used; otherwise it falls back to
// HomeDir/.claudeup.
func claudeupHomeFor(config *DevcontainerConfig) string {
	if config.ClaudeupHome != "" {
		return config.ClaudeupHome
	}
	return filepath.Join(config.HomeDir, ".claudeup")
}

func buildMounts(config *DevcontainerConfig) []string {
	id := config.ID
	home := config.HomeDir
	cupHome := claudeupHomeFor(config)

	mounts := []string{
		fmt.Sprintf("source=claudeup-lab-bashhistory-%s,target=/commandhistory,type=volume", id),
		fmt.Sprintf("source=claudeup-lab-config-%s,target=/home/node/.claude,type=volume", id),
		fmt.Sprintf("source=claudeup-lab-claudeup-%s,target=/home/node/.claudeup,type=volume", id),
	}

	// Optional bind mounts -- skip if source doesn't exist
	optionalMounts := []struct {
		source string
		target string
		opts   string
	}{
		{filepath.Join(cupHome, "profiles"), "/home/node/.claudeup/profiles", "type=bind,readonly"},
		{filepath.Join(cupHome, "ext"), "/home/node/.claudeup/ext", "type=bind,readonly"},
		{filepath.Join(home, ".claude-mem"), "/home/node/.claude-mem", "type=bind"},
		{filepath.Join(home, ".ssh"), "/home/node/.ssh", "type=bind,readonly"},
		{filepath.Join(home, ".claude", "settings.json"), "/tmp/base-settings.json", "type=bind,readonly"},
		{filepath.Join(home, ".claude.json"), "/home/node/.claude.json", "type=bind"},
	}

	for _, m := range optionalMounts {
		if _, err := os.Stat(m.source); err == nil {
			mounts = append(mounts, fmt.Sprintf("source=%s,target=%s,%s", m.source, m.target, m.opts))
		}
	}

	// Bare repo bind mount (required for git worktree resolution)
	mounts = append(mounts, fmt.Sprintf("source=%s,target=%s,type=bind", config.BareRepoPath, config.BareRepoPath))

	// Per-lab volumes
	mounts = append(mounts,
		fmt.Sprintf("source=claudeup-lab-npm-%s,target=/home/node/.npm-global,type=volume", id),
		fmt.Sprintf("source=claudeup-lab-local-%s,target=/home/node/.local,type=volume", id),
		fmt.Sprintf("source=claudeup-lab-bun-%s,target=/home/node/.bun,type=volume", id),
	)

	return mounts
}

func buildFeatures(specs []string) map[string]interface{} {
	if len(specs) == 0 {
		return map[string]interface{}{}
	}

	// Load feature registry
	var registry map[string]featureEntry
	if err := json.Unmarshal(assets.FeaturesJSON, &registry); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to parse feature registry: %v\n", err)
		return map[string]interface{}{}
	}

	features := make(map[string]interface{})
	for _, spec := range specs {
		name, version := parseFeatureSpec(spec)

		entry, ok := registry[name]
		if !ok {
			continue
		}

		if version == "" {
			version = entry.DefaultVersion
		}

		features[entry.Feature] = map[string]string{"version": version}
	}

	return features
}

func parseFeatureSpec(spec string) (name, version string) {
	idx := strings.IndexByte(spec, ':')
	if idx >= 0 {
		return spec[:idx], spec[idx+1:]
	}
	return spec, ""
}
