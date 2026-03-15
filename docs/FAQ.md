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
