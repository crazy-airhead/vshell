#!/bin/bash
set -e

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT_DIR"

# Ensure wails3 is in PATH
export PATH="$HOME/go/bin:$PATH"

if ! command -v wails3 &>/dev/null; then
    echo "ERROR: wails3 not found in PATH. Please install Wails 3 first."
    exit 1
fi

# Defaults
ARCH="${ARCH:-amd64}"
PACKAGE="${PACKAGE:-false}"
FORMAT="${FORMAT:-nsis}"

echo "==> Building Windows ${ARCH} binary..."
wails3 task windows:build "ARCH=${ARCH}"

BIN_PATH="bin/vshell.exe"
if [ ! -f "$BIN_PATH" ]; then
    echo "ERROR: Build output not found at $BIN_PATH"
    exit 1
fi

echo "==> Binary: $BIN_PATH"
echo "    Size: $(du -h "$BIN_PATH" | cut -f1)"

if [ "$PACKAGE" = "true" ]; then
    echo "==> Packaging as ${FORMAT} installer..."
    wails3 task windows:package "ARCH=${ARCH}" "FORMAT=${FORMAT}"
    echo "==> Done: bin/"
fi

echo ""
echo "==> Build complete!"
echo "    Output: $BIN_PATH"
