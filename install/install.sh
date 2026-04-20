#!/usr/bin/env bash
set -euo pipefail

OWNER_REPO="ssyamv/claude-code-skills"
INSTALL_DIR="${HOME}/.local/bin"
BINARY_NAME="xfchat-bootstrapper"
VERSION="latest"
# Installs to ~/.local/bin/xfchat-bootstrapper.

usage() {
  cat <<'EOF'
Usage: install.sh [--version <tag>]
EOF
}

resolve_latest_version() {
  curl -fsSL "https://api.github.com/repos/${OWNER_REPO}/releases/latest" \
    | sed -n 's/.*"tag_name":[[:space:]]*"\([^"]*\)".*/\1/p' \
    | head -n 1
}

ensure_path_export() {
  local rc_file="$1"
  local block_begin="# xfchat-bootstrapper installer PATH"
  local block_end="# xfchat-bootstrapper installer PATH end"
  local path_line='export PATH="$HOME/.local/bin:$PATH"'
  local tmp_file
  local found_block=0
  local in_block=0
  local saw_content=0

  [[ -f "$rc_file" ]] || touch "$rc_file"
  tmp_file="$(mktemp "${TMPDIR:-/tmp}/xfchat-bootstrapper-rc.XXXXXX")"

  {
    while IFS= read -r line || [[ -n "$line" ]]; do
      saw_content=1
      if [[ "$line" == "$block_begin" ]]; then
        if [[ $found_block -eq 0 ]]; then
          printf '%s\n%s\n%s\n' "$block_begin" "$path_line" "$block_end"
          found_block=1
        fi
        in_block=1
        continue
      fi

      if [[ $in_block -eq 1 ]]; then
        if [[ "$line" == "$block_end" ]]; then
          in_block=0
        fi
        continue
      fi

      printf '%s\n' "$line"
    done <"$rc_file"

    if [[ $in_block -eq 1 ]]; then
      :
    elif [[ $found_block -eq 0 ]]; then
      if [[ $saw_content -eq 1 ]]; then
        printf '\n'
      fi
      printf '%s\n%s\n%s\n' "$block_begin" "$path_line" "$block_end"
    fi
  } >"$tmp_file"

  if [[ $in_block -eq 1 ]]; then
    rm -f "$tmp_file"
    printf 'warning: malformed PATH block in %s; leaving file unchanged\n' "$rc_file" >&2
    return 0
  fi

  mv "$tmp_file" "$rc_file"
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --version)
      [[ $# -ge 2 ]] || { echo "missing value for --version" >&2; usage >&2; exit 1; }
      VERSION="$2"
      shift 2
      ;;
    --help|-h)
      usage
      exit 0
      ;;
    *)
      echo "unknown argument: $1" >&2
      usage >&2
      exit 1
      ;;
  esac
done

case "$(uname -s)" in
  Darwin) ;;
  *)
    echo "xfchat-bootstrapper installer is only supported on macOS" >&2
    exit 1
    ;;
esac

case "$(uname -m)" in
  arm64|aarch64)
    ASSET_NAME="xfchat-bootstrapper-darwin-arm64"
    ;;
  x86_64)
    ASSET_NAME="xfchat-bootstrapper-darwin-amd64"
    ;;
  *)
    echo "unsupported macOS architecture: $(uname -m)" >&2
    exit 1
    ;;
esac

if [[ "$VERSION" == "latest" ]]; then
  VERSION="$(resolve_latest_version)"
fi

if [[ -z "$VERSION" ]]; then
  echo "failed to resolve release version" >&2
  exit 1
fi

DOWNLOAD_URL="https://github.com/${OWNER_REPO}/releases/download/${VERSION}/${ASSET_NAME}"
TARGET_PATH="${INSTALL_DIR}/${BINARY_NAME}"
TEMP_PATH="$(mktemp "${TMPDIR:-/tmp}/xfchat-bootstrapper.XXXXXX")"

trap 'rm -f "$TEMP_PATH"' EXIT

mkdir -p "$INSTALL_DIR"
curl -fsSL "$DOWNLOAD_URL" -o "$TEMP_PATH"
chmod +x "$TEMP_PATH"
mv "$TEMP_PATH" "$TARGET_PATH"
trap - EXIT

ensure_path_export "${HOME}/.zshrc"
ensure_path_export "${HOME}/.bashrc"

"$TARGET_PATH"
