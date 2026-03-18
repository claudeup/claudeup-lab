# Fix MCP Server "Already Exists" Error During Profile Apply

**Date:** 2026-03-18
**Status:** Approved
**Issue:** Saving current user settings as a profile and immediately applying them produces a spurious error for MCP servers that already exist.

## Problem

When `claude mcp add` encounters a server that already exists at the target scope, it exits with status 1 and outputs `"MCP server <name> already exists in user config"`. Unlike plugins and marketplaces (which detect "already installed" output and treat it as a skip), all MCP install paths treat any error as fatal.

Additionally, `--replace` clears plugins via `clearScope()` but does not clear MCP servers, so a replace-mode apply always hits the "already exists" condition for MCP.

## Affected Code Paths

| #   | Path                   | File                  | Lines            | Description                                        |
| --- | ---------------------- | --------------------- | ---------------- | -------------------------------------------------- |
| 1   | `installMCPServersCLI` | `apply.go`            | 964-970          | Used by `ApplyAllScopes` for user/local scope      |
| 2   | User-scope diff apply  | `apply.go`            | 691-698          | Inline loop in `ApplyWithOptions` user-scope path  |
| 3   | Local-scope apply      | `apply.go`            | 452-462          | Inline loop in `ApplyWithOptions` local-scope path |
| 4   | Concurrent apply       | `apply_concurrent.go` | 157-164, 197-202 | MCP jobs in worker pool (`ApplyConcurrently`)      |

**Note:** `ApplyAllScopes` routes user and local MCP through path #1 (`installMCPServersCLI`). Paths #2 and #3 are separate inline loops only reached via `ApplyWithOptions`. All four paths need the fix independently.

## Design

### 1. Sentinel Error and Helper

Add to `apply.go`:

```go
var errMCPAlreadyExists = errors.New("MCP server already exists")

// checkMCPAlreadyExists inspects the output and error from a `claude mcp add`
// command. Returns nil if the command succeeded, errMCPAlreadyExists if the
// server was already configured, or the original error with output context.
func checkMCPAlreadyExists(output string, err error) error {
    if err == nil {
        return nil
    }
    if strings.Contains(output, "already exists") {
        return errMCPAlreadyExists
    }
    return fmt.Errorf("%w\n  Output: %s", err, strings.TrimSpace(output))
}
```

### 2. Result Type Changes

**`ApplyResult`** -- add field:

```go
MCPServersAlreadyPresent []string // MCP servers that were already configured
```

**`ConcurrentApplyResult`** -- add field:

```go
MCPServersSkipped []string // MCP servers skipped (already configured)
```

### 3. Fix All Four Install Paths

All paths use `checkMCPAlreadyExists` to classify the error:

- `nil` -- append to `MCPServersInstalled`
- `errMCPAlreadyExists` -- append to `MCPServersAlreadyPresent` (or `MCPServersSkipped` in concurrent path)
- other error -- append to `Errors`

**Concurrent path detail:** The `Execute` closure currently discards output. Change it to capture output, call `checkMCPAlreadyExists` inside the closure, and return `errMCPAlreadyExists` when detected. The helper cannot be called in the result-processing loop because `JobResult` does not carry output -- only the error. The result-processing loop checks `errors.Is(jr.Error, errMCPAlreadyExists)` and routes to `MCPServersSkipped`.

**Naming convention:** `ApplyResult` uses "AlreadyPresent" (matching `PluginsAlreadyPresent`). `ConcurrentApplyResult` uses "Skipped" (matching `PluginsSkipped`, `MarketplacesSkipped`). Each struct follows its own established convention.

### 4. Make `--replace` Clear MCP Servers

Handle MCP clearing inside `ApplyAllScopes` (option b) rather than modifying `clearScope`, because the executor is available there.

When `opts.ReplaceUserScope` is true and user scope has MCP servers to install:

1. Read existing user-scope MCP servers via `ReadMCPServersForScope(claudeJSONPath, "", "user")`
2. Remove each via `executor.RunWithOutput("mcp", "remove", name)` -- ignore errors (server may already be gone)
3. Proceed with `installMCPServersCLI` as normal

The "already exists" detection from step 3 acts as a safety net if removal fails.

### 5. Test Strategy

**`apply_test.go`:**

- `checkMCPAlreadyExists`: nil error, "already exists" sentinel, other error pass-through
- `installMCPServersCLI` with "already exists" output populates `MCPServersAlreadyPresent`
- User-scope diff apply and local-scope apply with "already exists" output

**`apply_allscopes_test.go`:**

- `ApplyAllScopes` with `ReplaceUserScope=true` -- verify `mcp remove` before `mcp add`
- `ApplyAllScopes` where `mcp add` returns "already exists" -- no error in result

**`apply_concurrent_test.go` (or existing file):**

- Concurrent apply with MCP "already exists" populates `MCPServersSkipped`, no errors

All tests use existing `FakeExecutor` patterns.
