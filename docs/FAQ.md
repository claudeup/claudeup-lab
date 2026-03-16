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

## Does my host ~/.claude directory get carried into the container?

No. The container's `~/.claude` is a fresh Docker volume (`claudeup-lab-config-{id}`), not a bind mount of your host directory. Your host `~/.claude` is never mounted into the container.

What does get carried in from the host (each is a conditional bind mount, only added if the source exists):

- `~/.claudeup/profiles` (readonly) -- so named and snapshot profiles are available for the entrypoint to apply
- `~/.claudeup/ext` (readonly) -- your extensions
- `~/.claude/settings.json` -- mounted to `/tmp/base-settings.json`, not directly into `~/.claude`
- `~/.claude.json` -- your credentials
- `~/.ssh` (readonly)
- `~/.claude-mem` (read-write)

## What ends up in the container's ~/.claude directory?

The container's `~/.claude` starts as an empty Docker volume. During the `postCreateCommand`, several init scripts populate it:

1. **init-claude-config.sh** -- configures git identity, GitHub auth, and merges your host `settings.json` (from `/tmp/base-settings.json`) into the container's `~/.claude/settings.json`, stripping `statusLine`, `enabledPlugins`, and notification hooks
2. **init-config-repo.sh** -- if `CLAUDE_CONFIG_REPO` is set, clones it and deploys shared config (`.library/`, `CLAUDE.md`, `enabled.json`, `Makefile`)
3. **init-claudeup.sh** -- installs claudeup, applies `CLAUDE_BASE_PROFILE` at user scope (if set), then applies `CLAUDE_PROFILE` at user or project scope, generates `enabled.json` from profile extension lists, and syncs extension symlinks

When Claude Code launches for the first time, it reads this config and installs any marketplace plugins listed in `settings.json`. Plugins are downloaded fresh into the container -- they are not copied from the host.

**Without `--profile`**: claudeup-lab snapshots your current host config via `claudeup profile save` before starting. The snapshot captures:

- **Plugins** -- the list of enabled plugin names (e.g., `episodic-memory@superpowers-marketplace`), not plugin binaries or `node_modules`
- **MCP servers** -- configured server names, commands, and args, with secret references (not secret values)
- **Marketplaces** -- plugin marketplace sources referenced by your plugins
- **Extensions** -- which agents, commands, skills, hooks, rules, and output-styles are enabled

The snapshot does **not** capture conversation history, project-specific state, or the full contents of `~/.claude/settings.json` (settings are handled separately via the base-settings bind mount). When the profile is applied inside the container, Claude Code downloads any marketplace plugins fresh -- they are not copied from the host.

**With `--profile`**: only the named profile is applied. No snapshot of your current config is taken.

To start a lab with no inherited config at all, create and use an empty profile:

```bash
claudeup profile create blank
claudeup-lab start --profile blank --project ~/code/myproject
```

## What's the difference between --profile and --base-profile?

They control which profiles get applied and at which Claude Code scope.

**`--profile` only:**

```bash
claudeup-lab start --profile base --project ~/code/myproject
```

No snapshot is taken. The `base` profile is applied at user scope (`~/.claude/settings.json`). Nothing is written to project scope.

**`--base-profile` only (no `--profile`):**

```bash
claudeup-lab start --base-profile base --project ~/code/myproject
```

Your current claudeup config is snapshotted as the main profile. The entrypoint applies `base` at user scope (`~/.claude/settings.json`), then applies the snapshot at project scope (`<project>/.claude/settings.json`). Because these are separate scopes writing to separate files, the snapshot does not overwrite the base -- both are active. Claude Code merges settings across scopes at runtime, with project scope taking precedence over user scope for conflicting keys.

**Both flags together:**

```bash
claudeup-lab start --base-profile base --profile solo-test --project ~/code/myproject
```

No snapshot is taken. `base` is applied at user scope, `solo-test` at project scope. This gives you a foundation layer with targeted overrides on top.

**Summary:**

| Flags                                | Snapshot? | User scope                 | Project scope              |
| ------------------------------------ | --------- | -------------------------- | -------------------------- |
| `--profile base`                     | No        | `base`                     | (empty)                    |
| `--base-profile base`                | Yes       | `base`                     | snapshot of current config |
| `--base-profile base --profile solo` | No        | `base`                     | `solo`                     |
| (neither)                            | Yes       | snapshot of current config | (empty)                    |

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

Yes, but it's quick. You can create a profile that only includes the plugin you want to test, then start a lab with it.

For Claude plugins (from the marketplace), create a profile with the plugin included:

```bash
claudeup profile create solo-test \
    --description "test a single plugin" \
    --marketplace your-org/your-plugin \
    --plugin "your-plugin@your-plugin-dev"
claudeup-lab start --profile solo-test
```

Alternatively, you can start a lab with a bare config and install the plugin manually:

```bash
claudeup-lab start
claude plugin marketplace add your-org/your-plugin
claude plugin install your-plugin@your-plugin-dev
```

For claudeup extensions (skills, agents, hooks, rules, output-styles), use `claudeup ext enable`:

```bash
claudeup profile create solo-test
claudeup ext enable skills my-skill
claudeup profile save solo-test
claudeup-lab start --profile solo-test
```

Each lab starts with a fresh `~/.claude` volume, so only the extensions from the specified profile get applied. Without `--profile`, a lab snapshots your current config -- which includes all your existing plugins.

If you want to test a plugin layered on top of a base config, use `--base-profile`:

```bash
claudeup-lab start --base-profile default --profile solo-test
```

This applies `default` at user scope, then `solo-test` at project scope, so you get your foundation settings plus just the plugin under test.
