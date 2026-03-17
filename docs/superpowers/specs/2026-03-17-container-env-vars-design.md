# Container Environment Variables

## Summary

Add `--env KEY=VALUE` and `--env-file FILE` flags to `claudeup-lab start` so users can inject arbitrary environment variables into devcontainers. User-provided vars can override any existing var, including hardcoded infrastructure vars.

## CLI Surface

Two new flags on `claudeup-lab start`:

```
--env KEY=VALUE    Set container environment variable (repeatable)
--env-file FILE    Read environment variables from file
```

Both are optional and combinable. When both are provided, `--env` values take precedence over `--env-file` values (same key in both means the CLI flag wins).

## Env File Format

Docker-style, one `KEY=VALUE` per line:

```
# Comment lines ignored
GITHUB_PERSONAL_ACCESS_TOKEN=ghp_abc123

# Blank lines ignored
MY_API_KEY=sk-xyz
```

Parsing rules:

- Lines starting with `#` are comments
- Blank/whitespace-only lines are skipped
- Each non-comment line must contain `=` -- lines without `=` are an error
- Key is everything before the first `=`, value is everything after (values can contain `=`)
- No quoting support (quotes are literal) -- keeps it simple and matches Docker behavior
- File must exist or it's an error

## Data Flow

The env vars flow through existing structures with minimal additions:

1. `StartOptions` gets two new fields: `EnvVars map[string]string` and `EnvFile string`
2. The `start` command uses `StringArrayVar` (not `StringSliceVar` -- values may contain commas) to collect `--env` flags into a `[]string`. Note: this intentionally differs from the existing `--feature` flag which uses `StringSliceVar`, because env values like `FOO=a,b` would be incorrectly split by `StringSliceVar`. In `RunE`, the slice is parsed into `opts.EnvVars` (a `map[string]string`), validating `KEY=VALUE` format and non-empty keys. The `--env-file` path is passed through as-is.
3. `Manager.Start` reads the env file (if provided), merges with `EnvVars` (CLI flags win over file values), and passes the merged result into `DevcontainerConfig.ExtraEnv`
4. `DevcontainerConfig` gets one new field: `ExtraEnv map[string]string`
5. `buildDevcontainerJSON` builds the default env map as today (hardcoded + auto-forwarded), then applies `ExtraEnv` on top (last-write-wins for overrides)

The merge happens in two sites:

- `Manager.Start` merges levels 3 and 4 (env file + CLI flags) into a single `ExtraEnv` map
- `buildDevcontainerJSON` applies levels 1 and 2 (hardcoded + auto-forwarded) as today, then overlays `ExtraEnv`

The env file parsing happens in `Manager.Start` rather than the command layer so the logic is testable without cobra.

## Merge Precedence (lowest to highest)

1. Hardcoded defaults (`CLAUDE_CONFIG_DIR`, `NODE_OPTIONS`, etc.) -- applied in `buildDevcontainerJSON`
2. Auto-forwarded host vars (`GITHUB_TOKEN`, `CONTEXT7_API_KEY`, etc.) -- applied in `buildDevcontainerJSON`
3. Env file values (`--env-file`) -- merged in `Manager.Start` into `ExtraEnv`
4. CLI flag values (`--env`) -- merged in `Manager.Start` into `ExtraEnv`

## Error Handling

- `--env` value without `=`: error `invalid --env format "FOO": expected KEY=VALUE`
- `--env` value with empty key (e.g., `--env =value`): error `invalid --env format "=value": empty key`
- `--env-file` path doesn't exist: error `env file not found: /path/to/file`
- Env file line without `=` (and not a comment/blank): error `invalid line in env file %s (line %d): expected KEY=VALUE`
- Empty key in env file (e.g., `=value`): error `empty key in env file %s (line %d): "=value"`
- No warnings for overriding hardcoded vars -- full user control

## Testing

- **Env file parser**: unit tests covering comments, blank lines, valid pairs, values with `=`, missing `=` error, empty key error, missing file error
- **`buildDevcontainerJSON`**: test that `ExtraEnv` overrides default vars and adds new ones
- **Env var merge in `Manager.Start`**: test that CLI flags override env file values for the same key (lives in `internal/lab/manager_test.go`)

All unit-level, no E2E needed for this feature.

## Files to Modify

- `internal/commands/start.go` -- add `--env` and `--env-file` flags, parse `[]string` into map in `RunE`
- `internal/lab/manager.go` -- read env file, merge with CLI env vars, pass to config
- `internal/lab/manager_test.go` -- test env var merge precedence
- `internal/lab/devcontainer.go` -- add `ExtraEnv` field, apply in `buildDevcontainerJSON`
- `internal/lab/devcontainer_test.go` -- tests for env override behavior in `buildDevcontainerJSON`
- New: `internal/lab/envfile.go` -- env file parser
- New: `internal/lab/envfile_test.go` -- env file parser tests
