#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
mkdir -p "$ROOT/.cache"
(cd "$ROOT/server" && go build -o "$ROOT/.cache/pp-piano-api-dev" .)
(cd "$ROOT/server" && exec "$ROOT/.cache/pp-piano-api-dev") &
API_PID=$!
trap 'kill "$API_PID" 2>/dev/null || true' EXIT INT TERM
cd "$ROOT/web-toy"
npm run dev
