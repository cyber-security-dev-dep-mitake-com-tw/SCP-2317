#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
OUT_DIR="$ROOT_DIR/dist/macos"
PAYLOAD_DIR="$OUT_DIR/payload/usr/local/bin"
APP_DIR="$OUT_DIR/payload/Applications/SCP-2317.app"
PKG_ROOT="$OUT_DIR/pkgroot"
DMG_ROOT="$OUT_DIR/dmgroot"

cd "$ROOT_DIR"
rm -rf "$OUT_DIR"
mkdir -p "$PAYLOAD_DIR" "$APP_DIR/Contents/MacOS" "$APP_DIR/Contents/Resources" "$PKG_ROOT" "$DMG_ROOT"

GOOS=darwin GOARCH="${GOARCH:-$(uname -m | sed 's/x86_64/amd64/;s/arm64/arm64/')}" \
  go build -ldflags="-s -w" -o "$PAYLOAD_DIR/scp2317-api" ./cmd/api
cp "$PAYLOAD_DIR/scp2317-api" "$APP_DIR/Contents/MacOS/scp2317-api"

if command -v swift >/dev/null 2>&1; then
  swift build --package-path "$ROOT_DIR/platform/macos-es" -c release
  if [[ -f "$ROOT_DIR/platform/macos-es/.build/release/SCP2317ES" ]]; then
    cp "$ROOT_DIR/platform/macos-es/.build/release/SCP2317ES" "$PAYLOAD_DIR/SCP2317ES"
  fi
fi

if [[ -f "$ROOT_DIR/assets/logo.icns" ]]; then
  cp "$ROOT_DIR/assets/logo.icns" "$APP_DIR/Contents/Resources/AppIcon.icns"
fi
if [[ -f "$ROOT_DIR/assets/logo.png" ]]; then
  cp "$ROOT_DIR/assets/logo.png" "$APP_DIR/Contents/Resources/logo.png"
fi

cat >"$APP_DIR/Contents/Info.plist" <<'PLIST'
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>CFBundleName</key>
  <string>SCP-2317</string>
  <key>CFBundleDisplayName</key>
  <string>SCP-2317</string>
  <key>CFBundleIdentifier</key>
  <string>com.example.scp2317.agent</string>
  <key>CFBundleVersion</key>
  <string>0.1.0</string>
  <key>CFBundleShortVersionString</key>
  <string>0.1.0</string>
  <key>CFBundleExecutable</key>
  <string>scp2317-api</string>
  <key>CFBundlePackageType</key>
  <string>APPL</string>
  <key>CFBundleIconFile</key>
  <string>AppIcon</string>
  <key>LSMinimumSystemVersion</key>
  <string>13.0</string>
</dict>
</plist>
PLIST

cp -R "$OUT_DIR/payload/." "$PKG_ROOT/"

pkgbuild \
  --root "$PKG_ROOT" \
  --identifier "com.example.scp2317.agent" \
  --version "0.1.0" \
  --install-location "/" \
  "$OUT_DIR/SCP-2317.pkg"

cp "$OUT_DIR/SCP-2317.pkg" "$DMG_ROOT/"
cp -R "$APP_DIR" "$DMG_ROOT/"
cp "$ROOT_DIR/assets/logo.png" "$DMG_ROOT/SCP-2317-logo.png" 2>/dev/null || true
echo "SCP-2317 macOS package (unsigned). Drag SCP-2317.app or install SCP-2317.pkg." >"$DMG_ROOT/README.txt"

hdiutil create \
  -volname "SCP-2317" \
  -srcfolder "$DMG_ROOT" \
  -ov \
  -format UDZO \
  "$OUT_DIR/SCP-2317.dmg"

echo "Wrote $OUT_DIR/SCP-2317.pkg and $OUT_DIR/SCP-2317.dmg"
