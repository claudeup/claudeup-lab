package docker_test

import (
	"testing"

	"github.com/claudeup/claudeup-lab/internal/docker"
)

func TestImageExistsLocally(t *testing.T) {
	im := docker.NewImageManager()
	exists := im.ExistsLocally("docker.io/library/hello-world:latest")
	t.Logf("hello-world exists locally: %v", exists)
}

func TestImageNameConstants(t *testing.T) {
	if docker.DefaultImage == "" {
		t.Error("DefaultImage should not be empty")
	}
}
