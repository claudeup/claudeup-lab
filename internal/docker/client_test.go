package docker_test

import (
	"os/exec"
	"testing"

	"github.com/claudeup/claudeup-lab/internal/docker"
)

func requireDocker(t *testing.T) {
	t.Helper()
	if err := exec.Command("docker", "info").Run(); err != nil {
		t.Skip("skipping: Docker is not available")
	}
}

func TestIsRunning(t *testing.T) {
	requireDocker(t)
	client := docker.NewClient()
	running := client.IsRunning()
	if !running {
		t.Error("expected Docker to be running")
	}
}

func TestFindContainerNoMatch(t *testing.T) {
	requireDocker(t)
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
	requireDocker(t)
	client := docker.NewClient()
	vols, err := client.ListVolumes("claudeup-lab-test-nonexistent-pattern")
	if err != nil {
		t.Fatalf("ListVolumes: %v", err)
	}
	if len(vols) != 0 {
		t.Errorf("expected 0 volumes, got %d", len(vols))
	}
}
