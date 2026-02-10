# claudeup-lab

Ephemeral devcontainer environments for testing Claude Code configurations.

Start a lab, experiment with plugins, skills, agents, and hooks, destroy it when you're done. Your host configuration stays untouched.

> **Status:** Early development. See the [design document](docs/plans/2026-02-10-claudeup-lab-design.md) for architecture details.

## What is this?

claudeup-lab creates isolated Docker containers pre-loaded with Claude Code and a [claudeup](https://github.com/claudeup/claudeup) profile of your choice. Each lab gets its own git worktree, its own Claude configuration, and its own set of extensions -- completely separate from your host.

This is different from Claude Code's built-in [sandbox mode](https://docs.anthropic.com/en/docs/claude-code/security#sandbox), which provides OS-level process isolation (filesystem and network restrictions) for security during a session. claudeup-lab provides **configuration isolation** -- different profiles, plugins, and extensions in throwaway containers. The two are complementary.

## Prerequisites

- [Docker Desktop](https://www.docker.com/products/docker-desktop/) (or Docker Engine)
- [devcontainer CLI](https://github.com/devcontainers/cli) (`npm install -g @devcontainers/cli`)
- Git

## Install

```bash
curl -fsSL https://raw.githubusercontent.com/claudeup/claudeup-lab/main/scripts/install.sh | bash
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
```

## Commands

| Command  | Description                           |
| -------- | ------------------------------------- |
| `start`  | Create and start a lab                |
| `list`   | Show all labs and their status        |
| `exec`   | Run a command inside a running lab    |
| `open`   | Attach VS Code to a running lab       |
| `stop`   | Stop a lab (volumes persist)          |
| `rm`     | Destroy a lab and all its data        |
| `doctor` | Check system health and prerequisites |

### `start` flags

| Flag                     | Default               | Description                                               |
| ------------------------ | --------------------- | --------------------------------------------------------- |
| `--project <path>`       | Current directory     | Project to create the lab from (must be a git repo)       |
| `--profile <name>`       | Current config        | claudeup profile to apply                                 |
| `--branch <name>`        | `lab/<profile>`       | Git branch name for the worktree                          |
| `--name <name>`          | `<project>-<profile>` | Display name for the lab                                  |
| `--feature <name[:ver]>` | None                  | Devcontainer feature to include (repeatable)              |
| `--base-profile <name>`  | None                  | Apply a base profile first, then overlay with `--profile` |

### Lab resolution

Labs can be identified by display name, UUID, partial UUID prefix, project name, or profile name. When run from inside a lab worktree, the lab is inferred automatically.

```bash
claudeup-lab exec --lab myproject-experimental   # display name
claudeup-lab exec --lab 976ae3b3                 # partial UUID
claudeup-lab exec                                # inferred from cwd
```

## How It Works

Each lab creates:

1. **A bare git clone** of your project (shared across labs of the same project)
2. **A git worktree** with its own branch, giving the lab isolated git state
3. **A devcontainer** with Docker volumes scoped by UUID, ensuring parallel labs don't interfere
4. **A claudeup profile** applied inside the container, installing the specified plugins, skills, and extensions

Labs store their data in `~/.claudeup-lab/` -- separate from both `~/.claude/` and `~/.claudeup/`.

## License

MIT
