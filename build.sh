#!/usr/bin/env bash
set -ue

./prebuild.sh
mkdir -p dist
for arch in amd64 arm64; do
  CGO_ENABLED=0 GOOS=linux GOARCH="$arch" go build -o dist/setup-wsl-open-"$arch" ./cmd/setup-wsl-open
done
