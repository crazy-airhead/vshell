#!/bin/bash
set -e

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
BIN_DIR="$ROOT_DIR/bin"
APP_NAME="vshell"
ARCH="${ARCH:-$(go env GOARCH)}"
APP_BUNDLE="$BIN_DIR/$APP_NAME.app"
DMG_NAME="${APP_NAME}-${ARCH}.dmg"

# Step 1: Build and package the app bundle
echo "==> Building macOS ${ARCH} app bundle with wails3..."
cd "$ROOT_DIR"
wails3 task darwin:package "ARCH=${ARCH}"

if [ ! -d "$APP_BUNDLE" ]; then
    echo "ERROR: App bundle not found at $APP_BUNDLE"
    exit 1
fi

echo "==> Done: $APP_BUNDLE"
echo "    Run with: open $APP_BUNDLE"

# Step 2: Create DMG
echo "==> Creating ${ARCH} DMG..."

TMP_DIR="$BIN_DIR/dmg"

rm -rf "$TMP_DIR"
mkdir -p "$TMP_DIR"

cp -R "$APP_BUNDLE" "$TMP_DIR"

# 可选：增加 Applications 快捷方式
ln -s /Applications "$TMP_DIR/Applications"

rm -f "$BIN_DIR/$DMG_NAME"

hdiutil create \
    -volname "$APP_NAME-$ARCH" \
    -srcfolder "$TMP_DIR" \
    -ov \
    -format UDZO \
    "$BIN_DIR/$DMG_NAME"

rm -rf "$TMP_DIR"

echo "==> DMG Created:"
echo "    $BIN_DIR/$DMG_NAME"
