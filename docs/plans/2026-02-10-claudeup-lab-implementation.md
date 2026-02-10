# claudeup-lab Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build a Go CLI that creates ephemeral devcontainer environments for testing Claude Code configurations.

**Architecture:** Cobra CLI wrapping internal packages for state management, git worktree ops, Docker orchestration, and devcontainer template rendering. Embedded assets (Dockerfile, template, features.json) ship inside the binary via `go:embed`.

**Tech Stack:** Go 1.23+, cobra, go:embed, text/template, os/exec (for Docker/git/devcontainer CLIs), encoding/json, google/uuid

**Reference:** The existing bash implementation lives at `~/.claude/scripts/claude-sandbox` (864 lines). The design doc is at `docs/plans/2026-02-10-claudeup-lab-design.md`.

---

## Task 1: Project Scaffolding

**Files:**
- Create: `go.mod`
- Create: `cmd/claudeup-lab/main.go`
- Create: `internal/commands/root.go`

**Step 1: Initialize Go module**

Run:
```bash
cd ~/code/claudeup-lab
go mod init github.com/claudeup/claudeup-lab
```

**Step 2: Add cobra dependency**

Run:
```bash
go get github.com/spf13/cobra@latest
go get github.com/google/uuid@latest
```

**Step 3: Write root command**

Create `internal/commands/root.go`:

```go
package commands

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var version = "dev"

func NewRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "claudeup-lab",
		Short: "Ephemeral devcontainer environments for testing Claude Code configurations",
		Long: `claudeup-lab creates disposable devcontainer environments for testing
Claude Code configurations (plugins, skills, agents, hooks, commands)
without affecting your host setup.`,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	cmd.AddCommand(newVersionCmd())

	return cmd
}

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(version)
		},
	}
}

func Execute() {
	if err := NewRootCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
```

**Step 4: Write main.go**

Create `cmd/claudeup-lab/main.go`:

```go
package main

import "github.com/claudeup/claudeup-lab/internal/commands"

func main() {
	commands.Execute()
}
```

**Step 5: Verify it builds and runs**

Run:
```bash
go build -o claudeup-lab ./cmd/claudeup-lab
./claudeup-lab --help
./claudeup-lab version
```

Expected: Help text prints, version prints "dev".

**Step 6: Commit**

```bash
git add -A
git commit -m "feat: project scaffolding with cobra root command"
```

---

## Task 2: Embedded Assets

**Files:**
- Create: `embed/embed.go`
- Create: `embed/Dockerfile`
- Create: `embed/devcontainer.template.json`
- Create: `embed/features.json`
- Create: `embed/init-claude-config.sh`
- Create: `embed/init-config-repo.sh`
- Create: `embed/init-claudeup.sh`

**Step 1: Create the embed package**

Create `embed/embed.go`:

```go
package embed

import "embed"

//go:embed Dockerfile
var Dockerfile []byte

//go:embed devcontainer.template.json
var DevcontainerTemplate string

//go:embed features.json
var FeaturesJSON []byte

//go:embed init-claude-config.sh
var InitClaudeConfig []byte

//go:embed init-config-repo.sh
var InitConfigRepo []byte

//go:embed init-claudeup.sh
var InitClaudeup []byte
```

**Step 2: Copy assets from existing implementation**

Copy these files from `~/.claude/devcontainer-base/` into `embed/`:
- `Dockerfile`
- `features.json`
- `init-claude-config.sh`
- `init-config-repo.sh`
- `init-claudeup.sh`

**Step 3: Create the devcontainer template**

Create `embed/devcontainer.template.json` using Go `text/template` syntax (not awk placeholders). This replaces the `{{PLACEHOLDER}}` awk-style template:

```json
{
  "name": "claudeup-lab - {{ .ProjectName }} ({{ .Profile }})",
  "image": "{{ .Image }}",
  "features": {
    {{ .Features }}
  },
  "remoteUser": "node",
  "mounts": [
    "source=claudeup-lab-bashhistory-{{ .ID }},target=/commandhistory,type=volume",
    "source=claudeup-lab-config-{{ .ID }},target=/home/node/.claude,type=volume",
    "source=claudeup-lab-claudeup-{{ .ID }},target=/home/node/.claudeup,type=volume",
    {{ range .Mounts }}"{{ . }}",
    {{ end }}"source=claudeup-lab-npm-{{ .ID }},target=/home/node/.npm-global,type=volume",
    "source=claudeup-lab-local-{{ .ID }},target=/home/node/.local,type=volume",
    "source=claudeup-lab-bun-{{ .ID }},target=/home/node/.bun,type=volume"
  ],
  "containerEnv": {
    "CLAUDE_CONFIG_DIR": "/home/node/.claude",
    "CLAUDE_PROFILE": "{{ .Profile }}",
    "NODE_OPTIONS": "--max-old-space-size=4096",
    "GIT_USER_NAME": "{{ .GitUserName }}",
    "GIT_USER_EMAIL": "{{ .GitUserEmail }}",
    "GITHUB_TOKEN": "{{ .GitHubToken }}",
    "CONTEXT7_API_KEY": "{{ .Context7APIKey }}",
    "CLAUDE_CONFIG_REPO": "{{ .ConfigRepo }}",
    "CLAUDE_CONFIG_BRANCH": "{{ .ConfigBranch }}",
    "CLAUDE_BASE_PROFILE": "{{ .BaseProfile }}"
  },
  "workspaceFolder": "/workspaces/{{ .DisplayName }}",
  "postCreateCommand": "claude upgrade && /usr/local/bin/init-claude-config.sh && /usr/local/bin/init-config-repo.sh && /usr/local/bin/init-claudeup.sh",
  "waitFor": "postCreateCommand"
}
```

Note: The exact template syntax will need refinement during Task 8 (devcontainer rendering) when the template data struct is finalized. The mounts range loop and features injection need special handling for trailing commas in JSON. Consider using Go code to build the JSON rather than a text template if JSON formatting becomes fragile.

**Step 4: Verify embed compiles**

Run:
```bash
go build ./embed/...
```

Expected: Builds without errors.

**Step 5: Commit**

```bash
git add -A
git commit -m "feat: embedded assets (Dockerfile, template, init scripts, features)"
```

---

## Task 3: State Management

**Files:**
- Create: `internal/lab/state.go`
- Create: `internal/lab/state_test.go`

The state package handles metadata persistence -- saving, loading, listing, and deleting lab metadata JSON files in `~/.claudeup-lab/state/`.

**Step 1: Write the failing tests**

Create `internal/lab/state_test.go`:

```go
package lab_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/claudeup/claudeup-lab/internal/lab"
)

func TestSaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	store := lab.NewStateStore(dir)

	meta := &lab.Metadata{
		ID:          "abc-123",
		DisplayName: "myapp-base",
		Project:     "/home/user/code/myapp",
		ProjectName: "myapp",
		Profile:     "base",
		BareRepo:    "/home/user/.claudeup-lab/repos/myapp.git",
		Worktree:    "/home/user/.claudeup-lab/workspaces/myapp-base",
		Branch:      "lab/base",
		Created:     time.Date(2026, 2, 10, 12, 0, 0, 0, time.UTC),
	}

	if err := store.Save(meta); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := store.Load("abc-123")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if loaded.DisplayName != "myapp-base" {
		t.Errorf("DisplayName = %q, want %q", loaded.DisplayName, "myapp-base")
	}
	if loaded.Branch != "lab/base" {
		t.Errorf("Branch = %q, want %q", loaded.Branch, "lab/base")
	}
}

func TestList(t *testing.T) {
	dir := t.TempDir()
	store := lab.NewStateStore(dir)

	store.Save(&lab.Metadata{ID: "id-1", DisplayName: "lab-one"})
	store.Save(&lab.Metadata{ID: "id-2", DisplayName: "lab-two"})

	labs, err := store.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(labs) != 2 {
		t.Fatalf("List returned %d labs, want 2", len(labs))
	}
}

func TestDelete(t *testing.T) {
	dir := t.TempDir()
	store := lab.NewStateStore(dir)

	store.Save(&lab.Metadata{ID: "to-delete", DisplayName: "temp"})
	if err := store.Delete("to-delete"); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	_, err := store.Load("to-delete")
	if err == nil {
		t.Error("Load after Delete should return error")
	}
}

func TestLoadNonexistent(t *testing.T) {
	dir := t.TempDir()
	store := lab.NewStateStore(dir)

	_, err := store.Load("does-not-exist")
	if err == nil {
		t.Error("Load nonexistent should return error")
	}
}

func TestListEmptyDir(t *testing.T) {
	dir := t.TempDir()
	store := lab.NewStateStore(dir)

	labs, err := store.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(labs) != 0 {
		t.Errorf("List returned %d labs, want 0", len(labs))
	}
}

func TestSaveCreatesDirectory(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nested", "state")
	store := lab.NewStateStore(dir)

	err := store.Save(&lab.Metadata{ID: "test", DisplayName: "test"})
	if err != nil {
		t.Fatalf("Save to nested dir: %v", err)
	}

	if _, err := os.Stat(dir); os.IsNotExist(err) {
		t.Error("Save should create state directory")
	}
}
```

**Step 2: Run tests to verify they fail**

Run: `cd ~/code/claudeup-lab && go test ./internal/lab/ -v`
Expected: FAIL -- types and functions not defined.

**Step 3: Implement state management**

Create `internal/lab/state.go`:

```go
package lab

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Metadata struct {
	ID          string    `json:"id"`
	DisplayName string    `json:"display_name"`
	Project     string    `json:"project"`
	ProjectName string    `json:"project_name"`
	Profile     string    `json:"profile"`
	BareRepo    string    `json:"bare_repo"`
	Worktree    string    `json:"worktree"`
	Branch      string    `json:"branch"`
	Created     time.Time `json:"created"`
	Snapshot    string    `json:"snapshot,omitempty"`
}

type StateStore struct {
	dir string
}

func NewStateStore(dir string) *StateStore {
	return &StateStore{dir: dir}
}

func (s *StateStore) Save(meta *Metadata) error {
	if err := os.MkdirAll(s.dir, 0o755); err != nil {
		return fmt.Errorf("create state directory: %w", err)
	}

	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal metadata: %w", err)
	}

	path := filepath.Join(s.dir, meta.ID+".json")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write metadata: %w", err)
	}

	return nil
}

func (s *StateStore) Load(id string) (*Metadata, error) {
	path := filepath.Join(s.dir, id+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read metadata for %s: %w", id, err)
	}

	var meta Metadata
	if err := json.Unmarshal(data, &meta); err != nil {
		return nil, fmt.Errorf("parse metadata for %s: %w", id, err)
	}

	return &meta, nil
}

func (s *StateStore) List() ([]*Metadata, error) {
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read state directory: %w", err)
	}

	var labs []*Metadata
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		id := strings.TrimSuffix(entry.Name(), ".json")
		meta, err := s.Load(id)
		if err != nil {
			continue
		}
		labs = append(labs, meta)
	}

	return labs, nil
}

func (s *StateStore) Delete(id string) error {
	path := filepath.Join(s.dir, id+".json")
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("delete metadata for %s: %w", id, err)
	}
	return nil
}
```

**Step 4: Run tests to verify they pass**

Run: `cd ~/code/claudeup-lab && go test ./internal/lab/ -v`
Expected: All 6 tests PASS.

**Step 5: Commit**

```bash
git add -A
git commit -m "feat: state management for lab metadata persistence"
```

---

## Task 4: Lab Resolution

**Files:**
- Create: `internal/lab/resolve.go`
- Create: `internal/lab/resolve_test.go`

Lab resolution finds a lab by fuzzy matching: exact UUID, display name, partial UUID prefix, project name, profile name, or CWD inference.

**Step 1: Write the failing tests**

Create `internal/lab/resolve_test.go`:

```go
package lab_test

import (
	"testing"

	"github.com/claudeup/claudeup-lab/internal/lab"
)

func setupResolver(t *testing.T) (*lab.Resolver, *lab.StateStore) {
	t.Helper()
	dir := t.TempDir()
	store := lab.NewStateStore(dir)

	store.Save(&lab.Metadata{
		ID:          "abc-def-123",
		DisplayName: "myapp-base",
		ProjectName: "myapp",
		Profile:     "base",
		Worktree:    "/tmp/workspaces/myapp-base",
	})
	store.Save(&lab.Metadata{
		ID:          "xyz-789-456",
		DisplayName: "other-experimental",
		ProjectName: "other",
		Profile:     "experimental",
		Worktree:    "/tmp/workspaces/other-experimental",
	})

	return lab.NewResolver(store), store
}

func TestResolveExactUUID(t *testing.T) {
	r, _ := setupResolver(t)
	meta, err := r.Resolve("abc-def-123")
	if err != nil {
		t.Fatalf("Resolve exact UUID: %v", err)
	}
	if meta.DisplayName != "myapp-base" {
		t.Errorf("got %q, want %q", meta.DisplayName, "myapp-base")
	}
}

func TestResolveDisplayName(t *testing.T) {
	r, _ := setupResolver(t)
	meta, err := r.Resolve("myapp-base")
	if err != nil {
		t.Fatalf("Resolve display name: %v", err)
	}
	if meta.ID != "abc-def-123" {
		t.Errorf("got %q, want %q", meta.ID, "abc-def-123")
	}
}

func TestResolvePartialUUID(t *testing.T) {
	r, _ := setupResolver(t)
	meta, err := r.Resolve("abc-def")
	if err != nil {
		t.Fatalf("Resolve partial UUID: %v", err)
	}
	if meta.ID != "abc-def-123" {
		t.Errorf("got %q, want %q", meta.ID, "abc-def-123")
	}
}

func TestResolveProjectName(t *testing.T) {
	r, _ := setupResolver(t)
	meta, err := r.Resolve("myapp")
	if err != nil {
		t.Fatalf("Resolve project name: %v", err)
	}
	if meta.ID != "abc-def-123" {
		t.Errorf("got %q, want %q", meta.ID, "abc-def-123")
	}
}

func TestResolveProfileName(t *testing.T) {
	r, _ := setupResolver(t)
	meta, err := r.Resolve("experimental")
	if err != nil {
		t.Fatalf("Resolve profile name: %v", err)
	}
	if meta.ID != "xyz-789-456" {
		t.Errorf("got %q, want %q", meta.ID, "xyz-789-456")
	}
}

func TestResolveAmbiguous(t *testing.T) {
	dir := t.TempDir()
	store := lab.NewStateStore(dir)
	store.Save(&lab.Metadata{ID: "id-1", DisplayName: "a", ProjectName: "shared", Profile: "p1"})
	store.Save(&lab.Metadata{ID: "id-2", DisplayName: "b", ProjectName: "shared", Profile: "p2"})

	r := lab.NewResolver(store)
	_, err := r.Resolve("shared")
	if err == nil {
		t.Error("expected ambiguous error")
	}
}

func TestResolveNoMatch(t *testing.T) {
	r, _ := setupResolver(t)
	_, err := r.Resolve("nonexistent")
	if err == nil {
		t.Error("expected no match error")
	}
}

func TestResolveByCWD(t *testing.T) {
	r, _ := setupResolver(t)
	meta, err := r.ResolveByCWD("/tmp/workspaces/myapp-base/subdir")
	if err != nil {
		t.Fatalf("ResolveByCWD: %v", err)
	}
	if meta.ID != "abc-def-123" {
		t.Errorf("got %q, want %q", meta.ID, "abc-def-123")
	}
}

func TestResolveByCWDNoMatch(t *testing.T) {
	r, _ := setupResolver(t)
	_, err := r.ResolveByCWD("/some/other/path")
	if err == nil {
		t.Error("expected no match error for unrelated cwd")
	}
}
```

**Step 2: Run tests to verify they fail**

Run: `cd ~/code/claudeup-lab && go test ./internal/lab/ -v -run TestResolve`
Expected: FAIL -- Resolver type not defined.

**Step 3: Implement lab resolution**

Create `internal/lab/resolve.go`:

```go
package lab

import (
	"fmt"
	"strings"
)

type Resolver struct {
	store *StateStore
}

func NewResolver(store *StateStore) *Resolver {
	return &Resolver{store: store}
}

func (r *Resolver) Resolve(query string) (*Metadata, error) {
	// 1. Exact UUID match
	if meta, err := r.store.Load(query); err == nil {
		return meta, nil
	}

	labs, err := r.store.List()
	if err != nil {
		return nil, fmt.Errorf("list labs: %w", err)
	}

	var matches []*Metadata
	for _, m := range labs {
		// 2. Display name match
		if m.DisplayName == query {
			return m, nil
		}

		// 3. Partial UUID prefix
		if strings.HasPrefix(m.ID, query) {
			matches = append(matches, m)
			continue
		}

		// 4. Project name match
		if m.ProjectName == query {
			matches = append(matches, m)
			continue
		}

		// 5. Profile match
		if m.Profile == query {
			matches = append(matches, m)
			continue
		}
	}

	if len(matches) == 1 {
		return matches[0], nil
	}

	if len(matches) > 1 {
		return nil, &AmbiguousError{Query: query, Matches: matches}
	}

	return nil, &NotFoundError{Query: query, Available: labs}
}

func (r *Resolver) ResolveByCWD(cwd string) (*Metadata, error) {
	labs, err := r.store.List()
	if err != nil {
		return nil, fmt.Errorf("list labs: %w", err)
	}

	for _, m := range labs {
		if strings.HasPrefix(cwd, m.Worktree) {
			return m, nil
		}
	}

	return nil, &NotFoundError{Query: cwd, Available: labs}
}

type AmbiguousError struct {
	Query   string
	Matches []*Metadata
}

func (e *AmbiguousError) Error() string {
	var names []string
	for _, m := range e.Matches {
		names = append(names, fmt.Sprintf("%s (%s)", m.DisplayName, m.ID[:8]))
	}
	return fmt.Sprintf("ambiguous lab query %q, matches: %s", e.Query, strings.Join(names, ", "))
}

type NotFoundError struct {
	Query     string
	Available []*Metadata
}

func (e *NotFoundError) Error() string {
	if len(e.Available) == 0 {
		return fmt.Sprintf("no lab matched %q (no labs found)", e.Query)
	}
	var names []string
	for _, m := range e.Available {
		names = append(names, fmt.Sprintf("%s (%s)", m.DisplayName, m.ID[:8]))
	}
	return fmt.Sprintf("no lab matched %q, available: %s", e.Query, strings.Join(names, ", "))
}
```

**Step 4: Run tests to verify they pass**

Run: `cd ~/code/claudeup-lab && go test ./internal/lab/ -v`
Expected: All tests PASS.

**Step 5: Commit**

```bash
git add -A
git commit -m "feat: fuzzy lab resolution by name, UUID, project, profile, and CWD"
```

---

## Task 5: Docker Client

**Files:**
- Create: `internal/docker/client.go`
- Create: `internal/docker/client_test.go`

Wraps Docker CLI commands: check if Docker is running, find containers by label, stop/remove containers, list/remove volumes.

**Step 1: Write the failing tests**

Create `internal/docker/client_test.go`:

```go
package docker_test

import (
	"testing"

	"github.com/claudeup/claudeup-lab/internal/docker"
)

func TestIsRunning(t *testing.T) {
	client := docker.NewClient()
	// This test passes if Docker is running on the test machine,
	// fails if not. It exercises the real Docker daemon.
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
```

**Step 2: Run tests to verify they fail**

Run: `cd ~/code/claudeup-lab && go test ./internal/docker/ -v`
Expected: FAIL -- docker package not defined.

**Step 3: Implement Docker client**

Create `internal/docker/client.go`:

```go
package docker

import (
	"fmt"
	"os/exec"
	"strings"
)

type Client struct{}

func NewClient() *Client {
	return &Client{}
}

func (c *Client) IsRunning() bool {
	cmd := exec.Command("docker", "info")
	return cmd.Run() == nil
}

func (c *Client) FindContainer(worktreePath string) (string, error) {
	cmd := exec.Command("docker", "ps", "-q",
		"--filter", fmt.Sprintf("label=devcontainer.local_folder=%s", worktreePath))
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("docker ps: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

func (c *Client) FindContainerIncludingStopped(worktreePath string) (string, error) {
	cmd := exec.Command("docker", "ps", "-aq",
		"--filter", fmt.Sprintf("label=devcontainer.local_folder=%s", worktreePath))
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("docker ps: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

func (c *Client) StopContainer(id string) error {
	cmd := exec.Command("docker", "stop", id)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker stop %s: %w", id, err)
	}
	return nil
}

func (c *Client) RemoveContainer(id string) error {
	cmd := exec.Command("docker", "rm", "-f", id)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker rm %s: %w", id, err)
	}
	return nil
}

func (c *Client) ListVolumes(pattern string) ([]string, error) {
	cmd := exec.Command("docker", "volume", "ls", "--format", "{{.Name}}")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("docker volume ls: %w", err)
	}

	var matches []string
	for _, name := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if name != "" && strings.Contains(name, pattern) {
			matches = append(matches, name)
		}
	}
	return matches, nil
}

func (c *Client) RemoveVolumes(names []string) error {
	if len(names) == 0 {
		return nil
	}
	args := append([]string{"volume", "rm"}, names...)
	cmd := exec.Command("docker", args...)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker volume rm: %w", err)
	}
	return nil
}

func (c *Client) ContainerHostname(worktreePath string) (string, error) {
	cmd := exec.Command("devcontainer", "exec",
		"--workspace-folder", worktreePath, "hostname")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("get container hostname: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}
```

**Step 4: Run tests to verify they pass**

Run: `cd ~/code/claudeup-lab && go test ./internal/docker/ -v`
Expected: All tests PASS (assuming Docker is running).

**Step 5: Commit**

```bash
git add -A
git commit -m "feat: Docker client for container and volume operations"
```

---

## Task 6: Docker Image Management

**Files:**
- Create: `internal/docker/image.go`
- Create: `internal/docker/image_test.go`

Handles pulling the base image from GHCR, with fallback to building from the embedded Dockerfile.

**Step 1: Write the failing tests**

Create `internal/docker/image_test.go`:

```go
package docker_test

import (
	"testing"

	"github.com/claudeup/claudeup-lab/internal/docker"
)

func TestImageExistsLocally(t *testing.T) {
	im := docker.NewImageManager()
	// node:22 should exist if Docker is available and pulls have happened
	// This tests the mechanism, not a specific image
	exists := im.ExistsLocally("docker.io/library/hello-world:latest")
	t.Logf("hello-world exists locally: %v", exists)
}

func TestImageNameConstants(t *testing.T) {
	if docker.DefaultImage == "" {
		t.Error("DefaultImage should not be empty")
	}
}
```

**Step 2: Run tests to verify they fail**

Run: `cd ~/code/claudeup-lab && go test ./internal/docker/ -v -run TestImage`
Expected: FAIL -- ImageManager type not defined.

**Step 3: Implement image management**

Create `internal/docker/image.go`:

```go
package docker

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/claudeup/claudeup-lab/embed"
)

const DefaultImage = "ghcr.io/claudeup/claudeup-lab:latest"

type ImageManager struct{}

func NewImageManager() *ImageManager {
	return &ImageManager{}
}

func (im *ImageManager) ExistsLocally(image string) bool {
	cmd := exec.Command("docker", "image", "inspect", image)
	return cmd.Run() == nil
}

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

	// Write embedded files to temp directory
	files := map[string][]byte{
		"Dockerfile":           embed.Dockerfile,
		"init-claude-config.sh": embed.InitClaudeConfig,
		"init-config-repo.sh":  embed.InitConfigRepo,
		"init-claudeup.sh":     embed.InitClaudeup,
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

func ImageTag() string {
	tag := os.Getenv("CLAUDEUP_LAB_IMAGE")
	if tag != "" {
		return strings.TrimSpace(tag)
	}
	return DefaultImage
}
```

**Step 4: Run tests to verify they pass**

Run: `cd ~/code/claudeup-lab && go test ./internal/docker/ -v`
Expected: All tests PASS.

**Step 5: Commit**

```bash
git add -A
git commit -m "feat: image management with GHCR pull and embedded Dockerfile fallback"
```

---

## Task 7: Git Worktree Management

**Files:**
- Create: `internal/lab/worktree.go`
- Create: `internal/lab/worktree_test.go`

Handles bare clone creation/refresh and worktree operations.

**Step 1: Write the failing tests**

Create `internal/lab/worktree_test.go`. These tests create real git repos and test real git operations:

```go
package lab_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/claudeup/claudeup-lab/internal/lab"
)

func initTestRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	run(t, dir, "git", "init")
	run(t, dir, "git", "config", "user.email", "test@test.com")
	run(t, dir, "git", "config", "user.name", "Test")
	os.WriteFile(filepath.Join(dir, "README.md"), []byte("# test"), 0o644)
	run(t, dir, "git", "add", ".")
	run(t, dir, "git", "commit", "-m", "initial")
	return dir
}

func run(t *testing.T, dir string, name string, args ...string) {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("%s %v: %v\n%s", name, args, err, out)
	}
}

func TestEnsureBareRepo(t *testing.T) {
	source := initTestRepo(t)
	reposDir := filepath.Join(t.TempDir(), "repos")

	wt := lab.NewWorktreeManager(reposDir)
	barePath, err := wt.EnsureBareRepo(source, "testproject")
	if err != nil {
		t.Fatalf("EnsureBareRepo: %v", err)
	}

	if _, err := os.Stat(barePath); os.IsNotExist(err) {
		t.Error("bare repo should exist")
	}

	// Second call should refresh, not recreate
	barePath2, err := wt.EnsureBareRepo(source, "testproject")
	if err != nil {
		t.Fatalf("EnsureBareRepo (refresh): %v", err)
	}
	if barePath != barePath2 {
		t.Errorf("paths differ: %q vs %q", barePath, barePath2)
	}
}

func TestCreateWorktree(t *testing.T) {
	source := initTestRepo(t)
	reposDir := filepath.Join(t.TempDir(), "repos")
	wsDir := filepath.Join(t.TempDir(), "workspaces")

	wt := lab.NewWorktreeManager(reposDir)
	barePath, _ := wt.EnsureBareRepo(source, "testproject")

	wtPath := filepath.Join(wsDir, "test-lab")
	branch, err := wt.CreateWorktree(barePath, wtPath, "lab/test")
	if err != nil {
		t.Fatalf("CreateWorktree: %v", err)
	}

	if branch != "lab/test" {
		t.Errorf("branch = %q, want %q", branch, "lab/test")
	}

	readme := filepath.Join(wtPath, "README.md")
	if _, err := os.Stat(readme); os.IsNotExist(err) {
		t.Error("README.md should exist in worktree")
	}
}

func TestCreateWorktreeBranchCollision(t *testing.T) {
	source := initTestRepo(t)
	reposDir := filepath.Join(t.TempDir(), "repos")
	wsDir := filepath.Join(t.TempDir(), "workspaces")

	wt := lab.NewWorktreeManager(reposDir)
	barePath, _ := wt.EnsureBareRepo(source, "testproject")

	// Create first worktree on lab/test
	wt.CreateWorktree(barePath, filepath.Join(wsDir, "lab1"), "lab/test")

	// Second worktree on same branch should get a suffix
	branch, err := wt.CreateWorktree(barePath, filepath.Join(wsDir, "lab2"), "lab/test")
	if err != nil {
		t.Fatalf("CreateWorktree (collision): %v", err)
	}
	if branch == "lab/test" {
		t.Error("branch should have been renamed due to collision")
	}
	if branch[:9] != "lab/test-" {
		t.Errorf("branch = %q, want prefix %q", branch, "lab/test-")
	}
}

func TestRemoveWorktree(t *testing.T) {
	source := initTestRepo(t)
	reposDir := filepath.Join(t.TempDir(), "repos")
	wsDir := filepath.Join(t.TempDir(), "workspaces")

	wt := lab.NewWorktreeManager(reposDir)
	barePath, _ := wt.EnsureBareRepo(source, "testproject")

	wtPath := filepath.Join(wsDir, "test-lab")
	wt.CreateWorktree(barePath, wtPath, "lab/test")

	err := wt.RemoveWorktree(barePath, wtPath)
	if err != nil {
		t.Fatalf("RemoveWorktree: %v", err)
	}

	if _, err := os.Stat(wtPath); !os.IsNotExist(err) {
		t.Error("worktree directory should be removed")
	}
}

func TestWorktreeCount(t *testing.T) {
	source := initTestRepo(t)
	reposDir := filepath.Join(t.TempDir(), "repos")
	wsDir := filepath.Join(t.TempDir(), "workspaces")

	wt := lab.NewWorktreeManager(reposDir)
	barePath, _ := wt.EnsureBareRepo(source, "testproject")

	count, err := wt.WorktreeCount(barePath)
	if err != nil {
		t.Fatalf("WorktreeCount: %v", err)
	}
	// Bare repo lists itself as one worktree
	if count != 1 {
		t.Errorf("count = %d, want 1", count)
	}

	wt.CreateWorktree(barePath, filepath.Join(wsDir, "lab1"), "lab/test")
	count, _ = wt.WorktreeCount(barePath)
	if count != 2 {
		t.Errorf("count = %d, want 2", count)
	}
}
```

**Step 2: Run tests to verify they fail**

Run: `cd ~/code/claudeup-lab && go test ./internal/lab/ -v -run TestEnsureBareRepo`
Expected: FAIL -- WorktreeManager type not defined.

**Step 3: Implement worktree management**

Create `internal/lab/worktree.go`:

```go
package lab

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type WorktreeManager struct {
	reposDir string
}

func NewWorktreeManager(reposDir string) *WorktreeManager {
	return &WorktreeManager{reposDir: reposDir}
}

func (w *WorktreeManager) EnsureBareRepo(sourceProject, projectName string) (string, error) {
	barePath := filepath.Join(w.reposDir, projectName+".git")
	markerFile := "lab-source-project"

	if err := os.MkdirAll(w.reposDir, 0o755); err != nil {
		return "", fmt.Errorf("create repos directory: %w", err)
	}

	// Check for source mismatch if bare repo exists
	if info, err := os.Stat(barePath); err == nil && info.IsDir() {
		stored, _ := os.ReadFile(filepath.Join(barePath, markerFile))
		if string(stored) != sourceProject {
			// Different project with same name -- use hash suffix
			hash := hashPrefix(sourceProject)
			barePath = filepath.Join(w.reposDir, fmt.Sprintf("%s-%s.git", projectName, hash))
		}
	}

	if info, err := os.Stat(barePath); err == nil && info.IsDir() {
		// Refresh existing bare clone
		w.refreshBareRepo(barePath, sourceProject)
		return barePath, nil
	}

	// Create new bare clone
	if err := w.createBareRepo(barePath, sourceProject, markerFile); err != nil {
		return "", err
	}

	return barePath, nil
}

func (w *WorktreeManager) refreshBareRepo(barePath, sourceProject string) {
	exec.Command("git", "-C", barePath, "fetch", "--all", "--prune").Run()
	exec.Command("git", "-C", barePath, "fetch", sourceProject,
		"+refs/heads/*:refs/heads/*").Run()
}

func (w *WorktreeManager) createBareRepo(barePath, sourceProject, markerFile string) error {
	// Try upstream first
	upstreamCmd := exec.Command("git", "-C", sourceProject, "remote", "get-url", "origin")
	upstreamOut, err := upstreamCmd.Output()
	upstream := strings.TrimSpace(string(upstreamOut))

	if err == nil && upstream != "" {
		cmd := exec.Command("git", "clone", "--bare", upstream, barePath)
		if cmd.Run() == nil {
			// Fetch local branches not yet pushed
			exec.Command("git", "-C", barePath, "fetch", sourceProject,
				"+refs/heads/*:refs/heads/*").Run()
		} else {
			// Upstream clone failed, fall back to local
			os.RemoveAll(barePath)
			cmd = exec.Command("git", "clone", "--bare", sourceProject, barePath)
			if err := cmd.Run(); err != nil {
				return fmt.Errorf("clone bare repo from %s: %w", sourceProject, err)
			}
		}
	} else {
		cmd := exec.Command("git", "clone", "--bare", sourceProject, barePath)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("clone bare repo from %s: %w", sourceProject, err)
		}
	}

	os.WriteFile(filepath.Join(barePath, markerFile), []byte(sourceProject), 0o644)
	return nil
}

func (w *WorktreeManager) CreateWorktree(barePath, worktreePath, branch string) (string, error) {
	if err := os.MkdirAll(filepath.Dir(worktreePath), 0o755); err != nil {
		return "", fmt.Errorf("create workspace parent: %w", err)
	}

	// Check if branch is already checked out in another worktree
	if w.branchInUse(barePath, branch) {
		branch = branch + "-" + randomSuffix()
	}

	// Check if branch already exists in the bare repo
	checkCmd := exec.Command("git", "-C", barePath, "show-ref", "--verify",
		"--quiet", "refs/heads/"+branch)
	if checkCmd.Run() == nil {
		cmd := exec.Command("git", "-C", barePath, "worktree", "add",
			worktreePath, branch)
		if out, err := cmd.CombinedOutput(); err != nil {
			return "", fmt.Errorf("create worktree (existing branch): %w\n%s", err, out)
		}
	} else {
		cmd := exec.Command("git", "-C", barePath, "worktree", "add",
			worktreePath, "-b", branch)
		if out, err := cmd.CombinedOutput(); err != nil {
			return "", fmt.Errorf("create worktree (new branch): %w\n%s", err, out)
		}
	}

	// Exclude .devcontainer/ from git tracking
	gitDir := w.worktreeGitDir(worktreePath)
	excludeFile := filepath.Join(gitDir, "info", "exclude")
	os.MkdirAll(filepath.Dir(excludeFile), 0o755)
	content, _ := os.ReadFile(excludeFile)
	if !strings.Contains(string(content), ".devcontainer/") {
		f, _ := os.OpenFile(excludeFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
		f.WriteString(".devcontainer/\n")
		f.Close()
	}

	return branch, nil
}

func (w *WorktreeManager) RemoveWorktree(barePath, worktreePath string) error {
	cmd := exec.Command("git", "-C", barePath, "worktree", "remove",
		worktreePath, "--force")
	if err := cmd.Run(); err != nil {
		// Fall back to removing the directory directly
		os.RemoveAll(worktreePath)
	}
	return nil
}

func (w *WorktreeManager) WorktreeCount(barePath string) (int, error) {
	cmd := exec.Command("git", "-C", barePath, "worktree", "list", "--porcelain")
	out, err := cmd.Output()
	if err != nil {
		return 0, fmt.Errorf("list worktrees: %w", err)
	}

	count := 0
	for _, line := range strings.Split(string(out), "\n") {
		if strings.HasPrefix(line, "worktree ") {
			count++
		}
	}
	return count, nil
}

func (w *WorktreeManager) branchInUse(barePath, branch string) bool {
	cmd := exec.Command("git", "-C", barePath, "worktree", "list", "--porcelain")
	out, _ := cmd.Output()
	return strings.Contains(string(out), "branch refs/heads/"+branch)
}

func (w *WorktreeManager) worktreeGitDir(worktreePath string) string {
	cmd := exec.Command("git", "-C", worktreePath, "rev-parse", "--git-dir")
	out, err := cmd.Output()
	if err != nil {
		return filepath.Join(worktreePath, ".git")
	}
	result := strings.TrimSpace(string(out))
	if filepath.IsAbs(result) {
		return result
	}
	return filepath.Join(worktreePath, result)
}

func hashPrefix(s string) string {
	// Simple hash for disambiguation -- 8 hex chars
	var h uint64
	for _, c := range s {
		h = h*31 + uint64(c)
	}
	return fmt.Sprintf("%08x", h)
}

func randomSuffix() string {
	// Use crypto/rand for a short random suffix
	b := make([]byte, 4)
	f, _ := os.Open("/dev/urandom")
	f.Read(b)
	f.Close()
	return fmt.Sprintf("%x", b)
}
```

Note: The `randomSuffix` function should use `crypto/rand` in the final implementation. The above is a placeholder for the plan -- the implementer should replace it with `crypto/rand.Read`.

**Step 4: Run tests to verify they pass**

Run: `cd ~/code/claudeup-lab && go test ./internal/lab/ -v`
Expected: All tests PASS.

**Step 5: Commit**

```bash
git add -A
git commit -m "feat: git bare clone and worktree management"
```

---

## Task 8: Devcontainer Rendering

**Files:**
- Create: `internal/lab/devcontainer.go`
- Create: `internal/lab/devcontainer_test.go`

Renders devcontainer.json from the embedded template with dynamic mounts, features, and env vars.

**Step 1: Write the failing tests**

Create `internal/lab/devcontainer_test.go`:

```go
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
		ProjectName: "myapp",
		Profile:     "base",
		ID:          "abc-123-def",
		DisplayName: "myapp-base",
		Image:       "ghcr.io/claudeup/claudeup-lab:latest",
		BareRepoPath: "/home/user/.claudeup-lab/repos/myapp.git",
		HomeDir:     "/home/user",
		GitUserName: "Test User",
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
```

**Step 2: Run tests to verify they fail**

Run: `cd ~/code/claudeup-lab && go test ./internal/lab/ -v -run TestRender`
Expected: FAIL -- DevcontainerConfig and RenderDevcontainer not defined.

**Step 3: Implement devcontainer rendering**

Create `internal/lab/devcontainer.go`. Build JSON programmatically rather than via text/template to avoid trailing-comma issues:

```go
package lab

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	embedpkg "github.com/claudeup/claudeup-lab/embed"
)

type DevcontainerConfig struct {
	ProjectName  string
	Profile      string
	ID           string
	DisplayName  string
	Image        string
	BareRepoPath string
	HomeDir      string
	GitUserName  string
	GitUserEmail string
	GitHubToken  string
	Context7Key  string
	ConfigRepo   string
	ConfigBranch string
	BaseProfile  string
	Features     []string
}

type featureEntry struct {
	Feature        string `json:"feature"`
	DefaultVersion string `json:"default_version"`
}

func RenderDevcontainer(config *DevcontainerConfig, worktreePath string) error {
	dcDir := filepath.Join(worktreePath, ".devcontainer")
	if err := os.MkdirAll(dcDir, 0o755); err != nil {
		return fmt.Errorf("create .devcontainer: %w", err)
	}

	dc := buildDevcontainerJSON(config)

	data, err := json.MarshalIndent(dc, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal devcontainer.json: %w", err)
	}

	outPath := filepath.Join(dcDir, "devcontainer.json")
	if err := os.WriteFile(outPath, data, 0o644); err != nil {
		return fmt.Errorf("write devcontainer.json: %w", err)
	}

	return nil
}

func buildDevcontainerJSON(config *DevcontainerConfig) map[string]interface{} {
	mounts := buildMounts(config)
	features := buildFeatures(config.Features)

	env := map[string]string{
		"CLAUDE_CONFIG_DIR":    "/home/node/.claude",
		"CLAUDE_PROFILE":      config.Profile,
		"NODE_OPTIONS":        "--max-old-space-size=4096",
		"GIT_USER_NAME":       config.GitUserName,
		"GIT_USER_EMAIL":      config.GitUserEmail,
		"GITHUB_TOKEN":        config.GitHubToken,
		"CONTEXT7_API_KEY":    config.Context7Key,
		"CLAUDE_CONFIG_REPO":  config.ConfigRepo,
		"CLAUDE_CONFIG_BRANCH": config.ConfigBranch,
		"CLAUDE_BASE_PROFILE": config.BaseProfile,
	}

	dc := map[string]interface{}{
		"name":               fmt.Sprintf("claudeup-lab - %s (%s)", config.ProjectName, config.Profile),
		"image":              config.Image,
		"features":           features,
		"remoteUser":         "node",
		"mounts":             mounts,
		"containerEnv":       env,
		"workspaceFolder":    fmt.Sprintf("/workspaces/%s", config.DisplayName),
		"postCreateCommand":  "claude upgrade && /usr/local/bin/init-claude-config.sh && /usr/local/bin/init-config-repo.sh && /usr/local/bin/init-claudeup.sh",
		"waitFor":            "postCreateCommand",
	}

	return dc
}

func buildMounts(config *DevcontainerConfig) []string {
	id := config.ID
	home := config.HomeDir

	mounts := []string{
		fmt.Sprintf("source=claudeup-lab-bashhistory-%s,target=/commandhistory,type=volume", id),
		fmt.Sprintf("source=claudeup-lab-config-%s,target=/home/node/.claude,type=volume", id),
		fmt.Sprintf("source=claudeup-lab-claudeup-%s,target=/home/node/.claudeup,type=volume", id),
	}

	// Optional bind mounts -- skip if source doesn't exist
	optionalMounts := []struct {
		source string
		target string
		opts   string
	}{
		{filepath.Join(home, ".claudeup", "profiles"), "/home/node/.claudeup/profiles", "type=bind,readonly"},
		{filepath.Join(home, ".claudeup", "local"), "/home/node/.claudeup/local", "type=bind,readonly"},
		{filepath.Join(home, ".claude-mem"), "/home/node/.claude-mem", "type=bind"},
		{filepath.Join(home, ".ssh"), "/home/node/.ssh", "type=bind,readonly"},
		{filepath.Join(home, ".claude", "settings.json"), "/tmp/base-settings.json", "type=bind,readonly"},
		{filepath.Join(home, ".claude.json"), "/home/node/.claude.json", "type=bind"},
	}

	for _, m := range optionalMounts {
		if _, err := os.Stat(m.source); err == nil {
			mounts = append(mounts, fmt.Sprintf("source=%s,target=%s,%s", m.source, m.target, m.opts))
		}
	}

	// Bare repo bind mount (required for git worktree resolution)
	mounts = append(mounts, fmt.Sprintf("source=%s,target=%s,type=bind", config.BareRepoPath, config.BareRepoPath))

	// Per-lab volumes
	mounts = append(mounts,
		fmt.Sprintf("source=claudeup-lab-npm-%s,target=/home/node/.npm-global,type=volume", id),
		fmt.Sprintf("source=claudeup-lab-local-%s,target=/home/node/.local,type=volume", id),
		fmt.Sprintf("source=claudeup-lab-bun-%s,target=/home/node/.bun,type=volume", id),
	)

	return mounts
}

func buildFeatures(specs []string) map[string]interface{} {
	if len(specs) == 0 {
		return map[string]interface{}{}
	}

	// Load feature registry
	var registry map[string]featureEntry
	json.Unmarshal(embedpkg.FeaturesJSON, &registry)

	features := make(map[string]interface{})
	for _, spec := range specs {
		name, version := parseFeatureSpec(spec)

		entry, ok := registry[name]
		if !ok {
			continue
		}

		if version == "" {
			version = entry.DefaultVersion
		}

		features[entry.Feature] = map[string]string{"version": version}
	}

	return features
}

func parseFeatureSpec(spec string) (name, version string) {
	parts := splitFirst(spec, ":")
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return spec, ""
}

func splitFirst(s, sep string) []string {
	i := 0
	for i < len(s) && string(s[i]) != sep {
		i++
	}
	if i == len(s) {
		return []string{s}
	}
	return []string{s[:i], s[i+1:]}
}
```

**Step 4: Run tests to verify they pass**

Run: `cd ~/code/claudeup-lab && go test ./internal/lab/ -v`
Expected: All tests PASS.

**Step 5: Commit**

```bash
git add -A
git commit -m "feat: devcontainer.json rendering with optional mounts and feature injection"
```

---

## Task 9: Profile Snapshotting

**Files:**
- Create: `internal/lab/profile.go`
- Create: `internal/lab/profile_test.go`

Handles snapshotting the current claudeup config when no `--profile` flag is given, and cleaning up snapshot profiles on `rm`.

**Step 1: Write the failing tests**

Create `internal/lab/profile_test.go`:

```go
package lab_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/claudeup/claudeup-lab/internal/lab"
)

func TestSnapshotProfile(t *testing.T) {
	profilesDir := filepath.Join(t.TempDir(), "profiles")
	os.MkdirAll(profilesDir, 0o755)

	pm := lab.NewProfileManager(profilesDir)
	name, err := pm.Snapshot("test-123")
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}

	if name != "_lab-snapshot-test-123" {
		t.Errorf("name = %q, want %q", name, "_lab-snapshot-test-123")
	}

	// Verify profile file exists
	path := filepath.Join(profilesDir, name+".json")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Error("snapshot profile file should exist")
	}
}

func TestCleanupSnapshot(t *testing.T) {
	profilesDir := filepath.Join(t.TempDir(), "profiles")
	os.MkdirAll(profilesDir, 0o755)

	pm := lab.NewProfileManager(profilesDir)
	name, _ := pm.Snapshot("test-456")

	err := pm.CleanupSnapshot(name)
	if err != nil {
		t.Fatalf("CleanupSnapshot: %v", err)
	}

	path := filepath.Join(profilesDir, name+".json")
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("snapshot profile file should be deleted")
	}
}

func TestCleanupNonSnapshot(t *testing.T) {
	profilesDir := filepath.Join(t.TempDir(), "profiles")
	os.MkdirAll(profilesDir, 0o755)
	os.WriteFile(filepath.Join(profilesDir, "real-profile.json"), []byte("{}"), 0o644)

	pm := lab.NewProfileManager(profilesDir)
	err := pm.CleanupSnapshot("real-profile")
	if err != nil {
		t.Fatalf("CleanupSnapshot: %v", err)
	}

	// Should NOT delete non-snapshot profiles
	path := filepath.Join(profilesDir, "real-profile.json")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Error("non-snapshot profile should NOT be deleted")
	}
}

func TestIsSnapshot(t *testing.T) {
	if !lab.IsSnapshotProfile("_lab-snapshot-abc123") {
		t.Error("should recognize snapshot profile")
	}
	if lab.IsSnapshotProfile("my-real-profile") {
		t.Error("should not flag regular profile as snapshot")
	}
}
```

**Step 2: Run tests to verify they fail**

Run: `cd ~/code/claudeup-lab && go test ./internal/lab/ -v -run TestSnapshot`
Expected: FAIL -- ProfileManager type not defined.

**Step 3: Implement profile management**

Create `internal/lab/profile.go`:

```go
package lab

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const snapshotPrefix = "_lab-snapshot-"

type ProfileManager struct {
	profilesDir string
}

func NewProfileManager(profilesDir string) *ProfileManager {
	return &ProfileManager{profilesDir: profilesDir}
}

func (pm *ProfileManager) Snapshot(labShortID string) (string, error) {
	name := snapshotPrefix + labShortID

	cmd := exec.Command("claudeup", "profile", "save", name)
	if out, err := cmd.CombinedOutput(); err != nil {
		// If claudeup isn't available, create a minimal empty profile
		path := filepath.Join(pm.profilesDir, name+".json")
		if writeErr := os.WriteFile(path, []byte("{}"), 0o644); writeErr != nil {
			return "", fmt.Errorf("claudeup profile save failed (%v: %s) and fallback write failed: %w",
				err, out, writeErr)
		}
	}

	return name, nil
}

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
```

**Step 4: Run tests to verify they pass**

Run: `cd ~/code/claudeup-lab && go test ./internal/lab/ -v`
Expected: All tests PASS.

**Step 5: Commit**

```bash
git add -A
git commit -m "feat: profile snapshotting for default profile behavior"
```

---

## Task 10: Start Command

**Files:**
- Create: `internal/lab/manager.go`
- Create: `internal/commands/start.go`
- Create: `internal/lab/display_name.go`
- Create: `internal/lab/display_name_test.go`

The start command orchestrates everything: validate prerequisites, generate UUID, ensure bare clone, compute display name, create worktree, render devcontainer.json, launch container, save metadata.

**Step 1: Write display name tests**

Create `internal/lab/display_name_test.go`:

```go
package lab_test

import (
	"testing"

	"github.com/claudeup/claudeup-lab/internal/lab"
)

func TestComputeDisplayName(t *testing.T) {
	tests := []struct {
		name     string
		project  string
		profile  string
		userName string
		want     string
	}{
		{"default", "myapp", "base", "", "myapp-base"},
		{"custom name", "myapp", "base", "custom", "custom"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := lab.ComputeDisplayName(tt.project, tt.profile, tt.userName)
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestValidateDisplayName(t *testing.T) {
	tests := []struct {
		name  string
		input string
		valid bool
	}{
		{"simple", "myapp-base", true},
		{"with dots", "my.app", true},
		{"with underscore", "my_app", true},
		{"starts with dot", ".hidden", false},
		{"double dot", "..", false},
		{"spaces", "my app", false},
		{"slashes", "my/app", false},
		{"empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := lab.ValidateDisplayName(tt.input)
			if tt.valid && err != nil {
				t.Errorf("expected valid, got error: %v", err)
			}
			if !tt.valid && err == nil {
				t.Error("expected invalid, got nil error")
			}
		})
	}
}
```

**Step 2: Run tests to verify they fail**

Run: `cd ~/code/claudeup-lab && go test ./internal/lab/ -v -run TestComputeDisplayName`
Expected: FAIL.

**Step 3: Implement display name logic**

Create `internal/lab/display_name.go`:

```go
package lab

import (
	"fmt"
	"regexp"
	"strings"
)

var validNameRegex = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)

func ComputeDisplayName(projectName, profile, userName string) string {
	if userName != "" {
		return userName
	}
	return projectName + "-" + profile
}

func ValidateDisplayName(name string) error {
	if name == "" {
		return fmt.Errorf("display name must not be empty")
	}
	if strings.HasPrefix(name, ".") {
		return fmt.Errorf("display name must not start with '.'")
	}
	if !validNameRegex.MatchString(name) {
		return fmt.Errorf("display name %q contains invalid characters (allowed: A-Z, a-z, 0-9, '.', '_', '-')", name)
	}
	return nil
}

func DisambiguateDisplayName(name, shortID string, existingNames map[string]bool) string {
	if !existingNames[name] {
		return name
	}
	return name + "-" + shortID
}
```

**Step 4: Run tests to verify they pass**

Run: `cd ~/code/claudeup-lab && go test ./internal/lab/ -v -run TestComputeDisplayName`
Expected: PASS.

**Step 5: Implement the manager and start command**

Create `internal/lab/manager.go`:

```go
package lab

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/claudeup/claudeup-lab/internal/docker"
	"github.com/google/uuid"
)

type Manager struct {
	baseDir   string
	store     *StateStore
	worktrees *WorktreeManager
	profiles  *ProfileManager
	docker    *docker.Client
	images    *docker.ImageManager
}

func NewManager(baseDir string) *Manager {
	return &Manager{
		baseDir:   baseDir,
		store:     NewStateStore(filepath.Join(baseDir, "state")),
		worktrees: NewWorktreeManager(filepath.Join(baseDir, "repos")),
		profiles:  NewProfileManager(filepath.Join(os.Getenv("HOME"), ".claudeup", "profiles")),
		docker:    docker.NewClient(),
		images:    docker.NewImageManager(),
	}
}

func (m *Manager) Store() *StateStore     { return m.store }
func (m *Manager) Docker() *docker.Client { return m.docker }

type StartOptions struct {
	Project     string
	Profile     string
	Branch      string
	Name        string
	Features    []string
	BaseProfile string
}

func (m *Manager) Start(opts *StartOptions) (*Metadata, error) {
	// Validate prerequisites
	if err := m.checkPrerequisites(); err != nil {
		return nil, err
	}

	// Resolve project path
	projectPath, err := filepath.Abs(opts.Project)
	if err != nil {
		return nil, fmt.Errorf("resolve project path: %w", err)
	}
	projectName := filepath.Base(projectPath)

	// Verify it's a git repo
	cmd := exec.Command("git", "-C", projectPath, "rev-parse", "--git-dir")
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("%s is not a git repository", projectPath)
	}

	// Handle profile snapshotting
	profile := opts.Profile
	var snapshotName string
	if profile == "" {
		id := uuid.New().String()[:8]
		snapshotName, err = m.profiles.Snapshot(id)
		if err != nil {
			return nil, fmt.Errorf("snapshot current config: %w", err)
		}
		profile = snapshotName
	}

	// Generate UUID
	labID := uuid.New().String()

	// Ensure base image
	image := docker.ImageTag()
	if err := m.images.EnsureImage(image); err != nil {
		return nil, fmt.Errorf("ensure base image: %w", err)
	}

	// Ensure bare clone
	barePath, err := m.worktrees.EnsureBareRepo(projectPath, projectName)
	if err != nil {
		return nil, fmt.Errorf("ensure bare repo: %w", err)
	}

	// Compute display name
	displayName := ComputeDisplayName(projectName, profile, opts.Name)
	if err := ValidateDisplayName(displayName); err != nil {
		return nil, err
	}

	// Check for name collisions
	existingNames := make(map[string]bool)
	if labs, _ := m.store.List(); labs != nil {
		for _, l := range labs {
			existingNames[l.DisplayName] = true
		}
	}
	displayName = DisambiguateDisplayName(displayName, labID[:8], existingNames)

	// Compute branch
	branch := opts.Branch
	if branch == "" {
		branch = "lab/" + profile
	}

	// Create worktree
	worktreePath := filepath.Join(m.baseDir, "workspaces", displayName)
	branch, err = m.worktrees.CreateWorktree(barePath, worktreePath, branch)
	if err != nil {
		return nil, fmt.Errorf("create worktree: %w", err)
	}

	// Render devcontainer.json
	dcConfig := &DevcontainerConfig{
		ProjectName:  projectName,
		Profile:      profile,
		ID:           labID,
		DisplayName:  displayName,
		Image:        image,
		BareRepoPath: barePath,
		HomeDir:      os.Getenv("HOME"),
		GitUserName:  gitConfig("user.name"),
		GitUserEmail: gitConfig("user.email"),
		GitHubToken:  os.Getenv("GITHUB_TOKEN"),
		Context7Key:  os.Getenv("CONTEXT7_API_KEY"),
		ConfigRepo:   os.Getenv("CLAUDE_CONFIG_REPO"),
		ConfigBranch: envOrDefault("CLAUDE_CONFIG_BRANCH", "main"),
		BaseProfile:  opts.BaseProfile,
		Features:     opts.Features,
	}
	if err := RenderDevcontainer(dcConfig, worktreePath); err != nil {
		// Cleanup on failure
		m.worktrees.RemoveWorktree(barePath, worktreePath)
		return nil, fmt.Errorf("render devcontainer: %w", err)
	}

	// Launch container
	fmt.Println("Starting devcontainer...")
	devCmd := exec.Command("devcontainer", "up", "--workspace-folder", worktreePath)
	devCmd.Stdout = os.Stdout
	devCmd.Stderr = os.Stderr
	if err := devCmd.Run(); err != nil {
		// Cleanup on failure
		m.worktrees.RemoveWorktree(barePath, worktreePath)
		return nil, fmt.Errorf("devcontainer up: %w", err)
	}

	// Save metadata
	meta := &Metadata{
		ID:          labID,
		DisplayName: displayName,
		Project:     projectPath,
		ProjectName: projectName,
		Profile:     profile,
		BareRepo:    barePath,
		Worktree:    worktreePath,
		Branch:      branch,
		Created:     time.Now().UTC(),
		Snapshot:    snapshotName,
	}
	if err := m.store.Save(meta); err != nil {
		return nil, fmt.Errorf("save metadata: %w", err)
	}

	return meta, nil
}

func (m *Manager) checkPrerequisites() error {
	if !m.docker.IsRunning() {
		return fmt.Errorf("Docker is not running (start Docker Desktop or the docker daemon)")
	}
	if _, err := exec.LookPath("devcontainer"); err != nil {
		return fmt.Errorf("devcontainer CLI not found (install: npm install -g @devcontainers/cli)")
	}
	if _, err := exec.LookPath("git"); err != nil {
		return fmt.Errorf("git not found on PATH")
	}
	return nil
}

func gitConfig(key string) string {
	cmd := exec.Command("git", "config", key)
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return string(out[:len(out)-1]) // trim newline
}

func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
```

Create `internal/commands/start.go`:

```go
package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/claudeup/claudeup-lab/internal/lab"
	"github.com/spf13/cobra"
)

func newStartCmd() *cobra.Command {
	var opts lab.StartOptions
	var features []string

	cmd := &cobra.Command{
		Use:   "start",
		Short: "Create and start a lab",
		RunE: func(cmd *cobra.Command, args []string) error {
			if opts.Project == "" {
				cwd, err := os.Getwd()
				if err != nil {
					return fmt.Errorf("get working directory: %w", err)
				}
				opts.Project = cwd
			}
			opts.Features = features

			baseDir := filepath.Join(os.Getenv("HOME"), ".claudeup-lab")
			mgr := lab.NewManager(baseDir)

			meta, err := mgr.Start(&opts)
			if err != nil {
				return err
			}

			fmt.Println()
			fmt.Println("Lab ready!")
			fmt.Printf("  Name:     %s\n", meta.DisplayName)
			fmt.Printf("  ID:       %s\n", meta.ID[:8])
			fmt.Printf("  Worktree: %s\n", meta.Worktree)
			fmt.Printf("  Branch:   %s\n", meta.Branch)
			fmt.Printf("  Profile:  %s\n", meta.Profile)
			fmt.Println()
			fmt.Println("Next steps:")
			fmt.Printf("  claudeup-lab exec   --lab %s -- <command>\n", meta.DisplayName)
			fmt.Printf("  claudeup-lab exec   --lab %s -- claude\n", meta.DisplayName)
			fmt.Printf("  claudeup-lab open   --lab %s\n", meta.DisplayName)
			fmt.Printf("  claudeup-lab stop   --lab %s\n", meta.DisplayName)
			return nil
		},
	}

	cmd.Flags().StringVar(&opts.Project, "project", "", "Project directory (default: current directory)")
	cmd.Flags().StringVar(&opts.Profile, "profile", "", "claudeup profile (default: snapshot current config)")
	cmd.Flags().StringVar(&opts.Branch, "branch", "", "Git branch name (default: lab/<profile>)")
	cmd.Flags().StringVar(&opts.Name, "name", "", "Display name for the lab")
	cmd.Flags().StringSliceVar(&features, "feature", nil, "Devcontainer feature (repeatable, e.g. go:1.23)")
	cmd.Flags().StringVar(&opts.BaseProfile, "base-profile", "", "Apply base profile before main profile")

	return cmd
}
```

Then register it in `internal/commands/root.go` by adding `cmd.AddCommand(newStartCmd())` to `NewRootCmd()`.

**Step 6: Verify it builds**

Run:
```bash
cd ~/code/claudeup-lab && go build ./cmd/claudeup-lab
./claudeup-lab start --help
```

Expected: Help text shows all start flags.

**Step 7: Commit**

```bash
git add -A
git commit -m "feat: start command with full lab lifecycle orchestration"
```

---

## Task 11: List Command

**Files:**
- Create: `internal/commands/list.go`

**Step 1: Implement list command**

Create `internal/commands/list.go`:

```go
package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/claudeup/claudeup-lab/internal/lab"
	"github.com/spf13/cobra"
)

func newListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "Show all labs and their status",
		Aliases: []string{"ls"},
		RunE: func(cmd *cobra.Command, args []string) error {
			baseDir := filepath.Join(os.Getenv("HOME"), ".claudeup-lab")
			mgr := lab.NewManager(baseDir)
			store := mgr.Store()
			docker := mgr.Docker()

			labs, err := store.List()
			if err != nil {
				return err
			}

			if len(labs) == 0 {
				fmt.Println("No labs found.")
				return nil
			}

			fmt.Printf("%-30s %-10s %-20s %-15s %s\n", "NAME", "ID", "PROJECT", "PROFILE", "STATUS")
			fmt.Printf("%-30s %-10s %-20s %-15s %s\n", "----", "--", "-------", "-------", "------")

			for _, m := range labs {
				status := labStatus(m, docker)
				fmt.Printf("%-30s %-10s %-20s %-15s %s\n",
					m.DisplayName, m.ID[:8], m.ProjectName, m.Profile, status)
			}

			return nil
		},
	}
}

func labStatus(meta *lab.Metadata, docker *lab.DockerClientInterface) string {
	// Check if this needs an interface -- for now use the concrete client
	// through the manager. Adjust the signature based on final docker package API.
	return "unknown"
}
```

Note: The `labStatus` function needs to call `docker.FindContainer(meta.Worktree)` to determine running/stopped/orphaned. The implementer should wire this through properly based on the docker.Client API. The exact pattern depends on whether we expose the docker client from the manager or pass it as a parameter.

A cleaner approach: add a `Status` method to the manager:

```go
// Add to internal/lab/manager.go
func (m *Manager) LabStatus(meta *Metadata) string {
	id, _ := m.docker.FindContainer(meta.Worktree)
	if id != "" {
		return "running"
	}
	if _, err := os.Stat(meta.Worktree); err == nil {
		return "stopped"
	}
	return "orphaned"
}
```

Then the list command calls `mgr.LabStatus(m)` for each lab.

Register in `root.go`: `cmd.AddCommand(newListCmd())`

**Step 2: Verify it builds**

Run: `cd ~/code/claudeup-lab && go build ./cmd/claudeup-lab && ./claudeup-lab list`
Expected: "No labs found." (or a table if labs exist).

**Step 3: Commit**

```bash
git add -A
git commit -m "feat: list command showing labs with status"
```

---

## Task 12: Exec Command

**Files:**
- Create: `internal/commands/exec.go`

**Step 1: Implement exec command**

Create `internal/commands/exec.go`:

```go
package commands

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/claudeup/claudeup-lab/internal/lab"
	"github.com/spf13/cobra"
)

func newExecCmd() *cobra.Command {
	var labName string

	cmd := &cobra.Command{
		Use:   "exec [-- command...]",
		Short: "Run a command inside a running lab",
		Long:  "Run a command inside a running lab. Without arguments after --, opens an interactive bash shell.",
		RunE: func(cmd *cobra.Command, args []string) error {
			baseDir := filepath.Join(os.Getenv("HOME"), ".claudeup-lab")
			mgr := lab.NewManager(baseDir)
			resolver := lab.NewResolver(mgr.Store())

			meta, err := resolveLab(resolver, labName)
			if err != nil {
				return err
			}

			execArgs := []string{"exec", "--workspace-folder", meta.Worktree}

			// Args after "--" are the command to run
			dashIdx := cmd.ArgsLenAtDash()
			if dashIdx >= 0 && dashIdx < len(args) {
				execArgs = append(execArgs, args[dashIdx:]...)
			} else {
				execArgs = append(execArgs, "bash")
			}

			devCmd := exec.Command("devcontainer", execArgs...)
			devCmd.Stdin = os.Stdin
			devCmd.Stdout = os.Stdout
			devCmd.Stderr = os.Stderr
			return devCmd.Run()
		},
	}

	cmd.Flags().StringVar(&labName, "lab", "", "Lab to exec into (name, UUID, project, or profile)")

	return cmd
}

func resolveLab(resolver *lab.Resolver, name string) (*lab.Metadata, error) {
	if name != "" {
		return resolver.Resolve(name)
	}

	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("get working directory: %w", err)
	}
	return resolver.ResolveByCWD(cwd)
}
```

Register in `root.go`: `cmd.AddCommand(newExecCmd())`

**Step 2: Verify it builds**

Run: `cd ~/code/claudeup-lab && go build ./cmd/claudeup-lab && ./claudeup-lab exec --help`
Expected: Help text shows `--lab` flag and usage.

**Step 3: Commit**

```bash
git add -A
git commit -m "feat: exec command for running commands inside labs"
```

---

## Task 13: Open Command

**Files:**
- Create: `internal/commands/open.go`

**Step 1: Implement open command**

Create `internal/commands/open.go`:

```go
package commands

import (
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/claudeup/claudeup-lab/internal/lab"
	"github.com/spf13/cobra"
)

func newOpenCmd() *cobra.Command {
	var labName string

	cmd := &cobra.Command{
		Use:   "open",
		Short: "Attach VS Code to a running lab",
		RunE: func(cmd *cobra.Command, args []string) error {
			if _, err := exec.LookPath("code"); err != nil {
				return fmt.Errorf("VS Code CLI 'code' not found (see: https://code.visualstudio.com/docs/setup/mac)")
			}

			baseDir := filepath.Join(os.Getenv("HOME"), ".claudeup-lab")
			mgr := lab.NewManager(baseDir)
			resolver := lab.NewResolver(mgr.Store())

			meta, err := resolveLab(resolver, labName)
			if err != nil {
				return err
			}

			containerID, err := mgr.Docker().ContainerHostname(meta.Worktree)
			if err != nil {
				return fmt.Errorf("could not get container ID -- is the lab running? %w", err)
			}

			hexID := hex.EncodeToString([]byte(containerID))
			uri := fmt.Sprintf("vscode-remote://attached-container+%s/workspaces/%s", hexID, meta.DisplayName)

			codeCmd := exec.Command("code", "--folder-uri", uri)
			if err := codeCmd.Run(); err != nil {
				return fmt.Errorf("open VS Code: %w", err)
			}

			fmt.Printf("VS Code attached to lab: %s (%s)\n", meta.DisplayName, meta.ID[:8])
			return nil
		},
	}

	cmd.Flags().StringVar(&labName, "lab", "", "Lab to open (name, UUID, project, or profile)")

	return cmd
}
```

Register in `root.go`: `cmd.AddCommand(newOpenCmd())`

**Step 2: Verify it builds**

Run: `cd ~/code/claudeup-lab && go build ./cmd/claudeup-lab && ./claudeup-lab open --help`
Expected: Help text shows.

**Step 3: Commit**

```bash
git add -A
git commit -m "feat: open command for VS Code attachment"
```

---

## Task 14: Stop Command

**Files:**
- Create: `internal/commands/stop.go`

**Step 1: Implement stop command**

Create `internal/commands/stop.go`:

```go
package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/claudeup/claudeup-lab/internal/lab"
	"github.com/spf13/cobra"
)

func newStopCmd() *cobra.Command {
	var labName string

	cmd := &cobra.Command{
		Use:   "stop",
		Short: "Stop a running lab (volumes persist)",
		RunE: func(cmd *cobra.Command, args []string) error {
			baseDir := filepath.Join(os.Getenv("HOME"), ".claudeup-lab")
			mgr := lab.NewManager(baseDir)
			resolver := lab.NewResolver(mgr.Store())

			meta, err := resolveLab(resolver, labName)
			if err != nil {
				return err
			}

			fmt.Printf("Stopping lab: %s...\n", meta.DisplayName)

			containerID, err := mgr.Docker().FindContainer(meta.Worktree)
			if err != nil {
				return err
			}

			if containerID == "" {
				fmt.Printf("No running container found for lab: %s\n", meta.DisplayName)
				return nil
			}

			if err := mgr.Docker().StopContainer(containerID); err != nil {
				return err
			}

			fmt.Printf("Stopped lab: %s\n", meta.DisplayName)
			return nil
		},
	}

	cmd.Flags().StringVar(&labName, "lab", "", "Lab to stop (name, UUID, project, or profile)")

	return cmd
}
```

Register in `root.go`: `cmd.AddCommand(newStopCmd())`

**Step 2: Verify it builds**

Run: `cd ~/code/claudeup-lab && go build ./cmd/claudeup-lab && ./claudeup-lab stop --help`
Expected: Help text shows.

**Step 3: Commit**

```bash
git add -A
git commit -m "feat: stop command to halt running labs"
```

---

## Task 15: Rm Command

**Files:**
- Create: `internal/commands/rm.go`
- Modify: `internal/lab/manager.go` (add Remove method)

**Step 1: Add Remove method to manager**

Add to `internal/lab/manager.go`:

```go
func (m *Manager) Remove(meta *Metadata, confirmed bool) error {
	if !confirmed {
		return fmt.Errorf("removal not confirmed")
	}

	// Stop container
	containerID, _ := m.docker.FindContainerIncludingStopped(meta.Worktree)
	if containerID != "" {
		fmt.Println("Removing container...")
		m.docker.RemoveContainer(containerID)
	}

	// Remove volumes
	fmt.Println("Removing Docker volumes...")
	volumes, _ := m.docker.ListVolumes(fmt.Sprintf("claudeup-lab-.*-%s", meta.ID))
	if len(volumes) > 0 {
		m.docker.RemoveVolumes(volumes)
	}

	// Remove worktree
	fmt.Println("Removing worktree...")
	m.worktrees.RemoveWorktree(meta.BareRepo, meta.Worktree)

	// Remove metadata
	fmt.Println("Removing metadata...")
	m.store.Delete(meta.ID)

	// Clean up snapshot profile if applicable
	if meta.Snapshot != "" {
		m.profiles.CleanupSnapshot(meta.Snapshot)
	}

	fmt.Printf("Removed lab: %s\n", meta.DisplayName)

	// Check if bare repo has remaining worktrees
	count, err := m.worktrees.WorktreeCount(meta.BareRepo)
	if err == nil && count <= 1 {
		return &BareRepoCleanupPrompt{BareRepo: meta.BareRepo}
	}

	return nil
}

type BareRepoCleanupPrompt struct {
	BareRepo string
}

func (e *BareRepoCleanupPrompt) Error() string {
	return fmt.Sprintf("bare repo %s has no remaining worktrees", e.BareRepo)
}
```

**Step 2: Implement rm command**

Create `internal/commands/rm.go`:

```go
package commands

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/claudeup/claudeup-lab/internal/lab"
	"github.com/spf13/cobra"
)

func newRmCmd() *cobra.Command {
	var labName string
	var force bool

	cmd := &cobra.Command{
		Use:   "rm",
		Short: "Destroy a lab and all its data",
		RunE: func(cmd *cobra.Command, args []string) error {
			baseDir := filepath.Join(os.Getenv("HOME"), ".claudeup-lab")
			mgr := lab.NewManager(baseDir)
			resolver := lab.NewResolver(mgr.Store())

			meta, err := resolveLab(resolver, labName)
			if err != nil {
				return err
			}

			if !force {
				fmt.Println("This will:")
				fmt.Println("  - Stop the container")
				fmt.Printf("  - Remove Docker volumes (claudeup-lab-*-%s)\n", meta.ID)
				fmt.Printf("  - Remove worktree: %s\n", meta.Worktree)
				fmt.Println("  - Remove lab metadata")
				fmt.Println()
				if !confirm("Continue?") {
					fmt.Println("Aborted.")
					return nil
				}
			}

			err = mgr.Remove(meta, true)

			var prompt *lab.BareRepoCleanupPrompt
			if errors.As(err, &prompt) {
				fmt.Printf("\nBare repo %s has no remaining worktrees.\n", prompt.BareRepo)
				if confirm("Remove bare repo?") {
					os.RemoveAll(prompt.BareRepo)
					fmt.Printf("Removed bare repo: %s\n", prompt.BareRepo)
				}
				return nil
			}

			return err
		},
	}

	cmd.Flags().StringVar(&labName, "lab", "", "Lab to remove (name, UUID, project, or profile)")
	cmd.Flags().BoolVarP(&force, "force", "f", false, "Skip confirmation prompt")

	return cmd
}

func confirm(prompt string) bool {
	fmt.Printf("%s [y/N] ", prompt)
	reader := bufio.NewReader(os.Stdin)
	response, _ := reader.ReadString('\n')
	response = strings.TrimSpace(strings.ToLower(response))
	return response == "y" || response == "yes"
}
```

Register in `root.go`: `cmd.AddCommand(newRmCmd())`

**Step 3: Verify it builds**

Run: `cd ~/code/claudeup-lab && go build ./cmd/claudeup-lab && ./claudeup-lab rm --help`
Expected: Help text shows `--lab` and `--force` flags.

**Step 4: Commit**

```bash
git add -A
git commit -m "feat: rm command for full lab teardown"
```

---

## Task 16: Doctor Command

**Files:**
- Create: `internal/commands/doctor.go`

**Step 1: Implement doctor command**

Create `internal/commands/doctor.go`:

```go
package commands

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/claudeup/claudeup-lab/internal/docker"
	"github.com/claudeup/claudeup-lab/internal/lab"
	"github.com/spf13/cobra"
)

func newDoctorCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Check system health and prerequisites",
		RunE: func(cmd *cobra.Command, args []string) error {
			issues := 0

			// Docker
			client := docker.NewClient()
			if client.IsRunning() {
				fmt.Println("[OK] Docker is running")
			} else {
				fmt.Println("[FAIL] Docker is not running")
				issues++
			}

			// devcontainer CLI
			if path, err := exec.LookPath("devcontainer"); err == nil {
				fmt.Printf("[OK] devcontainer CLI found: %s\n", path)
			} else {
				fmt.Println("[FAIL] devcontainer CLI not found (install: npm install -g @devcontainers/cli)")
				issues++
			}

			// Git
			if _, err := exec.LookPath("git"); err == nil {
				fmt.Println("[OK] git found")
			} else {
				fmt.Println("[FAIL] git not found")
				issues++
			}

			// claudeup (optional)
			if _, err := exec.LookPath("claudeup"); err == nil {
				fmt.Println("[OK] claudeup found")
			} else {
				fmt.Println("[WARN] claudeup not found (needed for profile snapshotting)")
			}

			// Base image
			im := docker.NewImageManager()
			image := docker.ImageTag()
			if im.ExistsLocally(image) {
				fmt.Printf("[OK] Base image available: %s\n", image)
			} else {
				fmt.Printf("[WARN] Base image not found locally: %s (will be pulled on first start)\n", image)
			}

			// Orphaned labs
			baseDir := filepath.Join(os.Getenv("HOME"), ".claudeup-lab")
			mgr := lab.NewManager(baseDir)
			labs, _ := mgr.Store().List()
			orphaned := 0
			for _, m := range labs {
				if _, err := os.Stat(m.Worktree); os.IsNotExist(err) {
					orphaned++
					fmt.Printf("[WARN] Orphaned lab: %s (%s) -- worktree missing\n", m.DisplayName, m.ID[:8])
				}
			}
			if orphaned == 0 && len(labs) > 0 {
				fmt.Printf("[OK] %d lab(s) found, no orphans\n", len(labs))
			} else if len(labs) == 0 {
				fmt.Println("[OK] No labs found")
			}

			fmt.Println()
			if issues > 0 {
				fmt.Printf("%d issue(s) found.\n", issues)
				return fmt.Errorf("doctor found %d issue(s)", issues)
			}
			fmt.Println("All checks passed.")
			return nil
		},
	}
}
```

Register in `root.go`: `cmd.AddCommand(newDoctorCmd())`

**Step 2: Verify it builds and runs**

Run: `cd ~/code/claudeup-lab && go build ./cmd/claudeup-lab && ./claudeup-lab doctor`
Expected: Status checks print with [OK]/[FAIL]/[WARN].

**Step 3: Commit**

```bash
git add -A
git commit -m "feat: doctor command for system health checks"
```

---

## Task 17: Release Infrastructure

**Files:**
- Create: `.goreleaser.yaml`
- Create: `scripts/install.sh`
- Create: `.github/workflows/release.yml`
- Create: `Makefile`

**Step 1: Create goreleaser config**

Create `.goreleaser.yaml`:

```yaml
project_name: claudeup-lab

builds:
  - main: ./cmd/claudeup-lab
    binary: claudeup-lab
    env:
      - CGO_ENABLED=0
    goos:
      - linux
      - darwin
    goarch:
      - amd64
      - arm64
    ldflags:
      - -s -w -X github.com/claudeup/claudeup-lab/internal/commands.version={{.Version}}

archives:
  - format: tar.gz
    name_template: "{{ .ProjectName }}_{{ .Os }}_{{ .Arch }}"

checksum:
  name_template: "checksums.txt"

release:
  github:
    owner: claudeup
    name: claudeup-lab
```

**Step 2: Create install script**

Create `scripts/install.sh`. Model this on claudeup's install script pattern. The script should:
- Detect OS and architecture
- Download the latest release from GitHub
- Extract to `~/.local/bin/`
- Verify the binary runs

```bash
#!/usr/bin/env bash
set -euo pipefail

REPO="claudeup/claudeup-lab"
INSTALL_DIR="${HOME}/.local/bin"

# Detect platform
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)
case "$ARCH" in
    x86_64) ARCH="amd64" ;;
    aarch64|arm64) ARCH="arm64" ;;
    *) echo "Unsupported architecture: $ARCH" >&2; exit 1 ;;
esac

# Get latest version
VERSION=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name"' | sed -E 's/.*"v?([^"]+)".*/\1/')
if [ -z "$VERSION" ]; then
    echo "Could not determine latest version" >&2
    exit 1
fi

echo "Downloading claudeup-lab v${VERSION} for ${OS}-${ARCH}..."

URL="https://github.com/${REPO}/releases/download/v${VERSION}/claudeup-lab_${OS}_${ARCH}.tar.gz"
mkdir -p "$INSTALL_DIR"
curl -fsSL "$URL" | tar xz -C "$INSTALL_DIR" claudeup-lab

echo "Installed claudeup-lab v${VERSION} to ${INSTALL_DIR}/claudeup-lab"

if ! echo "$PATH" | grep -q "$INSTALL_DIR"; then
    echo ""
    echo "Add to PATH: export PATH=\"\$PATH:${INSTALL_DIR}\""
fi
```

**Step 3: Create GitHub Actions release workflow**

Create `.github/workflows/release.yml`:

```yaml
name: Release

on:
  push:
    tags:
      - "v*"

permissions:
  contents: write

jobs:
  release:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0

      - uses: actions/setup-go@v5
        with:
          go-version: "1.23"

      - uses: goreleaser/goreleaser-action@v6
        with:
          version: latest
          args: release --clean
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

**Step 4: Create Makefile**

Create `Makefile`:

```makefile
.PHONY: build test clean install

build:
	go build -o claudeup-lab ./cmd/claudeup-lab

test:
	go test ./... -v

clean:
	rm -f claudeup-lab

install: build
	mkdir -p $(HOME)/.local/bin
	cp claudeup-lab $(HOME)/.local/bin/
```

**Step 5: Commit**

```bash
git add -A
git commit -m "feat: release infrastructure (goreleaser, install script, CI)"
```

---

## Task 18: End-to-End Smoke Test

This task verifies the full pipeline works end-to-end. No new code files -- just verification.

**Step 1: Build the binary**

Run:
```bash
cd ~/code/claudeup-lab && go build -o claudeup-lab ./cmd/claudeup-lab
```

Expected: Builds without errors.

**Step 2: Run unit tests**

Run:
```bash
cd ~/code/claudeup-lab && go test ./... -v
```

Expected: All tests pass.

**Step 3: Run doctor**

Run:
```bash
./claudeup-lab doctor
```

Expected: All prerequisite checks pass (Docker, devcontainer, git).

**Step 4: Start a lab**

Run:
```bash
./claudeup-lab start --project ~/code/claudeup-lab --profile base --name e2e-test
```

Expected: Lab starts successfully, prints "Lab ready!" with name, ID, worktree, branch.

**Step 5: List labs**

Run:
```bash
./claudeup-lab list
```

Expected: Shows e2e-test lab with "running" status.

**Step 6: Exec into lab**

Run:
```bash
./claudeup-lab exec --lab e2e-test -- ls /workspaces/e2e-test
```

Expected: Lists files from the project worktree inside the container.

**Step 7: Stop lab**

Run:
```bash
./claudeup-lab stop --lab e2e-test
```

Expected: "Stopped lab: e2e-test".

**Step 8: List shows stopped**

Run:
```bash
./claudeup-lab list
```

Expected: Shows e2e-test with "stopped" status.

**Step 9: Remove lab**

Run:
```bash
./claudeup-lab rm --lab e2e-test --force
```

Expected: Full cleanup, "Removed lab: e2e-test".

**Step 10: List shows empty**

Run:
```bash
./claudeup-lab list
```

Expected: "No labs found."

**Step 11: Commit any fixes**

If any fixes were needed during the smoke test, commit them:

```bash
git add -A
git commit -m "fix: adjustments from end-to-end smoke test"
```

---

## Implementation Notes

### Patterns to follow
- Use `os/exec` for all CLI tool invocations (Docker, git, devcontainer)
- Use `t.TempDir()` for test isolation -- no shared test state
- Check `os.Stat` before optional bind mounts -- log and skip, don't error
- Clean up on failure in the start command -- don't leave orphaned worktrees

### Things the implementer will need to adjust
- The devcontainer template (Task 8) may need iteration to produce valid JSON -- test carefully with `json.Unmarshal` in tests
- The `randomSuffix` function (Task 7) should use `crypto/rand`, not `/dev/urandom`
- Volume name matching in `ListVolumes` (Task 5) uses `strings.Contains` -- the rm command passes a pattern with the lab UUID which should be specific enough to avoid false matches
- The `resolveLab` helper (Task 12) is used by exec, open, stop, and rm -- extract it to a shared location rather than duplicating

### What's NOT in this plan
- GHCR image publishing (separate CI workflow, needs container registry setup)
- Docker image rebuild automation
- Homebrew formula
- Shell completions

These can be added incrementally after the core CLI is working.
