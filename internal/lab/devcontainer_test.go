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

func TestClaudeupHomeOverridesMountPaths(t *testing.T) {
	dir := t.TempDir()

	// Create a custom claudeup home with profiles and extension dirs
	customClaudeupHome := t.TempDir()
	os.MkdirAll(filepath.Join(customClaudeupHome, "profiles"), 0o755)
	os.MkdirAll(filepath.Join(customClaudeupHome, "ext"), 0o755)

	config := &lab.DevcontainerConfig{
		ProjectName:  "myapp",
		Profile:      "base",
		ID:           "abc-123",
		DisplayName:  "myapp-base",
		Image:        "test:latest",
		BareRepoPath: "/tmp/bare.git",
		HomeDir:      "/home/user",
		ClaudeupHome: customClaudeupHome,
	}

	err := lab.RenderDevcontainer(config, dir)
	if err != nil {
		t.Fatalf("RenderDevcontainer: %v", err)
	}

	outPath := filepath.Join(dir, ".devcontainer", "devcontainer.json")
	data, _ := os.ReadFile(outPath)
	content := string(data)

	// Bind mounts should use the custom claudeup home, not $HOME/.claudeup
	expectedProfilesSource := filepath.Join(customClaudeupHome, "profiles")
	if !strings.Contains(content, expectedProfilesSource) {
		t.Errorf("should mount profiles from custom CLAUDEUP_HOME %s, got:\n%s", expectedProfilesSource, content)
	}

	expectedExtSource := filepath.Join(customClaudeupHome, "ext")
	if !strings.Contains(content, expectedExtSource) {
		t.Errorf("should mount ext from custom CLAUDEUP_HOME %s, got:\n%s", expectedExtSource, content)
	}

	// Should NOT contain the default $HOME/.claudeup path
	defaultPath := filepath.Join("/home/user", ".claudeup", "profiles")
	if strings.Contains(content, defaultPath) {
		t.Error("should not use default $HOME/.claudeup path when ClaudeupHome is set")
	}
}

func TestDefaultClaudeupHomeFallback(t *testing.T) {
	dir := t.TempDir()

	// Create profiles/local under the fake home's .claudeup
	fakeHome := t.TempDir()
	os.MkdirAll(filepath.Join(fakeHome, ".claudeup", "profiles"), 0o755)
	os.MkdirAll(filepath.Join(fakeHome, ".claudeup", "ext"), 0o755)

	config := &lab.DevcontainerConfig{
		ProjectName:  "myapp",
		Profile:      "base",
		ID:           "abc-123",
		DisplayName:  "myapp-base",
		Image:        "test:latest",
		BareRepoPath: "/tmp/bare.git",
		HomeDir:      fakeHome,
		ClaudeupHome: "", // empty = should fall back to HomeDir/.claudeup
	}

	err := lab.RenderDevcontainer(config, dir)
	if err != nil {
		t.Fatalf("RenderDevcontainer: %v", err)
	}

	outPath := filepath.Join(dir, ".devcontainer", "devcontainer.json")
	data, _ := os.ReadFile(outPath)
	content := string(data)

	expectedProfilesSource := filepath.Join(fakeHome, ".claudeup", "profiles")
	if !strings.Contains(content, expectedProfilesSource) {
		t.Errorf("should fall back to $HOME/.claudeup/profiles when ClaudeupHome is empty, expected %s in:\n%s",
			expectedProfilesSource, content)
	}
}

func TestClaudeupHomeHelper(t *testing.T) {
	t.Run("returns CLAUDEUP_HOME when set", func(t *testing.T) {
		t.Setenv("CLAUDEUP_HOME", "/custom/claudeup")
		result := lab.ClaudeupHome()
		if result != "/custom/claudeup" {
			t.Errorf("expected /custom/claudeup, got %s", result)
		}
	})

	t.Run("falls back to HOME/.claudeup when unset", func(t *testing.T) {
		t.Setenv("CLAUDEUP_HOME", "")
		home := os.Getenv("HOME")
		expected := filepath.Join(home, ".claudeup")
		result := lab.ClaudeupHome()
		if result != expected {
			t.Errorf("expected %s, got %s", expected, result)
		}
	})
}

func TestVSCodeCustomizations(t *testing.T) {
	dir := t.TempDir()

	config := &lab.DevcontainerConfig{
		ProjectName:  "myapp",
		Profile:      "base",
		ID:           "abc-123",
		DisplayName:  "myapp-base",
		Image:        "test:latest",
		BareRepoPath: "/tmp/bare.git",
		HomeDir:      t.TempDir(),
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

	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	customizations, ok := parsed["customizations"].(map[string]interface{})
	if !ok {
		t.Fatal("devcontainer.json should have a 'customizations' key")
	}

	vscode, ok := customizations["vscode"].(map[string]interface{})
	if !ok {
		t.Fatal("customizations should have a 'vscode' key")
	}

	t.Run("extensions", func(t *testing.T) {
		extensions, ok := vscode["extensions"].([]interface{})
		if !ok {
			t.Fatal("vscode customizations should have an 'extensions' array")
		}

		want := []string{
			"anthropic.claude-code",
			"dbaeumer.vscode-eslint",
			"esbenp.prettier-vscode",
			"eamodio.gitlens",
		}

		extSet := make(map[string]bool)
		for _, ext := range extensions {
			extSet[ext.(string)] = true
		}

		for _, w := range want {
			if !extSet[w] {
				t.Errorf("extensions should include %q, got: %v", w, extensions)
			}
		}
	})

	t.Run("settings", func(t *testing.T) {
		settings, ok := vscode["settings"].(map[string]interface{})
		if !ok {
			t.Fatal("vscode customizations should have a 'settings' map")
		}

		defaultProfile, ok := settings["terminal.integrated.defaultProfile.linux"].(string)
		if !ok || defaultProfile != "zsh" {
			t.Errorf("default terminal profile should be 'zsh', got: %v", settings["terminal.integrated.defaultProfile.linux"])
		}
	})
}

func TestFirewallEnabled(t *testing.T) {
	dir := t.TempDir()

	config := &lab.DevcontainerConfig{
		ProjectName:  "myapp",
		Profile:      "base",
		ID:           "abc-123",
		DisplayName:  "myapp-base",
		Image:        "test:latest",
		BareRepoPath: "/tmp/bare.git",
		HomeDir:      t.TempDir(),
		Firewall:     true,
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

	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	// runArgs should include NET_ADMIN and NET_RAW capabilities
	runArgs, ok := parsed["runArgs"].([]interface{})
	if !ok {
		t.Fatal("devcontainer.json should have 'runArgs' when firewall is enabled")
	}
	hasNetAdmin := false
	hasNetRaw := false
	for _, arg := range runArgs {
		switch arg {
		case "--cap-add=NET_ADMIN":
			hasNetAdmin = true
		case "--cap-add=NET_RAW":
			hasNetRaw = true
		}
	}
	if !hasNetAdmin {
		t.Error("runArgs should include --cap-add=NET_ADMIN")
	}
	if !hasNetRaw {
		t.Error("runArgs should include --cap-add=NET_RAW")
	}

	// postStartCommand should run the firewall script
	postStart, ok := parsed["postStartCommand"].(string)
	if !ok {
		t.Fatal("devcontainer.json should have 'postStartCommand' when firewall is enabled")
	}
	if !strings.Contains(postStart, "init-firewall.sh") {
		t.Errorf("postStartCommand should reference init-firewall.sh, got: %s", postStart)
	}

}

func TestFirewallDisabledByDefault(t *testing.T) {
	dir := t.TempDir()

	config := &lab.DevcontainerConfig{
		ProjectName:  "myapp",
		Profile:      "base",
		ID:           "abc-123",
		DisplayName:  "myapp-base",
		Image:        "test:latest",
		BareRepoPath: "/tmp/bare.git",
		HomeDir:      t.TempDir(),
	}

	err := lab.RenderDevcontainer(config, dir)
	if err != nil {
		t.Fatalf("RenderDevcontainer: %v", err)
	}

	outPath := filepath.Join(dir, ".devcontainer", "devcontainer.json")
	data, _ := os.ReadFile(outPath)
	content := string(data)

	if strings.Contains(content, "runArgs") {
		t.Error("devcontainer.json should not have runArgs when firewall is disabled")
	}
	if strings.Contains(content, "postStartCommand") {
		t.Error("devcontainer.json should not have postStartCommand when firewall is disabled")
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
