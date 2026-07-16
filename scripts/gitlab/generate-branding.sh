#!/usr/bin/env bash
# Regenerate installer icons from assets/branding/scp-2317-source.jpg (optional).
# Committed assets/logo.{png,ico,icns} are used directly by CI packaging.
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
SRC="$ROOT_DIR/assets/branding/scp-2317-source.jpg"
BRAND="$ROOT_DIR/assets/branding"

if [[ ! -f "$SRC" ]]; then
  echo "Source art not found: $SRC" >&2
  echo "Keep committed assets/logo.* or download art per assets/ATTRIBUTION.md" >&2
  exit 1
fi

mkdir -p "$BRAND"
for s in 16 32 48 64 80 128 256 512 1024; do
  sips -s format png -z "$s" "$s" "$SRC" --out "$BRAND/logo-$s.png" >/dev/null
done

cp "$BRAND/logo-512.png" "$ROOT_DIR/assets/logo.png"

# Ribbon command icons required by outlook-addin/manifest.xml (16/32/80),
# the legacy 64px add-in icon referenced by IconUrl, and the 128px
# HighResolutionIconUrl required by newer Outlook clients.
cp "$BRAND/logo-16.png" "$ROOT_DIR/outlook-addin/icon-16.png"
cp "$BRAND/logo-32.png" "$ROOT_DIR/outlook-addin/icon-32.png"
cp "$BRAND/logo-80.png" "$ROOT_DIR/outlook-addin/icon-80.png"
cp "$BRAND/logo-64.png" "$ROOT_DIR/outlook-addin/icon-64.png"
cp "$BRAND/logo-128.png" "$ROOT_DIR/outlook-addin/icon-128.png"

ICONSET="$BRAND/SCP2317.iconset"
rm -rf "$ICONSET" && mkdir -p "$ICONSET"
cp "$BRAND/logo-16.png"   "$ICONSET/icon_16x16.png"
cp "$BRAND/logo-32.png"   "$ICONSET/icon_16x16@2x.png"
cp "$BRAND/logo-32.png"   "$ICONSET/icon_32x32.png"
cp "$BRAND/logo-64.png"   "$ICONSET/icon_32x32@2x.png"
cp "$BRAND/logo-128.png"  "$ICONSET/icon_128x128.png"
cp "$BRAND/logo-256.png"  "$ICONSET/icon_128x128@2x.png"
cp "$BRAND/logo-256.png"  "$ICONSET/icon_256x256.png"
cp "$BRAND/logo-512.png"  "$ICONSET/icon_256x256@2x.png"
cp "$BRAND/logo-512.png"  "$ICONSET/icon_512x512.png"
cp "$BRAND/logo-1024.png" "$ICONSET/icon_512x512@2x.png"
iconutil -c icns "$ICONSET" -o "$ROOT_DIR/assets/logo.icns"

python3 -m venv /tmp/scp2317-branding-venv
/tmp/scp2317-branding-venv/bin/pip install -q Pillow
/tmp/scp2317-branding-venv/bin/python - <<'PY'
from PIL import Image
img = Image.open('assets/branding/logo-256.png').convert('RGBA')
img.save('assets/logo.ico', format='ICO', sizes=[(16,16),(32,32),(48,48),(64,64),(128,128),(256,256)])
print('generated assets/logo.ico')
PY

echo "Branding assets updated."
