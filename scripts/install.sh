#!/bin/bash
set -e

# Detect OS and architecture
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

if [ "$OS" != "darwin" ]; then
    echo "Error: silibox currently only supports macOS"
    exit 1
fi

case "$ARCH" in
    x86_64) ARCH="amd64" ;;
    arm64|aarch64) ARCH="arm64" ;;
    *)
        echo "Error: Unsupported architecture: $ARCH"
        exit 1
        ;;
esac

# Get latest release version
echo "Fetching latest release..."
VERSION=$(curl -s https://api.github.com/repos/coheez/silibox/releases/latest | grep '"tag_name"' | cut -d'"' -f4)
if [ -z "$VERSION" ]; then
    echo "Error: Could not determine latest version"
    exit 1
fi

echo "Installing silibox ${VERSION}..."

# Download binary
DOWNLOAD_URL="https://github.com/coheez/silibox/releases/download/${VERSION}/sili-${OS}-${ARCH}"
TMP_FILE="/tmp/sili"

echo "Downloading from ${DOWNLOAD_URL}..."
if ! curl -L -o "$TMP_FILE" "$DOWNLOAD_URL"; then
    echo "Error: Failed to download binary"
    exit 1
fi

chmod +x "$TMP_FILE"

# Install to /usr/local/bin
INSTALL_DIR="/usr/local/bin"
if [ ! -w "$INSTALL_DIR" ]; then
    echo "Installing to $INSTALL_DIR (requires sudo)..."
    sudo mv "$TMP_FILE" "$INSTALL_DIR/sili"
else
    mv "$TMP_FILE" "$INSTALL_DIR/sili"
fi

echo ""
echo "✓ silibox installed successfully!"
echo ""
echo "Next steps:"
echo "  1. Install Lima: brew install lima"
echo "  2. Run 'sili doctor' to verify your environment"
echo "  3. Run 'sili vm up' to start the Linux VM"
echo "  4. Run 'sili create --name dev' to create a development environment"
echo ""
echo "For more information, visit https://github.com/coheez/silibox"
