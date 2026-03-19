package lab_test

import (
	"strings"
	"testing"

	"github.com/claudeup/claudeup-lab/internal/lab"
)

func TestMergeWithOverridesNoChangedFlags(t *testing.T) {
	base := &lab.StartOptions{
		Project: "/home/user/code/myapp",
		Profile: "zsh",
		Branch:  "lab/zsh",
	}
	flags := &lab.StartOptions{}
	changed := map[string]bool{}

	merged, overrides := lab.MergeWithOverrides(base, flags, changed)

	if merged.Project != base.Project {
		t.Errorf("Project = %q, want %q", merged.Project, base.Project)
	}
	if merged.Profile != base.Profile {
		t.Errorf("Profile = %q, want %q", merged.Profile, base.Profile)
	}
	if merged.Branch != base.Branch {
		t.Errorf("Branch = %q, want %q", merged.Branch, base.Branch)
	}
	if len(overrides) != 0 {
		t.Errorf("overrides = %v, want empty", overrides)
	}
}

func TestMergeWithOverridesSingleStringField(t *testing.T) {
	base := &lab.StartOptions{
		Project: "/home/user/code/myapp",
		Branch:  "lab/zsh",
	}
	flags := &lab.StartOptions{
		Branch: "lab/experiment",
	}
	changed := map[string]bool{"branch": true}

	merged, overrides := lab.MergeWithOverrides(base, flags, changed)

	if merged.Branch != "lab/experiment" {
		t.Errorf("Branch = %q, want %q", merged.Branch, "lab/experiment")
	}
	if merged.Project != base.Project {
		t.Errorf("Project = %q, want %q (unchanged)", merged.Project, base.Project)
	}
	if len(overrides) != 1 {
		t.Fatalf("overrides count = %d, want 1", len(overrides))
	}
	if overrides[0].Field != "branch" {
		t.Errorf("Field = %q, want %q", overrides[0].Field, "branch")
	}
	if overrides[0].OldValue != "lab/zsh" {
		t.Errorf("OldValue = %q, want %q", overrides[0].OldValue, "lab/zsh")
	}
	if overrides[0].NewValue != "lab/experiment" {
		t.Errorf("NewValue = %q, want %q", overrides[0].NewValue, "lab/experiment")
	}
}

func TestMergeWithOverridesBoolField(t *testing.T) {
	base := &lab.StartOptions{
		Firewall: false,
	}
	flags := &lab.StartOptions{
		Firewall: true,
	}
	changed := map[string]bool{"firewall": true}

	merged, overrides := lab.MergeWithOverrides(base, flags, changed)

	if !merged.Firewall {
		t.Error("Firewall = false, want true")
	}
	if len(overrides) != 1 {
		t.Fatalf("overrides count = %d, want 1", len(overrides))
	}
	if overrides[0].Field != "firewall" {
		t.Errorf("Field = %q, want %q", overrides[0].Field, "firewall")
	}
	if overrides[0].OldValue != "false" {
		t.Errorf("OldValue = %q, want %q", overrides[0].OldValue, "false")
	}
	if overrides[0].NewValue != "true" {
		t.Errorf("NewValue = %q, want %q", overrides[0].NewValue, "true")
	}
}

func TestMergeWithOverridesSliceField(t *testing.T) {
	base := &lab.StartOptions{
		Features: []string{"node:20"},
	}
	flags := &lab.StartOptions{
		Features: []string{"go:1.23"},
	}
	changed := map[string]bool{"feature": true}

	merged, overrides := lab.MergeWithOverrides(base, flags, changed)

	if len(merged.Features) != 1 || merged.Features[0] != "go:1.23" {
		t.Errorf("Features = %v, want [go:1.23]", merged.Features)
	}
	if len(overrides) != 1 {
		t.Fatalf("overrides count = %d, want 1", len(overrides))
	}
	if overrides[0].Field != "feature" {
		t.Errorf("Field = %q, want %q", overrides[0].Field, "feature")
	}
	if overrides[0].OldValue != "node:20" {
		t.Errorf("OldValue = %q, want %q", overrides[0].OldValue, "node:20")
	}
	if overrides[0].NewValue != "go:1.23" {
		t.Errorf("NewValue = %q, want %q", overrides[0].NewValue, "go:1.23")
	}
}

func TestMergeWithOverridesMapField(t *testing.T) {
	base := &lab.StartOptions{
		EnvVars: map[string]string{"API_KEY": "sk-1234"},
	}
	flags := &lab.StartOptions{
		EnvVars: map[string]string{"VERBOSE": "1"},
	}
	changed := map[string]bool{"env": true}

	merged, overrides := lab.MergeWithOverrides(base, flags, changed)

	if len(merged.EnvVars) != 1 || merged.EnvVars["VERBOSE"] != "1" {
		t.Errorf("EnvVars = %v, want map[VERBOSE:1]", merged.EnvVars)
	}
	if len(overrides) != 1 {
		t.Fatalf("overrides count = %d, want 1", len(overrides))
	}
	if overrides[0].Field != "env" {
		t.Errorf("Field = %q, want %q", overrides[0].Field, "env")
	}
	if overrides[0].OldValue != "API_KEY=sk-1234" {
		t.Errorf("OldValue = %q, want %q", overrides[0].OldValue, "API_KEY=sk-1234")
	}
	if overrides[0].NewValue != "VERBOSE=1" {
		t.Errorf("NewValue = %q, want %q", overrides[0].NewValue, "VERBOSE=1")
	}
}

func TestMergeWithOverridesMapFieldSortedKeys(t *testing.T) {
	base := &lab.StartOptions{
		EnvVars: map[string]string{"ZZZ": "last", "AAA": "first"},
	}
	flags := &lab.StartOptions{
		EnvVars: map[string]string{"MMM": "mid", "BBB": "second"},
	}
	changed := map[string]bool{"env": true}

	_, overrides := lab.MergeWithOverrides(base, flags, changed)

	if len(overrides) != 1 {
		t.Fatalf("overrides count = %d, want 1", len(overrides))
	}
	if overrides[0].OldValue != "AAA=first,ZZZ=last" {
		t.Errorf("OldValue = %q, want %q", overrides[0].OldValue, "AAA=first,ZZZ=last")
	}
	if overrides[0].NewValue != "BBB=second,MMM=mid" {
		t.Errorf("NewValue = %q, want %q", overrides[0].NewValue, "BBB=second,MMM=mid")
	}
}

func TestMergeWithOverridesAllFieldsOverridden(t *testing.T) {
	base := &lab.StartOptions{
		Project:     "/old/project",
		Profile:     "bash",
		Branch:      "lab/old",
		Name:        "old-name",
		Features:    []string{"node:18"},
		BaseProfile: "old-base",
		Firewall:    false,
		InitScript:  "old.sh",
		EnvVars:     map[string]string{"OLD": "val"},
		EnvFile:     "old.env",
	}
	flags := &lab.StartOptions{
		Project:     "/new/project",
		Profile:     "zsh",
		Branch:      "lab/new",
		Name:        "new-name",
		Features:    []string{"go:1.23"},
		BaseProfile: "new-base",
		Firewall:    true,
		InitScript:  "new.sh",
		EnvVars:     map[string]string{"NEW": "val"},
		EnvFile:     "new.env",
	}
	changed := map[string]bool{
		"project":      true,
		"profile":      true,
		"branch":       true,
		"name":         true,
		"feature":      true,
		"base-profile": true,
		"firewall":     true,
		"init-script":  true,
		"env":          true,
		"env-file":     true,
	}

	merged, overrides := lab.MergeWithOverrides(base, flags, changed)

	if merged.Project != "/new/project" {
		t.Errorf("Project = %q, want %q", merged.Project, "/new/project")
	}
	if merged.Profile != "zsh" {
		t.Errorf("Profile = %q, want %q", merged.Profile, "zsh")
	}
	if merged.Branch != "lab/new" {
		t.Errorf("Branch = %q, want %q", merged.Branch, "lab/new")
	}
	if merged.Name != "new-name" {
		t.Errorf("Name = %q, want %q", merged.Name, "new-name")
	}
	if len(merged.Features) != 1 || merged.Features[0] != "go:1.23" {
		t.Errorf("Features = %v, want [go:1.23]", merged.Features)
	}
	if merged.BaseProfile != "new-base" {
		t.Errorf("BaseProfile = %q, want %q", merged.BaseProfile, "new-base")
	}
	if !merged.Firewall {
		t.Error("Firewall = false, want true")
	}
	if merged.InitScript != "new.sh" {
		t.Errorf("InitScript = %q, want %q", merged.InitScript, "new.sh")
	}
	if merged.EnvVars["NEW"] != "val" {
		t.Errorf("EnvVars = %v, want map[NEW:val]", merged.EnvVars)
	}
	if merged.EnvFile != "new.env" {
		t.Errorf("EnvFile = %q, want %q", merged.EnvFile, "new.env")
	}

	if len(overrides) != 10 {
		t.Errorf("overrides count = %d, want 10", len(overrides))
	}

	fields := make(map[string]bool)
	for _, o := range overrides {
		fields[o.Field] = true
	}
	for flag := range changed {
		if !fields[flag] {
			t.Errorf("missing override for flag %q", flag)
		}
	}
}

func TestMergeWithOverridesBaseNotMutated(t *testing.T) {
	base := &lab.StartOptions{
		Branch:   "lab/zsh",
		Features: []string{"node:20", "go:1.23"},
		EnvVars:  map[string]string{"KEY": "val"},
	}
	flags := &lab.StartOptions{
		Branch:   "lab/experiment",
		Features: []string{"rust:1"},
		EnvVars:  map[string]string{"OTHER": "new"},
	}
	changed := map[string]bool{
		"branch":  true,
		"feature": true,
		"env":     true,
	}

	lab.MergeWithOverrides(base, flags, changed)

	if base.Branch != "lab/zsh" {
		t.Errorf("base.Branch mutated to %q, want %q", base.Branch, "lab/zsh")
	}
	if len(base.Features) != 2 || base.Features[0] != "node:20" || base.Features[1] != "go:1.23" {
		t.Errorf("base.Features mutated to %v", base.Features)
	}
	if base.EnvVars["KEY"] != "val" || len(base.EnvVars) != 1 {
		t.Errorf("base.EnvVars mutated to %v", base.EnvVars)
	}
}

func TestFormatOverridesWithOverrides(t *testing.T) {
	preset := &lab.Preset{
		Name:        "my-gstack",
		Project:     "/Users/mark/code/gstack",
		Profile:     "zsh",
		Branch:      "lab/zsh",
		BaseProfile: "base",
		Firewall:    true,
	}

	overrides := []lab.Override{
		{Field: "branch", OldValue: "lab/zsh", NewValue: "lab/experiment"},
	}

	output := lab.FormatOverrides(preset, overrides)

	if !strings.Contains(output, "Using preset 'my-gstack'") {
		t.Errorf("missing header line in:\n%s", output)
	}
	if !strings.Contains(output, "  project:") {
		t.Errorf("missing unchanged project line in:\n%s", output)
	}
	if !strings.Contains(output, "  profile:") {
		t.Errorf("missing unchanged profile line in:\n%s", output)
	}
	if !strings.Contains(output, "- branch:") {
		t.Errorf("missing - branch line in:\n%s", output)
	}
	if !strings.Contains(output, "+ branch:") {
		t.Errorf("missing + branch line in:\n%s", output)
	}

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, "- branch:") {
			if !strings.Contains(line, "lab/zsh") {
				t.Errorf("- branch line should contain old value, got: %q", line)
			}
		}
		if strings.Contains(line, "+ branch:") {
			if !strings.Contains(line, "lab/experiment") {
				t.Errorf("+ branch line should contain new value, got: %q", line)
			}
		}
	}
}

func TestFormatOverridesNoOverrides(t *testing.T) {
	preset := &lab.Preset{
		Name:    "my-gstack",
		Project: "/Users/mark/code/gstack",
		Profile: "zsh",
		Branch:  "lab/zsh",
	}

	output := lab.FormatOverrides(preset, nil)

	if !strings.Contains(output, "Using preset 'my-gstack'") {
		t.Errorf("missing header line in:\n%s", output)
	}
	if !strings.Contains(output, "  project:") {
		t.Errorf("missing project line in:\n%s", output)
	}
	if strings.Contains(output, "- ") || strings.Contains(output, "+ ") {
		t.Errorf("should have no diff markers in:\n%s", output)
	}
}

func TestFormatOverridesOmitsEmptyFields(t *testing.T) {
	preset := &lab.Preset{
		Name:    "minimal",
		Project: "/Users/mark/code/proj",
		Profile: "zsh",
	}

	output := lab.FormatOverrides(preset, nil)

	if !strings.Contains(output, "project:") {
		t.Errorf("missing project in:\n%s", output)
	}
	if !strings.Contains(output, "profile:") {
		t.Errorf("missing profile in:\n%s", output)
	}
	if strings.Contains(output, "branch:") {
		t.Errorf("should omit empty branch in:\n%s", output)
	}
	if strings.Contains(output, "feature:") {
		t.Errorf("should omit empty features in:\n%s", output)
	}
	if strings.Contains(output, "firewall:") {
		t.Errorf("should omit false firewall in:\n%s", output)
	}
	if strings.Contains(output, "env:") {
		t.Errorf("should omit empty env in:\n%s", output)
	}
	if strings.Contains(output, "env-file:") {
		t.Errorf("should omit empty env-file in:\n%s", output)
	}
}

func TestFormatOverridesBoolTrueToFalse(t *testing.T) {
	preset := &lab.Preset{
		Name:     "fw-test",
		Project:  "/tmp/proj",
		Firewall: true,
	}

	overrides := []lab.Override{
		{Field: "firewall", OldValue: "true", NewValue: "false"},
	}

	output := lab.FormatOverrides(preset, overrides)

	if !strings.Contains(output, "- firewall:") {
		t.Errorf("missing - firewall line in:\n%s", output)
	}
	// false is zero value, so no + line
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, "+ firewall:") {
			t.Errorf("should not show + firewall for false value in:\n%s", output)
		}
	}
}

func TestFormatOverridesAddedField(t *testing.T) {
	preset := &lab.Preset{
		Name:    "no-env",
		Project: "/tmp/proj",
	}

	overrides := []lab.Override{
		{Field: "env", OldValue: "", NewValue: "VERBOSE=1"},
	}

	output := lab.FormatOverrides(preset, overrides)

	if strings.Contains(output, "- env:") {
		t.Errorf("should not show - env line for empty old value in:\n%s", output)
	}
	if !strings.Contains(output, "+ env:") {
		t.Errorf("missing + env line in:\n%s", output)
	}
	if !strings.Contains(output, "VERBOSE=1") {
		t.Errorf("missing new env value in:\n%s", output)
	}
}

func TestMergeWithOverridesAddedFieldEmptyOldValue(t *testing.T) {
	base := &lab.StartOptions{
		Project: "/tmp/proj",
	}
	flags := &lab.StartOptions{
		Branch: "lab/new",
	}
	changed := map[string]bool{"branch": true}

	_, overrides := lab.MergeWithOverrides(base, flags, changed)

	if len(overrides) != 1 {
		t.Fatalf("overrides count = %d, want 1", len(overrides))
	}
	if overrides[0].OldValue != "" {
		t.Errorf("OldValue = %q, want empty string", overrides[0].OldValue)
	}
	if overrides[0].NewValue != "lab/new" {
		t.Errorf("NewValue = %q, want %q", overrides[0].NewValue, "lab/new")
	}
}

func TestMergeWithOverridesRemovedField(t *testing.T) {
	base := &lab.StartOptions{
		Branch: "lab/old",
	}
	flags := &lab.StartOptions{
		Branch: "",
	}
	changed := map[string]bool{"branch": true}

	merged, overrides := lab.MergeWithOverrides(base, flags, changed)

	if merged.Branch != "" {
		t.Errorf("Branch = %q, want empty", merged.Branch)
	}
	if len(overrides) != 1 {
		t.Fatalf("overrides count = %d, want 1", len(overrides))
	}
	if overrides[0].OldValue != "lab/old" {
		t.Errorf("OldValue = %q, want %q", overrides[0].OldValue, "lab/old")
	}
	if overrides[0].NewValue != "" {
		t.Errorf("NewValue = %q, want empty string", overrides[0].NewValue)
	}
}

func TestFormatOverridesFieldOrder(t *testing.T) {
	preset := &lab.Preset{
		Name:        "ordered",
		Project:     "/tmp/proj",
		Profile:     "zsh",
		Branch:      "lab/main",
		BaseProfile: "base",
		Firewall:    true,
		EnvFile:     ".env",
	}

	output := lab.FormatOverrides(preset, nil)
	lines := strings.Split(output, "\n")

	// Find positions of each field, using the formatted label prefix to avoid
	// substring collisions (e.g. "profile:" matching "base-profile:").
	type labelPos struct {
		label string
		match string // unique substring to look for in each line
	}
	checks := []labelPos{
		{"project:", "  project:"},
		{"profile:", "  profile:"},
		{"branch:", "  branch:"},
		{"base-profile:", "  base-profile:"},
		{"firewall:", "  firewall:"},
		{"env-file:", "  env-file:"},
	}

	positions := make(map[string]int)
	for i, line := range lines {
		for _, c := range checks {
			if strings.Contains(line, c.match) {
				positions[c.label] = i
			}
		}
	}

	order := []string{"project:", "profile:", "branch:", "base-profile:", "firewall:", "env-file:"}
	for i := 1; i < len(order); i++ {
		prev := order[i-1]
		curr := order[i]
		if positions[prev] >= positions[curr] {
			t.Errorf("%s (line %d) should come before %s (line %d)", prev, positions[prev], curr, positions[curr])
		}
	}
}

func TestFormatOverridesEnvWithOverride(t *testing.T) {
	preset := &lab.Preset{
		Name:    "env-test",
		Project: "/tmp/proj",
		EnvVars: map[string]string{"API_KEY": "sk-1234"},
	}

	overrides := []lab.Override{
		{Field: "env", OldValue: "API_KEY=sk-1234", NewValue: "VERBOSE=1"},
	}

	output := lab.FormatOverrides(preset, overrides)

	if !strings.Contains(output, "- env:") {
		t.Errorf("missing - env line in:\n%s", output)
	}
	if !strings.Contains(output, "+ env:") {
		t.Errorf("missing + env line in:\n%s", output)
	}
}

func TestMergeWithOverridesSameValueProducesNoOverride(t *testing.T) {
	base := &lab.StartOptions{
		Project:  "/Users/mark/code/gstack",
		Branch:   "lab/zsh",
		Firewall: true,
		Features: []string{"go:1.23"},
		EnvVars:  map[string]string{"KEY": "val"},
	}
	flags := &lab.StartOptions{
		Branch:   "lab/zsh",                       // same value
		Firewall: true,                            // same value
		Features: []string{"go:1.23"},             // same value
		EnvVars:  map[string]string{"KEY": "val"}, // same value
	}
	changed := map[string]bool{
		"branch":   true,
		"firewall": true,
		"feature":  true,
		"env":      true,
	}

	merged, overrides := lab.MergeWithOverrides(base, flags, changed)

	if len(overrides) != 0 {
		t.Errorf("expected 0 overrides for same values, got %d: %+v", len(overrides), overrides)
	}
	// Merged values should still reflect the flags even though no override was recorded.
	if merged.Branch != "lab/zsh" {
		t.Errorf("merged.Branch = %q, want %q", merged.Branch, "lab/zsh")
	}
	if !merged.Firewall {
		t.Error("merged.Firewall = false, want true")
	}
}

func TestMergeWithOverridesFlagsNotMutated(t *testing.T) {
	base := &lab.StartOptions{
		Features: []string{"node:20"},
		EnvVars:  map[string]string{"OLD": "val"},
	}
	flags := &lab.StartOptions{
		Features: []string{"go:1.23"},
		EnvVars:  map[string]string{"NEW": "val"},
	}
	changed := map[string]bool{
		"feature": true,
		"env":     true,
	}

	merged, _ := lab.MergeWithOverrides(base, flags, changed)

	// Mutating merged should not affect flags.
	merged.Features[0] = "MUTATED"
	merged.EnvVars["INJECTED"] = "bad"

	if flags.Features[0] != "go:1.23" {
		t.Errorf("flags.Features mutated to %v", flags.Features)
	}
	if len(flags.EnvVars) != 1 || flags.EnvVars["NEW"] != "val" {
		t.Errorf("flags.EnvVars mutated to %v", flags.EnvVars)
	}
}
