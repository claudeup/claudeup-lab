# Dockerfile Version Pinning and Checksum Verification

## Problem

The embedded Dockerfile fetches and runs remote install scripts for Bun and Claude CLI without version pinning or checksum verification (`embed/Dockerfile` lines 44, 49). This makes builds non-reproducible and carries supply-chain risk from unverified binaries.

## Decision

Download binaries directly from their release URLs, verify SHA-256 checksums before extraction/installation, and skip the live install scripts entirely. All versions and checksums are hardcoded in the Dockerfile using `ARG` declarations with defaults. No Go code changes required.

## Approach: Direct Download + Checksum Verification

Instead of piping remote install scripts to bash, download the release artifacts directly, verify their checksums, then extract and place the binaries manually. Nothing executes before checksum verification passes. Architecture is detected at build time via `uname -m` to support both x64 and arm64.

### Bun

- Checksum covers the zip archive (matches `SHASUMS256.txt` format)
- Checksum source: `https://github.com/oven-sh/bun/releases/download/bun-v{VERSION}/SHASUMS256.txt`
- Platforms: `linux-x64` (x86_64) and `linux-aarch64` (arm64)
- `unzip -j` with a glob pattern extracts the binary regardless of internal directory structure
- Requires `unzip` -- must be added to the `apt-get install` line
- Version discovery: check `https://github.com/oven-sh/bun/releases/latest`

### Claude CLI

- The Claude CLI release is a single binary (not an archive)
- Checksum is hex-encoded SHA-256, covers the binary directly
- Checksum source: `https://storage.googleapis.com/claude-code-dist-86c565f3-f756-42ad-8dfa-d59b1c096819/claude-code-releases/{VERSION}/manifest.json` -- use the platform entry's `checksum` field
- Platforms: `linux-x64` (x86_64) and `linux-arm64` (aarch64)
- `install -m 755` sets permissions and moves atomically (consistent with the pattern used in #10 for `scripts/install.sh`)
- Version discovery: fetch `https://storage.googleapis.com/claude-code-dist-86c565f3-f756-42ad-8dfa-d59b1c096819/claude-code-releases/latest` for the current version string, or run `claude --version` from an existing installation

### Multi-Architecture Support

Each `RUN` layer detects the build platform via `uname -m` and selects the correct download URL and checksum. Separate `ARG` declarations are provided for x64 and arm64 checksums (e.g., `BUN_SHA256_X64`, `BUN_SHA256_ARM64`). Unsupported architectures fail the build with a clear error message.

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

- **Two supported architectures**: x64 and arm64 are supported. Other architectures will fail the build with a clear error.
- **Manual version bumps**: No automation for updating versions. Acceptable for a fallback build path that is rarely exercised (the primary path pulls a pre-built image from the registry).
- **No install script side effects**: By skipping the install scripts, we may miss setup steps they perform (e.g., shell completions, PATH modifications). The Dockerfile already handles PATH via `ENV` on line 42, so this is acceptable.

## References

- GitHub issue: #4
- PR #1 Copilot review comment
- PR #10 (install script checksum verification) -- uses the same download-verify-install pattern
