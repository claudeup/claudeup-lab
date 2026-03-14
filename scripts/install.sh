#!/usr/bin/env bash
set -Eeuo pipefail

REPO="claudeup/claudeup-lab"
INSTALL_DIR="${HOME}/.local/bin"
BINARY="claudeup-lab"

TEMP_DIR=""
cleanup() {
    [[ -n "$TEMP_DIR" && -d "$TEMP_DIR" ]] && rm -rf "$TEMP_DIR"
}
trap cleanup EXIT

# Detect platform
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)
case "$ARCH" in
    x86_64) ARCH="amd64" ;;
    aarch64|arm64) ARCH="arm64" ;;
    *) echo "Unsupported architecture: $ARCH" >&2; exit 1 ;;
esac

# Get version (override with VERSION env var)
if [[ -z "${VERSION:-}" ]]; then
    VERSION=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name"' | sed -E 's/.*"v?([^"]+)".*/\1/')
fi
if [[ -z "$VERSION" ]]; then
    VERSION="0.1.0"
fi

ARCHIVE="${BINARY}_${OS}_${ARCH}.tar.gz"
BASE_URL="https://github.com/${REPO}/releases/download/v${VERSION}"

echo "Downloading ${BINARY} v${VERSION} for ${OS}-${ARCH}..."

TEMP_DIR=$(mktemp -d)
curl -fsSL -o "${TEMP_DIR}/${ARCHIVE}" "${BASE_URL}/${ARCHIVE}"
curl -fsSL -o "${TEMP_DIR}/checksums.txt" "${BASE_URL}/checksums.txt"

# Verify checksum
EXPECTED=$(grep "${ARCHIVE}" "${TEMP_DIR}/checksums.txt" | awk '{print $1}')
if [[ -z "$EXPECTED" ]]; then
    echo "No checksum found for ${ARCHIVE} in checksums.txt" >&2
    exit 1
fi

if command -v sha256sum >/dev/null 2>&1; then
    ACTUAL=$(sha256sum "${TEMP_DIR}/${ARCHIVE}" | awk '{print $1}')
elif command -v shasum >/dev/null 2>&1; then
    ACTUAL=$(shasum -a 256 "${TEMP_DIR}/${ARCHIVE}" | awk '{print $1}')
else
    echo "No sha256sum or shasum found" >&2
    exit 1
fi

if [[ "$EXPECTED" != "$ACTUAL" ]]; then
    echo "Checksum verification failed" >&2
    echo "  expected: ${EXPECTED}" >&2
    echo "  actual:   ${ACTUAL}" >&2
    exit 1
fi

# Extract and install atomically
tar xz -C "$TEMP_DIR" -f "${TEMP_DIR}/${ARCHIVE}" "$BINARY"
mkdir -p "$INSTALL_DIR"
mv "${TEMP_DIR}/${BINARY}" "${INSTALL_DIR}/${BINARY}"

echo "Installed ${BINARY} v${VERSION} to ${INSTALL_DIR}/${BINARY}"

if ! echo "$PATH" | grep -q "$INSTALL_DIR"; then
    echo ""
    echo "Add to PATH: export PATH=\"\$PATH:${INSTALL_DIR}\""
fi
