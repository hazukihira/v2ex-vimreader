$ErrorActionPreference = "SilentlyContinue"
$root = $PSScriptRoot
$exe = Join-Path $root "bin\v2ex-vimreader.exe"
$url = "http://127.0.0.1:7788/"

if (-not (Test-Path $exe)) {
  Write-Host "v2ex-vimreader.exe not found; build it first or download a release."
  exit 1
}

$running = $false
try {
  $running = [bool](Get-NetTCPConnection -LocalPort 7788 -State Listen -ErrorAction Stop)
} catch {
  $running = $false
}

if (-not $running) {
  Start-Process -FilePath $exe -WorkingDirectory $root -WindowStyle Hidden
  Start-Sleep -Milliseconds 900
}

Start-Process $url
