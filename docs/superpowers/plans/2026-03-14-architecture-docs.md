# Architecture Documentation Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the stale design doc with a current architecture reference and FAQ.

**Architecture:** Pure documentation change. Create `docs/ARCHITECTURE.md` and `docs/FAQ.md`, delete the stale design doc, link both from the README.

**Tech Stack:** Markdown

**Spec:** `docs/superpowers/specs/2026-03-14-architecture-docs-design.md`

---

## Chunk 1: Architecture and FAQ docs

### Task 1: Create docs/ARCHITECTURE.md

**Files:**

- Create: `docs/ARCHITECTURE.md`

- [ ] **Step 1: Write the architecture document**

Create `docs/ARCHITECTURE.md` with six sections as specified. Full content:

```markdown
# Architecture

claudeup-lab creates ephemeral devcontainer environments for testing Claude Code configurations. The codebase has three layers:

- **CLI** (`cmd/claudeup-lab/`, `internal/commands/`) -- Cobra command handlers that parse flags and delegate to the orchestration layer
- **Orchestration** (`internal/lab/Manager`) -- coordinates the lab lifecycle: creating bare repos, worktrees, containers, and profiles
- **Infrastructure** (`internal/lab/`, `internal/docker/`) -- git worktree operations, devcontainer rendering, Docker container/volume management, and profile snapshotting

## Package Layout
```

cmd/claudeup-lab/ Entry point
internal/
commands/ Cobra command handlers
lab/ Core domain: Manager, Resolver, WorktreeManager,
ProfileManager, DevcontainerConfig, StateStore
docker/ Docker CLI wrapper: Client, ImageManager
embed/ Embedded Dockerfile, init scripts, features registry
scripts/ Install script

```

## Key Types

| Type | Package | Responsibility |
|------|---------|----------------|
| Manager | lab | Orchestrates lab lifecycle (start, remove, status) |
| Metadata | lab | Identity and paths for a single lab (ID, display name, project, profile, bare repo, worktree, branch) |
| Resolver | lab | Fuzzy lab lookup by UUID, display name, UUID prefix, project name, profile name, or current working directory |
| WorktreeManager | lab | Bare clone creation/refresh, git worktree operations, branch collision handling |
| ProfileManager | lab | Snapshots current config when no profile is specified, cleanup of temporary snapshots |
| DevcontainerConfig | lab | Renders devcontainer.json with mounts, environment variables, and features |
| StateStore | lab | Reads and writes lab metadata as JSON files |
| Client | docker | Container and volume operations via the docker CLI |
| ImageManager | docker | Pulls images from GHCR, falls back to building from the embedded Dockerfile |

## Data Flow

The `start` command is the most complex path and touches every layer:

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

Other commands (list, exec, stop, rm) follow a simpler pattern: resolve the lab, then perform a single operation.

## Storage Layout

```

~/.claudeup-lab/
state/<uuid>.json Lab metadata
repos/<project>.git Bare clone (shared across labs of same project)
workspaces/<display-name>/ Git worktree per lab

```

Each lab also creates Docker volumes scoped by UUID to prevent parallel labs from interfering with each other:

- `claudeup-lab-config-<uuid>` -- `~/.claude` inside the container
- `claudeup-lab-claudeup-<uuid>` -- `~/.claudeup` inside the container
- `claudeup-lab-npm-<uuid>` -- npm global packages
- `claudeup-lab-local-<uuid>` -- `~/.local` binaries
- `claudeup-lab-bun-<uuid>` -- bun runtime
- `claudeup-lab-bashhistory-<uuid>` -- shell history

All lab data lives under `~/.claudeup-lab/`, deliberately separate from `~/.claude/` and `~/.claudeup/`, so labs never touch host configuration.

## Design Decisions

- **Bare clone per project** -- avoids re-cloning the repository for every lab. Worktrees branch off the bare clone cheaply. A hash suffix on the directory name handles project name collisions from different source paths.

- **UUID-scoped Docker volumes** -- each lab's volumes include its UUID, so parallel labs for the same project can run without interfering with each other.

- **Optional mounts with existence checks** -- host paths like `~/.ssh` and `~/.claude.json` are only mounted if they exist. This means labs degrade gracefully on machines with minimal configuration.

- **Profile snapshotting** -- when no `--profile` is given, the current claudeup configuration is snapshotted so the lab captures a point-in-time copy. If the `claudeup` CLI is unavailable, the snapshot falls back to an empty `{}`.

- **Image pull with build fallback** -- the image manager tries to pull from GHCR first. If the registry is unreachable, it builds locally from the embedded Dockerfile. This keeps the tool usable offline.

- **Fuzzy lab resolution** -- labs can be identified by exact UUID, display name, UUID prefix, project name, profile name, or inferred from the current working directory. The resolver tries each in order, so users rarely need to type a full UUID.
```

- [ ] **Step 2: Commit**

```bash
git add docs/ARCHITECTURE.md
git commit -m "docs: add architecture reference"
```

### Task 2: Create docs/FAQ.md

**Files:**

- Create: `docs/FAQ.md`

- [ ] **Step 1: Write the FAQ document**

Create `docs/FAQ.md` with eight questions as specified. Full content:

````markdown
# FAQ

## What happens when I specify a profile that doesn't exist?

The profile name is passed through to the container without validation. The lab starts and the container receives `CLAUDE_PROFILE=<name>` as an environment variable, but `claudeup profile apply` will fail inside the container if the profile doesn't exist on the host. The worktree and container are still created -- you just won't have the expected extensions applied.

## What dev features can I include?

Five features are available from the embedded registry:

| Feature | OCI Reference                           |
| ------- | --------------------------------------- |
| go      | ghcr.io/devcontainers/features/go:1     |
| rust    | ghcr.io/devcontainers/features/rust:1   |
| python  | ghcr.io/devcontainers/features/python:1 |
| node    | ghcr.io/devcontainers/features/node:1   |
| java    | ghcr.io/devcontainers/features/java:1   |

Use `--feature <name>` or `--feature <name>:<version>`:

```bash
claudeup-lab start --feature python
claudeup-lab start --feature python:3.12
claudeup-lab start --feature go --feature rust
```
````

Without a version, the default from the registry is used. Unknown feature names are silently ignored.

## What volumes get mounted by default?

**Docker volumes (always created, scoped by lab UUID):**

| Volume                        | Container Path  |
| ----------------------------- | --------------- |
| claudeup-lab-config-{id}      | ~/.claude       |
| claudeup-lab-claudeup-{id}    | ~/.claudeup     |
| claudeup-lab-npm-{id}         | ~/.npm-global   |
| claudeup-lab-local-{id}       | ~/.local        |
| claudeup-lab-bun-{id}         | ~/.bun          |
| claudeup-lab-bashhistory-{id} | /commandhistory |

**Conditional bind mounts (only added if the source exists on the host):**

| Source                  | Container Path          | Access     |
| ----------------------- | ----------------------- | ---------- |
| ~/.claudeup/profiles    | ~/.claudeup/profiles    | readonly   |
| ~/.claudeup/ext         | ~/.claudeup/ext         | readonly   |
| ~/.claude-mem           | ~/.claude-mem           | read-write |
| ~/.ssh                  | ~/.ssh                  | readonly   |
| ~/.claude/settings.json | /tmp/base-settings.json | readonly   |
| ~/.claude.json          | ~/.claude.json          | read-write |

**Required bind mount:**

The bare repo clone is always mounted at its host path so that git worktree resolution works inside the container.

## What's the difference between stop and rm?

`stop` halts the container but preserves it along with all Docker volumes and the git worktree. Restarting a stopped lab is fast because nothing needs to be recreated.

`rm` destroys everything: the container, all Docker volumes, the git worktree, and the metadata file. The bare clone is kept because it may be shared with other labs of the same project.

## Can I run multiple labs for the same project?

Yes. Each lab gets its own UUID, Docker volumes, worktree, and branch. They share the bare clone for the project but are otherwise fully isolated. Use different `--profile` or `--name` values to distinguish them.

## Can I use claudeup-lab offline?

Mostly. The image manager tries to pull from GHCR first but falls back to building from the embedded Dockerfile if the registry is unreachable. The bare repo clone requires the source project to already be on disk. The only hard network dependency is if devcontainer features need to be pulled on first use.

## How do I get my changes out of a lab?

Two ways:

- The git worktree lives on the host at `~/.claudeup-lab/workspaces/<name>/`, so changes are directly accessible from the host filesystem
- From inside the container, `git push` works if SSH keys or tokens are mounted (both are included as conditional bind mounts)

## I want to try a new plugin without interference from other plugins. Do I need a new profile?

Yes, but it's quick. Create a profile that only includes the plugin you want to test, then start a lab with it:

```bash
claudeup profile create solo-test
claudeup ext enable ...              # enable just the plugin you want
claudeup profile save solo-test
claudeup-lab start --profile solo-test
```

Each lab starts with a fresh `~/.claude` volume, so only the extensions from the specified profile get applied. Without `--profile`, a lab snapshots your current config -- which includes all your existing plugins.

If you want to test a plugin layered on top of a base config, use `--base-profile`:

```bash
claudeup-lab start --base-profile default --profile solo-test
```

This applies `default` at user scope, then `solo-test` at project scope, so you get your foundation settings plus just the plugin under test.

````

- [ ] **Step 2: Commit**

```bash
git add docs/FAQ.md
git commit -m "docs: add FAQ for common usage questions"
````

### Task 3: Delete the stale design doc

**Files:**

- Delete: `docs/plans/2026-02-10-claudeup-lab-design.md`

- [ ] **Step 1: Remove the stale design doc**

```bash
git rm docs/plans/2026-02-10-claudeup-lab-design.md
```

- [ ] **Step 2: Remove the empty docs/plans/ directory if it's now empty**

Check if `docs/plans/` has any remaining files. If empty, git will clean it up automatically on the next checkout. No action needed.

- [ ] **Step 3: Commit**

```bash
git commit -m "docs: remove stale design doc from initial PR"
```

### Task 4: Link both docs from the README

**Files:**

- Modify: `README.md:5-6`

- [ ] **Step 1: Add links after the intro paragraph**

In `README.md`, after line 6 (the "Start a lab..." paragraph), add:

```markdown
See [Architecture](docs/ARCHITECTURE.md) for how it works under the hood, or the [FAQ](docs/FAQ.md) for common questions.
```

This goes between the intro paragraph and the `## What is this?` heading.

- [ ] **Step 2: Commit**

```bash
git add README.md
git commit -m "docs: link architecture and FAQ from README"
```

### Task 5: Push branch and update PR

- [ ] **Step 1: Push changes to remote**

```bash
git push
```

The branch `fix/remove-stale-design-doc-ref` already has PR #15 open. The new commits will be added to the existing PR.

- [ ] **Step 2: Update PR description**

Update PR #15 description to reflect the expanded scope:

```bash
gh pr edit 15 --title "docs: replace stale design doc with architecture and FAQ" --body "$(cat <<'EOF'
## Summary

- Creates `docs/ARCHITECTURE.md` with package layout, key types, data flow, storage layout, and design decisions
- Creates `docs/FAQ.md` with eight common usage questions and answers
- Removes stale `docs/plans/2026-02-10-claudeup-lab-design.md` (outdated paths, no longer authoritative)
- Links both new docs from the README

Closes #11
EOF
)"
```
