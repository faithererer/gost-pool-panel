#!/usr/bin/env sh
set -eu

ROOT="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"
DIST="$ROOT/dist"

mkdir -p "$DIST"
cd "$ROOT"

go build -o "$DIST/gost-pool-panel" ./cmd/panel
GOOS=linux GOARCH=amd64 go build -o "$DIST/gost-pool-agent-linux-amd64" ./cmd/agent
GOOS=linux GOARCH=arm64 go build -o "$DIST/gost-pool-agent-linux-arm64" ./cmd/agent

ls -lah "$DIST"
