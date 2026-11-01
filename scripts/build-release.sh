#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
OUT="$ROOT/release"
mkdir -p "$OUT/api" "$OUT/toy-dist" "$OUT/deploy"
(
 cd "$ROOT/web-toy"
 npm ci --no-audit --no-fund
 npm run test:timing
 npm run build
)
cp -R "$ROOT/web-toy/dist/." "$OUT/toy-dist/"
(
 cd "$ROOT/server"
 GOOS="$(go env GOHOSTOS)" GOARCH="$(go env GOHOSTARCH)" go test ./...
 CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -o "$OUT/api/pp-piano-api" .
)
cp "$ROOT/server/.env.example" "$OUT/api/.env.example"
cp "$ROOT/deploy/Caddyfile" "$ROOT/deploy/pp-piano-server.service" "$OUT/deploy/"
cp "$ROOT/community-docs/IMPLEMENTATION.md" "$OUT/README.md"
(
 cd "$OUT"
 shasum -a 256 api/pp-piano-api > SHA256SUMS
)
printf 'Community artifacts: %s\n' "$OUT"
