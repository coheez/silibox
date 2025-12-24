#!/usr/bin/env bash
set -euo pipefail

echo "Checking Go..."
if ! command -v go >/dev/null; then
  echo "Install Go 1.22+: https://go.dev/dl/"
  exit 1
fi

echo "Checking lima..."
if ! command -v limactl >/dev/null; then
  echo "Install Lima: brew install lima"
  exit 1
fi

echo "OK — deps present."