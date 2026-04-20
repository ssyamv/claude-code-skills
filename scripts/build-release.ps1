$ErrorActionPreference = "Stop"
Set-StrictMode -Version Latest

New-Item -ItemType Directory -Force -Path dist | Out-Null

$env:GOOS = "windows"
$env:GOARCH = "amd64"
go build -o dist/xfchat-bootstrapper-windows-amd64.exe ./cmd/xfchat-bootstrapper
