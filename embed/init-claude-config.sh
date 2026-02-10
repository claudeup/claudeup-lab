#!/usr/bin/env bash
# ABOUTME: Configures git identity, GitHub auth, and dotfiles inside sandbox containers.
# ABOUTME: Reads GIT_USER_NAME, GIT_USER_EMAIL, GITHUB_TOKEN, and DOTFILES_REPO env vars.

set -euo pipefail

CLAUDE_HOME="/home/node/.claude"
DOTFILES_DIR="/home/node/dotfiles"

echo "Initializing Claude Code configuration..."

validate_email() {
    local email="$1"
    [[ "$email" =~ ^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$ ]]
}

validate_git_name() {
    local name="$1"
    if [[ "$name" =~ [\;\|\&\$\`\'\"\<\>\(\)\{\}\[\]\\] ]] || [[ "$name" =~ [[:cntrl:]] ]]; then
        return 1
    fi
    [ -n "$name" ] && [ ${#name} -le 256 ]
}

validate_git_url() {
    local url="$1"
    [[ "$url" =~ ^https://(github\.com|gitlab\.com|bitbucket\.org)/ ]] || \
    [[ "$url" =~ ^git@(github\.com|gitlab\.com|bitbucket\.org): ]]
}

# Configure git identity
pushd /tmp > /dev/null
if [ -n "${GIT_USER_NAME:-}" ]; then
    if validate_git_name "$GIT_USER_NAME"; then
        git config --global user.name "$GIT_USER_NAME"
        echo "[OK] Git user.name: $GIT_USER_NAME"
    else
        echo "[WARN] GIT_USER_NAME contains invalid characters, skipping"
    fi
fi

if [ -n "${GIT_USER_EMAIL:-}" ]; then
    if validate_email "$GIT_USER_EMAIL"; then
        git config --global user.email "$GIT_USER_EMAIL"
        echo "[OK] Git user.email: $GIT_USER_EMAIL"
    else
        echo "[WARN] GIT_USER_EMAIL is not a valid email format, skipping"
    fi
fi

# Configure GitHub auth
if [ -n "${GITHUB_TOKEN:-}" ]; then
    if command -v gh &> /dev/null; then
        gh auth setup-git 2>/dev/null || true
        echo "[OK] GitHub auth configured via gh CLI"
    else
        git config --global credential.helper \
            "!f() { echo \"username=x-access-token\"; echo \"password=\${GITHUB_TOKEN}\"; }; f"
        echo "[OK] GitHub credential helper configured"
    fi
fi
popd > /dev/null

# Clone dotfiles if configured
if [ -n "${DOTFILES_REPO:-}" ] && [ -z "$(ls -A "$DOTFILES_DIR" 2>/dev/null)" ]; then
    if validate_git_url "$DOTFILES_REPO"; then
        echo "Cloning dotfiles from $DOTFILES_REPO..."
        git clone --branch "${DOTFILES_BRANCH:-main}" "$DOTFILES_REPO" "$DOTFILES_DIR"
        echo "[OK] Dotfiles cloned"

        if [ -f "$DOTFILES_DIR/install.sh" ]; then
            echo "[SECURITY] Executing install.sh from $DOTFILES_REPO"
            # shellcheck disable=SC2086
            cd "$DOTFILES_DIR" && chmod +x install.sh && ./install.sh ${DOTFILES_INSTALL_ARGS:-}
            echo "[OK] Dotfiles install script completed"
        fi
    else
        echo "[WARN] DOTFILES_REPO must be a valid GitHub/GitLab/Bitbucket URL, skipping"
    fi
elif [ -n "${DOTFILES_REPO:-}" ]; then
    echo "[SKIP] Dotfiles directory not empty, preserving"
fi

mkdir -p "$CLAUDE_HOME"

# Seed settings.json from host base settings, stripping plugin-dependent keys
# (enabledPlugins and statusLine are managed by the claudeup profile)
if [ -f /tmp/base-settings.json ] && [ ! -f "$CLAUDE_HOME/settings.json" ]; then
    jq 'del(.statusLine, .enabledPlugins, .hooks.Notification)' /tmp/base-settings.json > "$CLAUDE_HOME/settings.json"
    echo "[OK] Base settings.json deployed (permissions, hooks)"
elif [ -f /tmp/base-settings.json ]; then
    echo "[SKIP] Existing settings.json found, not overwriting"
fi

mkdir -p /home/node/.npm-global/lib

echo "Claude configuration complete"
