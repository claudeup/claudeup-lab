package docker_test

import (
	"testing"

	"github.com/claudeup/claudeup-lab/internal/docker"
)

func TestImageExistsLocally(t *testing.T) {
	requireDocker(t)
	im := docker.NewImageManager()
	exists := im.ExistsLocally("claudeup-test-image-that-should-not-exist:latest")
	if exists {
		t.Error("expected non-existent image to return false")
	}
}

func TestImageNameConstants(t *testing.T) {
	if docker.DefaultImage == "" {
		t.Error("DefaultImage should not be empty")
	}
}
