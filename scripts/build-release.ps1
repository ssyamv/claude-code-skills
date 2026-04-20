$ErrorActionPreference = "Stop"
Set-StrictMode -Version Latest

Remove-Item -Recurse -Force dist -ErrorAction SilentlyContinue
New-Item -ItemType Directory -Force -Path dist | Out-Null

$env:GOOS = "darwin"
$env:GOARCH = "arm64"
go build -o dist/xfchat-bootstrapper-darwin-arm64 ./cmd/xfchat-bootstrapper

$env:GOOS = "darwin"
$env:GOARCH = "amd64"
go build -o dist/xfchat-bootstrapper-darwin-amd64 ./cmd/xfchat-bootstrapper

$env:GOOS = "windows"
$env:GOARCH = "amd64"
go build -o dist/xfchat-bootstrapper-windows-amd64.exe ./cmd/xfchat-bootstrapper

$requiredFiles = @(
  "dist/xfchat-bootstrapper-darwin-arm64",
  "dist/xfchat-bootstrapper-darwin-amd64",
  "dist/xfchat-bootstrapper-windows-amd64.exe"
)

foreach ($file in $requiredFiles) {
  $item = Get-Item $file -ErrorAction SilentlyContinue
  if ($null -eq $item -or $item.Length -le 0) {
    throw "missing or empty release artifact: $file"
  }
}

$expectedFiles = $requiredFiles | Sort-Object
$actualFiles = Get-ChildItem -Path dist -Recurse -File | ForEach-Object {
  $_.FullName.Substring((Get-Location).Path.Length + 1) -replace '\\', '/'
} | Sort-Object

if (($actualFiles -join "`n") -ne ($expectedFiles -join "`n")) {
  throw @"
unexpected release artifact set
expected:
$($expectedFiles -join "`n")
actual:
$($actualFiles -join "`n")
"@
}
