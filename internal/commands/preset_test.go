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

func TestPresetDelete_Force(t *testing.T) {
	base, mgr := setupPresetTestEnv(t)

	// Save a preset first
	p := &lab.Preset{Name: "del-test", Project: "/tmp/proj", Profile: "zsh"}
	if err := mgr.Presets().Save(p); err != nil {
		t.Fatalf("failed to save preset: %v", err)
	}

	cmd := NewRootCmd()
	cmd.SetArgs([]string{"preset", "delete", "del-test", "--force"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify preset is gone
	store := lab.NewPresetStore(filepath.Join(base, "presets"))
	_, err = store.Load("del-test")
	if err == nil {
		t.Fatal("preset should have been deleted")
	}
}

func TestPresetDelete_NotFound(t *testing.T) {
	setupPresetTestEnv(t)

	cmd := NewRootCmd()
	cmd.SetArgs([]string{"preset", "delete", "nonexistent", "--force"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for nonexistent preset")
	}
	if !strings.Contains(err.Error(), "preset 'nonexistent' not found") {
		t.Errorf("error %q should contain \"preset 'nonexistent' not found\"", err.Error())
	}
}

func TestPresetDelete_AliasRm(t *testing.T) {
	base, mgr := setupPresetTestEnv(t)

	p := &lab.Preset{Name: "alias-test", Project: "/tmp/proj", Profile: "zsh"}
	if err := mgr.Presets().Save(p); err != nil {
		t.Fatalf("failed to save preset: %v", err)
	}

	cmd := NewRootCmd()
	cmd.SetArgs([]string{"preset", "rm", "alias-test", "--force"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	store := lab.NewPresetStore(filepath.Join(base, "presets"))
	_, err = store.Load("alias-test")
	if err == nil {
		t.Fatal("preset should have been deleted via rm alias")
	}
}

func TestPresetDelete_TooManyArgs(t *testing.T) {
	setupPresetTestEnv(t)

	cmd := NewRootCmd()
	cmd.SetArgs([]string{"preset", "delete", "foo", "bar"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for too many args")
	}
	if !strings.Contains(err.Error(), "accepts 1 arg(s), received 2") {
		t.Errorf("error %q should contain cobra's arg count message", err.Error())
	}
}

func TestPresetDelete_SuccessOutput(t *testing.T) {
	_, mgr := setupPresetTestEnv(t)

	p := &lab.Preset{Name: "output-del", Project: "/tmp/proj", Profile: "zsh"}
	if err := mgr.Presets().Save(p); err != nil {
		t.Fatalf("failed to save preset: %v", err)
	}

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	cmd := NewRootCmd()
	cmd.SetArgs([]string{"preset", "delete", "output-del", "--force"})
	err := cmd.Execute()

	w.Close()
	os.Stdout = old

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	buf := make([]byte, 4096)
	n, _ := r.Read(buf)
	output := string(buf[:n])

	if !strings.Contains(output, "Deleted preset: output-del") {
		t.Errorf("output should contain 'Deleted preset: output-del', got:\n%s", output)
	}
}

// captureStdout runs fn while capturing stdout, returning the captured output.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	fn()

	w.Close()
	os.Stdout = old

	buf := make([]byte, 8192)
	n, _ := r.Read(buf)
	return string(buf[:n])
}

// savePresetDirect saves a preset directly via the store for test setup.
func savePresetDirect(t *testing.T, mgr *lab.Manager, p *lab.Preset) {
	t.Helper()
	if err := mgr.Presets().Save(p); err != nil {
		t.Fatalf("save preset %q: %v", p.Name, err)
	}
}

func TestPresetList_WithPresets(t *testing.T) {
	_, mgr := setupPresetTestEnv(t)

	savePresetDirect(t, mgr, &lab.Preset{
		Name:        "my-gstack",
		Project:     "/Users/mark/code/gstack",
		Profile:     "zsh",
		Firewall:    true,
		BaseProfile: "base",
	})
	savePresetDirect(t, mgr, &lab.Preset{
		Name:    "quick-test",
		Project: "/Users/mark/code/myapp",
		Profile: "default",
		EnvFile: "/Users/mark/code/myapp/lab.env",
	})

	output := captureStdout(t, func() {
		cmd := NewRootCmd()
		cmd.SetArgs([]string{"preset", "list"})
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	// Check header
	if !strings.Contains(output, "NAME") || !strings.Contains(output, "PROJECT") ||
		!strings.Contains(output, "PROFILE") || !strings.Contains(output, "FLAGS") {
		t.Errorf("missing header columns in output:\n%s", output)
	}
	// Check separator
	if !strings.Contains(output, "----") {
		t.Errorf("missing separator line in output:\n%s", output)
	}
	// Check preset rows
	if !strings.Contains(output, "my-gstack") || !strings.Contains(output, "gstack") || !strings.Contains(output, "zsh") {
		t.Errorf("missing my-gstack row in output:\n%s", output)
	}
	if !strings.Contains(output, "quick-test") || !strings.Contains(output, "myapp") {
		t.Errorf("missing quick-test row in output:\n%s", output)
	}
}

func TestPresetList_NoPresets(t *testing.T) {
	setupPresetTestEnv(t)

	output := captureStdout(t, func() {
		cmd := NewRootCmd()
		cmd.SetArgs([]string{"preset", "list"})
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	expected := "No presets found."
	if !strings.Contains(output, expected) {
		t.Errorf("output = %q, want %q", output, expected)
	}
}

func TestPresetList_Alias(t *testing.T) {
	setupPresetTestEnv(t)

	output := captureStdout(t, func() {
		cmd := NewRootCmd()
		cmd.SetArgs([]string{"preset", "ls"})
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(output, "No presets found.") {
		t.Errorf("alias 'ls' should work, got output:\n%s", output)
	}
}

func TestPresetList_FlagsBooleanFlag(t *testing.T) {
	_, mgr := setupPresetTestEnv(t)

	savePresetDirect(t, mgr, &lab.Preset{
		Name:     "fw-test",
		Project:  "/tmp/proj",
		Profile:  "zsh",
		Firewall: true,
	})

	output := captureStdout(t, func() {
		cmd := NewRootCmd()
		cmd.SetArgs([]string{"preset", "list"})
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(output, "--firewall") {
		t.Errorf("FLAGS should contain --firewall, got:\n%s", output)
	}
}

func TestPresetList_FlagsStringPathFlag(t *testing.T) {
	_, mgr := setupPresetTestEnv(t)

	savePresetDirect(t, mgr, &lab.Preset{
		Name:       "script-test",
		Project:    "/tmp/proj",
		Profile:    "zsh",
		InitScript: "/Users/mark/scripts/setup.sh",
	})

	output := captureStdout(t, func() {
		cmd := NewRootCmd()
		cmd.SetArgs([]string{"preset", "list"})
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(output, "--init-script=setup.sh") {
		t.Errorf("FLAGS should contain --init-script=setup.sh (basename), got:\n%s", output)
	}
}

func TestPresetList_FlagsEnvKeysOnly(t *testing.T) {
	_, mgr := setupPresetTestEnv(t)

	savePresetDirect(t, mgr, &lab.Preset{
		Name:    "env-list-test",
		Project: "/tmp/proj",
		Profile: "zsh",
		EnvVars: map[string]string{"API_KEY": "secret", "DEBUG": "true"},
	})

	output := captureStdout(t, func() {
		cmd := NewRootCmd()
		cmd.SetArgs([]string{"preset", "list"})
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(output, "--env=API_KEY,DEBUG") {
		t.Errorf("FLAGS should contain --env=API_KEY,DEBUG (sorted keys only), got:\n%s", output)
	}
	// Must NOT contain the secret value
	if strings.Contains(output, "secret") {
		t.Errorf("FLAGS must not contain env values, got:\n%s", output)
	}
}

func TestPresetList_FlagsNone(t *testing.T) {
	_, mgr := setupPresetTestEnv(t)

	savePresetDirect(t, mgr, &lab.Preset{
		Name:    "bare",
		Project: "/tmp/proj",
		Profile: "zsh",
	})

	output := captureStdout(t, func() {
		cmd := NewRootCmd()
		cmd.SetArgs([]string{"preset", "list"})
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	// The FLAGS column should show (none)
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, "bare") {
			if !strings.Contains(line, "(none)") {
				t.Errorf("FLAGS for bare preset should be (none), got line:\n%s", line)
			}
			break
		}
	}
}

func TestPresetList_ProfileNone(t *testing.T) {
	_, mgr := setupPresetTestEnv(t)

	savePresetDirect(t, mgr, &lab.Preset{
		Name:    "no-profile",
		Project: "/tmp/proj",
	})

	output := captureStdout(t, func() {
		cmd := NewRootCmd()
		cmd.SetArgs([]string{"preset", "list"})
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, "no-profile") {
			if !strings.Contains(line, "(none)") {
				t.Errorf("PROFILE for no-profile preset should be (none), got line:\n%s", line)
			}
			break
		}
	}
}

func TestPresetShow_FullDetails(t *testing.T) {
	_, mgr := setupPresetTestEnv(t)

	savePresetDirect(t, mgr, &lab.Preset{
		Name:        "my-gstack",
		Project:     "/Users/mark/code/gstack",
		Profile:     "zsh",
		Branch:      "lab/zsh",
		BaseProfile: "base",
		Features:    []string{"go:1.23", "node:20"},
		Firewall:    true,
		InitScript:  "/Users/mark/scripts/setup.sh",
		EnvVars:     map[string]string{"API_KEY": "sk-1234abcd", "DEBUG": "true"},
		EnvFile:     "/Users/mark/code/gstack/lab.env",
	})

	output := captureStdout(t, func() {
		cmd := NewRootCmd()
		cmd.SetArgs([]string{"preset", "show", "my-gstack"})
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	// Header
	if !strings.Contains(output, "Preset: my-gstack") {
		t.Errorf("missing 'Preset: my-gstack' header, got:\n%s", output)
	}
	// Full paths
	if !strings.Contains(output, "/Users/mark/code/gstack") {
		t.Errorf("missing full project path, got:\n%s", output)
	}
	if !strings.Contains(output, "/Users/mark/scripts/setup.sh") {
		t.Errorf("missing full init-script path, got:\n%s", output)
	}
	// Env values shown (not hidden)
	if !strings.Contains(output, "API_KEY=sk-1234abcd") {
		t.Errorf("missing env var value, got:\n%s", output)
	}
	// Profile
	if !strings.Contains(output, "zsh") {
		t.Errorf("missing profile, got:\n%s", output)
	}
	// Firewall
	if !strings.Contains(output, "true") {
		t.Errorf("missing firewall=true, got:\n%s", output)
	}
}

func TestPresetShow_RepeatedValues(t *testing.T) {
	_, mgr := setupPresetTestEnv(t)

	savePresetDirect(t, mgr, &lab.Preset{
		Name:     "multi-test",
		Project:  "/tmp/proj",
		Profile:  "zsh",
		Features: []string{"go:1.23", "node:20"},
		EnvVars:  map[string]string{"A": "1", "B": "2"},
	})

	output := captureStdout(t, func() {
		cmd := NewRootCmd()
		cmd.SetArgs([]string{"preset", "show", "multi-test"})
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	// Each feature should get its own line
	if strings.Count(output, "go:1.23") < 1 {
		t.Errorf("missing feature go:1.23, got:\n%s", output)
	}
	if strings.Count(output, "node:20") < 1 {
		t.Errorf("missing feature node:20, got:\n%s", output)
	}
	// Each env var on its own line
	if strings.Count(output, "A=1") < 1 {
		t.Errorf("missing env A=1, got:\n%s", output)
	}
	if strings.Count(output, "B=2") < 1 {
		t.Errorf("missing env B=2, got:\n%s", output)
	}
}

func TestPresetShow_Nonexistent(t *testing.T) {
	setupPresetTestEnv(t)

	cmd := NewRootCmd()
	cmd.SetArgs([]string{"preset", "show", "nonexistent"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for nonexistent preset")
	}
	if !strings.Contains(err.Error(), "preset 'nonexistent' not found") {
		t.Errorf("error = %q, want 'preset 'nonexistent' not found'", err.Error())
	}
}

func TestPresetShow_OmitsEmptyFields(t *testing.T) {
	_, mgr := setupPresetTestEnv(t)

	savePresetDirect(t, mgr, &lab.Preset{
		Name:    "minimal",
		Project: "/tmp/proj",
		Profile: "zsh",
	})

	output := captureStdout(t, func() {
		cmd := NewRootCmd()
		cmd.SetArgs([]string{"preset", "show", "minimal"})
		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if strings.Contains(output, "branch:") {
		t.Errorf("should not show empty branch, got:\n%s", output)
	}
	if strings.Contains(output, "firewall:") {
		t.Errorf("should not show false firewall, got:\n%s", output)
	}
	if strings.Contains(output, "init-script:") {
		t.Errorf("should not show empty init-script, got:\n%s", output)
	}
	if strings.Contains(output, "env-vars:") {
		t.Errorf("should not show empty env-vars, got:\n%s", output)
	}
	if strings.Contains(output, "env-file:") {
		t.Errorf("should not show empty env-file, got:\n%s", output)
	}
}
