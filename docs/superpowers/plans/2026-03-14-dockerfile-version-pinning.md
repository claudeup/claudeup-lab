# Dockerfile Version Pinning Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Pin Bun and Claude CLI versions in the embedded Dockerfile with SHA-256 checksum verification.

**Architecture:** Replace the two piped install scripts with direct artifact downloads, checksum verification via `sha256sum`, and manual binary placement with multi-arch support (x64/arm64). Code change to `embed/Dockerfile`.

**Tech Stack:** Docker, sha256sum, curl, unzip

**Spec:** `docs/superpowers/specs/2026-03-14-dockerfile-version-pinning-design.md`

---

## Chunk 1: Implementation

### Task 1: Add unzip to apt-get install line

**Files:**

- Modify: `embed/Dockerfile:9-18`

- [ ] **Step 1: Add `unzip` to the apt-get install list**

Add `unzip` to the existing package list in the `apt-get install` line. Place it alphabetically near the end of the list:

```dockerfile
RUN apt-get update && apt-get install -y --no-install-recommends \
    curl wget ca-certificates gnupg2 \
    git git-lfs gh \
    procps htop \
    jq yq \
    ripgrep fd-find \
    build-essential make \
    python3 \
    nano vim unzip \
    && apt-get clean && rm -rf /var/lib/apt/lists/*
```

- [ ] **Step 2: Commit**

```bash
git add embed/Dockerfile
git commit -m "feat(dockerfile): add unzip for Bun archive extraction (#4)"
```

---

### Task 2: Pin Bun with checksum verification

**Files:**

- Modify: `embed/Dockerfile:44`

- [ ] **Step 1: Replace the Bun install line**

Replace line 44:

```dockerfile
RUN curl -fsSL https://bun.sh/install | bash
```

With:

```dockerfile
ARG BUN_VERSION=1.3.10
ARG BUN_SHA256_X64=f57bc0187e39623de716ba3a389fda5486b2d7be7131a980ba54dc7b733d2e08
ARG BUN_SHA256_ARM64=fa5ecb25cafa8e8f5c87a0f833719d46dd0af0a86c7837d806531212d55636d3
RUN ARCH=$(uname -m) \
    && if [ "$ARCH" = "x86_64" ]; then \
        BUN_ARCH="x64"; BUN_SHA256="${BUN_SHA256_X64}"; \
    elif [ "$ARCH" = "aarch64" ] || [ "$ARCH" = "arm64" ]; then \
        BUN_ARCH="aarch64"; BUN_SHA256="${BUN_SHA256_ARM64}"; \
    else echo "Unsupported architecture: $ARCH" >&2; exit 1; fi \
    && curl -fsSL -o /tmp/bun.zip \
        "https://github.com/oven-sh/bun/releases/download/bun-v${BUN_VERSION}/bun-linux-${BUN_ARCH}.zip" \
    && echo "${BUN_SHA256}  /tmp/bun.zip" | sha256sum -c - \
    && mkdir -p /home/node/.bun/bin \
    && unzip -j /tmp/bun.zip "*/bun" -d /home/node/.bun/bin \
    && test -f /home/node/.bun/bin/bun \
    && chmod 755 /home/node/.bun/bin/bun \
    && rm -f /tmp/bun.zip
```

- [ ] **Step 2: Commit**

```bash
git add embed/Dockerfile
git commit -m "feat(dockerfile): pin Bun v1.3.10 with SHA-256 verification (#4)"
```

---

### Task 3: Pin Claude CLI with checksum verification

**Files:**

- Modify: `embed/Dockerfile:49`

- [ ] **Step 1: Replace the Claude CLI install line**

Replace:

```dockerfile
RUN curl -fsSL https://claude.ai/install.sh | bash
```

With:

```dockerfile
ARG CLAUDE_VERSION=2.1.76
ARG CLAUDE_SHA256_X64=801a085676c3d54392c42e8e43c44947df7c52132356575f7d9267c4f22d6992
ARG CLAUDE_SHA256_ARM64=40f753c07f070df34ca83e400f746a8279a3fd343967a453d9fbfab2f3ca7acd
RUN ARCH=$(uname -m) \
    && if [ "$ARCH" = "x86_64" ]; then \
        CLAUDE_ARCH="x64"; CLAUDE_SHA256="${CLAUDE_SHA256_X64}"; \
    elif [ "$ARCH" = "aarch64" ] || [ "$ARCH" = "arm64" ]; then \
        CLAUDE_ARCH="arm64"; CLAUDE_SHA256="${CLAUDE_SHA256_ARM64}"; \
    else echo "Unsupported architecture: $ARCH" >&2; exit 1; fi \
    && curl -fsSL -o /tmp/claude \
        "https://storage.googleapis.com/claude-code-dist-86c565f3-f756-42ad-8dfa-d59b1c096819/claude-code-releases/${CLAUDE_VERSION}/linux-${CLAUDE_ARCH}/claude" \
    && echo "${CLAUDE_SHA256}  /tmp/claude" | sha256sum -c - \
    && install -m 755 /tmp/claude /home/node/.local/bin/claude \
    && rm /tmp/claude
```

- [ ] **Step 2: Commit**

```bash
git add embed/Dockerfile
git commit -m "feat(dockerfile): pin Claude CLI v2.1.76 with SHA-256 verification (#4)"
```

---

### Task 4: Build and verify the Dockerfile

- [ ] **Step 1: Build the image**

```bash
docker build -t claudeup-lab-test:pinned embed/
```

Expected: Build completes successfully with both checksum verifications passing.

- [ ] **Step 2: Verify Bun version**

```bash
docker run --rm claudeup-lab-test:pinned bun --version
```

Expected: `1.3.10`

- [ ] **Step 3: Verify Claude CLI version**

```bash
docker run --rm claudeup-lab-test:pinned claude --version
```

Expected: Output contains `2.1.76`

- [ ] **Step 4: Verify bad checksum fails the build**

Temporarily change one character in `BUN_SHA256` and rebuild:

```bash
docker build --build-arg BUN_SHA256_ARM64=0000000000000000000000000000000000000000000000000000000000000000 --build-arg BUN_SHA256_X64=0000000000000000000000000000000000000000000000000000000000000000 -t claudeup-lab-test:bad embed/
```

Expected: Build fails with `sha256sum: WARNING: 1 computed checksum did NOT match`

- [ ] **Step 5: Clean up test images**

```bash
docker rmi claudeup-lab-test:pinned claudeup-lab-test:bad 2>/dev/null || true
```

- [ ] **Step 6: Commit any fixes if needed, then final commit**

If all verifications pass with no changes needed:

```bash
git log --oneline -4
```

Verify the commit history looks clean. No additional commit needed if tasks 1-3 are already committed.
