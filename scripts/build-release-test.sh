#!/usr/bin/env bash
set -euo pipefail

required_files=(
  dist/xfchat-bootstrapper-darwin-arm64
  dist/xfchat-bootstrapper-darwin-amd64
  dist/xfchat-bootstrapper-windows-amd64.exe
)

for file in "${required_files[@]}"; do
  if [[ ! -s "$file" ]]; then
    echo "missing or empty release artifact: $file" >&2
    exit 1
  fi
done

expected_files=$(printf '%s\n' "${required_files[@]}" | sort)
actual_files=$(find dist -type f | sort)

if [[ "$actual_files" != "$expected_files" ]]; then
  echo "unexpected release artifact set" >&2
  echo "expected:" >&2
  printf '%s\n' "${required_files[@]}" >&2
  echo "actual:" >&2
  find dist -type f | sort >&2
  exit 1
fi

echo "release artifacts match expected set"
