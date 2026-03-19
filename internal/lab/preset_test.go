package lab_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/claudeup/claudeup-lab/internal/lab"
)

func TestPresetSaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	store := lab.NewPresetStore(dir)

	preset := &lab.Preset{
		Name:        "my-preset",
		CreatedAt:   time.Date(2026, 3, 19, 12, 0, 0, 0, time.UTC),
		Project:     "/home/user/code/myapp",
		Profile:     "zsh",
		Branch:      "lab/zsh",
		LabName:     "myapp-zsh",
		Features:    []string{"ghcr.io/devcontainers/features/go:1"},
		BaseProfile: "base",
		Firewall:    true,
		InitScript:  "setup.sh",
		EnvVars:     map[string]string{"FOO": "bar", "BAZ": "qux"},
		EnvFile:     ".env.local",
	}

	if err := store.Save(preset); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := store.Load("my-preset")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if loaded.Name != preset.Name {
		t.Errorf("Name = %q, want %q", loaded.Name, preset.Name)
	}
	if !loaded.CreatedAt.Equal(preset.CreatedAt) {
		t.Errorf("CreatedAt = %v, want %v", loaded.CreatedAt, preset.CreatedAt)
	}
	if loaded.Project != preset.Project {
		t.Errorf("Project = %q, want %q", loaded.Project, preset.Project)
	}
	if loaded.Profile != preset.Profile {
		t.Errorf("Profile = %q, want %q", loaded.Profile, preset.Profile)
	}
	if loaded.Branch != preset.Branch {
		t.Errorf("Branch = %q, want %q", loaded.Branch, preset.Branch)
	}
	if loaded.LabName != preset.LabName {
		t.Errorf("LabName = %q, want %q", loaded.LabName, preset.LabName)
	}
	if len(loaded.Features) != 1 || loaded.Features[0] != preset.Features[0] {
		t.Errorf("Features = %v, want %v", loaded.Features, preset.Features)
	}
	if loaded.BaseProfile != preset.BaseProfile {
		t.Errorf("BaseProfile = %q, want %q", loaded.BaseProfile, preset.BaseProfile)
	}
	if !loaded.Firewall {
		t.Error("Firewall = false, want true")
	}
	if loaded.InitScript != preset.InitScript {
		t.Errorf("InitScript = %q, want %q", loaded.InitScript, preset.InitScript)
	}
	if loaded.EnvVars["FOO"] != "bar" || loaded.EnvVars["BAZ"] != "qux" {
		t.Errorf("EnvVars = %v, want %v", loaded.EnvVars, preset.EnvVars)
	}
	if loaded.EnvFile != preset.EnvFile {
		t.Errorf("EnvFile = %q, want %q", loaded.EnvFile, preset.EnvFile)
	}
}

func TestPresetListAll(t *testing.T) {
	dir := t.TempDir()
	store := lab.NewPresetStore(dir)

	store.Save(&lab.Preset{Name: "alpha", Project: "/a"})
	store.Save(&lab.Preset{Name: "beta", Project: "/b"})
	store.Save(&lab.Preset{Name: "gamma", Project: "/c"})

	presets, err := store.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(presets) != 3 {
		t.Fatalf("List returned %d presets, want 3", len(presets))
	}

	names := make(map[string]bool)
	for _, p := range presets {
		names[p.Name] = true
	}
	for _, want := range []string{"alpha", "beta", "gamma"} {
		if !names[want] {
			t.Errorf("List missing preset %q", want)
		}
	}
}

func TestPresetListNonexistentDirectory(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "does", "not", "exist")
	store := lab.NewPresetStore(dir)

	presets, err := store.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if presets == nil {
		t.Error("List returned nil, want empty slice")
	}
	if len(presets) != 0 {
		t.Errorf("List returned %d presets, want 0", len(presets))
	}
}

func TestPresetListEmptyDirectory(t *testing.T) {
	dir := t.TempDir()
	store := lab.NewPresetStore(dir)

	presets, err := store.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if presets == nil {
		t.Error("List returned nil, want empty slice")
	}
	if len(presets) != 0 {
		t.Errorf("List returned %d presets, want 0", len(presets))
	}
}

func TestPresetDelete(t *testing.T) {
	dir := t.TempDir()
	store := lab.NewPresetStore(dir)

	store.Save(&lab.Preset{Name: "to-delete", Project: "/tmp"})

	if err := store.Delete("to-delete"); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	_, err := store.Load("to-delete")
	if err == nil {
		t.Error("Load after Delete should return error")
	}
}

func TestPresetLoadNonexistent(t *testing.T) {
	dir := t.TempDir()
	store := lab.NewPresetStore(dir)

	_, err := store.Load("does-not-exist")
	if err == nil {
		t.Error("Load nonexistent should return error")
	}
}

func TestPresetSaveCreatesDirectory(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nested", "presets")
	store := lab.NewPresetStore(dir)

	err := store.Save(&lab.Preset{Name: "test", Project: "/tmp"})
	if err != nil {
		t.Fatalf("Save to nested dir: %v", err)
	}

	if _, err := os.Stat(dir); os.IsNotExist(err) {
		t.Error("Save should create preset directory")
	}
}

func TestPresetOverwrite(t *testing.T) {
	dir := t.TempDir()
	store := lab.NewPresetStore(dir)

	store.Save(&lab.Preset{Name: "foo", Project: "/original"})
	store.Save(&lab.Preset{Name: "foo", Project: "/updated"})

	loaded, err := store.Load("foo")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if loaded.Project != "/updated" {
		t.Errorf("Project = %q, want %q", loaded.Project, "/updated")
	}
}

func TestPresetNameValidationRejectsInvalid(t *testing.T) {
	dir := t.TempDir()
	store := lab.NewPresetStore(dir)

	invalid := []string{"", "../etc", "foo/bar", ".hidden", "has spaces", "has@special"}

	for _, name := range invalid {
		t.Run("save_"+name, func(t *testing.T) {
			err := store.Save(&lab.Preset{Name: name, Project: "/tmp"})
			if err == nil {
				t.Errorf("Save(%q) should return error", name)
			}
		})
		t.Run("load_"+name, func(t *testing.T) {
			_, err := store.Load(name)
			if err == nil {
				t.Errorf("Load(%q) should return error", name)
			}
		})
		t.Run("delete_"+name, func(t *testing.T) {
			err := store.Delete(name)
			if err == nil {
				t.Errorf("Delete(%q) should return error", name)
			}
		})
	}
}

func TestPresetNameValidationAcceptsValid(t *testing.T) {
	dir := t.TempDir()
	store := lab.NewPresetStore(dir)

	valid := []string{"my-preset", "test_config", "v1.0", "ABC123"}

	for _, name := range valid {
		t.Run(name, func(t *testing.T) {
			err := store.Save(&lab.Preset{Name: name, Project: "/tmp"})
			if err != nil {
				t.Errorf("Save(%q) should succeed, got: %v", name, err)
			}
			_, err = store.Load(name)
			if err != nil {
				t.Errorf("Load(%q) should succeed, got: %v", name, err)
			}
		})
	}
}

func TestPresetFromStartOptions(t *testing.T) {
	opts := &lab.StartOptions{
		Project:     "/home/user/code/gstack",
		Profile:     "zsh",
		Branch:      "lab/zsh",
		Name:        "gstack-zsh",
		Features:    []string{"ghcr.io/devcontainers/features/go:1"},
		BaseProfile: "base",
		Firewall:    true,
		InitScript:  "setup.sh",
		EnvVars:     map[string]string{"KEY": "val"},
		EnvFile:     ".env",
	}

	before := time.Now().UTC()
	preset := lab.PresetFromStartOptions("my-config", opts)
	after := time.Now().UTC()

	if preset.Name != "my-config" {
		t.Errorf("Name = %q, want %q", preset.Name, "my-config")
	}
	if preset.CreatedAt.Before(before) || preset.CreatedAt.After(after) {
		t.Errorf("CreatedAt = %v, want between %v and %v", preset.CreatedAt, before, after)
	}
	if preset.Project != opts.Project {
		t.Errorf("Project = %q, want %q", preset.Project, opts.Project)
	}
	if preset.Profile != opts.Profile {
		t.Errorf("Profile = %q, want %q", preset.Profile, opts.Profile)
	}
	if preset.Branch != opts.Branch {
		t.Errorf("Branch = %q, want %q", preset.Branch, opts.Branch)
	}
	if preset.LabName != opts.Name {
		t.Errorf("LabName = %q, want %q", preset.LabName, opts.Name)
	}
	if len(preset.Features) != 1 || preset.Features[0] != opts.Features[0] {
		t.Errorf("Features = %v, want %v", preset.Features, opts.Features)
	}
	if preset.BaseProfile != opts.BaseProfile {
		t.Errorf("BaseProfile = %q, want %q", preset.BaseProfile, opts.BaseProfile)
	}
	if !preset.Firewall {
		t.Error("Firewall = false, want true")
	}
	if preset.InitScript != opts.InitScript {
		t.Errorf("InitScript = %q, want %q", preset.InitScript, opts.InitScript)
	}
	if preset.EnvVars["KEY"] != "val" {
		t.Errorf("EnvVars = %v, want %v", preset.EnvVars, opts.EnvVars)
	}
	if preset.EnvFile != opts.EnvFile {
		t.Errorf("EnvFile = %q, want %q", preset.EnvFile, opts.EnvFile)
	}
}

func TestPresetToStartOptions(t *testing.T) {
	preset := &lab.Preset{
		Name:        "my-config",
		Project:     "/home/user/code/gstack",
		Profile:     "zsh",
		Branch:      "lab/zsh",
		LabName:     "gstack-zsh",
		Features:    []string{"ghcr.io/devcontainers/features/go:1"},
		BaseProfile: "base",
		Firewall:    true,
		InitScript:  "setup.sh",
		EnvVars:     map[string]string{"KEY": "val"},
		EnvFile:     ".env",
	}

	opts := preset.ToStartOptions()

	if opts.Project != preset.Project {
		t.Errorf("Project = %q, want %q", opts.Project, preset.Project)
	}
	if opts.Profile != preset.Profile {
		t.Errorf("Profile = %q, want %q", opts.Profile, preset.Profile)
	}
	if opts.Branch != preset.Branch {
		t.Errorf("Branch = %q, want %q", opts.Branch, preset.Branch)
	}
	if opts.Name != preset.LabName {
		t.Errorf("Name = %q, want %q", opts.Name, preset.LabName)
	}
	if len(opts.Features) != 1 || opts.Features[0] != preset.Features[0] {
		t.Errorf("Features = %v, want %v", opts.Features, preset.Features)
	}
	if opts.BaseProfile != preset.BaseProfile {
		t.Errorf("BaseProfile = %q, want %q", opts.BaseProfile, preset.BaseProfile)
	}
	if !opts.Firewall {
		t.Error("Firewall = false, want true")
	}
	if opts.InitScript != preset.InitScript {
		t.Errorf("InitScript = %q, want %q", opts.InitScript, preset.InitScript)
	}
	if opts.EnvVars["KEY"] != "val" {
		t.Errorf("EnvVars = %v, want %v", opts.EnvVars, preset.EnvVars)
	}
	if opts.EnvFile != preset.EnvFile {
		t.Errorf("EnvFile = %q, want %q", opts.EnvFile, preset.EnvFile)
	}
}

func TestFormatPresetFieldsNonZero(t *testing.T) {
	preset := &lab.Preset{
		Project:  "/Users/mark/code/gstack",
		Profile:  "zsh",
		Branch:   "lab/zsh",
		Firewall: true,
	}

	output := lab.FormatPresetFields(preset)

	expected := []string{
		"  project:      /Users/mark/code/gstack",
		"  profile:      zsh",
		"  branch:       lab/zsh",
		"  firewall:     true",
	}

	for _, line := range expected {
		if !strings.Contains(output, line) {
			t.Errorf("output missing line %q\ngot:\n%s", line, output)
		}
	}
}

func TestFormatPresetFieldsOmitsZero(t *testing.T) {
	preset := &lab.Preset{
		Project: "/Users/mark/code/gstack",
		Profile: "zsh",
	}

	output := lab.FormatPresetFields(preset)

	if strings.Contains(output, "branch:") {
		t.Errorf("output should omit branch, got:\n%s", output)
	}
	if strings.Contains(output, "feature:") {
		t.Errorf("output should omit feature, got:\n%s", output)
	}
	if strings.Contains(output, "firewall:") {
		t.Errorf("output should omit firewall, got:\n%s", output)
	}
	if strings.Contains(output, "env:") {
		t.Errorf("output should omit env, got:\n%s", output)
	}
	if strings.Contains(output, "env-file:") {
		t.Errorf("output should omit env-file, got:\n%s", output)
	}
	if strings.Contains(output, "init-script:") {
		t.Errorf("output should omit init-script, got:\n%s", output)
	}
}

func TestFormatPresetFieldsRepeatedValues(t *testing.T) {
	preset := &lab.Preset{
		Project:  "/tmp/proj",
		Features: []string{"feat-a", "feat-b"},
		EnvVars:  map[string]string{"ZZZ": "last", "AAA": "first", "MMM": "mid"},
	}

	output := lab.FormatPresetFields(preset)

	// Each feature on its own line
	if strings.Count(output, "feature:") != 2 {
		t.Errorf("expected 2 feature: lines, got %d in:\n%s", strings.Count(output, "feature:"), output)
	}

	// Env vars sorted alphabetically
	if strings.Count(output, "env:") != 3 {
		t.Errorf("expected 3 env: lines, got %d in:\n%s", strings.Count(output, "env:"), output)
	}

	lines := strings.Split(output, "\n")
	var envLines []string
	for _, line := range lines {
		if strings.Contains(line, "env:") {
			envLines = append(envLines, line)
		}
	}
	if len(envLines) != 3 {
		t.Fatalf("expected 3 env-var lines, got %d", len(envLines))
	}
	if !strings.Contains(envLines[0], "AAA=first") {
		t.Errorf("first env-var line should contain AAA=first, got %q", envLines[0])
	}
	if !strings.Contains(envLines[1], "MMM=mid") {
		t.Errorf("second env-var line should contain MMM=mid, got %q", envLines[1])
	}
	if !strings.Contains(envLines[2], "ZZZ=last") {
		t.Errorf("third env-var line should contain ZZZ=last, got %q", envLines[2])
	}
}
