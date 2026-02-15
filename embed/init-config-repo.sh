#!/usr/bin/env bash
# ABOUTME: Clones a Claude configuration repo and deploys extensions into ~/.claude.
# ABOUTME: Reads CLAUDE_CONFIG_REPO and CLAUDE_CONFIG_BRANCH env vars.

set -euo pipefail

CLAUDE_HOME="/home/node/.claude"
MARKER_FILE="$CLAUDE_HOME/.config-repo-deployed"

if [ -f "$MARKER_FILE" ]; then
    echo "[SKIP] Config repo already deployed"
    exit 0
fi

if [ -z "${CLAUDE_CONFIG_REPO:-}" ]; then
    echo "[SKIP] CLAUDE_CONFIG_REPO not set, skipping config deployment"
    exit 0
fi

BRANCH="${CLAUDE_CONFIG_BRANCH:-main}"
TEMP_DIR=$(mktemp -d)
trap 'rm -rf "$TEMP_DIR"' EXIT

echo "Cloning config repo ($BRANCH)..."
git clone --branch "$BRANCH" --depth 1 "$CLAUDE_CONFIG_REPO" "$TEMP_DIR"

# Deploy .library/ (all extensions) -- skip if already bind-mounted from host
if [ -d "$TEMP_DIR/.library" ] && [ ! -d "$CLAUDE_HOME/.library" ]; then
    cp -a "$TEMP_DIR/.library" "$CLAUDE_HOME/.library"
    echo "[OK] .library/ deployed"
elif [ -d "$CLAUDE_HOME/.library" ]; then
    echo "[SKIP] .library/ already mounted"
fi

# Deploy config files (not settings.json -- seeded from host, updated by claudeup profile)
for file in CLAUDE.md enabled.json Makefile; do
    if [ -f "$TEMP_DIR/$file" ]; then
        cp "$TEMP_DIR/$file" "$CLAUDE_HOME/$file"
        echo "[OK] $file deployed"
    fi
done

# Create category directories for symlinks
for dir in skills agents commands hooks output-styles; do
    mkdir -p "$CLAUDE_HOME/$dir"
done

# Sync symlinks from enabled.json
if command -v claudeup &> /dev/null && [ -f "$CLAUDE_HOME/enabled.json" ]; then
    claudeup ext sync -y
    echo "[OK] Extension symlinks synced"
fi

touch "$MARKER_FILE"
echo "Config repo deployment complete"
