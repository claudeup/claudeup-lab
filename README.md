# claudeup-lab

Ephemeral devcontainer environments for testing Claude Code configurations.

Start a lab, experiment with plugins, skills, agents, and hooks, destroy it when you're done. Your host configuration stays untouched.

See [Architecture](docs/ARCHITECTURE.md) for how it works under the hood, or the [FAQ](docs/FAQ.md) for common questions.

## What is this?

claudeup-lab creates isolated Docker containers pre-loaded with Claude Code and a [claudeup](https://github.com/claudeup/claudeup) profile of your choice. Each lab gets its own git worktree, its own Claude configuration, and its own set of extensions -- completely separate from your host.

This is different from Claude Code's built-in [sandbox mode](https://docs.anthropic.com/en/docs/claude-code/security#sandbox), which provides OS-level process isolation (filesystem and network restrictions) for security during a session. claudeup-lab provides **configuration isolation** -- different profiles, plugins, and extensions in throwaway containers. The two are complementary.

## Prerequisites

- [Docker Desktop](https://www.docker.com/products/docker-desktop/) (or Docker Engine)
- [devcontainer CLI](https://github.com/devcontainers/cli) (`npm install -g @devcontainers/cli`)
- Git

## Install

```bash
# One-liner install (macOS/Linux)
curl -fsSL https://raw.githubusercontent.com/claudeup/claudeup-lab/main/scripts/install.sh | bash

# Install a specific version
VERSION=0.1.0 curl -fsSL https://raw.githubusercontent.com/claudeup/claudeup-lab/main/scripts/install.sh | bash

# Or build from source (version will show "dev")
go install github.com/claudeup/claudeup-lab/cmd/claudeup-lab@latest
```

## Quick Start

```bash
# Start a lab using your current configuration
cd ~/code/myproject
claudeup-lab start

# Start a lab with a specific profile
claudeup-lab start --profile experimental

# List running labs
claudeup-lab list

# Open a shell inside a lab
claudeup-lab exec --lab myproject-experimental

# Run Claude Code inside a lab
claudeup-lab exec --lab myproject-experimental -- claude

# Attach VS Code to a lab
claudeup-lab open --lab myproject-experimental

# Stop a lab (preserves state for fast restart)
claudeup-lab stop --lab myproject-experimental

# Destroy a lab completely
claudeup-lab rm --lab myproject-experimental

# Save a lab's configuration as a reusable preset
claudeup-lab preset save my-config --from-lab myproject-experimental

# Start a lab from a saved preset
claudeup-lab start my-config

# Start with overrides
claudeup-lab start my-config --branch lab/experiment

# List saved presets
claudeup-lab preset list

# View preset details
claudeup-lab preset show my-config
```

## Commands

| Command  | Description                                              |
| -------- | -------------------------------------------------------- |
| `start`  | Create and start a lab                                   |
| `list`   | Show all labs and their status                           |
| `exec`   | Run a command inside a running lab                       |
| `open`   | Attach VS Code to a running lab                          |
| `stop`   | Stop a lab (volumes persist)                             |
| `rm`     | Destroy a lab and all its data                           |
| `preset` | Save, list, show, and delete reusable lab configurations |
| `doctor` | Check system health and prerequisites                    |

### `start` flags

Usage: `claudeup-lab start [preset-name] [flags]`

When a preset name is given as a positional argument, its saved values are used as defaults. Any flags provided on the command line override the preset values. Changed fields are displayed with diff-style `-`/`+` markers before the lab starts.

| Flag                     | Default               | Description                                                                |
| ------------------------ | --------------------- | -------------------------------------------------------------------------- |
| `--project <path>`       | Current directory     | Project to create the lab from (must be a git repo)                        |
| `--profile <name>`       | Current config        | claudeup profile to apply                                                  |
| `--branch <name>`        | `lab/<profile>`       | Git branch name for the worktree                                           |
| `--name <name>`          | `<project>-<profile>` | Display name for the lab                                                   |
| `--feature <name[:ver]>` | None                  | Devcontainer feature to include (repeatable)                               |
| `--base-profile <name>`  | None                  | Apply a base profile first, then overlay with `--profile`                  |
| `--firewall`             | Off                   | Enable container firewall restricting network to allowed services          |
| `--init-script <path>`   | None                  | Host script to run after devcontainer setup (runs as node, sudo available) |
| `--env <KEY=VALUE>`      | None                  | Set container environment variable (repeatable)                            |
| `--env-file <path>`      | None                  | Read container environment variables from file                             |

### Environment variables

The container receives environment variables from the host at creation time.

**Always set:**

| Variable               | Source                  | Purpose                                                             |
| ---------------------- | ----------------------- | ------------------------------------------------------------------- |
| `CLAUDE_CONFIG_DIR`    | Hardcoded               | Claude Code config path inside the container (`/home/node/.claude`) |
| `CLAUDE_PROFILE`       | `--profile` flag        | Profile to apply during container initialization                    |
| `NODE_OPTIONS`         | Hardcoded               | V8 memory limit for Claude Code (`--max-old-space-size=4096`)       |
| `CLAUDE_CONFIG_BRANCH` | `$CLAUDE_CONFIG_BRANCH` | Branch of the config repo to use (default: `main`)                  |

**Conditional** (only set when the host value is non-empty):

| Variable              | Source                  | Purpose                                                   |
| --------------------- | ----------------------- | --------------------------------------------------------- |
| `GIT_USER_NAME`       | `git config user.name`  | Git identity inside the container                         |
| `GIT_USER_EMAIL`      | `git config user.email` | Git identity inside the container                         |
| `GITHUB_TOKEN`        | `$GITHUB_TOKEN`         | GitHub API authentication for `gh` CLI and git operations |
| `CONTEXT7_API_KEY`    | `$CONTEXT7_API_KEY`     | API key for the Context7 MCP documentation server         |
| `CLAUDE_CONFIG_REPO`  | `$CLAUDE_CONFIG_REPO`   | Git repo to clone for shared Claude Code configuration    |
| `CLAUDE_BASE_PROFILE` | `--base-profile` flag   | Foundation profile applied before the main profile        |

The container's `~/.claude` starts as an empty Docker volume populated during init -- not a copy of your host directory. Without `--profile`, your current config is snapshotted and applied. With `--profile`, only the named profile is applied. See the [FAQ](docs/FAQ.md#what-ends-up-in-the-containers-claude-directory) for details.

### Lab resolution

Labs can be identified by display name, UUID, partial UUID prefix, project name, or profile name. When run from inside a lab worktree, the lab is inferred automatically.

```bash
claudeup-lab exec --lab myproject-experimental   # display name
claudeup-lab exec --lab 976ae3b3                 # partial UUID
claudeup-lab exec                                # inferred from cwd
```

## Presets

Presets are saved lab configurations that you can reuse. Instead of remembering which flags to pass every time you start a lab, save the configuration once and start from it by name.

### Saving a preset

The easiest way to create a preset is from an existing lab:

```bash
claudeup-lab preset save my-config --from-lab myproject-experimental
```

This captures the lab's resolved start options -- project, profile, branch, features, environment variables, and all other flags that were used when the lab was created.

You can also create a preset from explicit flags without a running lab:

```bash
claudeup-lab preset save gpu-lab --project ~/code/ml-project --profile gpu --feature python --env CUDA_VISIBLE_DEVICES=0
```

### Starting from a preset

Pass the preset name as a positional argument to `start`:

```bash
claudeup-lab start my-config
```

Any flag provided on the command line overrides the preset value. Changed fields are displayed with diff-style markers before the lab starts:

```bash
claudeup-lab start my-config --branch lab/experiment
# Using preset 'my-config'
#   project:      /Users/you/code/myproject
#   profile:      experimental
# - branch:       lab/experimental
# + branch:       lab/experiment
```

Flags that match the preset value produce no diff output.

### Managing presets

```bash
# List all saved presets
claudeup-lab preset list

# View details of a specific preset
claudeup-lab preset show my-config

# Delete a preset
claudeup-lab preset delete my-config
```

### Sharing presets

Preset files are human-readable JSON stored in `~/.claudeup-lab/presets/`. Each preset is a single `<name>.json` file that can be copied between machines.

## How It Works

Each lab creates:

1. **A bare git clone** of your project (shared across labs of the same project)
2. **A git worktree** with its own branch, giving the lab isolated git state
3. **A devcontainer** with Docker volumes scoped by UUID, ensuring parallel labs don't interfere
4. **A claudeup profile** applied inside the container, installing the specified plugins, skills, and extensions

Labs store their data in `~/.claudeup-lab/` -- separate from both `~/.claude/` and `~/.claudeup/`.

## License

MIT
