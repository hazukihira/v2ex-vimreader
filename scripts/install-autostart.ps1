$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $PSScriptRoot
$autostart = Join-Path $root "scripts\autostart.ps1"
$exe = Join-Path $root "bin\v2ex-vimreader.exe"
if (-not (Test-Path $exe)) {
  $exe = Join-Path $root "v2ex-vimreader.exe"
}
if (-not (Test-Path $autostart)) {
  throw "autostart.ps1 not found"
}
if (-not (Test-Path $exe)) {
  throw "v2ex-vimreader executable not found; build it or unpack a release first"
}
$startup = [Environment]::GetFolderPath("Startup")
$shortcutPath = Join-Path $startup "v2ex-vimreader.lnk"
$shell = New-Object -ComObject WScript.Shell
$shortcut = $shell.CreateShortcut($shortcutPath)
$shortcut.TargetPath = Join-Path $env:SystemRoot "System32\WindowsPowerShell\v1.0\powershell.exe"
$shortcut.Arguments = '-NoProfile -WindowStyle Hidden -ExecutionPolicy Bypass -File "' + $autostart + '"'
$shortcut.WorkingDirectory = $root
$shortcut.WindowStyle = 7
$shortcut.IconLocation = $exe + ",0"
$shortcut.Description = "v2ex-vimreader startup"
$shortcut.Save()
Write-Host "Installed: $shortcutPath"
