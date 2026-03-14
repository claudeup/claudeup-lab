# Dockerfile Version Pinning and Checksum Verification

## Problem

The embedded Dockerfile fetches and runs remote install scripts for Bun and Claude CLI without version pinning or checksum verification (`embed/Dockerfile` lines 44, 49). This makes builds non-reproducible and carries supply-chain risk from unverified binaries.

## Decision

Download binaries directly from their release URLs, verify SHA-256 checksums before extraction/installation, and skip the live install scripts entirely. All versions and checksums are hardcoded in the Dockerfile using `ARG` declarations with defaults. No Go code changes required.

## Approach: Direct Download + Checksum Verification

Instead of piping remote install scripts to bash, download the release artifacts directly, verify their checksums, then extract and place the binaries manually. Nothing executes before checksum verification passes.

### Bun

```dockerfile
ARG BUN_VERSION=<latest stable>
ARG BUN_SHA256=<hash from SHASUMS256.txt>
RUN curl -fsSL -o /tmp/bun.zip \
        "https://github.com/oven-sh/bun/releases/download/bun-v${BUN_VERSION}/bun-linux-x64.zip" \
    && echo "${BUN_SHA256}  /tmp/bun.zip" | sha256sum -c - \
    && mkdir -p /home/node/.bun/bin \
    && unzip -j /tmp/bun.zip "*/bun" -d /home/node/.bun/bin \
    && chmod 755 /home/node/.bun/bin/bun \
    && rm -f /tmp/bun.zip
```

- Checksum covers the zip archive (matches `SHASUMS256.txt` format)
- Checksum source: `https://github.com/oven-sh/bun/releases/download/bun-v{VERSION}/SHASUMS256.txt`
- Platform: `linux-x64` (the Dockerfile base image is `node:22`, Debian amd64)
- `unzip -j` with a glob pattern extracts the binary regardless of internal directory structure
- Requires `unzip` -- must be added to the `apt-get install` line
- Version discovery: check `https://github.com/oven-sh/bun/releases/latest`

### Claude CLI

```dockerfile
ARG CLAUDE_VERSION=<latest stable>
ARG CLAUDE_SHA256=<hash from manifest.json>
RUN curl -fsSL -o /tmp/claude \
        "https://storage.googleapis.com/claude-code-dist-86c565f3-f756-42ad-8dfa-d59b1c096819/claude-code-releases/${CLAUDE_VERSION}/linux-x64/claude" \
    && echo "${CLAUDE_SHA256}  /tmp/claude" | sha256sum -c - \
    && install -m 755 /tmp/claude /home/node/.local/bin/claude \
    && rm /tmp/claude
```

- The Claude CLI release is a single binary (not an archive)
- Checksum is hex-encoded SHA-256, covers the binary directly
- Checksum source: `https://storage.googleapis.com/claude-code-dist-86c565f3-f756-42ad-8dfa-d59b1c096819/claude-code-releases/{VERSION}/manifest.json` -- use the `linux-x64` entry's `checksum` field
- `install -m 755` sets permissions and moves atomically (consistent with the pattern used in #10 for `scripts/install.sh`)
- Version discovery: fetch `https://storage.googleapis.com/claude-code-dist-86c565f3-f756-42ad-8dfa-d59b1c096819/claude-code-releases/latest` for the current version string, or run `claude --version` from an existing installation

### ARG Placement

The `ARG` declarations go immediately before their corresponding `RUN` layer, after the `USER node` switch (line 39). This keeps each version/checksum co-located with the layer that uses it and does not affect caching of earlier layers.

### Prerequisites

`sha256sum` is available in `node:22` (Debian-based, provided by `coreutils`). `unzip` must be added to the `apt-get install` line for the Bun zip extraction.

## Scope

- **Changed file**: `embed/Dockerfile` only
- **No Go code changes**: The Dockerfile is a static embedded asset; `buildFallback()` in `image.go` does not pass `--build-arg` flags and does not need to
- **No new files**: No helper scripts or verification utilities

## Version Selection

Actual version numbers and checksums will be determined at implementation time by fetching the latest stable releases and their published hashes. Bumping versions in the future means updating the `ARG` defaults in the Dockerfile.

## Testing

- Build the Dockerfile and verify both installs succeed with checksum verification
- Verify `bun --version` and `claude --version` report the pinned versions
- Verify a deliberately wrong checksum causes the build to fail

## Trade-offs

- **Platform-specific**: The direct download URLs target `linux-x64`. This is fine because Docker builds always run on Linux, but it means the Dockerfile cannot be built natively on macOS/ARM without emulation. This is already the case with `node:22` (amd64).
- **Manual version bumps**: No automation for updating versions. Acceptable for a fallback build path that is rarely exercised (the primary path pulls a pre-built image from the registry).
- **No install script side effects**: By skipping the install scripts, we may miss setup steps they perform (e.g., shell completions, PATH modifications). The Dockerfile already handles PATH via `ENV` on line 42, so this is acceptable.

## References

- GitHub issue: #4
- PR #1 Copilot review comment
- PR #10 (install script checksum verification) -- uses the same download-verify-install pattern
