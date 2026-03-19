package lab_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/claudeup/claudeup-lab/internal/lab"
)

func TestSaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	store := lab.NewStateStore(dir)

	meta := &lab.Metadata{
		ID:          "abc-123",
		DisplayName: "myapp-base",
		Project:     "/home/user/code/myapp",
		ProjectName: "myapp",
		Profile:     "base",
		BareRepo:    "/home/user/.claudeup-lab/repos/myapp.git",
		Worktree:    "/home/user/.claudeup-lab/workspaces/myapp-base",
		Branch:      "lab/base",
		Created:     time.Date(2026, 2, 10, 12, 0, 0, 0, time.UTC),
	}

	if err := store.Save(meta); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := store.Load("abc-123")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if loaded.DisplayName != "myapp-base" {
		t.Errorf("DisplayName = %q, want %q", loaded.DisplayName, "myapp-base")
	}
	if loaded.Branch != "lab/base" {
		t.Errorf("Branch = %q, want %q", loaded.Branch, "lab/base")
	}
}

func TestList(t *testing.T) {
	dir := t.TempDir()
	store := lab.NewStateStore(dir)

	store.Save(&lab.Metadata{ID: "id-1", DisplayName: "lab-one"})
	store.Save(&lab.Metadata{ID: "id-2", DisplayName: "lab-two"})

	labs, err := store.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(labs) != 2 {
		t.Fatalf("List returned %d labs, want 2", len(labs))
	}
}

func TestDelete(t *testing.T) {
	dir := t.TempDir()
	store := lab.NewStateStore(dir)

	store.Save(&lab.Metadata{ID: "to-delete", DisplayName: "temp"})
	if err := store.Delete("to-delete"); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	_, err := store.Load("to-delete")
	if err == nil {
		t.Error("Load after Delete should return error")
	}
}

func TestLoadNonexistent(t *testing.T) {
	dir := t.TempDir()
	store := lab.NewStateStore(dir)

	_, err := store.Load("does-not-exist")
	if err == nil {
		t.Error("Load nonexistent should return error")
	}
}

func TestListEmptyDir(t *testing.T) {
	dir := t.TempDir()
	store := lab.NewStateStore(dir)

	labs, err := store.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(labs) != 0 {
		t.Errorf("List returned %d labs, want 0", len(labs))
	}
}

func TestSaveCreatesDirectory(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nested", "state")
	store := lab.NewStateStore(dir)

	err := store.Save(&lab.Metadata{ID: "test", DisplayName: "test"})
	if err != nil {
		t.Fatalf("Save to nested dir: %v", err)
	}

	if _, err := os.Stat(dir); os.IsNotExist(err) {
		t.Error("Save should create state directory")
	}
}

func TestStartConfigRoundTrip(t *testing.T) {
	dir := t.TempDir()
	store := lab.NewStateStore(dir)

	envFilePath := "/home/user/config/vars"
	meta := &lab.Metadata{
		ID:          "cfg-001",
		DisplayName: "myapp-zsh",
		Project:     "/home/user/code/myapp",
		ProjectName: "myapp",
		Profile:     "zsh",
		BareRepo:    "/home/user/.claudeup-lab/repos/myapp.git",
		Worktree:    "/home/user/.claudeup-lab/workspaces/myapp-zsh",
		Branch:      "lab/zsh",
		Created:     time.Date(2026, 3, 19, 12, 0, 0, 0, time.UTC),
		StartConfig: &lab.StartConfig{
			Project:     "/home/user/code/myapp",
			Profile:     "zsh",
			Branch:      "lab/zsh",
			LabName:     "myapp-zsh",
			Features:    []string{"ghcr.io/devcontainers/features/node:1"},
			BaseProfile: "base",
			Firewall:    true,
			InitScript:  "/home/user/init.sh",
			EnvVars:     map[string]string{"FOO": "bar", "BAZ": "qux"},
			EnvFile:     envFilePath,
		},
	}

	if err := store.Save(meta); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := store.Load("cfg-001")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if loaded.StartConfig == nil {
		t.Fatal("StartConfig is nil after round-trip")
	}

	sc := loaded.StartConfig
	if sc.Project != "/home/user/code/myapp" {
		t.Errorf("Project = %q, want %q", sc.Project, "/home/user/code/myapp")
	}
	if sc.Profile != "zsh" {
		t.Errorf("Profile = %q, want %q", sc.Profile, "zsh")
	}
	if sc.Branch != "lab/zsh" {
		t.Errorf("Branch = %q, want %q", sc.Branch, "lab/zsh")
	}
	if sc.LabName != "myapp-zsh" {
		t.Errorf("LabName = %q, want %q", sc.LabName, "myapp-zsh")
	}
	if len(sc.Features) != 1 || sc.Features[0] != "ghcr.io/devcontainers/features/node:1" {
		t.Errorf("Features = %v, want [ghcr.io/devcontainers/features/node:1]", sc.Features)
	}
	if sc.BaseProfile != "base" {
		t.Errorf("BaseProfile = %q, want %q", sc.BaseProfile, "base")
	}
	if !sc.Firewall {
		t.Error("Firewall = false, want true")
	}
	if sc.InitScript != "/home/user/init.sh" {
		t.Errorf("InitScript = %q, want %q", sc.InitScript, "/home/user/init.sh")
	}
	if sc.EnvVars["FOO"] != "bar" || sc.EnvVars["BAZ"] != "qux" {
		t.Errorf("EnvVars = %v, want map[FOO:bar BAZ:qux]", sc.EnvVars)
	}
	if sc.EnvFile != envFilePath {
		t.Errorf("EnvFile = %q, want %q", sc.EnvFile, envFilePath)
	}
}

func TestStartConfigNilWhenUnset(t *testing.T) {
	dir := t.TempDir()
	store := lab.NewStateStore(dir)

	meta := &lab.Metadata{
		ID:          "no-cfg",
		DisplayName: "plain-lab",
		Project:     "/tmp/project",
		ProjectName: "project",
		Profile:     "default",
		Branch:      "lab/default",
		Created:     time.Date(2026, 3, 19, 12, 0, 0, 0, time.UTC),
	}

	if err := store.Save(meta); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := store.Load("no-cfg")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if loaded.StartConfig != nil {
		t.Errorf("StartConfig = %+v, want nil", loaded.StartConfig)
	}
}

func TestStartConfigBackwardCompatibility(t *testing.T) {
	dir := t.TempDir()

	// Simulate a pre-feature state file with no start_config key
	rawJSON := `{
  "id": "legacy-001",
  "display_name": "old-lab",
  "project": "/home/user/code/old",
  "project_name": "old",
  "profile": "base",
  "bare_repo": "/home/user/.claudeup-lab/repos/old.git",
  "worktree": "/home/user/.claudeup-lab/workspaces/old-lab",
  "branch": "lab/base",
  "created": "2026-01-01T00:00:00Z"
}`
	if err := os.WriteFile(filepath.Join(dir, "legacy-001.json"), []byte(rawJSON), 0o644); err != nil {
		t.Fatalf("write raw JSON: %v", err)
	}

	store := lab.NewStateStore(dir)
	loaded, err := store.Load("legacy-001")
	if err != nil {
		t.Fatalf("Load legacy state: %v", err)
	}

	if loaded.StartConfig != nil {
		t.Errorf("StartConfig = %+v, want nil for legacy state file", loaded.StartConfig)
	}
	if loaded.DisplayName != "old-lab" {
		t.Errorf("DisplayName = %q, want %q", loaded.DisplayName, "old-lab")
	}
	if loaded.Branch != "lab/base" {
		t.Errorf("Branch = %q, want %q", loaded.Branch, "lab/base")
	}
}

func TestStartConfigFromResolved(t *testing.T) {
	envFilePath := "/home/user/config/local-vars"
	opts := &lab.StartOptions{
		Project:     "./relative/path", // raw, should be ignored
		Profile:     "",                // raw, should be ignored
		Branch:      "",                // raw, should be ignored
		Name:        "custom-name",
		Features:    []string{"feat-a", "feat-b"},
		BaseProfile: "minimal",
		Firewall:    true,
		InitScript:  "/scripts/init.sh",
		EnvVars:     map[string]string{"KEY": "val"},
		EnvFile:     envFilePath,
	}

	cfg := lab.StartConfigFromResolved("/abs/path/to/project", "zsh", "lab/zsh", opts)

	// Resolved params take priority over opts fields
	if cfg.Project != "/abs/path/to/project" {
		t.Errorf("Project = %q, want %q", cfg.Project, "/abs/path/to/project")
	}
	if cfg.Profile != "zsh" {
		t.Errorf("Profile = %q, want %q", cfg.Profile, "zsh")
	}
	if cfg.Branch != "lab/zsh" {
		t.Errorf("Branch = %q, want %q", cfg.Branch, "lab/zsh")
	}

	// Remaining fields from opts
	if cfg.LabName != "custom-name" {
		t.Errorf("LabName = %q, want %q", cfg.LabName, "custom-name")
	}
	if len(cfg.Features) != 2 || cfg.Features[0] != "feat-a" || cfg.Features[1] != "feat-b" {
		t.Errorf("Features = %v, want [feat-a feat-b]", cfg.Features)
	}
	if cfg.BaseProfile != "minimal" {
		t.Errorf("BaseProfile = %q, want %q", cfg.BaseProfile, "minimal")
	}
	if !cfg.Firewall {
		t.Error("Firewall = false, want true")
	}
	if cfg.InitScript != "/scripts/init.sh" {
		t.Errorf("InitScript = %q, want %q", cfg.InitScript, "/scripts/init.sh")
	}
	if cfg.EnvVars["KEY"] != "val" {
		t.Errorf("EnvVars = %v, want map[KEY:val]", cfg.EnvVars)
	}
	if cfg.EnvFile != envFilePath {
		t.Errorf("EnvFile = %q, want %q", cfg.EnvFile, envFilePath)
	}
}

func TestStartConfigFromResolvedOmitsEmptyOptionals(t *testing.T) {
	opts := &lab.StartOptions{
		Project: ".",
		Profile: "default",
		Branch:  "main",
		Name:    "",
	}

	cfg := lab.StartConfigFromResolved("/abs/project", "default", "lab/default", opts)

	if cfg.LabName != "" {
		t.Errorf("LabName = %q, want empty", cfg.LabName)
	}
	if cfg.Features != nil {
		t.Errorf("Features = %v, want nil", cfg.Features)
	}
	if cfg.BaseProfile != "" {
		t.Errorf("BaseProfile = %q, want empty", cfg.BaseProfile)
	}
	if cfg.Firewall {
		t.Error("Firewall = true, want false")
	}
	if cfg.InitScript != "" {
		t.Errorf("InitScript = %q, want empty", cfg.InitScript)
	}
	if cfg.EnvVars != nil {
		t.Errorf("EnvVars = %v, want nil", cfg.EnvVars)
	}
	if cfg.EnvFile != "" {
		t.Errorf("EnvFile = %q, want empty", cfg.EnvFile)
	}
}
