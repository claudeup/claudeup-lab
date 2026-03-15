# Architecture Documentation Design

## Problem

The README referenced `docs/plans/2026-02-10-claudeup-lab-design.md` as the architecture
reference, but that design doc is a point-in-time artifact from PR #1 with stale information
(e.g., `~/.claudeup/local` instead of `~/.claudeup/ext`). Issue #11 removed the link.
We need a current, maintained architecture document to replace it, plus a FAQ for common
usage questions.

## Decision

Create two new documents and remove the stale one:

- `docs/ARCHITECTURE.md` -- living architecture reference
- `docs/FAQ.md` -- answers to common usage questions
- Delete `docs/plans/2026-02-10-claudeup-lab-design.md`
- Add links to both from the README

## Audience

Contributors and curious users. Contributor-focused (package layout, key types, data flow)
but with enough high-level framing that a user deciding whether to use the tool can follow it.

## docs/ARCHITECTURE.md

Six sections, each scaled to its complexity:

### 1. Overview

Brief paragraph explaining what claudeup-lab is and the three-layer architecture:

- CLI commands (`cmd/`, `internal/commands/`)
- Orchestration (`internal/lab/Manager`)
- Infrastructure (`internal/lab/{worktree,devcontainer,profile}`, `internal/docker/`)

### 2. Package Layout

Tree diagram of Go packages with one-line responsibility per package:

```
cmd/claudeup-lab/       Entry point
internal/
  commands/             Cobra command handlers
  lab/                  Core domain: Manager, Resolver, WorktreeManager,
                        ProfileManager, DevcontainerConfig, StateStore
  docker/               Docker CLI wrapper: Client, ImageManager
embed/                  Embedded Dockerfile, init scripts, features registry
scripts/                Install script
```

### 3. Key Types

One-liner per domain object:

- Manager -- orchestrates lab lifecycle (start, remove, status)
- Metadata -- identity and paths for a single lab
- Resolver -- fuzzy lab lookup (UUID, display name, prefix, project, profile, or current working directory)
- WorktreeManager -- bare clone creation/refresh, git worktree ops, branch collision handling
- ProfileManager -- snapshots current config when no profile specified
- DevcontainerConfig -- renders devcontainer.json with mounts, env vars, features
- StateStore -- reads/writes lab metadata as JSON
- docker.Client -- container and volume operations via docker CLI
- docker.ImageManager -- pull from GHCR, fallback to embedded Dockerfile build

### 4. Data Flow

Walkthrough of `start` command as the most complex path:

```
claudeup-lab start --project ~/myapp --profile experimental
  -> StartCmd parses flags into StartOptions
  -> Manager.Start() validates prerequisites, generates UUID
  -> WorktreeManager: EnsureBareRepo -> CreateWorktree
  -> ProfileManager: Snapshot (if no --profile given)
  -> ImageManager: EnsureImage (pull or build fallback)
  -> DevcontainerConfig: Render devcontainer.json
  -> StateStore: Save metadata
  -> devcontainer up
```

Other commands (list, exec, stop, rm) are straightforward resolver + single operation
and don't need dedicated flow diagrams.

### 5. Storage Layout

```
~/.claudeup-lab/
  state/<uuid>.json                    Lab metadata
  repos/<project>.git                  Bare clone (shared across labs of same project)
  workspaces/<display-name>/           Git worktree per lab
```

Docker volumes scoped by UUID: `claudeup-lab-{category}-<uuid>` where category is
one of: config, claudeup, npm, local, bun, bashhistory.

Call out that this is deliberately separate from `~/.claude/` and `~/.claudeup/`.

### 6. Design Decisions

Short list of non-obvious choices with rationale:

- Bare clone per project (avoids re-cloning; worktrees branch cheaply; hash suffix for collisions)
- UUID-scoped Docker volumes (parallel lab isolation)
- Optional mounts with existence checks (graceful degradation for missing host paths)
- Profile snapshotting (point-in-time copy when no --profile; falls back to empty `{}` if the `claudeup` CLI is unavailable)
- Image pull with build fallback (GHCR first, embedded Dockerfile if unreachable)
- Fuzzy lab resolution (cascading lookup so users don't need UUIDs)

## docs/FAQ.md

Eight questions:

### 1. What happens when I specify a profile that doesn't exist?

The profile name is passed through without validation. The lab starts and the container
gets `CLAUDE_PROFILE=<name>`, but `claudeup profile apply` will fail inside the container
if the profile doesn't exist on the host. The worktree and container are still created.

### 2. What dev features can I include?

Five registered features from `embed/features.json`:

- go, rust, python, node, java

Format: `--feature <name>` or `--feature <name>:<version>`. Without a version, the
default from the registry is used. Unknown feature names are silently ignored.

### 3. What volumes get mounted by default?

**Docker volumes (always created, scoped by lab UUID):**

- bashhistory, config, claudeup, npm, local, bun

**Conditional bind mounts (only if source exists on host):**

- claudeup profiles directory (readonly)
- claudeup ext directory (readonly)
- ~/.claude-mem (read-write)
- ~/.ssh (readonly)
- ~/.claude/settings.json (readonly, mounted to /tmp/base-settings.json)
- ~/.claude.json (read-write)

**Required bind mount:**

- Bare repo path (for git worktree resolution)

### 4. What's the difference between stop and rm?

`stop` halts the container but preserves it along with all Docker volumes and the
git worktree. Restarting is fast because nothing needs to be recreated.

`rm` destroys everything: container, all Docker volumes, the git worktree, and the
metadata file. The bare clone is kept (it's shared with other labs of the same project).

### 5. Can I run multiple labs for the same project?

Yes. Each lab gets its own UUID, Docker volumes, worktree, and branch. They share
the bare clone for the project but are otherwise fully isolated. Use different
`--profile` or `--name` values to distinguish them.

### 6. Can I use claudeup-lab offline?

Mostly. The image manager tries to pull from GHCR first but falls back to building
from the embedded Dockerfile if the registry is unreachable. Git clone for the bare
repo requires the source project to be local. The only hard network dependency is if
devcontainer features need to be pulled on first use.

### 7. How do I get my changes out of a lab?

Two ways:

- The git worktree lives on the host at `~/.claudeup-lab/workspaces/<name>/`, so
  changes are directly accessible from the host filesystem
- From inside the container, `git push` works if SSH keys or tokens are mounted

### 8. I want to try a new plugin without interference from other plugins. Do I need a new profile?

Yes, but it's quick. Create a profile that only includes the plugin you want to test,
then start a lab with it:

```bash
claudeup profile create solo-test    # create a minimal profile
claudeup ext enable ...              # enable just the plugin you want
claudeup profile save solo-test      # save it
claudeup-lab start --profile solo-test
```

Each lab starts with a fresh `~/.claude` volume, so only the extensions from the
specified profile get applied. Without `--profile`, a lab snapshots your current
config -- which includes all your existing plugins.

If you want to test a plugin layered on top of a base config, use `--base-profile`:

```bash
claudeup-lab start --base-profile default --profile solo-test
```

This applies `default` at user scope, then `solo-test` at project scope, so you get
your foundation settings plus just the plugin under test.

## README Changes

Add a short line after the intro paragraph linking to both docs:

```markdown
See [Architecture](docs/ARCHITECTURE.md) for how it works under the hood,
or the [FAQ](docs/FAQ.md) for common questions.
```

## Files Changed

- Create `docs/ARCHITECTURE.md`
- Create `docs/FAQ.md`
- Delete `docs/plans/2026-02-10-claudeup-lab-design.md`
- Edit `README.md` (add links after intro)
