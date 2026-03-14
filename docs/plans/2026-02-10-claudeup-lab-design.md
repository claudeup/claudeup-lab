# claudeup-lab: Ephemeral Configuration Isolation for Claude Code

## Purpose

claudeup-lab creates disposable devcontainer environments for testing Claude Code configurations (plugins, skills, agents, hooks, commands) without affecting your host setup. Start a lab, experiment with a profile, destroy it when you're done.

This is distinct from Claude Code's built-in sandbox, which provides OS-level process isolation (filesystem/network restrictions) for security during a session. claudeup-lab provides configuration isolation -- different profiles, plugins, and extensions in ephemeral containers. The two are complementary: you'd use Claude's sandbox inside a lab for security, and claudeup-lab around the whole session for config isolation.

## Distribution

- **Repo:** `github.com/claudeup/claudeup-lab`
- **Language:** Go
- **Binary:** `claudeup-lab`
- **Install:** GitHub releases with install script (matching claudeup's pattern)
- **Image:** `ghcr.io/claudeup/claudeup-lab:latest` with embedded Dockerfile fallback

## Architecture

Three layers:

1. **Go CLI (`claudeup-lab`)** -- Handles the full lifecycle: git worktree setup, devcontainer.json generation, container orchestration, state management. Distributed as a single binary via GitHub releases.

2. **Base Docker image (`ghcr.io/claudeup/claudeup-lab`)** -- Pre-built image with Claude Code, claudeup, Node.js, and common dev tools. Init scripts baked in. Published to GHCR, with an embedded Dockerfile fallback if the registry is unreachable.

3. **claudeup profiles** -- Source of truth for what gets installed. The CLI reads profiles from `~/.claudeup/profiles/`. When no profile is specified, the user's current configuration is snapshotted automatically.

## Command Structure

```
claudeup-lab start    [--project <path>] [--profile <name>] [--branch <name>]
                      [--name <name>] [--feature <name[:ver]>] [--base-profile <name>]
claudeup-lab list
claudeup-lab exec     --lab <name> [-- <command>]
claudeup-lab open     --lab <name>
claudeup-lab stop     --lab <name>
claudeup-lab rm       --lab <name>
claudeup-lab doctor
```

### Command Details

**`start`** -- Create and start a lab. `--project` defaults to cwd (must be a git repo). `--profile` defaults to a snapshot of the current config. Creates a bare clone, git worktree, generates devcontainer.json, and launches the container.

**`list`** -- Show all labs with display name, short UUID, project, profile, and status (running/stopped/orphaned).

**`exec`** -- Run a command inside a running lab. Without `--` args, opens an interactive bash shell. Subsumes the old `claude` subcommand -- use `claudeup-lab exec --lab myapp -- claude` to launch Claude Code.

**`open`** -- Attach VS Code to a running lab container.

**`stop`** -- Stop the container. Docker volumes persist so the next `start` is fast.

**`rm`** -- Full teardown with confirmation: stop container, remove Docker volumes, remove worktree, remove metadata. Cleans up snapshot profiles if applicable. Offers to remove the bare repo if no remaining worktrees use it.

**`doctor`** -- Check system health: Docker status, devcontainer CLI, base image availability, orphaned labs, stale worktrees, disk usage.

### Lab Resolution

Labs are resolved by `--lab` flag with fuzzy matching:

1. Exact UUID match
2. Display name match
3. Partial UUID prefix
4. Project name match
5. Profile name match
6. CWD inference (when run from inside a lab worktree)

Ambiguous matches list available options.

### Defaults

```bash
claudeup-lab start                              # cwd project, current config snapshot
claudeup-lab start --profile experimental       # cwd project, named profile
claudeup-lab start --project ~/code/myapp       # explicit project, current config snapshot
```

## Storage and State

All lab runtime data lives under `~/.claudeup-lab/`, separate from both claudeup and Claude Code:

```
~/.claudeup-lab/
├── state/                    # metadata (one JSON file per lab)
│   └── <uuid>.json
├── repos/                    # bare clones (one per source project)
│   └── <project-name>.git/
└── workspaces/               # git worktrees (one per lab)
    └── <display-name>/
        └── .devcontainer/
            └── devcontainer.json
```

### Metadata Format

```json
{
  "id": "976ae3b3-b311-4a5f-bd8a-f1dbbcc462ad",
  "display_name": "myapp-base",
  "project": "/Users/mark/code/myapp",
  "project_name": "myapp",
  "profile": "base",
  "bare_repo": "/Users/mark/.claudeup-lab/repos/myapp.git",
  "worktree": "/Users/mark/.claudeup-lab/workspaces/myapp-base",
  "branch": "lab/base",
  "created": "2026-02-10T12:34:56Z"
}
```

## Mount Strategy

### Required (fail if missing)

- `~/.claude.json` -- authentication state
- Docker running

### Optional (skip silently if missing)

Each optional mount is checked with a path existence test. Missing mounts are logged at info level (e.g., `skipping ~/.claude-mem mount (directory not found)`).

- `~/.claudeup/profiles` -- profile definitions (required only if `--profile` is explicitly named)
- `~/.claudeup/local` -- local extensions (bind readonly)
- `~/.claude/settings.json` -- base settings seed (bind readonly)
- `~/.claude-mem` -- persistent memory across sessions
- `~/.ssh` -- git access (bind readonly)

### Per-Lab Volumes (Docker volumes scoped by UUID)

- `claudeup-lab-config-<uuid>` -- `~/.claude`
- `claudeup-lab-claudeup-<uuid>` -- `~/.claudeup`
- `claudeup-lab-npm-<uuid>` -- `~/.npm-global`
- `claudeup-lab-local-<uuid>` -- `~/.local`
- `claudeup-lab-bun-<uuid>` -- `~/.bun`
- `claudeup-lab-bashhistory-<uuid>` -- `/commandhistory`

## Git Isolation

- **One bare clone per source project** in `~/.claudeup-lab/repos/`. Shared across all labs of the same project.
- **Each lab gets its own worktree** created from the bare clone. Isolated branch, index, and HEAD.
- **Source marker file** in each bare repo prevents project name collisions. Hash suffix disambiguates when two different projects share a name.
- **Branch collision handling** -- if a branch is already checked out in another worktree, append a short UUID suffix.
- **Default branch prefix** is `lab/` (e.g., `lab/base`, `lab/experimental`).
- **Clone strategy** -- clone from the project's `origin` remote when available, falling back to cloning the local project directly.

## Profile Snapshotting

When `--profile` is omitted:

1. CLI calls `claudeup profile save` to a temporary profile name (e.g., `_lab-snapshot-<short-uuid>`)
2. Snapshot is saved to `~/.claudeup/profiles/`
3. Lab container applies that snapshot via `claudeup setup --profile <name>`
4. Snapshot profile is cleaned up when the lab is `rm`'d

When `--profile` is explicitly provided, it's used directly -- no snapshotting.

## Feature Injection

`--feature go:1.23` maps to a devcontainer feature OCI reference. The feature registry ships as an embedded JSON file in the binary via `go:embed`. The devcontainer.json template is rendered using Go's `text/template`.

## Base Docker Image

Pre-built and published to `ghcr.io/claudeup/claudeup-lab:latest`.

Contents:

- Node.js 22 (base)
- Claude Code (official installer)
- claudeup
- Bun
- Common dev tools: git, git-lfs, gh, jq, yq, ripgrep, fd-find, make, python3
- Init scripts baked in at `/usr/local/bin/`:
  - `init-claude-config.sh` -- git identity, GitHub auth, settings.json seeding
  - `init-config-repo.sh` -- clone and deploy config repo
  - `init-claudeup.sh` -- install claudeup, apply profile, sync local items

The embedded Dockerfile in the Go binary serves as a fallback when the registry is unreachable. `claudeup-lab start` pulls from GHCR first, builds locally if the pull fails.

## Error Handling and Prerequisites

### Prerequisites checked at startup

- Docker running (hard requirement)
- `devcontainer` CLI on PATH (hard requirement -- provide install hint if missing)
- Git available (hard requirement)
- claudeup available (soft requirement -- needed only for profile snapshotting; named profiles work without it since the file is just JSON)

### Container startup failures

If `devcontainer up` fails, the CLI cleans up the worktree and metadata automatically rather than leaving orphaned state.

### Orphan detection

The `list` command detects orphaned labs (metadata exists, worktree or container missing) and flags them so `rm` can clean up.

## Project Structure

```
github.com/claudeup/claudeup-lab/
├── cmd/
│   └── claudeup-lab/
│       └── main.go                 # CLI entry point
├── internal/
│   ├── lab/
│   │   ├── manager.go              # core orchestration (start, stop, rm)
│   │   ├── state.go                # metadata read/write
│   │   ├── resolve.go              # fuzzy lab resolution
│   │   ├── worktree.go             # bare clone + worktree management
│   │   ├── devcontainer.go         # template rendering + mount logic
│   │   └── profile.go              # snapshot + profile handling
│   ├── docker/
│   │   ├── client.go               # Docker status checks, container ops
│   │   └── image.go                # GHCR pull + local build fallback
│   └── commands/
│       ├── start.go
│       ├── list.go
│       ├── exec.go
│       ├── open.go
│       ├── stop.go
│       ├── rm.go
│       └── doctor.go
├── embed/
│   ├── Dockerfile                  # embedded fallback
│   ├── devcontainer.template.json  # Go text/template
│   └── features.json               # feature registry
├── scripts/
│   └── install.sh                  # curl-pipe installer
├── .goreleaser.yaml                # cross-platform builds
├── Makefile
├── go.mod
├── go.sum
└── README.md
```

- **`internal/`** keeps the API surface private -- this is a CLI tool, not a library
- **`embed/`** holds all files bundled into the binary via `go:embed`
- **`.goreleaser.yaml`** handles cross-compilation for darwin/linux, amd64/arm64
- **CLI framework:** cobra (matching claudeup)

## Migration Path

### Phase 1: Build to feature parity

All six commands working (start, list, exec, open, stop, rm). Published to GHCR. Install script working. Tested on macOS and Linux.

### Phase 2: Run side-by-side

claudeup-lab uses `~/.claudeup-lab/` -- completely separate from `~/.claude-sandboxes/`. No migration tool needed since the two don't share state. Use claudeup-lab for new labs, existing sandboxes stay on the old script. Validate with real usage.

### Phase 3: Deprecate when comfortable

Clean up `~/.claude-sandboxes/` manually (old labs are ephemeral -- nothing to migrate). Remove `scripts/claude-sandbox`, `devcontainer-base/`, and Makefile sandbox targets from the config repo.
