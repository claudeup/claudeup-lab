package docker

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	assets "github.com/claudeup/claudeup-lab/embed"
)

const DefaultImage = "ghcr.io/claudeup/claudeup-lab:latest"

// ImageManager handles pulling and building the base container image.
type ImageManager struct{}

func NewImageManager() *ImageManager {
	return &ImageManager{}
}

func (im *ImageManager) ExistsLocally(image string) bool {
	cmd := exec.Command("docker", "image", "inspect", image)
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Run() == nil
}

// EnsureImage pulls the image from the registry, falling back to building
// from the embedded Dockerfile if the pull fails.
func (im *ImageManager) EnsureImage(image string) error {
	if im.ExistsLocally(image) {
		return nil
	}

	fmt.Printf("Pulling image %s...\n", image)
	if err := im.pull(image); err != nil {
		fmt.Printf("Pull failed (%v), building from embedded Dockerfile...\n", err)
		return im.buildFallback(image)
	}

	return nil
}

func (im *ImageManager) pull(image string) error {
	cmd := exec.Command("docker", "pull", image)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (im *ImageManager) buildFallback(tag string) error {
	dir, err := os.MkdirTemp("", "claudeup-lab-build-*")
	if err != nil {
		return fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(dir)

	files := map[string][]byte{
		"Dockerfile":            assets.Dockerfile,
		"init-claude-config.sh": assets.InitClaudeConfig,
		"init-config-repo.sh":   assets.InitConfigRepo,
		"init-claudeup.sh":      assets.InitClaudeup,
	}
	for name, content := range files {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, content, 0o755); err != nil {
			return fmt.Errorf("write %s: %w", name, err)
		}
	}

	fmt.Println("Building image from embedded Dockerfile...")
	cmd := exec.Command("docker", "build", "-t", tag, dir)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker build: %w", err)
	}

	return nil
}

// ImageTag returns the configured image tag, allowing override via
// the CLAUDEUP_LAB_IMAGE environment variable.
func ImageTag() string {
	tag := os.Getenv("CLAUDEUP_LAB_IMAGE")
	if tag != "" {
		return strings.TrimSpace(tag)
	}
	return DefaultImage
}
