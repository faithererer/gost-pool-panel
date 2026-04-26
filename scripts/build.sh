#!/usr/bin/env sh
set -eu

ROOT="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"
DIST="$ROOT/dist"

mkdir -p "$DIST"
mkdir -p "$ROOT/.gotmp" "$ROOT/.gocache"
export GOTMPDIR="${GOTMPDIR:-$ROOT/.gotmp}"
export GOCACHE="${GOCACHE:-$ROOT/.gocache}"
cd "$ROOT"

if [ -f "$ROOT/frontend/package.json" ]; then
  cd "$ROOT/frontend"
  if [ ! -d node_modules ]; then
    npm ci
  fi
  npm run build
  cd "$ROOT"
fi

go build -o "$DIST/gost-pool-panel" ./cmd/panel
GOOS=linux GOARCH=amd64 go build -o "$DIST/gost-pool-agent-linux-amd64" ./cmd/agent
GOOS=linux GOARCH=arm64 go build -o "$DIST/gost-pool-agent-linux-arm64" ./cmd/agent

ls -lah "$DIST"
