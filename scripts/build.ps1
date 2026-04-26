$ErrorActionPreference = "Stop"

$root = Split-Path -Parent $PSScriptRoot
$dist = Join-Path $root "dist"
New-Item -ItemType Directory -Force -Path $dist | Out-Null
New-Item -ItemType Directory -Force -Path (Join-Path $root ".gotmp"), (Join-Path $root ".gocache") | Out-Null

Push-Location $root
try {
  if (-not $env:GOTMPDIR) {
    $env:GOTMPDIR = Join-Path $root ".gotmp"
  }
  if (-not $env:GOCACHE) {
    $env:GOCACHE = Join-Path $root ".gocache"
  }

  if (Test-Path (Join-Path $root "frontend/package.json")) {
    Push-Location (Join-Path $root "frontend")
    try {
      if (-not (Test-Path "node_modules")) {
        npm ci
      }
      npm run build
    } finally {
      Pop-Location
    }
  }

  go build -o (Join-Path $dist "gost-pool-panel.exe") ./cmd/panel

  $env:GOOS = "linux"
  $env:GOARCH = "amd64"
  go build -o (Join-Path $dist "gost-pool-agent-linux-amd64") ./cmd/agent

  $env:GOARCH = "arm64"
  go build -o (Join-Path $dist "gost-pool-agent-linux-arm64") ./cmd/agent
} finally {
  Remove-Item Env:\GOOS -ErrorAction SilentlyContinue
  Remove-Item Env:\GOARCH -ErrorAction SilentlyContinue
  Pop-Location
}

Get-ChildItem -LiteralPath $dist
