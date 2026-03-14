#!/usr/bin/env bash
set -Eeuo pipefail

readonly REPO="claudeup/claudeup-lab"
readonly INSTALL_DIR="${HOME}/.local/bin"
readonly BINARY="claudeup-lab"

TEMP_DIR=""
cleanup() {
    [[ -n "$TEMP_DIR" && -d "$TEMP_DIR" ]] && rm -rf "$TEMP_DIR"
}
trap cleanup EXIT
trap 'echo "Error on line $LINENO" >&2; exit 1' ERR

# Detect platform
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)
case "$OS" in
    linux|darwin) ;;
    *) echo "Unsupported operating system: $OS" >&2; exit 1 ;;
esac
case "$ARCH" in
    x86_64) ARCH="amd64" ;;
    aarch64|arm64) ARCH="arm64" ;;
    *) echo "Unsupported architecture: $ARCH" >&2; exit 1 ;;
esac

# Get version (override with VERSION env var)
if [[ -z "${VERSION:-}" ]]; then
    VERSION=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name"' | sed -E 's/.*"v?([^"]+)".*/\1/') || true
fi
if [[ -z "$VERSION" ]]; then
    echo "Failed to detect latest version from GitHub API." >&2
    echo "Set VERSION explicitly: VERSION=0.2.0 bash scripts/install.sh" >&2
    exit 1
fi

readonly ARCHIVE="${BINARY}_${OS}_${ARCH}.tar.gz"
readonly BASE_URL="https://github.com/${REPO}/releases/download/v${VERSION}"

echo "Downloading ${BINARY} v${VERSION} for ${OS}-${ARCH}..."

TEMP_DIR=$(mktemp -d)

if ! curl -fsSL -o "${TEMP_DIR}/${ARCHIVE}" "${BASE_URL}/${ARCHIVE}"; then
    echo "Failed to download ${BASE_URL}/${ARCHIVE}" >&2
    echo "Check that version v${VERSION} exists and has a release for ${OS}-${ARCH}" >&2
    exit 1
fi

if ! curl -fsSL -o "${TEMP_DIR}/checksums.txt" "${BASE_URL}/checksums.txt"; then
    echo "Failed to download checksums file from ${BASE_URL}/checksums.txt" >&2
    exit 1
fi

# Verify checksum
EXPECTED=$(awk -v f="${ARCHIVE}" '$2 == f {print $1}' "${TEMP_DIR}/checksums.txt")
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

# Extract and install
if ! tar xz -C "$TEMP_DIR" -f "${TEMP_DIR}/${ARCHIVE}" "$BINARY"; then
    echo "Failed to extract ${BINARY} from ${ARCHIVE}" >&2
    exit 1
fi

mkdir -p "$INSTALL_DIR"
install -m 755 "${TEMP_DIR}/${BINARY}" "${INSTALL_DIR}/${BINARY}"

echo "Installed ${BINARY} v${VERSION} to ${INSTALL_DIR}/${BINARY}"

if ! echo "$PATH" | grep -q "$INSTALL_DIR"; then
    echo ""
    echo "Add to PATH: export PATH=\"\$PATH:${INSTALL_DIR}\""
fi
