# Container Environment Variables Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add `--env KEY=VALUE` and `--env-file FILE` flags to `claudeup-lab start` so users can inject arbitrary environment variables into devcontainers.

**Architecture:** New env file parser in `internal/lab/envfile.go`. CLI flags parsed in `start.go`, env file read and merged in `Manager.Start`, applied as `ExtraEnv` overlay in `buildDevcontainerJSON`. User vars override all defaults (last-write-wins).

**Tech Stack:** Go 1.23, cobra (CLI), standard library only (no new deps)

**Spec:** `docs/superpowers/specs/2026-03-17-container-env-vars-design.md`

---

## File Structure

| File                                | Action | Responsibility                                                                 |
| ----------------------------------- | ------ | ------------------------------------------------------------------------------ |
| `internal/lab/envfile.go`           | Create | Parse Docker-style env files into `map[string]string`                          |
| `internal/lab/envfile_test.go`      | Create | Unit tests for env file parser                                                 |
| `internal/lab/devcontainer.go`      | Modify | Add `ExtraEnv` field to `DevcontainerConfig`, apply in `buildDevcontainerJSON` |
| `internal/lab/devcontainer_test.go` | Modify | Tests for `ExtraEnv` override behavior                                         |
| `internal/lab/manager.go`           | Modify | Read env file, merge with CLI env vars, pass to `DevcontainerConfig`           |
| `internal/lab/manager_test.go`      | Create | Test env var merge precedence                                                  |
| `internal/commands/start.go`        | Modify | Add `--env` and `--env-file` flags, parse `[]string` into map                  |

---

### Task 1: Env File Parser

**Files:**

- Create: `internal/lab/envfile.go`
- Create: `internal/lab/envfile_test.go`

- [ ] **Step 1: Write failing tests for env file parser**

```go
// internal/lab/envfile_test.go
package lab

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseEnvFile(t *testing.T) {
	t.Run("parses valid key=value pairs", func(t *testing.T) {
		path := writeEnvFile(t, "FOO=bar\nBAZ=qux\n")
		got, err := ParseEnvFile(path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := map[string]string{"FOO": "bar", "BAZ": "qux"}
		assertEnvEqual(t, got, want)
	})

	t.Run("skips comments and blank lines", func(t *testing.T) {
		path := writeEnvFile(t, "# comment\nFOO=bar\n\n  \n# another\nBAZ=qux\n")
		got, err := ParseEnvFile(path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := map[string]string{"FOO": "bar", "BAZ": "qux"}
		assertEnvEqual(t, got, want)
	})

	t.Run("value can contain equals signs", func(t *testing.T) {
		path := writeEnvFile(t, "CONN=host=localhost;port=5432\n")
		got, err := ParseEnvFile(path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got["CONN"] != "host=localhost;port=5432" {
			t.Errorf("got %q, want %q", got["CONN"], "host=localhost;port=5432")
		}
	})

	t.Run("quotes are literal", func(t *testing.T) {
		path := writeEnvFile(t, `FOO="bar baz"` + "\n")
		got, err := ParseEnvFile(path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got["FOO"] != `"bar baz"` {
			t.Errorf("got %q, want %q", got["FOO"], `"bar baz"`)
		}
	})

	t.Run("empty value is valid", func(t *testing.T) {
		path := writeEnvFile(t, "FOO=\n")
		got, err := ParseEnvFile(path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got["FOO"] != "" {
			t.Errorf("got %q, want empty string", got["FOO"])
		}
	})

	t.Run("errors on line without equals", func(t *testing.T) {
		path := writeEnvFile(t, "FOO=bar\nINVALID\n")
		_, err := ParseEnvFile(path)
		if err == nil {
			t.Fatal("expected error for line without =")
		}
		wantMsg := "expected KEY=VALUE"
		if !containsSubstr(err.Error(), wantMsg) {
			t.Errorf("error %q should contain %q", err.Error(), wantMsg)
		}
		if !containsSubstr(err.Error(), "line 2") {
			t.Errorf("error %q should reference line 2", err.Error())
		}
	})

	t.Run("errors on empty key", func(t *testing.T) {
		path := writeEnvFile(t, "=value\n")
		_, err := ParseEnvFile(path)
		if err == nil {
			t.Fatal("expected error for empty key")
		}
		if !containsSubstr(err.Error(), "empty key") {
			t.Errorf("error %q should contain 'empty key'", err.Error())
		}
	})

	t.Run("errors on missing file", func(t *testing.T) {
		_, err := ParseEnvFile("/nonexistent/path/env.txt")
		if err == nil {
			t.Fatal("expected error for missing file")
		}
	})

	t.Run("last value wins for duplicate keys", func(t *testing.T) {
		path := writeEnvFile(t, "FOO=first\nFOO=second\n")
		got, err := ParseEnvFile(path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got["FOO"] != "second" {
			t.Errorf("got %q, want %q", got["FOO"], "second")
		}
	})
}

func writeEnvFile(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "env")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write env file: %v", err)
	}
	return path
}

func assertEnvEqual(t *testing.T, got, want map[string]string) {
	t.Helper()
	if len(got) != len(want) {
		t.Errorf("got %d keys, want %d", len(got), len(want))
	}
	for k, wv := range want {
		if gv, ok := got[k]; !ok {
			t.Errorf("missing key %q", k)
		} else if gv != wv {
			t.Errorf("key %q: got %q, want %q", k, gv, wv)
		}
	}
}

func containsSubstr(s, substr string) bool {
	return strings.Contains(s, substr)
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd /Users/markalston/code/claudeup-lab && go test ./internal/lab/ -run TestParseEnvFile -v`
Expected: FAIL -- `ParseEnvFile` not defined

- [ ] **Step 3: Write minimal implementation**

```go
// internal/lab/envfile.go
package lab

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// ParseEnvFile reads a Docker-style env file and returns key-value pairs.
// Lines starting with # are comments. Blank lines are skipped.
// Each non-comment line must be KEY=VALUE format.
func ParseEnvFile(path string) (map[string]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("env file not found: %s", path)
	}
	defer f.Close()

	env := make(map[string]string)
	scanner := bufio.NewScanner(f)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		// Skip blank lines and comments
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		idx := strings.IndexByte(trimmed, '=')
		if idx < 0 {
			return nil, fmt.Errorf("invalid line in env file %s (line %d): expected KEY=VALUE", path, lineNum)
		}

		key := trimmed[:idx]
		if key == "" {
			return nil, fmt.Errorf("empty key in env file %s (line %d): %q", path, lineNum, trimmed)
		}

		value := trimmed[idx+1:]
		env[key] = value
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read env file %s: %w", path, err)
	}

	return env, nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd /Users/markalston/code/claudeup-lab && go test ./internal/lab/ -run TestParseEnvFile -v`
Expected: All PASS

- [ ] **Step 5: Commit**

```bash
git add internal/lab/envfile.go internal/lab/envfile_test.go
git commit -m "feat: add Docker-style env file parser"
```

---

### Task 2: ExtraEnv Support in DevcontainerConfig

**Files:**

- Modify: `internal/lab/devcontainer.go:14-33` (add `ExtraEnv` field to struct)
- Modify: `internal/lab/devcontainer.go:63-134` (apply `ExtraEnv` in `buildDevcontainerJSON`)
- Modify: `internal/lab/devcontainer_test.go` (add tests)

- [ ] **Step 1: Write failing tests for ExtraEnv**

Add to `internal/lab/devcontainer_test.go`:

```go
func TestExtraEnvAddsNewVars(t *testing.T) {
	dir := t.TempDir()

	config := &lab.DevcontainerConfig{
		ProjectName:  "myapp",
		Profile:      "base",
		ID:           "abc-123",
		DisplayName:  "myapp-base",
		Image:        "test:latest",
		BareRepoPath: "/tmp/bare.git",
		HomeDir:      t.TempDir(),
		ExtraEnv: map[string]string{
			"MY_CUSTOM_VAR": "hello",
			"ANOTHER_VAR":   "world",
		},
	}

	err := lab.RenderDevcontainer(config, dir)
	if err != nil {
		t.Fatalf("RenderDevcontainer: %v", err)
	}

	outPath := filepath.Join(dir, ".devcontainer", "devcontainer.json")
	data, _ := os.ReadFile(outPath)

	var parsed map[string]interface{}
	json.Unmarshal(data, &parsed)

	envRaw, ok := parsed["containerEnv"].(map[string]interface{})
	if !ok {
		t.Fatal("containerEnv should be a map")
	}

	if envRaw["MY_CUSTOM_VAR"] != "hello" {
		t.Errorf("MY_CUSTOM_VAR = %v, want hello", envRaw["MY_CUSTOM_VAR"])
	}
	if envRaw["ANOTHER_VAR"] != "world" {
		t.Errorf("ANOTHER_VAR = %v, want world", envRaw["ANOTHER_VAR"])
	}

	// Hardcoded defaults should still be present
	if _, exists := envRaw["CLAUDE_CONFIG_DIR"]; !exists {
		t.Error("hardcoded CLAUDE_CONFIG_DIR should still be present")
	}
}

func TestExtraEnvOverridesDefaults(t *testing.T) {
	dir := t.TempDir()

	config := &lab.DevcontainerConfig{
		ProjectName:  "myapp",
		Profile:      "base",
		ID:           "abc-123",
		DisplayName:  "myapp-base",
		Image:        "test:latest",
		BareRepoPath: "/tmp/bare.git",
		HomeDir:      t.TempDir(),
		ExtraEnv: map[string]string{
			"NODE_OPTIONS": "--max-old-space-size=8192",
		},
	}

	err := lab.RenderDevcontainer(config, dir)
	if err != nil {
		t.Fatalf("RenderDevcontainer: %v", err)
	}

	outPath := filepath.Join(dir, ".devcontainer", "devcontainer.json")
	data, _ := os.ReadFile(outPath)

	var parsed map[string]interface{}
	json.Unmarshal(data, &parsed)

	envRaw, ok := parsed["containerEnv"].(map[string]interface{})
	if !ok {
		t.Fatal("containerEnv should be a map")
	}
	if envRaw["NODE_OPTIONS"] != "--max-old-space-size=8192" {
		t.Errorf("NODE_OPTIONS = %v, want --max-old-space-size=8192", envRaw["NODE_OPTIONS"])
	}
}

func TestExtraEnvNilIsNoOp(t *testing.T) {
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

	var parsed map[string]interface{}
	json.Unmarshal(data, &parsed)

	envRaw, ok := parsed["containerEnv"].(map[string]interface{})
	if !ok {
		t.Fatal("containerEnv should be a map")
	}

	// Should still have all defaults
	for _, key := range []string{"CLAUDE_CONFIG_DIR", "CLAUDE_PROFILE", "NODE_OPTIONS"} {
		if _, exists := envRaw[key]; !exists {
			t.Errorf("default %s should be present when ExtraEnv is nil", key)
		}
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd /Users/markalston/code/claudeup-lab && go test ./internal/lab/ -run "TestExtraEnv" -v`
Expected: FAIL -- `ExtraEnv` field unknown

- [ ] **Step 3: Add ExtraEnv field to DevcontainerConfig**

In `internal/lab/devcontainer.go`, add to the `DevcontainerConfig` struct (after `InitScript`):

```go
ExtraEnv   map[string]string // User-provided env vars that override all defaults
```

- [ ] **Step 4: Apply ExtraEnv in buildDevcontainerJSON**

In `internal/lab/devcontainer.go`, add after the optional env var loop (after line 87, before the `dc :=` map literal):

```go
	// Apply user-provided env vars (override anything)
	for k, v := range config.ExtraEnv {
		env[k] = v
	}
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `cd /Users/markalston/code/claudeup-lab && go test ./internal/lab/ -v`
Expected: All PASS (both new and existing tests)

- [ ] **Step 6: Commit**

```bash
git add internal/lab/devcontainer.go internal/lab/devcontainer_test.go
git commit -m "feat: add ExtraEnv support to devcontainer config"
```

---

### Task 3: Env Var Merge in Manager.Start

**Files:**

- Modify: `internal/lab/manager.go:43-52` (add fields to `StartOptions`)
- Modify: `internal/lab/manager.go:126-145` (merge env vars, pass to config)
- Create: `internal/lab/manager_test.go`

- [ ] **Step 1: Write failing test for merge logic**

Create `internal/lab/manager_test.go`. Since `Manager.Start` has heavy side effects (Docker, git, devcontainer CLI), test the merge logic via an unexported helper function. Both `envfile_test.go` and `manager_test.go` use `package lab` (internal tests), so `manager_test.go` can call the `writeEnvFile` helper already defined in `envfile_test.go`.

```go
// internal/lab/manager_test.go
package lab

import (
	"testing"
)

func TestMergeEnvVars(t *testing.T) {
	t.Run("file only", func(t *testing.T) {
		path := writeEnvFile(t, "FOO=from-file\nBAR=also-file\n")
		got, err := mergeEnvVars(path, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got["FOO"] != "from-file" || got["BAR"] != "also-file" {
			t.Errorf("unexpected result: %v", got)
		}
	})

	t.Run("flags only", func(t *testing.T) {
		got, err := mergeEnvVars("", map[string]string{"FOO": "from-flag"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got["FOO"] != "from-flag" {
			t.Errorf("got %q, want from-flag", got["FOO"])
		}
	})

	t.Run("flags override file", func(t *testing.T) {
		path := writeEnvFile(t, "FOO=from-file\nBAR=file-only\n")
		flags := map[string]string{"FOO": "from-flag"}
		got, err := mergeEnvVars(path, flags)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got["FOO"] != "from-flag" {
			t.Errorf("FOO = %q, want from-flag", got["FOO"])
		}
		if got["BAR"] != "file-only" {
			t.Errorf("BAR = %q, want file-only", got["BAR"])
		}
	})

	t.Run("both empty returns empty map", func(t *testing.T) {
		got, err := mergeEnvVars("", nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got) != 0 {
			t.Errorf("expected empty map, got %v", got)
		}
	})

	t.Run("invalid env file returns error", func(t *testing.T) {
		path := writeEnvFile(t, "INVALID\n")
		_, err := mergeEnvVars(path, nil)
		if err == nil {
			t.Fatal("expected error for invalid env file")
		}
	})
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd /Users/markalston/code/claudeup-lab && go test ./internal/lab/ -run TestMergeEnvVars -v`
Expected: FAIL -- `mergeEnvVars` not defined

- [ ] **Step 3: Add mergeEnvVars and update StartOptions**

In `internal/lab/manager.go`, add fields to `StartOptions`:

```go
type StartOptions struct {
	Project     string
	Profile     string
	Branch      string
	Name        string
	Features    []string
	BaseProfile string
	Firewall    bool
	InitScript  string
	EnvVars     map[string]string
	EnvFile     string
}
```

Add the unexported `mergeEnvVars` function (no need to export -- tested via `package lab` internal test):

```go
// mergeEnvVars reads an env file (if path is non-empty), then overlays
// flagVars on top. Flag values win over file values for the same key.
func mergeEnvVars(envFile string, flagVars map[string]string) (map[string]string, error) {
	merged := make(map[string]string)

	if envFile != "" {
		fileVars, err := ParseEnvFile(envFile)
		if err != nil {
			return nil, err
		}
		for k, v := range fileVars {
			merged[k] = v
		}
	}

	for k, v := range flagVars {
		merged[k] = v
	}

	return merged, nil
}
```

- [ ] **Step 4: Wire mergeEnvVars into Manager.Start**

In `Manager.Start`, add **before** the `dcConfig` struct literal (before line 126), after the worktree is created:

```go
	extraEnv, err := mergeEnvVars(opts.EnvFile, opts.EnvVars)
	if err != nil {
		m.worktrees.RemoveWorktree(barePath, worktreePath)
		return nil, fmt.Errorf("merge env vars: %w", err)
	}
```

Note: the worktree cleanup on error matches the existing pattern at lines 147-149 of `manager.go`. The worktree has already been created at this point (line 120), so we must clean it up on failure.

And add `ExtraEnv: extraEnv` to the `dcConfig` struct literal.

- [ ] **Step 5: Run tests to verify they pass**

Run: `cd /Users/markalston/code/claudeup-lab && go test ./internal/lab/ -v`
Expected: All PASS

- [ ] **Step 6: Commit**

```bash
git add internal/lab/manager.go internal/lab/manager_test.go
git commit -m "feat: merge env file and CLI env vars in Manager.Start"
```

---

### Task 4: CLI Flags

**Files:**

- Modify: `internal/commands/start.go` (add `--env` and `--env-file` flags, parse in `RunE`)

- [ ] **Step 1: Add flags and parsing to start command**

In `internal/commands/start.go`:

Add a local `[]string` variable alongside the existing `features` var:

```go
var envFlags []string
```

In `RunE`, after `opts.Features = features` and before the `InitScript` block, add:

```go
		// Parse --env KEY=VALUE flags into map
		if len(envFlags) > 0 {
			opts.EnvVars = make(map[string]string, len(envFlags))
			for _, e := range envFlags {
				idx := strings.IndexByte(e, '=')
				if idx < 0 {
					return fmt.Errorf("invalid --env format %q: expected KEY=VALUE", e)
				}
				key := e[:idx]
				if key == "" {
					return fmt.Errorf("invalid --env format %q: empty key", e)
				}
				opts.EnvVars[key] = e[idx+1:]
			}
		}
```

Add the `"strings"` import.

Add flag registrations at the bottom (after the `--init-script` flag):

```go
	cmd.Flags().StringArrayVar(&envFlags, "env", nil, "Set container environment variable as KEY=VALUE (repeatable)")
	cmd.Flags().StringVar(&opts.EnvFile, "env-file", "", "Read container environment variables from file")
```

- [ ] **Step 2: Run full test suite**

Run: `cd /Users/markalston/code/claudeup-lab && go test ./... -v`
Expected: All PASS (compilation confirms flags wire correctly)

- [ ] **Step 3: Commit**

```bash
git add internal/commands/start.go
git commit -m "feat: add --env and --env-file flags to start command"
```

---

### Task 5: Verify End-to-End Flag Wiring

- [ ] **Step 1: Run full test suite one more time**

Run: `cd /Users/markalston/code/claudeup-lab && go test ./... -v`
Expected: All PASS

- [ ] **Step 2: Verify CLI help output**

Run: `cd /Users/markalston/code/claudeup-lab && go run . start --help`
Expected: Output includes `--env` and `--env-file` flags with descriptions

- [ ] **Step 3: Verify build**

Run: `cd /Users/markalston/code/claudeup-lab && go build -o /dev/null .`
Expected: Clean build, no errors

- [ ] **Step 4: Commit any remaining changes if needed**

Only if verification steps required fixes.
