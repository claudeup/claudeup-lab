package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/claudeup/claudeup-lab/internal/lab"
)

// setupPresetTestEnv creates a temp base dir with state and presets subdirs,
// returning the base dir and a Manager rooted there.
func setupPresetTestEnv(t *testing.T) (string, *lab.Manager) {
	t.Helper()
	base := t.TempDir()
	t.Cleanup(setTestBaseDir(base))
	mgr := lab.NewManager(base)
	return base, mgr
}

func TestPresetSave_FromExplicitFlags(t *testing.T) {
	base, _ := setupPresetTestEnv(t)

	cmd := NewRootCmd()
	cmd.SetArgs([]string{
		"preset", "save", "test-preset",
		"--project", "/tmp/proj",
		"--profile", "zsh",
		"--branch", "lab/zsh",
	})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	store := lab.NewPresetStore(filepath.Join(base, "presets"))
	p, err := store.Load("test-preset")
	if err != nil {
		t.Fatalf("failed to load saved preset: %v", err)
	}
	if p.Project != "/tmp/proj" {
		t.Errorf("project = %q, want /tmp/proj", p.Project)
	}
	if p.Profile != "zsh" {
		t.Errorf("profile = %q, want zsh", p.Profile)
	}
	if p.Branch != "lab/zsh" {
		t.Errorf("branch = %q, want lab/zsh", p.Branch)
	}
}

func TestPresetSave_FromLab(t *testing.T) {
	_, mgr := setupPresetTestEnv(t)

	meta := &lab.Metadata{
		ID:          "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee",
		DisplayName: "mylab",
		Project:     "/tmp/proj",
		ProjectName: "proj",
		Profile:     "zsh",
		Branch:      "lab/zsh",
		StartConfig: &lab.StartConfig{
			Project:     "/tmp/proj",
			Profile:     "zsh",
			Branch:      "lab/zsh",
			BaseProfile: "base",
			Firewall:    true,
		},
	}
	if err := mgr.Store().Save(meta); err != nil {
		t.Fatalf("failed to save lab state: %v", err)
	}

	cmd := NewRootCmd()
	cmd.SetArgs([]string{
		"preset", "save", "from-lab-preset",
		"--from-lab", "mylab",
	})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	store := mgr.Presets()
	p, err := store.Load("from-lab-preset")
	if err != nil {
		t.Fatalf("failed to load saved preset: %v", err)
	}
	if p.Project != "/tmp/proj" {
		t.Errorf("project = %q, want /tmp/proj", p.Project)
	}
	if p.Profile != "zsh" {
		t.Errorf("profile = %q, want zsh", p.Profile)
	}
	if p.Branch != "lab/zsh" {
		t.Errorf("branch = %q, want lab/zsh", p.Branch)
	}
	if p.BaseProfile != "base" {
		t.Errorf("base-profile = %q, want base", p.BaseProfile)
	}
	if !p.Firewall {
		t.Error("firewall = false, want true")
	}
}

func TestPresetSave_FromLabNotFound(t *testing.T) {
	setupPresetTestEnv(t)

	cmd := NewRootCmd()
	cmd.SetArgs([]string{
		"preset", "save", "test-preset",
		"--from-lab", "nonexistent",
	})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for nonexistent lab")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error %q should contain 'not found'", err.Error())
	}
}

func TestPresetSave_FromLabNoStartConfig(t *testing.T) {
	_, mgr := setupPresetTestEnv(t)

	meta := &lab.Metadata{
		ID:          "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee",
		DisplayName: "oldlab",
		Project:     "/tmp/proj",
		ProjectName: "proj",
		Profile:     "zsh",
		Branch:      "lab/zsh",
	}
	if err := mgr.Store().Save(meta); err != nil {
		t.Fatalf("failed to save lab state: %v", err)
	}

	cmd := NewRootCmd()
	cmd.SetArgs([]string{
		"preset", "save", "test-preset",
		"--from-lab", "oldlab",
	})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for lab with no StartConfig")
	}
	if !strings.Contains(err.Error(), "has no tracked configuration") {
		t.Errorf("error %q should contain 'has no tracked configuration'", err.Error())
	}
}

func TestPresetSave_FromLabCombinedWithFlags(t *testing.T) {
	setupPresetTestEnv(t)

	cmd := NewRootCmd()
	cmd.SetArgs([]string{
		"preset", "save", "test-preset",
		"--from-lab", "mylab",
		"--profile", "zsh",
	})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error when combining --from-lab with other flags")
	}
	if !strings.Contains(err.Error(), "--from-lab cannot be combined with other flags") {
		t.Errorf("error %q should contain '--from-lab cannot be combined with other flags'", err.Error())
	}
}

func TestPresetSave_NoFlagsProvided(t *testing.T) {
	setupPresetTestEnv(t)

	cmd := NewRootCmd()
	cmd.SetArgs([]string{"preset", "save", "test-preset"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error when no flags are provided")
	}
	if !strings.Contains(err.Error(), "provide --from-lab or at least one start flag") {
		t.Errorf("error %q should contain 'provide --from-lab or at least one start flag'", err.Error())
	}
}

func TestPresetSave_InvalidName(t *testing.T) {
	setupPresetTestEnv(t)

	cmd := NewRootCmd()
	cmd.SetArgs([]string{
		"preset", "save", "has spaces",
		"--profile", "zsh",
	})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for invalid name")
	}
	if !strings.Contains(err.Error(), "invalid characters") {
		t.Errorf("error %q should contain 'invalid characters'", err.Error())
	}
}

func TestPresetSave_PathsResolvedToAbsolute(t *testing.T) {
	base, _ := setupPresetTestEnv(t)

	projDir := t.TempDir()

	cmd := NewRootCmd()
	cmd.SetArgs([]string{
		"preset", "save", "path-test",
		"--project", projDir,
		"--profile", "zsh",
	})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	store := lab.NewPresetStore(filepath.Join(base, "presets"))
	p, err := store.Load("path-test")
	if err != nil {
		t.Fatalf("failed to load preset: %v", err)
	}
	if !filepath.IsAbs(p.Project) {
		t.Errorf("project path %q should be absolute", p.Project)
	}
}

func TestPresetSave_InitScriptValidation(t *testing.T) {
	setupPresetTestEnv(t)

	cmd := NewRootCmd()
	cmd.SetArgs([]string{
		"preset", "save", "init-test",
		"--init-script", "/nonexistent/script.sh",
	})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for nonexistent init script")
	}
	if !strings.Contains(err.Error(), "init script not found") {
		t.Errorf("error %q should contain 'init script not found'", err.Error())
	}
}

func TestPresetSave_OverwriteWithForce(t *testing.T) {
	base, _ := setupPresetTestEnv(t)

	// Save initial preset
	cmd1 := NewRootCmd()
	cmd1.SetArgs([]string{
		"preset", "save", "overwrite-test",
		"--project", "/tmp/proj1",
		"--profile", "old-profile",
	})
	if err := cmd1.Execute(); err != nil {
		t.Fatalf("first save failed: %v", err)
	}

	// Overwrite with --force
	cmd2 := NewRootCmd()
	cmd2.SetArgs([]string{
		"preset", "save", "overwrite-test",
		"--project", "/tmp/proj2",
		"--profile", "new-profile",
		"--force",
	})
	if err := cmd2.Execute(); err != nil {
		t.Fatalf("force overwrite failed: %v", err)
	}

	store := lab.NewPresetStore(filepath.Join(base, "presets"))
	p, err := store.Load("overwrite-test")
	if err != nil {
		t.Fatalf("failed to load preset: %v", err)
	}
	if p.Project != "/tmp/proj2" {
		t.Errorf("project = %q, want /tmp/proj2", p.Project)
	}
	if p.Profile != "new-profile" {
		t.Errorf("profile = %q, want new-profile", p.Profile)
	}
}

func TestPresetSave_MissingName(t *testing.T) {
	setupPresetTestEnv(t)

	cmd := NewRootCmd()
	cmd.SetArgs([]string{"preset", "save"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for missing name")
	}
	if !strings.Contains(err.Error(), "accepts 1 arg(s), received 0") {
		t.Errorf("error %q should contain cobra's arg count message", err.Error())
	}
}

func TestPresetSave_SuccessOutput(t *testing.T) {
	setupPresetTestEnv(t)

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	cmd := NewRootCmd()
	cmd.SetArgs([]string{
		"preset", "save", "output-test",
		"--project", "/tmp/proj",
		"--profile", "zsh",
		"--branch", "lab/zsh",
	})
	err := cmd.Execute()

	w.Close()
	os.Stdout = old

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	buf := make([]byte, 4096)
	n, _ := r.Read(buf)
	output := string(buf[:n])

	if !strings.Contains(output, "Preset saved: output-test") {
		t.Errorf("output should contain 'Preset saved: output-test', got:\n%s", output)
	}
	if !strings.Contains(output, "project:") {
		t.Errorf("output should contain 'project:', got:\n%s", output)
	}
	if !strings.Contains(output, "/tmp/proj") {
		t.Errorf("output should contain '/tmp/proj', got:\n%s", output)
	}
}

func TestPresetSave_EnvFlags(t *testing.T) {
	base, _ := setupPresetTestEnv(t)

	cmd := NewRootCmd()
	cmd.SetArgs([]string{
		"preset", "save", "env-test",
		"--profile", "zsh",
		"--env", "FOO=bar",
		"--env", "BAZ=qux",
	})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	store := lab.NewPresetStore(filepath.Join(base, "presets"))
	p, err := store.Load("env-test")
	if err != nil {
		t.Fatalf("failed to load preset: %v", err)
	}
	if p.EnvVars["FOO"] != "bar" {
		t.Errorf("env FOO = %q, want bar", p.EnvVars["FOO"])
	}
	if p.EnvVars["BAZ"] != "qux" {
		t.Errorf("env BAZ = %q, want qux", p.EnvVars["BAZ"])
	}
}

func TestPresetSave_AllStartFlags(t *testing.T) {
	base, _ := setupPresetTestEnv(t)

	initScript := filepath.Join(t.TempDir(), "init.sh")
	if err := os.WriteFile(initScript, []byte("#!/bin/bash\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	cmd := NewRootCmd()
	cmd.SetArgs([]string{
		"preset", "save", "full-test",
		"--project", "/tmp/proj",
		"--profile", "zsh",
		"--branch", "lab/zsh",
		"--base-profile", "base",
		"--firewall",
		"--init-script", initScript,
		"--feature", "go:1.23",
		"--feature", "node:20",
		"--env", "FOO=bar",
	})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	store := lab.NewPresetStore(filepath.Join(base, "presets"))
	p, err := store.Load("full-test")
	if err != nil {
		t.Fatalf("failed to load preset: %v", err)
	}
	if p.Project != "/tmp/proj" {
		t.Errorf("project = %q", p.Project)
	}
	if p.Profile != "zsh" {
		t.Errorf("profile = %q", p.Profile)
	}
	if p.Branch != "lab/zsh" {
		t.Errorf("branch = %q", p.Branch)
	}
	if p.BaseProfile != "base" {
		t.Errorf("base-profile = %q", p.BaseProfile)
	}
	if !p.Firewall {
		t.Error("firewall = false")
	}
	if p.InitScript != initScript {
		t.Errorf("init-script = %q", p.InitScript)
	}
	if len(p.Features) != 2 || p.Features[0] != "go:1.23" || p.Features[1] != "node:20" {
		t.Errorf("features = %v", p.Features)
	}
	if p.EnvVars["FOO"] != "bar" {
		t.Errorf("env FOO = %q", p.EnvVars["FOO"])
	}
}
