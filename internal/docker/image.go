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

const imageBase = "ghcr.io/claudeup/claudeup-lab"

var buildVersion string

// SetVersion records the build version for constructing versioned image tags.
func SetVersion(v string) {
	buildVersion = v
}

func isReleaseVersion() bool {
	return buildVersion != "" && buildVersion != "dev"
}

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

// EnsureImage tries each tag from ImageTagsForPull in order, falling back
// to building from the embedded Dockerfile if all pulls fail. Returns the
// tag that is available locally for use by the caller.
func (im *ImageManager) EnsureImage(image string) (string, error) {
	if im.ExistsLocally(image) {
		return image, nil
	}

	tags := ImageTagsForPull()
	for _, tag := range tags {
		if im.ExistsLocally(tag) {
			return tag, nil
		}
		fmt.Printf("Pulling image %s...\n", tag)
		if err := im.pull(tag); err != nil {
			fmt.Printf("Pull failed (%v)\n", err)
			continue
		}
		return tag, nil
	}

	fmt.Println("All pulls failed, building from embedded Dockerfile...")
	if err := im.buildFallback(image); err != nil {
		return "", err
	}
	return image, nil
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
		"init-firewall.sh":      assets.InitFirewall,
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

// ImageTag returns the preferred image tag. If CLAUDEUP_LAB_IMAGE is set,
// it takes precedence. Otherwise returns a versioned tag for release builds
// or :latest for dev builds.
func ImageTag() string {
	tag := os.Getenv("CLAUDEUP_LAB_IMAGE")
	if tag != "" {
		return strings.TrimSpace(tag)
	}
	if isReleaseVersion() {
		return imageBase + ":v" + buildVersion
	}
	return DefaultImage
}

// ImageTagsForPull returns the ordered list of image tags to try when pulling.
// For release builds: versioned tag first, then :latest.
// For dev builds: only :latest.
func ImageTagsForPull() []string {
	if isReleaseVersion() {
		return []string{
			imageBase + ":v" + buildVersion,
			DefaultImage,
		}
	}
	return []string{DefaultImage}
}
