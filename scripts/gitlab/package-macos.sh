#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
OUT_DIR="$ROOT_DIR/dist/macos"
PAYLOAD_DIR="$OUT_DIR/payload/usr/local/bin"
PKG_ROOT="$OUT_DIR/pkgroot"
DMG_ROOT="$OUT_DIR/dmgroot"

cd "$ROOT_DIR"
mkdir -p "$PAYLOAD_DIR" "$PKG_ROOT" "$DMG_ROOT"

go build -o "$PAYLOAD_DIR/scp2317-api" ./cmd/api

if command -v swift >/dev/null 2>&1; then
  swift build --package-path "$ROOT_DIR/platform/macos-es" -c release
  if [[ -f "$ROOT_DIR/platform/macos-es/.build/release/SCP2317ES" ]]; then
    cp "$ROOT_DIR/platform/macos-es/.build/release/SCP2317ES" "$PAYLOAD_DIR/SCP2317ES"
  fi
fi

cp -R "$OUT_DIR/payload/." "$PKG_ROOT/"

pkgbuild \
  --root "$PKG_ROOT" \
  --identifier "com.example.scp2317.agent" \
  --version "0.1.0" \
  "$OUT_DIR/SCP-2317.pkg"

cp "$OUT_DIR/SCP-2317.pkg" "$DMG_ROOT/"
echo "SCP-2317 macOS package" >"$DMG_ROOT/README.txt"

hdiutil create \
  -volname "SCP-2317" \
  -srcfolder "$DMG_ROOT" \
  -ov \
  -format UDZO \
  "$OUT_DIR/SCP-2317.dmg"
