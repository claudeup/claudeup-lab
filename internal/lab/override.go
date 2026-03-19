package lab

import (
	"fmt"
	"sort"
	"strings"
)

// Override records a single field that was changed from its preset value.
type Override struct {
	Field    string // flag name, e.g. "branch"
	OldValue string // value from preset (formatted for display)
	NewValue string // value from CLI flag (formatted for display)
}

// MergeWithOverrides merges CLI flag values into a base StartOptions from a preset.
// Only fields whose flag names appear in the changed map are replaced. It returns
// the merged options and a list of overrides describing what changed.
func MergeWithOverrides(base *StartOptions, flags *StartOptions, changed map[string]bool) (*StartOptions, []Override) {
	merged := copyStartOptions(base)
	var overrides []Override

	type fieldDef struct {
		flag string
		get  func(*StartOptions) string
		set  func(*StartOptions, *StartOptions)
	}

	stringFields := []fieldDef{
		{"project", func(o *StartOptions) string { return o.Project }, func(m, f *StartOptions) { m.Project = f.Project }},
		{"profile", func(o *StartOptions) string { return o.Profile }, func(m, f *StartOptions) { m.Profile = f.Profile }},
		{"branch", func(o *StartOptions) string { return o.Branch }, func(m, f *StartOptions) { m.Branch = f.Branch }},
		{"name", func(o *StartOptions) string { return o.Name }, func(m, f *StartOptions) { m.Name = f.Name }},
		{"base-profile", func(o *StartOptions) string { return o.BaseProfile }, func(m, f *StartOptions) { m.BaseProfile = f.BaseProfile }},
		{"init-script", func(o *StartOptions) string { return o.InitScript }, func(m, f *StartOptions) { m.InitScript = f.InitScript }},
		{"env-file", func(o *StartOptions) string { return o.EnvFile }, func(m, f *StartOptions) { m.EnvFile = f.EnvFile }},
	}

	for _, f := range stringFields {
		if !changed[f.flag] {
			continue
		}
		oldVal := f.get(base)
		newVal := f.get(flags)
		f.set(merged, flags)
		overrides = append(overrides, Override{
			Field:    f.flag,
			OldValue: oldVal,
			NewValue: newVal,
		})
	}

	// Bool: firewall
	if changed["firewall"] {
		overrides = append(overrides, Override{
			Field:    "firewall",
			OldValue: fmt.Sprintf("%t", base.Firewall),
			NewValue: fmt.Sprintf("%t", flags.Firewall),
		})
		merged.Firewall = flags.Firewall
	}

	// Slice: feature -> Features
	if changed["feature"] {
		overrides = append(overrides, Override{
			Field:    "feature",
			OldValue: formatSlice(base.Features),
			NewValue: formatSlice(flags.Features),
		})
		merged.Features = copySlice(flags.Features)
	}

	// Map: env -> EnvVars
	if changed["env"] {
		overrides = append(overrides, Override{
			Field:    "env",
			OldValue: formatEnvMap(base.EnvVars),
			NewValue: formatEnvMap(flags.EnvVars),
		})
		merged.EnvVars = copyMap(flags.EnvVars)
	}

	return merged, overrides
}

// FormatOverrides renders a preset summary with diff markers for overridden fields.
func FormatOverrides(preset *Preset, overrides []Override) string {
	var lines []string
	lines = append(lines, fmt.Sprintf("Using preset '%s'", preset.Name))

	overrideMap := make(map[string]*Override)
	for i := range overrides {
		overrideMap[overrides[i].Field] = &overrides[i]
	}

	type fieldEntry struct {
		flag  string
		value string
	}

	// Collect fields from the preset in display order.
	var fields []fieldEntry
	addField := func(flag, value string) {
		fields = append(fields, fieldEntry{flag: flag, value: value})
	}

	if preset.Project != "" {
		addField("project", preset.Project)
	}
	if preset.Profile != "" {
		addField("profile", preset.Profile)
	}
	if preset.Branch != "" {
		addField("branch", preset.Branch)
	}
	if preset.LabName != "" {
		addField("name", preset.LabName)
	}
	if preset.BaseProfile != "" {
		addField("base-profile", preset.BaseProfile)
	}
	if preset.Firewall {
		addField("firewall", "true")
	}
	if preset.InitScript != "" {
		addField("init-script", preset.InitScript)
	}
	if len(preset.Features) > 0 {
		addField("feature", formatSlice(preset.Features))
	}
	if len(preset.EnvVars) > 0 {
		addField("env", formatEnvMap(preset.EnvVars))
	}
	if preset.EnvFile != "" {
		addField("env-file", preset.EnvFile)
	}

	// Track which overrides we've rendered (some add fields not in the preset).
	rendered := make(map[string]bool)

	for _, f := range fields {
		o, hasOverride := overrideMap[f.flag]
		if hasOverride {
			rendered[f.flag] = true
			if isDisplayableValue(f.flag, o.OldValue) {
				lines = append(lines, formatLine("-", f.flag, o.OldValue))
			}
			if isDisplayableValue(f.flag, o.NewValue) {
				lines = append(lines, formatLine("+", f.flag, o.NewValue))
			}
		} else {
			lines = append(lines, formatLine(" ", f.flag, f.value))
		}
	}

	// Render overrides for fields that were not in the preset (added fields).
	for _, o := range overrides {
		if rendered[o.Field] {
			continue
		}
		if isDisplayableValue(o.Field, o.OldValue) {
			lines = append(lines, formatLine("-", o.Field, o.OldValue))
		}
		if isDisplayableValue(o.Field, o.NewValue) {
			lines = append(lines, formatLine("+", o.Field, o.NewValue))
		}
	}

	return strings.Join(lines, "\n") + "\n"
}

// isDisplayableValue returns true if value is non-zero for the given field.
// For bool fields, "false" is the zero value.
func isDisplayableValue(flag, value string) bool {
	if value == "" {
		return false
	}
	if flag == "firewall" && value == "false" {
		return false
	}
	return true
}

func formatLine(prefix, flag, value string) string {
	return fmt.Sprintf("%s %-14s%s", prefix, flag+":", value)
}

func formatSlice(s []string) string {
	return strings.Join(s, ",")
}

func formatEnvMap(m map[string]string) string {
	if len(m) == 0 {
		return ""
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	pairs := make([]string, 0, len(m))
	for _, k := range keys {
		pairs = append(pairs, k+"="+m[k])
	}
	return strings.Join(pairs, ",")
}

func copyStartOptions(src *StartOptions) *StartOptions {
	dst := *src
	dst.Features = copySlice(src.Features)
	dst.EnvVars = copyMap(src.EnvVars)
	return &dst
}

func copySlice(s []string) []string {
	if s == nil {
		return nil
	}
	c := make([]string, len(s))
	copy(c, s)
	return c
}

func copyMap(m map[string]string) map[string]string {
	if m == nil {
		return nil
	}
	c := make(map[string]string, len(m))
	for k, v := range m {
		c[k] = v
	}
	return c
}
