#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
OUT_DIR="$ROOT_DIR/dist/releases"
cd "$ROOT_DIR"
mkdir -p "$OUT_DIR"

{
  echo "SCP-2317 artifact manifest"
  echo "Generated: $(date -u +"%Y-%m-%dT%H:%M:%SZ")"
  echo
  find "$ROOT_DIR/dist" -type f | sort | while read -r file; do
    rel="${file#$ROOT_DIR/}"
    sha256="$(sha256sum "$file" | awk '{print $1}')"
    size="$(wc -c <"$file" | tr -d ' ')"
    echo "$rel|$size|$sha256"
  done
} >"$OUT_DIR/manifest.txt"

cp "$OUT_DIR/manifest.txt" "$OUT_DIR/SCP-2317-manifest.txt"
