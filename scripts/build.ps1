# Build v2ex-vimreader for the current platform.
$ErrorActionPreference = "Stop"
Set-Location (Join-Path $PSScriptRoot "..")
$version = if ($env:VERSION) { $env:VERSION } else { "dev" }
New-Item -ItemType Directory -Force -Path bin | Out-Null
$env:CGO_ENABLED = "0"
go build -trimpath -ldflags "-s -w -X github.com/hazukihira/v2ex-vimreader/internal/config.Version=$version" -o bin/v2ex-vimreader.exe .
Write-Host "built bin/v2ex-vimreader.exe ($version)"
