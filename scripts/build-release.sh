#!/usr/bin/env bash
set -euo pipefail

rm -rf dist
mkdir -p dist

GOOS=darwin GOARCH=arm64 go build -o dist/xfchat-bootstrapper-darwin-arm64 ./cmd/xfchat-bootstrapper
GOOS=darwin GOARCH=amd64 go build -o dist/xfchat-bootstrapper-darwin-amd64 ./cmd/xfchat-bootstrapper
GOOS=windows GOARCH=amd64 go build -o dist/xfchat-bootstrapper-windows-amd64.exe ./cmd/xfchat-bootstrapper
