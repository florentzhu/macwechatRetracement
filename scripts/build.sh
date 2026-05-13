#!/usr/bin/env bash
#
# Build a macOS universal binary of wechattweak.
#
# Usage:
#   ./scripts/build.sh           # outputs ./wechattweak
#   OUT=dist/wechattweak ./scripts/build.sh
#

set -euo pipefail

PKG="./cmd/wechattweak"
OUT="${OUT:-wechattweak}"
LDFLAGS="-s -w"

cd "$(dirname "$0")/.."

if ! command -v lipo >/dev/null 2>&1; then
  echo "error: 'lipo' not found (macOS Xcode command line tools required)" >&2
  exit 1
fi

mkdir -p "$(dirname "$OUT")"
TMP_DIR="$(mktemp -d)"
trap 'rm -rf "$TMP_DIR"' EXIT

echo "==> building darwin/arm64"
GOOS=darwin GOARCH=arm64 go build -trimpath -ldflags "$LDFLAGS" -o "$TMP_DIR/wechattweak-arm64" "$PKG"

echo "==> building darwin/amd64"
GOOS=darwin GOARCH=amd64 go build -trimpath -ldflags "$LDFLAGS" -o "$TMP_DIR/wechattweak-amd64" "$PKG"

echo "==> creating universal binary -> $OUT"
lipo -create -output "$OUT" "$TMP_DIR/wechattweak-arm64" "$TMP_DIR/wechattweak-amd64"

echo "==> done"
file "$OUT"
ls -lh "$OUT"
