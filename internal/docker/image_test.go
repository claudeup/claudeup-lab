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

func TestImageTagReturnsVersionedTag(t *testing.T) {
	t.Setenv("CLAUDEUP_LAB_IMAGE", "")
	docker.SetVersion("1.2.3")
	defer docker.SetVersion("")

	got := docker.ImageTag()
	want := "ghcr.io/claudeup/claudeup-lab:v1.2.3"
	if got != want {
		t.Errorf("ImageTag() = %q, want %q", got, want)
	}
}

func TestImageTagDevVersionReturnsLatest(t *testing.T) {
	t.Setenv("CLAUDEUP_LAB_IMAGE", "")
	docker.SetVersion("dev")
	defer docker.SetVersion("")

	got := docker.ImageTag()
	want := "ghcr.io/claudeup/claudeup-lab:latest"
	if got != want {
		t.Errorf("ImageTag() = %q, want %q", got, want)
	}
}

func TestImageTagEmptyVersionReturnsLatest(t *testing.T) {
	t.Setenv("CLAUDEUP_LAB_IMAGE", "")
	docker.SetVersion("")

	got := docker.ImageTag()
	want := "ghcr.io/claudeup/claudeup-lab:latest"
	if got != want {
		t.Errorf("ImageTag() = %q, want %q", got, want)
	}
}

func TestImageTagEnvOverride(t *testing.T) {
	t.Setenv("CLAUDEUP_LAB_IMAGE", "my-registry.io/custom:tag")
	docker.SetVersion("1.2.3")
	defer docker.SetVersion("")

	got := docker.ImageTag()
	want := "my-registry.io/custom:tag"
	if got != want {
		t.Errorf("ImageTag() = %q, want %q", got, want)
	}
}

func TestImageTagsForPull(t *testing.T) {
	tests := []struct {
		name    string
		version string
		want    []string
	}{
		{
			name:    "release version returns versioned then latest",
			version: "1.2.3",
			want: []string{
				"ghcr.io/claudeup/claudeup-lab:v1.2.3",
				"ghcr.io/claudeup/claudeup-lab:latest",
			},
		},
		{
			name:    "dev version returns only latest",
			version: "dev",
			want:    []string{"ghcr.io/claudeup/claudeup-lab:latest"},
		},
		{
			name:    "empty version returns only latest",
			version: "",
			want:    []string{"ghcr.io/claudeup/claudeup-lab:latest"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			docker.SetVersion(tt.version)
			defer docker.SetVersion("")

			got := docker.ImageTagsForPull()
			if len(got) != len(tt.want) {
				t.Fatalf("ImageTagsForPull() returned %d tags, want %d: %v", len(got), len(tt.want), got)
			}
			for i, tag := range got {
				if tag != tt.want[i] {
					t.Errorf("ImageTagsForPull()[%d] = %q, want %q", i, tag, tt.want[i])
				}
			}
		})
	}
}
