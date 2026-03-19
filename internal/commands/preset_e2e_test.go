package commands

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/claudeup/claudeup-lab/internal/lab"
)

// TestE2E_SaveFromFlags_ThenListShowStart exercises the full journey of saving a
// preset from explicit flags, then listing, showing, and starting from it.
func TestE2E_SaveFromFlags_ThenListShowStart(t *testing.T) {
	base := t.TempDir()
	t.Cleanup(setTestBaseDir(base))

	projDir := t.TempDir()

	// Step 1: Save a preset from explicit flags
	saveOutput := captureStdout(t, func() {
		cmd := NewRootCmd()
		cmd.SetArgs([]string{
			"preset", "save", "my-config",
			"--project", projDir,
			"--profile", "zsh",
			"--branch", "lab/zsh",
			"--firewall",
		})
		if err := cmd.Execute(); err != nil {
			t.Fatalf("preset save failed: %v", err)
		}
	})

	if !strings.Contains(saveOutput, "Preset saved: my-config") {
		t.Errorf("save output should contain 'Preset saved: my-config', got:\n%s", saveOutput)
	}
	if !strings.Contains(saveOutput, "project:") {
		t.Errorf("save output should contain 'project:', got:\n%s", saveOutput)
	}
	if !strings.Contains(saveOutput, "zsh") {
		t.Errorf("save output should contain 'zsh', got:\n%s", saveOutput)
	}
	if !strings.Contains(saveOutput, "lab/zsh") {
		t.Errorf("save output should contain 'lab/zsh', got:\n%s", saveOutput)
	}
	if !strings.Contains(saveOutput, "firewall:") {
		t.Errorf("save output should contain 'firewall:', got:\n%s", saveOutput)
	}

	// Step 2: List presets and verify my-config appears
	listOutput := captureStdout(t, func() {
		cmd := NewRootCmd()
		cmd.SetArgs([]string{"preset", "list"})
		if err := cmd.Execute(); err != nil {
			t.Fatalf("preset list failed: %v", err)
		}
	})

	if !strings.Contains(listOutput, "my-config") {
		t.Errorf("list output should contain 'my-config', got:\n%s", listOutput)
	}
	projBase := filepath.Base(projDir)
	if !strings.Contains(listOutput, projBase) {
		t.Errorf("list output should contain project basename %q, got:\n%s", projBase, listOutput)
	}
	if !strings.Contains(listOutput, "zsh") {
		t.Errorf("list output should contain profile 'zsh', got:\n%s", listOutput)
	}
	if !strings.Contains(listOutput, "--firewall") {
		t.Errorf("list FLAGS should contain '--firewall', got:\n%s", listOutput)
	}

	// Step 3: Show the preset and verify full details
	showOutput := captureStdout(t, func() {
		cmd := NewRootCmd()
		cmd.SetArgs([]string{"preset", "show", "my-config"})
		if err := cmd.Execute(); err != nil {
			t.Fatalf("preset show failed: %v", err)
		}
	})

	if !strings.Contains(showOutput, "Preset: my-config") {
		t.Errorf("show output should contain 'Preset: my-config', got:\n%s", showOutput)
	}
	if !strings.Contains(showOutput, projDir) {
		t.Errorf("show output should contain full project path %q, got:\n%s", projDir, showOutput)
	}
	if !strings.Contains(showOutput, "zsh") {
		t.Errorf("show output should contain profile 'zsh', got:\n%s", showOutput)
	}
	if !strings.Contains(showOutput, "lab/zsh") {
		t.Errorf("show output should contain branch 'lab/zsh', got:\n%s", showOutput)
	}
	if !strings.Contains(showOutput, "firewall:") && !strings.Contains(showOutput, "true") {
		t.Errorf("show output should contain firewall=true, got:\n%s", showOutput)
	}

	// Step 4: Start from the preset (will fail at Docker, but verify preset loading output)
	var stderr bytes.Buffer
	cmd := NewRootCmd()
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"start", "my-config"})
	_ = cmd.Execute()

	startOutput := stderr.String()
	if !strings.Contains(startOutput, "Using preset 'my-config'") {
		t.Errorf("start stderr should contain \"Using preset 'my-config'\", got:\n%s", startOutput)
	}
	if !strings.Contains(startOutput, projDir) {
		t.Errorf("start stderr should contain project path %q, got:\n%s", projDir, startOutput)
	}
	if !strings.Contains(startOutput, "zsh") {
		t.Errorf("start stderr should contain profile 'zsh', got:\n%s", startOutput)
	}
	if !strings.Contains(startOutput, "lab/zsh") {
		t.Errorf("start stderr should contain branch 'lab/zsh', got:\n%s", startOutput)
	}
}

// TestE2E_SaveFromLab exercises saving a preset from a lab's tracked configuration.
func TestE2E_SaveFromLab(t *testing.T) {
	base, mgr := setupPresetTestEnv(t)

	// Set up a lab with StartConfig in its state file
	meta := &lab.Metadata{
		ID:          "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee",
		DisplayName: "source-lab",
		Project:     "/tmp/test-project",
		ProjectName: "test-project",
		Profile:     "zsh",
		Branch:      "lab/zsh",
		StartConfig: &lab.StartConfig{
			Project:     "/tmp/test-project",
			Profile:     "zsh",
			Branch:      "lab/zsh",
			BaseProfile: "base",
			Firewall:    true,
		},
	}
	if err := mgr.Store().Save(meta); err != nil {
		t.Fatalf("failed to save lab state: %v", err)
	}

	// Save a preset from the lab
	saveOutput := captureStdout(t, func() {
		cmd := NewRootCmd()
		cmd.SetArgs([]string{
			"preset", "save", "from-lab-test",
			"--from-lab", "source-lab",
		})
		if err := cmd.Execute(); err != nil {
			t.Fatalf("preset save --from-lab failed: %v", err)
		}
	})

	if !strings.Contains(saveOutput, "Preset saved: from-lab-test") {
		t.Errorf("save output should contain 'Preset saved: from-lab-test', got:\n%s", saveOutput)
	}
	if !strings.Contains(saveOutput, "/tmp/test-project") {
		t.Errorf("save output should contain project path, got:\n%s", saveOutput)
	}
	if !strings.Contains(saveOutput, "zsh") {
		t.Errorf("save output should contain profile 'zsh', got:\n%s", saveOutput)
	}

	// Verify the preset file exists at the expected location
	presetPath := filepath.Join(base, "presets", "from-lab-test.json")
	store := lab.NewPresetStore(filepath.Join(base, "presets"))
	p, err := store.Load("from-lab-test")
	if err != nil {
		t.Fatalf("preset file should exist at %s, load error: %v", presetPath, err)
	}
	if p.Project != "/tmp/test-project" {
		t.Errorf("preset project = %q, want /tmp/test-project", p.Project)
	}
	if p.Profile != "zsh" {
		t.Errorf("preset profile = %q, want zsh", p.Profile)
	}
	if p.Branch != "lab/zsh" {
		t.Errorf("preset branch = %q, want lab/zsh", p.Branch)
	}
	if p.BaseProfile != "base" {
		t.Errorf("preset base-profile = %q, want base", p.BaseProfile)
	}
	if !p.Firewall {
		t.Error("preset firewall = false, want true")
	}
}

// TestE2E_StartFromPresetWithOverrides exercises starting from a preset with
// command-line overrides, verifying the diff-style output.
func TestE2E_StartFromPresetWithOverrides(t *testing.T) {
	base := t.TempDir()
	t.Cleanup(setTestBaseDir(base))

	// Save a preset
	saveTestPreset(t, base, &lab.Preset{
		Name:    "override-test",
		Project: "/tmp/proj",
		Profile: "zsh",
		Branch:  "lab/zsh",
	})

	// Start with a branch override
	var stderr bytes.Buffer
	cmd := NewRootCmd()
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"start", "override-test", "--branch", "lab/experiment"})
	_ = cmd.Execute()

	output := stderr.String()

	// Verify diff markers for the overridden branch
	if !strings.Contains(output, "- branch:") {
		t.Errorf("output should contain '- branch:' for old value, got:\n%s", output)
	}
	if !strings.Contains(output, "lab/zsh") {
		t.Errorf("output should contain old branch value 'lab/zsh', got:\n%s", output)
	}
	if !strings.Contains(output, "+ branch:") {
		t.Errorf("output should contain '+ branch:' for new value, got:\n%s", output)
	}
	if !strings.Contains(output, "lab/experiment") {
		t.Errorf("output should contain new branch value 'lab/experiment', got:\n%s", output)
	}

	// Non-overridden fields should NOT have diff markers
	if strings.Contains(output, "- project:") || strings.Contains(output, "+ project:") {
		t.Errorf("project should not have diff markers, got:\n%s", output)
	}
	if strings.Contains(output, "- profile:") || strings.Contains(output, "+ profile:") {
		t.Errorf("profile should not have diff markers, got:\n%s", output)
	}

	// Non-overridden fields should appear plainly
	if !strings.Contains(output, "project:") || !strings.Contains(output, "/tmp/proj") {
		t.Errorf("output should show project field without markers, got:\n%s", output)
	}
	if !strings.Contains(output, "profile:") || !strings.Contains(output, "zsh") {
		t.Errorf("output should show profile field without markers, got:\n%s", output)
	}
}

// TestE2E_PresetLifecycle exercises the full lifecycle: save, overwrite with force,
// verify show reflects new values, delete, verify list is empty, verify show errors.
func TestE2E_PresetLifecycle(t *testing.T) {
	base := t.TempDir()
	t.Cleanup(setTestBaseDir(base))

	projDir := t.TempDir()

	// Step 1: Save a preset
	captureStdout(t, func() {
		cmd := NewRootCmd()
		cmd.SetArgs([]string{
			"preset", "save", "lifecycle-test",
			"--project", projDir,
			"--profile", "zsh",
			"--branch", "lab/original",
		})
		if err := cmd.Execute(); err != nil {
			t.Fatalf("initial save failed: %v", err)
		}
	})

	// Step 2: Overwrite with --force and different values
	captureStdout(t, func() {
		cmd := NewRootCmd()
		cmd.SetArgs([]string{
			"preset", "save", "lifecycle-test",
			"--project", projDir,
			"--profile", "fish",
			"--branch", "lab/updated",
			"--firewall",
			"--force",
		})
		if err := cmd.Execute(); err != nil {
			t.Fatalf("force overwrite failed: %v", err)
		}
	})

	// Step 3: Verify show reflects the new values
	showOutput := captureStdout(t, func() {
		cmd := NewRootCmd()
		cmd.SetArgs([]string{"preset", "show", "lifecycle-test"})
		if err := cmd.Execute(); err != nil {
			t.Fatalf("show after overwrite failed: %v", err)
		}
	})

	if !strings.Contains(showOutput, "fish") {
		t.Errorf("show should reflect updated profile 'fish', got:\n%s", showOutput)
	}
	if !strings.Contains(showOutput, "lab/updated") {
		t.Errorf("show should reflect updated branch 'lab/updated', got:\n%s", showOutput)
	}
	if strings.Contains(showOutput, "lab/original") {
		t.Errorf("show should NOT contain old branch 'lab/original', got:\n%s", showOutput)
	}
	if !strings.Contains(showOutput, "firewall:") {
		t.Errorf("show should contain firewall field, got:\n%s", showOutput)
	}

	// Step 4: Delete with --force
	captureStdout(t, func() {
		cmd := NewRootCmd()
		cmd.SetArgs([]string{"preset", "delete", "lifecycle-test", "--force"})
		if err := cmd.Execute(); err != nil {
			t.Fatalf("delete failed: %v", err)
		}
	})

	// Step 5: Verify list shows "No presets found."
	listOutput := captureStdout(t, func() {
		cmd := NewRootCmd()
		cmd.SetArgs([]string{"preset", "list"})
		if err := cmd.Execute(); err != nil {
			t.Fatalf("list after delete failed: %v", err)
		}
	})

	if !strings.Contains(listOutput, "No presets found.") {
		t.Errorf("list after delete should say 'No presets found.', got:\n%s", listOutput)
	}

	// Step 6: Verify show returns error for deleted preset
	cmd := NewRootCmd()
	cmd.SetArgs([]string{"preset", "show", "lifecycle-test"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("show after delete should return error")
	}
	if !strings.Contains(err.Error(), "preset 'lifecycle-test' not found") {
		t.Errorf("error = %q, want 'preset 'lifecycle-test' not found'", err.Error())
	}
}

// TestE2E_ErrorScenarios exercises all error paths described in the acceptance criteria.
func TestE2E_ErrorScenarios(t *testing.T) {
	t.Run("save_no_flags_no_from_lab", func(t *testing.T) {
		setupPresetTestEnv(t)

		cmd := NewRootCmd()
		cmd.SetArgs([]string{"preset", "save", "empty-test"})
		err := cmd.Execute()
		if err == nil {
			t.Fatal("expected error when no flags and no --from-lab")
		}
		if !strings.Contains(err.Error(), "provide --from-lab or at least one start flag") {
			t.Errorf("error = %q, should mention missing flags", err.Error())
		}
	})

	t.Run("save_from_lab_with_explicit_flags", func(t *testing.T) {
		setupPresetTestEnv(t)

		cmd := NewRootCmd()
		cmd.SetArgs([]string{
			"preset", "save", "conflict-test",
			"--from-lab", "some-lab",
			"--profile", "zsh",
		})
		err := cmd.Execute()
		if err == nil {
			t.Fatal("expected error when --from-lab combined with flags")
		}
		if !strings.Contains(err.Error(), "--from-lab cannot be combined with other flags") {
			t.Errorf("error = %q, should mention mutual exclusivity", err.Error())
		}
	})

	t.Run("start_nonexistent_preset", func(t *testing.T) {
		base := t.TempDir()
		t.Cleanup(setTestBaseDir(base))

		cmd := NewRootCmd()
		cmd.SetArgs([]string{"start", "foo"})
		err := cmd.Execute()
		if err == nil {
			t.Fatal("expected error for nonexistent preset")
		}
		if !strings.Contains(err.Error(), "preset 'foo' not found") {
			t.Errorf("error = %q, want \"preset 'foo' not found\"", err.Error())
		}
	})

	t.Run("show_nonexistent_preset", func(t *testing.T) {
		setupPresetTestEnv(t)

		cmd := NewRootCmd()
		cmd.SetArgs([]string{"preset", "show", "foo"})
		err := cmd.Execute()
		if err == nil {
			t.Fatal("expected error for nonexistent preset")
		}
		if !strings.Contains(err.Error(), "preset 'foo' not found") {
			t.Errorf("error = %q, want \"preset 'foo' not found\"", err.Error())
		}
	})

	t.Run("delete_nonexistent_preset", func(t *testing.T) {
		setupPresetTestEnv(t)

		cmd := NewRootCmd()
		cmd.SetArgs([]string{"preset", "delete", "foo", "--force"})
		err := cmd.Execute()
		if err == nil {
			t.Fatal("expected error for nonexistent preset")
		}
		if !strings.Contains(err.Error(), "preset 'foo' not found") {
			t.Errorf("error = %q, want \"preset 'foo' not found\"", err.Error())
		}
	})
}

// TestE2E_BackwardCompatibility verifies that pre-feature labs (without StartConfig)
// load without error, and that preset save --from-lab returns a clear error.
func TestE2E_BackwardCompatibility(t *testing.T) {
	t.Run("state_without_start_config_loads", func(t *testing.T) {
		_, mgr := setupPresetTestEnv(t)

		// Save a lab state without StartConfig (simulating pre-feature lab)
		meta := &lab.Metadata{
			ID:          "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee",
			DisplayName: "old-lab",
			Project:     "/tmp/old-project",
			ProjectName: "old-project",
			Profile:     "zsh",
			Branch:      "lab/zsh",
			// No StartConfig field
		}
		if err := mgr.Store().Save(meta); err != nil {
			t.Fatalf("failed to save lab state: %v", err)
		}

		// Verify the lab loads without error via the resolver
		resolver := lab.NewResolver(mgr.Store())
		loaded, err := resolver.Resolve("old-lab")
		if err != nil {
			t.Fatalf("pre-feature lab should load without error: %v", err)
		}
		if loaded.StartConfig != nil {
			t.Error("pre-feature lab should have nil StartConfig")
		}
	})

	t.Run("save_from_lab_without_start_config", func(t *testing.T) {
		_, mgr := setupPresetTestEnv(t)

		// Save a lab state without StartConfig
		meta := &lab.Metadata{
			ID:          "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee",
			DisplayName: "pre-feature-lab",
			Project:     "/tmp/old-project",
			ProjectName: "old-project",
			Profile:     "zsh",
			Branch:      "lab/zsh",
		}
		if err := mgr.Store().Save(meta); err != nil {
			t.Fatalf("failed to save lab state: %v", err)
		}

		cmd := NewRootCmd()
		cmd.SetArgs([]string{
			"preset", "save", "from-old",
			"--from-lab", "pre-feature-lab",
		})
		err := cmd.Execute()
		if err == nil {
			t.Fatal("expected error for lab without StartConfig")
		}
		if !strings.Contains(err.Error(), "has no tracked configuration") {
			t.Errorf("error = %q, should contain 'has no tracked configuration'", err.Error())
		}
	})
}
