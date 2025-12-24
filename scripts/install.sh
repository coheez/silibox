#!/bin/bash
set -euo pipefail

# Silibox Installation Script for Alpha Testers
# This script installs dependencies and builds Silibox

echo "🚀 Silibox Alpha Installation"
echo "=============================="

# Check if we're on macOS
if [[ "$OSTYPE" != "darwin"* ]]; then
    echo "❌ Error: Silibox requires macOS"
    exit 1
fi

# Check if Homebrew is installed
if ! command -v brew &> /dev/null; then
    echo "❌ Error: Homebrew is required but not installed"
    echo "Install Homebrew: https://brew.sh/"
    exit 1
fi

echo "✅ macOS detected"
echo "✅ Homebrew found"

# Install Lima
echo "📦 Installing Lima..."
if ! command -v limactl &> /dev/null; then
    brew install lima
    echo "✅ Lima installed"
else
    echo "✅ Lima already installed"
fi

# Check Go installation
echo "🔍 Checking Go installation..."
if ! command -v go &> /dev/null; then
    echo "❌ Error: Go is required but not installed"
    echo "Install Go: https://golang.org/dl/"
    exit 1
fi

GO_VERSION=$(go version | cut -d' ' -f3 | sed 's/go//')
echo "✅ Go $GO_VERSION found"

# Build Silibox
echo "🔨 Building Silibox..."
if ! make build; then
    echo "❌ Error: Failed to build Silibox"
    exit 1
fi

echo "✅ Silibox built successfully"

# Optional: Install globally
read -p "📦 Install Silibox globally? (y/N): " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    if make install; then
        echo "✅ Silibox installed globally"
        echo "You can now run 'sili' from anywhere"
    else
        echo "⚠️  Failed to install globally, but binary is available at ./bin/sili"
    fi
else
    echo "ℹ️  Binary available at ./bin/sili"
fi

# Run doctor check
echo ""
echo "🔍 Running health check..."
if ./bin/sili doctor; then
    echo ""
    echo "🎉 Installation complete!"
    echo ""
    echo "Next steps:"
    echo "  1. Start VM: ./bin/sili vm up"
    echo "  2. Create environment: ./bin/sili create --name my-project"
    echo "  3. Enter shell: ./bin/sili enter --name my-project"
    echo ""
    echo "For help: ./bin/sili --help"
else
    echo ""
    echo "⚠️  Installation complete but some issues detected"
    echo "Run './bin/sili doctor' to see details"
fi
