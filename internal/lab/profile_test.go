package lab_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/claudeup/claudeup-lab/internal/lab"
)

func TestSnapshotProfile(t *testing.T) {
	profilesDir := filepath.Join(t.TempDir(), "profiles")
	os.MkdirAll(profilesDir, 0o755)

	pm := lab.NewProfileManager(profilesDir)
	name, err := pm.Snapshot("test-123")
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}

	if name != "_lab-snapshot-test-123" {
		t.Errorf("name = %q, want %q", name, "_lab-snapshot-test-123")
	}

	// Verify profile file exists
	path := filepath.Join(profilesDir, name+".json")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Error("snapshot profile file should exist")
	}
}

func TestCleanupSnapshot(t *testing.T) {
	profilesDir := filepath.Join(t.TempDir(), "profiles")
	os.MkdirAll(profilesDir, 0o755)

	pm := lab.NewProfileManager(profilesDir)
	name, _ := pm.Snapshot("test-456")

	err := pm.CleanupSnapshot(name)
	if err != nil {
		t.Fatalf("CleanupSnapshot: %v", err)
	}

	path := filepath.Join(profilesDir, name+".json")
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("snapshot profile file should be deleted")
	}
}

func TestCleanupNonSnapshot(t *testing.T) {
	profilesDir := filepath.Join(t.TempDir(), "profiles")
	os.MkdirAll(profilesDir, 0o755)
	os.WriteFile(filepath.Join(profilesDir, "real-profile.json"), []byte("{}"), 0o644)

	pm := lab.NewProfileManager(profilesDir)
	err := pm.CleanupSnapshot("real-profile")
	if err != nil {
		t.Fatalf("CleanupSnapshot: %v", err)
	}

	// Should NOT delete non-snapshot profiles
	path := filepath.Join(profilesDir, "real-profile.json")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Error("non-snapshot profile should NOT be deleted")
	}
}

func TestIsSnapshot(t *testing.T) {
	if !lab.IsSnapshotProfile("_lab-snapshot-abc123") {
		t.Error("should recognize snapshot profile")
	}
	if lab.IsSnapshotProfile("my-real-profile") {
		t.Error("should not flag regular profile as snapshot")
	}
}
