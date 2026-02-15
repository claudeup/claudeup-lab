#!/usr/bin/env bash
# ABOUTME: Installs claudeup and applies the specified profile.
# ABOUTME: Reads CLAUDE_PROFILE and CLAUDE_BASE_PROFILE env vars.

set -euo pipefail

CLAUDEUP_HOME="/home/node/.claudeup"
CLAUDE_HOME="/home/node/.claude"
MARKER_FILE="$CLAUDEUP_HOME/.setup-complete"

echo "Initializing claudeup..."

if [ -f "$MARKER_FILE" ]; then
    echo "[SKIP] Claudeup setup already complete"
    exit 0
fi

if [ -z "${CLAUDE_PROFILE:-}" ]; then
    echo "[WARN] CLAUDE_PROFILE not set, skipping profile setup"
    exit 0
fi

mkdir -p "$CLAUDEUP_HOME/profiles"

if ! command -v claudeup &> /dev/null; then
    echo "Installing claudeup..."
    curl -fsSL https://raw.githubusercontent.com/claudeup/claudeup/main/install.sh | bash
    export PATH="$HOME/.local/bin:$PATH"
    echo "[OK] claudeup installed"
else
    echo "[SKIP] claudeup already installed"
fi

# Apply base profile at user scope (foundation layer)
if [ -n "${CLAUDE_BASE_PROFILE:-}" ]; then
    echo "Applying base profile: $CLAUDE_BASE_PROFILE (user scope)..."
    if claudeup profile apply "$CLAUDE_BASE_PROFILE" --user -y; then
        echo "[OK] Base profile '$CLAUDE_BASE_PROFILE' applied at user scope"
    else
        echo "[WARN] Base profile apply failed, will retry on next container start"
        exit 1
    fi
fi

# Apply profile: project scope when layering on a base, user scope otherwise
if [ -n "${CLAUDE_BASE_PROFILE:-}" ]; then
    apply_scope="--project"
    scope_label="project"
else
    apply_scope="--user"
    scope_label="user"
fi

echo "Applying profile: $CLAUDE_PROFILE ($scope_label scope)..."
if claudeup profile apply "$CLAUDE_PROFILE" $apply_scope -y; then
    echo "[OK] Profile '$CLAUDE_PROFILE' applied at $scope_label scope"
else
    echo "[WARN] claudeup profile apply failed, will retry on next container start"
    exit 1
fi

# Sync extensions (agents, commands, skills, hooks, output-styles) from profiles.
# Skip if enabled.json already exists (e.g., deployed by init-config-repo.sh).
if [ ! -f "$CLAUDE_HOME/enabled.json" ]; then
    # Generate enabled.json from profile extensions
    ext_base="{}"
    if [ -n "${CLAUDE_BASE_PROFILE:-}" ]; then
        base_file="$CLAUDEUP_HOME/profiles/$CLAUDE_BASE_PROFILE.json"
        if [ -f "$base_file" ] && jq -e '.extensions' "$base_file" > /dev/null 2>&1; then
            ext_base=$(jq '.extensions | with_entries(.value |= (map({(.): true}) | add // {}))' "$base_file")
        fi
    fi

    ext_profile="{}"
    profile_file="$CLAUDEUP_HOME/profiles/$CLAUDE_PROFILE.json"
    if [ -f "$profile_file" ] && jq -e '.extensions' "$profile_file" > /dev/null 2>&1; then
        ext_profile=$(jq '.extensions | with_entries(.value |= (map({(.): true}) | add // {}))' "$profile_file")
    fi

    # Merge base + profile items (profile wins on conflicts)
    merged_items=$(jq -n --argjson base "$ext_base" --argjson profile "$ext_profile" '$base * $profile')

    if [ "$merged_items" != "{}" ]; then
        echo "$merged_items" > "$CLAUDE_HOME/enabled.json"
        echo "[OK] enabled.json generated from profile extensions"
    else
        echo "[SKIP] No extensions in profile(s)"
    fi
else
    echo "[SKIP] enabled.json already exists"
fi

# Create category directories and sync symlinks
if [ -f "$CLAUDE_HOME/enabled.json" ]; then
    for dir in skills agents commands hooks output-styles rules; do
        mkdir -p "$CLAUDE_HOME/$dir"
    done

    if command -v claudeup &> /dev/null; then
        claudeup ext sync -y
        echo "[OK] Extension item symlinks synced"
    fi
fi

touch "$MARKER_FILE"

echo "Claudeup initialization complete"
