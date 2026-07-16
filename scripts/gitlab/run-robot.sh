#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
OUT_DIR="$ROOT_DIR/robot-output"
API_LOG="$OUT_DIR/api.log"
BASE_URL="${MAILGUARD_API_BASE_URL:-http://127.0.0.1:8080}"
API_PID=""

cd "$ROOT_DIR"
mkdir -p "$OUT_DIR"

cleanup() {
  if [[ -n "$API_PID" ]]; then
    kill "$API_PID" >/dev/null 2>&1 || true
    wait "$API_PID" >/dev/null 2>&1 || true
  fi
}

trap cleanup EXIT

go run ./cmd/api >"$API_LOG" 2>&1 &
API_PID="$!"

READY=0
for _ in $(seq 1 30); do
  if python3 -c 'import sys; from urllib.request import urlopen; value=urlopen(sys.argv[1], timeout=2).read().decode("utf-8").strip(); sys.exit(0 if value == "ok" else 1)' "$BASE_URL/healthz" >/dev/null 2>&1; then
    READY=1
    break
  fi
  sleep 1
done

if [[ "$READY" -ne 1 ]]; then
  echo "SCP-2317 API did not become ready" >&2
  exit 1
fi

python3 -m robot \
  --outputdir "$OUT_DIR" \
  --variable "BASE_URL:$BASE_URL" \
  "$ROOT_DIR/tests/robot/api.robot"
