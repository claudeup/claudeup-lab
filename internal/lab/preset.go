package lab

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Preset holds a reusable lab configuration.
type Preset struct {
	Name        string            `json:"name"`
	CreatedAt   time.Time         `json:"created_at"`
	Project     string            `json:"project"`
	Profile     string            `json:"profile"`
	Branch      string            `json:"branch"`
	LabName     string            `json:"name_flag"`
	Features    []string          `json:"features,omitempty"`
	BaseProfile string            `json:"base_profile,omitempty"`
	Firewall    bool              `json:"firewall,omitempty"`
	InitScript  string            `json:"init_script,omitempty"`
	EnvVars     map[string]string `json:"env_vars,omitempty"`
	EnvFile     string            `json:"env_file,omitempty"`
}

// PresetStore reads and writes preset JSON files in a directory.
type PresetStore struct {
	dir string
}

func NewPresetStore(dir string) *PresetStore {
	return &PresetStore{dir: dir}
}

func (s *PresetStore) Save(p *Preset) error {
	if err := validatePresetName(p.Name); err != nil {
		return err
	}

	if err := os.MkdirAll(s.dir, 0o700); err != nil {
		return fmt.Errorf("create preset directory: %w", err)
	}

	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal preset: %w", err)
	}

	path := filepath.Join(s.dir, p.Name+".json")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("write preset: %w", err)
	}

	return nil
}

func (s *PresetStore) Load(name string) (*Preset, error) {
	if err := validatePresetName(name); err != nil {
		return nil, err
	}

	path := filepath.Join(s.dir, name+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read preset %q: %w", name, err)
	}

	var p Preset
	if err := json.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("parse preset %q: %w", name, err)
	}

	return &p, nil
}

func (s *PresetStore) List() ([]*Preset, error) {
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		if os.IsNotExist(err) {
			return []*Preset{}, nil
		}
		return nil, fmt.Errorf("read preset directory: %w", err)
	}

	presets := make([]*Preset, 0)
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		name := strings.TrimSuffix(entry.Name(), ".json")
		p, err := s.Load(name)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: skipping unreadable preset %q: %v\n", name, err)
			continue
		}
		presets = append(presets, p)
	}

	return presets, nil
}

func (s *PresetStore) Delete(name string) error {
	if err := validatePresetName(name); err != nil {
		return err
	}

	path := filepath.Join(s.dir, name+".json")
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("delete preset %q: %w", name, err)
	}
	return nil
}

func validatePresetName(name string) error {
	if name == "" {
		return fmt.Errorf("preset name %q contains invalid characters (allowed: A-Z, a-z, 0-9, '.', '_', '-')", name)
	}
	if strings.HasPrefix(name, ".") {
		return fmt.Errorf("preset name %q contains invalid characters (allowed: A-Z, a-z, 0-9, '.', '_', '-')", name)
	}
	if strings.Contains(name, "/") || strings.Contains(name, "..") || strings.Contains(name, string(filepath.Separator)) {
		return fmt.Errorf("preset name %q contains invalid characters (allowed: A-Z, a-z, 0-9, '.', '_', '-')", name)
	}
	if !validNameRegex.MatchString(name) {
		return fmt.Errorf("preset name %q contains invalid characters (allowed: A-Z, a-z, 0-9, '.', '_', '-')", name)
	}
	return nil
}

// PresetFromStartOptions creates a Preset from a name and StartOptions.
func PresetFromStartOptions(name string, opts *StartOptions) *Preset {
	return &Preset{
		Name:        name,
		CreatedAt:   time.Now().UTC(),
		Project:     opts.Project,
		Profile:     opts.Profile,
		Branch:      opts.Branch,
		LabName:     opts.Name,
		Features:    opts.Features,
		BaseProfile: opts.BaseProfile,
		Firewall:    opts.Firewall,
		InitScript:  opts.InitScript,
		EnvVars:     opts.EnvVars,
		EnvFile:     opts.EnvFile,
	}
}

// ToStartOptions converts a Preset back to StartOptions.
func (p *Preset) ToStartOptions() *StartOptions {
	return &StartOptions{
		Project:     p.Project,
		Profile:     p.Profile,
		Branch:      p.Branch,
		Name:        p.LabName,
		Features:    p.Features,
		BaseProfile: p.BaseProfile,
		Firewall:    p.Firewall,
		InitScript:  p.InitScript,
		EnvVars:     p.EnvVars,
		EnvFile:     p.EnvFile,
	}
}

// FormatPresetFields renders a key:value aligned field listing for a preset.
// Only non-zero fields are included. Callers prepend their own header line.
func FormatPresetFields(p *Preset) string {
	var lines []string

	add := func(label, value string) {
		lines = append(lines, fmt.Sprintf("  %-14s%s", label+":", value))
	}

	if p.Project != "" {
		add("project", p.Project)
	}
	if p.Profile != "" {
		add("profile", p.Profile)
	}
	if p.Branch != "" {
		add("branch", p.Branch)
	}
	if p.LabName != "" {
		add("lab-name", p.LabName)
	}
	if p.BaseProfile != "" {
		add("base-profile", p.BaseProfile)
	}
	if p.Firewall {
		add("firewall", "true")
	}
	if p.InitScript != "" {
		add("init-script", p.InitScript)
	}
	for _, f := range p.Features {
		add("feature", f)
	}
	if len(p.EnvVars) > 0 {
		keys := make([]string, 0, len(p.EnvVars))
		for k := range p.EnvVars {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			add("env", k+"="+p.EnvVars[k])
		}
	}
	if p.EnvFile != "" {
		add("env-file", p.EnvFile)
	}

	return strings.Join(lines, "\n")
}
