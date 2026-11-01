$ErrorActionPreference = "SilentlyContinue"
$root = Split-Path -Parent $PSScriptRoot
$exe = Join-Path $root "bin\v2ex-vimreader.exe"
if (-not (Test-Path $exe)) {
  $exe = Join-Path $root "v2ex-vimreader.exe"
}
if (-not (Test-Path $exe)) {
  exit 1
}
$running = $false
try {
  $running = [bool](Get-NetTCPConnection -LocalPort 7788 -State Listen -ErrorAction Stop)
} catch {
  $running = $false
}
if ($running) {
  exit 0
}
Start-Process -FilePath $exe -WorkingDirectory $root -WindowStyle Hidden -ArgumentList "-no-open"
