package lab_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/claudeup/claudeup-lab/internal/lab"
)

func TestRenderDevcontainer(t *testing.T) {
	dir := t.TempDir()

	config := &lab.DevcontainerConfig{
		ProjectName:  "myapp",
		Profile:      "base",
		ID:           "abc-123-def",
		DisplayName:  "myapp-base",
		Image:        "ghcr.io/claudeup/claudeup-lab:latest",
		BareRepoPath: "/home/user/.claudeup-lab/repos/myapp.git",
		HomeDir:      "/home/user",
		GitUserName:  "Test User",
		GitUserEmail: "test@example.com",
	}

	err := lab.RenderDevcontainer(config, dir)
	if err != nil {
		t.Fatalf("RenderDevcontainer: %v", err)
	}

	outPath := filepath.Join(dir, ".devcontainer", "devcontainer.json")
	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "myapp-base") {
		t.Error("output should contain display name")
	}
	if !strings.Contains(content, "abc-123-def") {
		t.Error("output should contain lab ID")
	}
	if !strings.Contains(content, "ghcr.io/claudeup/claudeup-lab:latest") {
		t.Error("output should contain image")
	}

	// Verify it's valid JSON
	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}
}

func TestOptionalMountsSkipped(t *testing.T) {
	dir := t.TempDir()

	config := &lab.DevcontainerConfig{
		ProjectName:  "myapp",
		Profile:      "base",
		ID:           "abc-123",
		DisplayName:  "myapp-base",
		Image:        "test:latest",
		BareRepoPath: "/tmp/bare.git",
		HomeDir:      "/nonexistent/home",
	}

	err := lab.RenderDevcontainer(config, dir)
	if err != nil {
		t.Fatalf("RenderDevcontainer: %v", err)
	}

	outPath := filepath.Join(dir, ".devcontainer", "devcontainer.json")
	data, _ := os.ReadFile(outPath)
	content := string(data)

	// Optional mounts for nonexistent home should be absent
	if strings.Contains(content, ".claude-mem") {
		t.Error("should skip .claude-mem mount when dir doesn't exist")
	}
}

func TestFeatureInjection(t *testing.T) {
	dir := t.TempDir()

	config := &lab.DevcontainerConfig{
		ProjectName:  "myapp",
		Profile:      "base",
		ID:           "abc-123",
		DisplayName:  "myapp-base",
		Image:        "test:latest",
		BareRepoPath: "/tmp/bare.git",
		HomeDir:      t.TempDir(),
		Features:     []string{"go:1.23", "python"},
	}

	err := lab.RenderDevcontainer(config, dir)
	if err != nil {
		t.Fatalf("RenderDevcontainer: %v", err)
	}

	outPath := filepath.Join(dir, ".devcontainer", "devcontainer.json")
	data, _ := os.ReadFile(outPath)

	var parsed map[string]interface{}
	json.Unmarshal(data, &parsed)

	features, ok := parsed["features"].(map[string]interface{})
	if !ok {
		t.Fatal("features should be a map")
	}
	if _, ok := features["ghcr.io/devcontainers/features/go:1"]; !ok {
		t.Error("should contain go feature")
	}
}
