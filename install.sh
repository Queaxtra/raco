#!/bin/sh
# raco install script

set -e

REPO="Queaxtra/raco"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case "$ARCH" in
    x86_64) ARCH="amd64" ;;
    arm64|aarch64) ARCH="arm64" ;;
    *) echo "Unsupported arch: $ARCH"; exit 1 ;;
esac

case "$OS" in
    linux|darwin) ;;  # supported
    *) echo "Unsupported OS: $OS"; exit 1 ;;
esac

LATEST=$(curl -s "https://api.github.com/repos/$REPO/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')

if [ -z "$LATEST" ]; then
    echo "Failed to fetch latest release"
    exit 1
fi

echo "Installing raco $LATEST for ${OS}_${ARCH}..."

URL="https://github.com/$REPO/releases/download/$LATEST/raco_${OS}_${ARCH}.tar.gz"
CHECKSUM_URL="https://github.com/$REPO/releases/download/$LATEST/checksums.txt"
TMP_DIR=$(mktemp -d)

curl -sL "$URL" -o "$TMP_DIR/raco.tar.gz"
curl -sL "$CHECKSUM_URL" -o "$TMP_DIR/checksums.txt"

# Verify SHA256 checksum before extracting to prevent tampered binary execution.
EXPECTED=$(grep "raco_${OS}_${ARCH}.tar.gz" "$TMP_DIR/checksums.txt" | awk '{print $1}')
if [ -z "$EXPECTED" ]; then
    echo "Error: checksum not found for raco_${OS}_${ARCH}.tar.gz"
    rm -rf "$TMP_DIR"
    exit 1
fi

if command -v sha256sum >/dev/null 2>&1; then
    ACTUAL=$(sha256sum "$TMP_DIR/raco.tar.gz" | awk '{print $1}')
elif command -v shasum >/dev/null 2>&1; then
    ACTUAL=$(shasum -a 256 "$TMP_DIR/raco.tar.gz" | awk '{print $1}')
else
    echo "Error: no sha256sum or shasum found; cannot verify download"
    rm -rf "$TMP_DIR"
    exit 1
fi

if [ "$ACTUAL" != "$EXPECTED" ]; then
    echo "Error: checksum mismatch (expected $EXPECTED, got $ACTUAL)"
    rm -rf "$TMP_DIR"
    exit 1
fi

tar -xzf "$TMP_DIR/raco.tar.gz" -C "$TMP_DIR"

if [ -w "$INSTALL_DIR" ]; then
    mv "$TMP_DIR/raco" "$INSTALL_DIR/"
else
    sudo mv "$TMP_DIR/raco" "$INSTALL_DIR/"
fi

rm -rf "$TMP_DIR"

echo "raco installed to $INSTALL_DIR/raco"
echo "Run 'raco --version' to verify"
