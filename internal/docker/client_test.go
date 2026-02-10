package docker_test

import (
	"testing"

	"github.com/claudeup/claudeup-lab/internal/docker"
)

func TestIsRunning(t *testing.T) {
	client := docker.NewClient()
	running := client.IsRunning()
	t.Logf("Docker running: %v", running)
}

func TestFindContainerNoMatch(t *testing.T) {
	client := docker.NewClient()
	id, err := client.FindContainer("nonexistent-label-value-that-doesnt-exist")
	if err != nil {
		t.Fatalf("FindContainer: %v", err)
	}
	if id != "" {
		t.Errorf("expected empty container ID, got %q", id)
	}
}

func TestListVolumesNoMatch(t *testing.T) {
	client := docker.NewClient()
	vols, err := client.ListVolumes("claudeup-lab-test-nonexistent-pattern")
	if err != nil {
		t.Fatalf("ListVolumes: %v", err)
	}
	if len(vols) != 0 {
		t.Errorf("expected 0 volumes, got %d", len(vols))
	}
}
