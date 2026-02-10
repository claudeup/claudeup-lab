package lab

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const snapshotPrefix = "_lab-snapshot-"

// ProfileManager handles snapshotting and cleaning up claudeup profiles.
type ProfileManager struct {
	profilesDir string
}

func NewProfileManager(profilesDir string) *ProfileManager {
	return &ProfileManager{profilesDir: profilesDir}
}

// Snapshot saves the current claudeup config as a temporary profile.
// Returns the snapshot profile name.
func (pm *ProfileManager) Snapshot(labShortID string) (string, error) {
	name := snapshotPrefix + labShortID
	path := filepath.Join(pm.profilesDir, name+".json")

	cmd := exec.Command("claudeup", "profile", "save", name)
	if out, err := cmd.CombinedOutput(); err != nil {
		// claudeup not available -- write a minimal empty profile
		if writeErr := os.WriteFile(path, []byte("{}"), 0o644); writeErr != nil {
			return "", fmt.Errorf("claudeup profile save failed (%v: %s) and fallback write failed: %w",
				err, out, writeErr)
		}
	} else if _, statErr := os.Stat(path); statErr != nil {
		// claudeup succeeded but wrote to its default profiles dir, not ours.
		// Write a minimal profile in our configured dir so cleanup can find it.
		os.WriteFile(path, []byte("{}"), 0o644)
	}

	return name, nil
}

// CleanupSnapshot removes a snapshot profile file. Non-snapshot profiles
// are left untouched.
func (pm *ProfileManager) CleanupSnapshot(name string) error {
	if !IsSnapshotProfile(name) {
		return nil
	}

	path := filepath.Join(pm.profilesDir, name+".json")
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove snapshot profile: %w", err)
	}
	return nil
}

func IsSnapshotProfile(name string) bool {
	return strings.HasPrefix(name, snapshotPrefix)
}
