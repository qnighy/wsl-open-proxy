#!/usr/bin/env bash
set -ue

for arch in amd64 arm64; do
  GOOS=windows GOARCH="$arch" go build -o "./cmd/setup-wsl-open/assets/wsl-open-proxy-$arch.exe" ./cmd/wsl-open-proxy
done
