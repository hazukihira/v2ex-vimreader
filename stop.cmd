@echo off
setlocal enabledelayedexpansion
set "found="
for /f "tokens=5" %%p in ('netstat -ano ^| findstr ":7788" ^| findstr LISTENING') do (
  set "found=1"
  taskkill /F /PID %%p >nul 2>&1
)
if defined found (echo v2ex-vimreader stopped) else (echo v2ex-vimreader is not running)
