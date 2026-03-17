#!/usr/bin/env bash
# ABOUTME: Clones a Claude configuration repo and deploys shared config.
# ABOUTME: Reads CLAUDE_CONFIG_REPO and CLAUDE_CONFIG_BRANCH env vars.

set -euo pipefail

CLAUDE_CONFIG_DIR="/home/node/.claude"
CLAUDEUP_HOME="/home/node/.claudeup"
MARKER_FILE="$CLAUDE_CONFIG_DIR/.config-repo-deployed"

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

mkdir -p "$CLAUDE_CONFIG_DIR"

# Deploy config files
for file in CLAUDE.md Makefile; do
    if [ -f "$TEMP_DIR/$file" ]; then
        cp "$TEMP_DIR/$file" "$CLAUDE_CONFIG_DIR/$file"
        echo "[OK] $file deployed to $CLAUDE_CONFIG_DIR"
    fi
done

# enabled.json lives in CLAUDEUP_HOME for claudeup ext sync
if [ -f "$TEMP_DIR/enabled.json" ]; then
    mkdir -p "$CLAUDEUP_HOME"
    cp "$TEMP_DIR/enabled.json" "$CLAUDEUP_HOME/enabled.json"
    echo "[OK] enabled.json deployed to $CLAUDEUP_HOME"
fi

# Sync symlinks from enabled.json
if command -v claudeup &> /dev/null && [ -f "$CLAUDEUP_HOME/enabled.json" ]; then
    claudeup ext sync -y
    echo "[OK] Extension symlinks synced"
fi

touch "$MARKER_FILE"
echo "Config repo deployment complete"
