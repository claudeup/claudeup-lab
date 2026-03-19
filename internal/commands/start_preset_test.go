package commands

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/claudeup/claudeup-lab/internal/lab"
	"github.com/spf13/cobra"
)

// saveTestPreset creates a preset in the given base directory's preset store.
func saveTestPreset(t *testing.T, base string, p *lab.Preset) {
	t.Helper()
	store := lab.NewPresetStore(filepath.Join(base, "presets"))
	if err := store.Save(p); err != nil {
		t.Fatalf("save test preset: %v", err)
	}
}

func TestStartWithPreset_NotFound(t *testing.T) {
	base := t.TempDir()
	t.Cleanup(setTestBaseDir(base))

	cmd := NewRootCmd()
	cmd.SetArgs([]string{"start", "nonexistent-preset"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for nonexistent preset")
	}
	if !strings.Contains(err.Error(), "preset 'nonexistent-preset' not found") {
		t.Errorf("error %q should contain \"preset 'nonexistent-preset' not found\"", err.Error())
	}
}

func TestStartWithPreset_LoadsPresetAndPrintsSummary(t *testing.T) {
	base := t.TempDir()
	t.Cleanup(setTestBaseDir(base))

	saveTestPreset(t, base, &lab.Preset{
		Name:        "my-gstack",
		Project:     "/tmp/gstack",
		Profile:     "zsh",
		Branch:      "lab/zsh",
		BaseProfile: "base",
		Firewall:    true,
	})

	// Capture stderr
	var stderr bytes.Buffer
	cmd := NewRootCmd()
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"start", "my-gstack"})

	// Will fail at mgr.Start() (no Docker), but stderr should have preset info
	_ = cmd.Execute()

	output := stderr.String()
	if !strings.Contains(output, "Using preset 'my-gstack'") {
		t.Errorf("stderr should contain \"Using preset 'my-gstack'\", got:\n%s", output)
	}
	if !strings.Contains(output, "project:") || !strings.Contains(output, "/tmp/gstack") {
		t.Errorf("stderr should contain project field, got:\n%s", output)
	}
	if !strings.Contains(output, "profile:") || !strings.Contains(output, "zsh") {
		t.Errorf("stderr should contain profile field, got:\n%s", output)
	}
	if !strings.Contains(output, "branch:") || !strings.Contains(output, "lab/zsh") {
		t.Errorf("stderr should contain branch field, got:\n%s", output)
	}
	if !strings.Contains(output, "base-profile:") || !strings.Contains(output, "base") {
		t.Errorf("stderr should contain base-profile field, got:\n%s", output)
	}
	if !strings.Contains(output, "firewall:") || !strings.Contains(output, "true") {
		t.Errorf("stderr should contain firewall field, got:\n%s", output)
	}
}

func TestStartWithPreset_OverrideBranchShowsDiff(t *testing.T) {
	base := t.TempDir()
	t.Cleanup(setTestBaseDir(base))

	saveTestPreset(t, base, &lab.Preset{
		Name:    "my-gstack",
		Project: "/tmp/gstack",
		Profile: "zsh",
		Branch:  "lab/zsh",
	})

	var stderr bytes.Buffer
	cmd := NewRootCmd()
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"start", "my-gstack", "--branch", "lab/experiment"})

	_ = cmd.Execute()

	output := stderr.String()
	if !strings.Contains(output, "Using preset 'my-gstack'") {
		t.Errorf("stderr should contain preset header, got:\n%s", output)
	}
	// Override diff markers
	if !strings.Contains(output, "- branch:") {
		t.Errorf("stderr should contain '- branch:' for removed value, got:\n%s", output)
	}
	if !strings.Contains(output, "lab/zsh") {
		t.Errorf("stderr should contain old branch value 'lab/zsh', got:\n%s", output)
	}
	if !strings.Contains(output, "+ branch:") {
		t.Errorf("stderr should contain '+ branch:' for new value, got:\n%s", output)
	}
	if !strings.Contains(output, "lab/experiment") {
		t.Errorf("stderr should contain new branch value 'lab/experiment', got:\n%s", output)
	}
	// Non-overridden fields should NOT have diff markers
	if strings.Contains(output, "- project:") || strings.Contains(output, "+ project:") {
		t.Errorf("project should not have diff markers, got:\n%s", output)
	}
}

func TestStartWithPreset_NoOverridesShowsPlainFields(t *testing.T) {
	base := t.TempDir()
	t.Cleanup(setTestBaseDir(base))

	saveTestPreset(t, base, &lab.Preset{
		Name:    "simple",
		Project: "/tmp/proj",
		Profile: "zsh",
	})

	var stderr bytes.Buffer
	cmd := NewRootCmd()
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"start", "simple"})

	_ = cmd.Execute()

	output := stderr.String()
	if !strings.Contains(output, "Using preset 'simple'") {
		t.Errorf("stderr should contain preset header, got:\n%s", output)
	}
	// No diff markers when there are no overrides
	if strings.Contains(output, "- ") || strings.Contains(output, "+ ") {
		t.Errorf("no diff markers expected without overrides, got:\n%s", output)
	}
}

func TestStartWithPreset_ProjectFromPresetNotCwd(t *testing.T) {
	base := t.TempDir()
	t.Cleanup(setTestBaseDir(base))

	saveTestPreset(t, base, &lab.Preset{
		Name:    "proj-test",
		Project: "/tmp/specific-project",
		Profile: "zsh",
	})

	var stderr bytes.Buffer
	cmd := NewRootCmd()
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"start", "proj-test"})

	// The command will fail at mgr.Start, but the error should reference
	// the preset's project, not cwd.
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error (no Docker)")
	}

	// Check stderr has the preset's project, not cwd
	output := stderr.String()
	if !strings.Contains(output, "/tmp/specific-project") {
		t.Errorf("stderr should show preset project '/tmp/specific-project', got:\n%s", output)
	}
}

func TestStartWithPreset_UsageLineShowsOptionalPresetArg(t *testing.T) {
	cmd := NewRootCmd()

	// Find the start subcommand
	var startCmd *cobra.Command
	for _, c := range cmd.Commands() {
		if c.Name() == "start" {
			startCmd = c
			break
		}
	}
	if startCmd == nil {
		t.Fatal("start command not found")
	}

	if !strings.Contains(startCmd.Use, "[preset-name]") {
		t.Errorf("start command Use should contain '[preset-name]', got %q", startCmd.Use)
	}
}

func TestStartWithoutPreset_NoChangeToExistingBehavior(t *testing.T) {
	base := t.TempDir()
	t.Cleanup(setTestBaseDir(base))

	var stderr bytes.Buffer
	cmd := NewRootCmd()
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"start", "--project", "/tmp/proj", "--profile", "zsh"})

	// Will fail at mgr.Start (no Docker), but should NOT print preset info
	_ = cmd.Execute()

	output := stderr.String()
	if strings.Contains(output, "Using preset") {
		t.Errorf("without preset arg, should not print 'Using preset', got:\n%s", output)
	}
}

func TestStartWithPreset_MultipleOverrides(t *testing.T) {
	base := t.TempDir()
	t.Cleanup(setTestBaseDir(base))

	saveTestPreset(t, base, &lab.Preset{
		Name:    "multi",
		Project: "/tmp/proj",
		Profile: "zsh",
		Branch:  "lab/zsh",
		EnvVars: map[string]string{"API_KEY": "sk-1234"},
	})

	var stderr bytes.Buffer
	cmd := NewRootCmd()
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{
		"start", "multi",
		"--branch", "lab/experiment",
		"--env", "VERBOSE=1",
	})

	_ = cmd.Execute()

	output := stderr.String()
	// Branch should be overridden
	if !strings.Contains(output, "- branch:") || !strings.Contains(output, "+ branch:") {
		t.Errorf("branch should have diff markers, got:\n%s", output)
	}
	// Env should be overridden (preset env replaced by flag env)
	if !strings.Contains(output, "- env:") || !strings.Contains(output, "+ env:") {
		t.Errorf("env should have diff markers, got:\n%s", output)
	}
	// Project should not be overridden
	if strings.Contains(output, "- project:") || strings.Contains(output, "+ project:") {
		t.Errorf("project should not have diff markers, got:\n%s", output)
	}
}

func TestStartWithPreset_TooManyPositionalArgs(t *testing.T) {
	base := t.TempDir()
	t.Cleanup(setTestBaseDir(base))

	cmd := NewRootCmd()
	cmd.SetArgs([]string{"start", "preset1", "preset2"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for too many args")
	}
}
