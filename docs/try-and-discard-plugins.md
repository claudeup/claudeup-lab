# Try and Discard: Testing Claude Code Plugins in Ephemeral Labs

Test unfamiliar Claude Code plugins, skills, and agents in an isolated devcontainer without touching your host configuration. When you're done, destroy the lab and everything disappears.

## Prerequisites

- claudeup-lab installed (`claudeup-lab doctor` passes)
- A target project repo to test the plugin against (any git repo works)

## Choosing a profile mode

When starting a lab, you choose how much of your host config carries over:

| Mode                   | Command                               | What's in the container                                                      |
| ---------------------- | ------------------------------------- | ---------------------------------------------------------------------------- |
| **Snapshot** (default) | `claudeup-lab start`                  | Full copy of your current plugins, marketplaces, MCP servers, and extensions |
| **Named profile**      | `claudeup-lab start --profile <name>` | Only what's in that profile                                                  |
| **Clean slate**        | `claudeup-lab start --profile bare`   | Nothing -- just Claude Code and claudeup                                     |

**Snapshot mode** is convenient when you want your existing tools available alongside the new plugin. **Clean slate** is better when you want to test a plugin in isolation without interference from your current setup.

### Creating a bare profile (one-time setup)

```bash
# Create a minimal profile with just the official plugin marketplace
claudeup profile create bare \
  --description "empty profile for clean-slate labs" \
  --marketplace anthropics/claude-plugins-official
```

## Steps

### 1. Clone the plugin repo (if not already done)

```bash
git clone https://github.com/<org>/<plugin-repo>.git ~/code/<plugin-repo>
```

### 2. Start a lab for your target project

The target project is whatever codebase you want to test the plugin against. If you just want a sandbox, any git repo works (even `git init` on an empty directory).

```bash
cd ~/code/<target-project>

# With your current config carried over:
claudeup-lab start --name <target-project>-<plugin-name>

# Or with a clean slate:
claudeup-lab start --name <target-project>-<plugin-name> --profile bare
```

### 3. Install the plugin inside the container

Open a shell in the lab and clone/install the plugin. Where you clone it depends on what the plugin is:

| Plugin type                        | Clone to                   | Why                                                  |
| ---------------------------------- | -------------------------- | ---------------------------------------------------- |
| Claude Code skills/agents          | `~/.claude/skills/<repo>`  | Claude Code discovers skills from this directory     |
| Claude Code plugin (plugin.json)   | `~/.claude/plugins/<repo>` | Standard plugin install location                     |
| Standalone tool or repo to explore | `~/code/<repo>`            | Writable home directory, separate from the workspace |

**Important**: The `/workspaces/` parent directory is root-owned. Only the project subdirectory (`/workspaces/<lab-name>/`) is writable. Clone repos to your home directory (`~`), not to `/workspaces/`.

```bash
# Shell into the lab
claudeup-lab exec --lab <target-project>-<plugin-name>

# Once inside the container, clone to the appropriate location:
git clone https://github.com/<org>/<plugin-repo>.git ~/.claude/skills/<plugin-repo>
cd ~/.claude/skills/<plugin-repo>

# Run the plugin's setup (check its README for exact steps)
./setup        # or: bun install, npm install, make, etc.
```

### 4. Configure Claude Code to use the plugin

Most skill-based plugins need a mention in CLAUDE.md to activate their slash commands. Follow the plugin's README for what to add. Example:

```bash
# Still inside the container
cat >> ~/.claude/CLAUDE.md << 'EOF'

## <Plugin Name>
Use /<skill-name> for <description>. Available commands: /cmd1, /cmd2, /cmd3.
EOF
```

### 5. Test the plugin

Launch Claude Code inside the lab and try the commands:

```bash
# From inside the container shell
claude

# Or directly from the host
claudeup-lab exec --lab <target-project>-<plugin-name> -- claude
```

### 6. Tear down when done

```bash
# Stop the lab (preserves state if you want to come back)
claudeup-lab stop --lab <target-project>-<plugin-name>

# Or destroy everything -- container, volumes, worktree, branch
claudeup-lab rm --lab <target-project>-<plugin-name>
```

The plugin never touched your host `~/.claude/` directory.

## Automating prerequisites with `--init-script`

Some plugins need system packages or other setup that requires root. Instead of running these manually after every lab creation, use `--init-script` to run a host script automatically during container initialization:

```bash
claudeup-lab start --name mylab --profile bare --init-script ~/scripts/my-deps.sh
```

The script is bind-mounted read-only into the container and runs as the `node` user after all other initialization completes. Use `sudo` for operations that need root (passwordless sudo is available).

**Example init script for Playwright-based plugins:**

```bash
#!/bin/bash
# ~/scripts/playwright-deps.sh
sudo apt-get update -qq
sudo apt-get install -y -qq libnss3 libatk1.0-0 libatk-bridge2.0-0 \
  libdbus-1-3 libcups2 libatspi2.0-0 libxcomposite1 libxdamage1 \
  libxfixes3 libxrandr2 libgbm1 libasound2 libxkbcommon0
```

## Example: Testing gstack

gstack provides 9 specialized skills for Claude Code (planning, review, QA with headless browser, shipping, retrospectives).

**Requirements**: Git, Bun (pre-installed in claudeup-lab containers)

```bash
# 1. Clone gstack (already done if you have it locally)
git clone https://github.com/garrytan/gstack.git ~/code/gstack

# 2. Start a clean-slate lab with Playwright deps pre-installed
cd ~/code/myproject
claudeup-lab start --name myproject-gstack --profile bare \
  --init-script ~/scripts/playwright-deps.sh

# 3. Shell in and install gstack
claudeup-lab exec --lab myproject-gstack
# Inside container:
git clone https://github.com/garrytan/gstack.git ~/.claude/skills/gstack
cd ~/.claude/skills/gstack
./setup

# 4. Add gstack config to CLAUDE.md
cat >> ~/.claude/CLAUDE.md << 'GSTACK'

## gstack
Use the /browse skill from gstack for web browsing. Available skills:
/plan-ceo-review, /plan-eng-review, /review, /ship, /browse,
/qa, /qa-only, /setup-browser-cookies, /retro
GSTACK

# 5. Launch Claude Code and test
claude

# 6. When done, exit and destroy
exit
claudeup-lab rm --lab myproject-gstack
```

## Tips

- **Pick any target project**: The lab needs a git repo to create a worktree from. If you just want a playground, `git init` on an empty directory works.
- **Container has Bun and Node**: Most Claude Code plugins that need a JS runtime will work out of the box.
- **Volumes are isolated**: Each lab gets its own `~/.claude`, `~/.claudeup`, npm, and Bun directories. Nothing leaks between labs or back to the host.
- **Re-enter a stopped lab**: `claudeup-lab start` on a stopped lab resumes it. Your installed plugins persist across stop/start cycles.
- **List labs**: `claudeup-lab list` shows all labs and their status.
- **Cookie import won't work**: Plugins that import cookies from host browsers (like gstack's `/setup-browser-cookies`) can't access macOS Keychain from inside the container.
