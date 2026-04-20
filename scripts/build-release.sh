#!/usr/bin/env bash
set -euo pipefail

mkdir -p dist

GOOS=darwin GOARCH=arm64 go build -o dist/xfchat-bootstrapper-darwin-arm64 ./cmd/xfchat-bootstrapper
GOOS=windows GOARCH=amd64 go build -o dist/xfchat-bootstrapper-windows-amd64.exe ./cmd/xfchat-bootstrapper
