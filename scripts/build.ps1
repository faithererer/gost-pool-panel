$ErrorActionPreference = "Stop"

$root = Split-Path -Parent $PSScriptRoot
$dist = Join-Path $root "dist"
New-Item -ItemType Directory -Force -Path $dist | Out-Null

Push-Location $root
try {
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
