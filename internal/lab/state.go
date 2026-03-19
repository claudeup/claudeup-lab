package lab

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// StartConfig records the fully resolved start flags for a lab instance.
// Stored as part of Metadata so that preset save --from-lab can reconstruct
// the exact configuration that was used to start the lab.
type StartConfig struct {
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

// Metadata holds persisted state for a single lab instance.
type Metadata struct {
	ID          string       `json:"id"`
	DisplayName string       `json:"display_name"`
	Project     string       `json:"project"`
	ProjectName string       `json:"project_name"`
	Profile     string       `json:"profile"`
	BareRepo    string       `json:"bare_repo"`
	Worktree    string       `json:"worktree"`
	Branch      string       `json:"branch"`
	Created     time.Time    `json:"created"`
	Snapshot    string       `json:"snapshot,omitempty"`
	StartConfig *StartConfig `json:"start_config,omitempty"`
}

// StateStore reads and writes lab metadata JSON files in a directory.
type StateStore struct {
	dir string
}

func NewStateStore(dir string) *StateStore {
	return &StateStore{dir: dir}
}

func (s *StateStore) Save(meta *Metadata) error {
	if err := os.MkdirAll(s.dir, 0o755); err != nil {
		return fmt.Errorf("create state directory: %w", err)
	}

	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal metadata: %w", err)
	}

	path := filepath.Join(s.dir, meta.ID+".json")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write metadata: %w", err)
	}

	return nil
}

func (s *StateStore) Load(id string) (*Metadata, error) {
	if err := validateID(id); err != nil {
		return nil, err
	}
	path := filepath.Join(s.dir, id+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read metadata for %s: %w", id, err)
	}

	var meta Metadata
	if err := json.Unmarshal(data, &meta); err != nil {
		return nil, fmt.Errorf("parse metadata for %s: %w", id, err)
	}

	return &meta, nil
}

func (s *StateStore) List() ([]*Metadata, error) {
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read state directory: %w", err)
	}

	var labs []*Metadata
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		id := strings.TrimSuffix(entry.Name(), ".json")
		meta, err := s.Load(id)
		if err != nil {
			continue
		}
		labs = append(labs, meta)
	}

	return labs, nil
}

func validateID(id string) error {
	if strings.Contains(id, "/") || strings.Contains(id, "..") || strings.Contains(id, string(filepath.Separator)) {
		return fmt.Errorf("invalid lab ID %q: must not contain path separators or '..'", id)
	}
	return nil
}

func (s *StateStore) Delete(id string) error {
	if err := validateID(id); err != nil {
		return err
	}
	path := filepath.Join(s.dir, id+".json")
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("delete metadata for %s: %w", id, err)
	}
	return nil
}
