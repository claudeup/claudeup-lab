#!/usr/bin/env bash
# ABOUTME: Installs claudeup and applies the specified profile.
# ABOUTME: Reads CLAUDE_PROFILE and CLAUDE_BASE_PROFILE env vars. Detects
# ABOUTME: multi-scope profiles and applies them across all scope levels.

set -euo pipefail

CLAUDEUP_HOME="/home/node/.claudeup"
CLAUDE_CONFIG_DIR="/home/node/.claude"
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

# Check if a profile uses multi-scope format (has perScope key).
# Returns 0 if multi-scope, 1 if single-scope. Exits on invalid JSON.
is_multi_scope() {
    local profile_name="$1"
    local profile_file="$CLAUDEUP_HOME/profiles/$profile_name.json"
    [ -f "$profile_file" ] || return 1
    if ! jq empty "$profile_file" 2>/dev/null; then
        echo "[ERROR] Profile '$profile_name' has invalid JSON: $profile_file" >&2
        exit 1
    fi
    jq -e '.perScope' "$profile_file" > /dev/null 2>&1
}

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

# Multi-scope profiles define their own scope placement, so --base-profile
# layering is not compatible. Omitting the scope flag tells claudeup to apply
# each perScope section (user, project, local) to its respective location.
# Single-scope profiles use explicit scope flags for layering compatibility.
if is_multi_scope "$CLAUDE_PROFILE"; then
    if [ -n "${CLAUDE_BASE_PROFILE:-}" ]; then
        echo "[ERROR] --base-profile is not compatible with multi-scope profiles."
        echo "        Multi-scope profiles define their own scope placement."
        exit 1
    fi
    echo "Applying multi-scope profile: $CLAUDE_PROFILE..."
    if claudeup profile apply "$CLAUDE_PROFILE" -y; then
        echo "[OK] Profile '$CLAUDE_PROFILE' applied (all scopes)"
    else
        echo "[WARN] claudeup profile apply failed, will retry on next container start"
        exit 1
    fi
else
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
fi

# Sync extensions (agents, commands, skills, hooks, output-styles) from profiles.
# Skip if enabled.json already exists (e.g., deployed by init-config-repo.sh).
if [ ! -f "$CLAUDEUP_HOME/enabled.json" ]; then
    # Generate enabled.json from profile extensions
    ext_base="{}"
    if [ -n "${CLAUDE_BASE_PROFILE:-}" ]; then
        base_file="$CLAUDEUP_HOME/profiles/$CLAUDE_BASE_PROFILE.json"
        if [ -f "$base_file" ] && jq -e '.extensions' "$base_file" > /dev/null 2>&1; then
            ext_base=$(jq '.extensions | with_entries(.value |= (map({(.): true}) | add // {}))' "$base_file") || {
                echo "[ERROR] Failed to parse extensions from base profile: $base_file" >&2
                exit 1
            }
        fi
    fi

    ext_profile="{}"
    profile_file="$CLAUDEUP_HOME/profiles/$CLAUDE_PROFILE.json"
    if [ -f "$profile_file" ] && jq -e '.extensions' "$profile_file" > /dev/null 2>&1; then
        ext_profile=$(jq '.extensions | with_entries(.value |= (map({(.): true}) | add // {}))' "$profile_file") || {
            echo "[ERROR] Failed to parse extensions from profile: $profile_file" >&2
            exit 1
        }
    fi

    # Merge base + profile items (profile wins on conflicts)
    merged_items=$(jq -n --argjson base "$ext_base" --argjson profile "$ext_profile" '$base * $profile')

    if [ "$merged_items" != "{}" ]; then
        echo "$merged_items" > "$CLAUDEUP_HOME/enabled.json"
        echo "[OK] enabled.json generated from profile extensions"
    else
        echo "[SKIP] No extensions in profile(s)"
    fi
else
    echo "[SKIP] enabled.json already exists"
fi

# Create category directories and sync symlinks
if [ -f "$CLAUDEUP_HOME/enabled.json" ]; then
    for dir in skills agents commands hooks output-styles rules; do
        mkdir -p "$CLAUDE_CONFIG_DIR/$dir"
    done

    if command -v claudeup &> /dev/null; then
        claudeup ext sync -y
        echo "[OK] Extension item symlinks synced"
    fi
fi

touch "$MARKER_FILE"

echo "Claudeup initialization complete"
