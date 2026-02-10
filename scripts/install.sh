#!/usr/bin/env bash
set -euo pipefail

REPO="claudeup/claudeup-lab"
INSTALL_DIR="${HOME}/.local/bin"

# Detect platform
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)
case "$ARCH" in
    x86_64) ARCH="amd64" ;;
    aarch64|arm64) ARCH="arm64" ;;
    *) echo "Unsupported architecture: $ARCH" >&2; exit 1 ;;
esac

# Get version (override with VERSION env var)
if [ -z "${VERSION:-}" ]; then
    VERSION=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name"' | sed -E 's/.*"v?([^"]+)".*/\1/')
fi
if [ -z "$VERSION" ]; then
    VERSION="0.1.0"
fi

echo "Downloading claudeup-lab v${VERSION} for ${OS}-${ARCH}..."

URL="https://github.com/${REPO}/releases/download/v${VERSION}/claudeup-lab_${OS}_${ARCH}.tar.gz"
mkdir -p "$INSTALL_DIR"
curl -fsSL "$URL" | tar xz -C "$INSTALL_DIR" claudeup-lab

echo "Installed claudeup-lab v${VERSION} to ${INSTALL_DIR}/claudeup-lab"

if ! echo "$PATH" | grep -q "$INSTALL_DIR"; then
    echo ""
    echo "Add to PATH: export PATH=\"\$PATH:${INSTALL_DIR}\""
fi
