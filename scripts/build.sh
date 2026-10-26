#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
VERSION="${VERSION:-$(git describe --tags --always --dirty 2>/dev/null || echo dev)}"
mkdir -p bin
CGO_ENABLED=0 go build -trimpath \
  -ldflags "-s -w -X github.com/hazukihira/v2ex-vimreader/internal/config.Version=${VERSION}" \
  -o bin/v2ex-vimreader .
echo "built bin/v2ex-vimreader (${VERSION})"
