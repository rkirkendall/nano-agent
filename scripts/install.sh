#!/usr/bin/env bash
set -euo pipefail

REPO="rkirkendall/nano-agent"
BIN="nano-agent"

# Default to a user-writable location; allow override via DEST_DIR
DEST_DIR="${DEST_DIR:-"$HOME/.local/bin"}"

os=$(uname -s | tr '[:upper:]' '[:lower:]')
arch=$(uname -m)
case "$arch" in
  x86_64|amd64) arch=amd64;;
  aarch64|arm64) arch=arm64;;
  *) echo "Unsupported arch: $arch" >&2; exit 1;;
esac

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

latest=$(curl -fsSL https://api.github.com/repos/$REPO/releases/latest | grep -o '"tag_name": *"[^"]*"' | cut -d '"' -f4)
url="https://github.com/$REPO/releases/download/${latest}/${BIN}_${os}_${arch}.tar.gz"

curl -fsSL "$url" -o "$tmp/${BIN}.tar.gz"
tar -C "$tmp" -xzf "$tmp/${BIN}.tar.gz"

# Locate extracted binary (archive wraps in a directory)
SRC=$(find "$tmp" -type f -name "$BIN" -perm -111 -print -quit)
if [ -z "$SRC" ]; then
  echo "Failed to locate extracted $BIN in archive" >&2
  exit 1
fi

# Ensure destination exists
mkdir -p "$DEST_DIR"

DEST="$DEST_DIR/$BIN"

# Use cp + chmod for maximum portability (BSD/Linux)
if [ -w "$DEST_DIR" ]; then
  cp -f "$SRC" "$DEST" && chmod 0755 "$DEST"
elif command -v sudo >/dev/null 2>&1 && [ "${NO_SUDO:-}" != "1" ]; then
  sudo cp -f "$SRC" "$DEST" && sudo chmod 0755 "$DEST"
else
  echo "Destination not writable and sudo disabled; set DEST_DIR to a writable directory." >&2
  exit 1
fi

echo "Installed $BIN to $DEST_DIR/$BIN"

# PATH hint
case ":$PATH:" in
  *:"$DEST_DIR":*) ;;
  *) echo "Add to PATH: export PATH=\"$DEST_DIR:\$PATH\"";;
esac

