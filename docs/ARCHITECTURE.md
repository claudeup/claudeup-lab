# Architecture

claudeup-lab creates ephemeral devcontainer environments for testing Claude Code configurations. The codebase has three layers:

- **CLI** (`cmd/claudeup-lab/`, `internal/commands/`) -- Cobra command handlers that parse flags and delegate to the orchestration layer
- **Orchestration** (`internal/lab/Manager`) -- coordinates the lab lifecycle: creating bare repos, worktrees, containers, and profiles
- **Infrastructure** (`internal/lab/`, `internal/docker/`) -- git worktree operations, devcontainer rendering, Docker container/volume management, and profile snapshotting

## Package Layout

```
cmd/claudeup-lab/       Entry point
internal/
  commands/             Cobra command handlers
  lab/                  Core domain: Manager, Resolver, WorktreeManager,
                        ProfileManager, DevcontainerConfig, StateStore,
                        PresetStore (preset.go), override merging (override.go)
  docker/               Docker CLI wrapper: Client, ImageManager
embed/                  Embedded Dockerfile, init scripts, features registry
scripts/                Install script
```

## Key Types

| Type               | Package | Responsibility                                                                                                |
| ------------------ | ------- | ------------------------------------------------------------------------------------------------------------- |
| Manager            | lab     | Orchestrates lab lifecycle (start, remove, status)                                                            |
| Metadata           | lab     | Identity and paths for a single lab (ID, display name, project, profile, bare repo, worktree, branch)         |
| Resolver           | lab     | Fuzzy lab lookup by UUID, display name, UUID prefix, project name, profile name, or current working directory |
| WorktreeManager    | lab     | Bare clone creation/refresh, git worktree operations, branch collision handling                               |
| ProfileManager     | lab     | Snapshots current config when no profile is specified, cleanup of temporary snapshots                         |
| DevcontainerConfig | lab     | Renders devcontainer.json with mounts, environment variables, and features                                    |
| StateStore         | lab     | Reads and writes lab metadata as JSON files                                                                   |
| Client             | docker  | Container and volume operations via the docker CLI                                                            |
| Preset             | lab     | Reusable lab configuration with name, timestamps, and start options                                           |
| PresetStore        | lab     | CRUD for preset JSON files in ~/.claudeup-lab/presets/                                                        |
| StartConfig        | lab     | Resolved start options tracked on lab Metadata for --from-lab                                                 |
| Override           | lab     | Single field diff between preset and CLI flag values                                                          |
| ImageManager       | docker  | Pulls images from GHCR, falls back to building from the embedded Dockerfile                                   |

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

**Preset flow:**

```
preset save --from-lab <name>
  -> Resolver finds lab by name -> reads StartConfig from Metadata
  -> Creates Preset from StartConfig -> PresetStore writes JSON

start <preset-name> [--flag overrides]
  -> PresetStore loads Preset -> converts to StartOptions
  -> MergeWithOverrides applies CLI flags over preset defaults
  -> FormatOverrides renders diff-style summary to stderr
  -> Proceeds with normal start flow using merged options
```

## Storage Layout

```
~/.claudeup-lab/
  state/<uuid>.json                    Lab metadata
  repos/<project>.git                  Bare clone (shared across labs of same project)
  workspaces/<display-name>/           Git worktree per lab
  presets/<name>.json                  Saved preset configurations
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

- **Per-lab StartConfig tracking** -- resolved start options are stored on Metadata so any lab (running or stopped) can be a preset source. The StartConfig is deleted automatically when the lab is removed.

- **Explicit Preset struct** -- mirrors StartOptions fields but is a separate type for serialization stability. Conversion functions (`PresetFromStartOptions`, `Preset.ToStartOptions`) bridge the two.

- **Diff-style override display** -- when starting from a preset with overrides, changed values are shown with `-`/`+` markers. Flags that match the preset value produce no diff output.
