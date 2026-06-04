#!/bin/sh
set -e

REPO="TheRealShek/persephone"
BIN_NAME="purr"
INSTALL_DIR="/usr/local/bin"

echo "Installing Persephone ($BIN_NAME)..."

# Detect OS
OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
if [ "$OS" != "linux" ]; then
    echo "Error: This installer is only for Linux. Detected $OS."
    exit 1
fi

# Detect Architecture
ARCH="$(uname -m)"
case "$ARCH" in
    x86_64|amd64)
        ARCH="amd64"
        ;;
    aarch64|arm64)
        ARCH="arm64"
        ;;
    *)
        echo "Error: Unsupported architecture $ARCH"
        exit 1
        ;;
esac

echo "Detected OS: $OS, Architecture: $ARCH"

# Fetch latest release version from GitHub API
echo "Fetching latest release from GitHub..."
RELEASE_JSON=$(curl -s "https://api.github.com/repos/$REPO/releases/latest")
VERSION=$(echo "$RELEASE_JSON" | grep -m 1 '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')

if [ -z "$VERSION" ]; then
    echo "Error: Failed to fetch the latest release from $REPO"
    exit 1
fi

echo "Latest release is $VERSION"

ASSET_NAME="persephone_${OS}_${ARCH}.tar.gz"
DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${VERSION}/${ASSET_NAME}"

echo "Downloading $ASSET_NAME..."
TMP_DIR=$(mktemp -d)
# Ensure cleanup on exit
trap "rm -rf $TMP_DIR" EXIT

if ! curl -sfL -o "$TMP_DIR/$ASSET_NAME" "$DOWNLOAD_URL"; then
    echo "Error: Failed to download $DOWNLOAD_URL"
    exit 1
fi

echo "Extracting..."
tar -xzf "$TMP_DIR/$ASSET_NAME" -C "$TMP_DIR"

if [ ! -f "$TMP_DIR/$BIN_NAME" ]; then
    echo "Error: Binary '$BIN_NAME' not found in the downloaded archive."
    exit 1
fi

echo "Installing to $INSTALL_DIR..."
# Try installing without sudo first
if [ -w "$INSTALL_DIR" ]; then
    mv "$TMP_DIR/$BIN_NAME" "$INSTALL_DIR/$BIN_NAME"
    chmod +x "$INSTALL_DIR/$BIN_NAME"
else
    echo "Directory $INSTALL_DIR is not writable. Attempting sudo..."
    sudo mv "$TMP_DIR/$BIN_NAME" "$INSTALL_DIR/$BIN_NAME"
    sudo chmod +x "$INSTALL_DIR/$BIN_NAME"
fi

echo "Successfully installed $("$INSTALL_DIR/$BIN_NAME" --version 2>/dev/null || echo "$BIN_NAME $VERSION")"
echo "You can now run '$BIN_NAME --help'"
