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

# Ensure destination exists
mkdir -p "$DEST_DIR"

# Choose install strategy (BSD install doesn't support -t). Always specify dest filename.
SRC="$tmp/${BIN}/${BIN}"
DEST="$DEST_DIR/$BIN"

if command -v install >/dev/null 2>&1; then
  if [ -w "$DEST_DIR" ]; then
    install -m 0755 "$SRC" "$DEST"
  elif command -v sudo >/dev/null 2>&1 && [ "${NO_SUDO:-}" != "1" ]; then
    sudo install -m 0755 "$SRC" "$DEST"
  else
    echo "Destination not writable and sudo disabled; set DEST_DIR to a writable directory." >&2
    exit 1
  fi
else
  # Fallback to cp + chmod
  if [ -w "$DEST_DIR" ]; then
    cp -f "$SRC" "$DEST" && chmod 0755 "$DEST"
  elif command -v sudo >/dev/null 2>&1 && [ "${NO_SUDO:-}" != "1" ]; then
    sudo cp -f "$SRC" "$DEST" && sudo chmod 0755 "$DEST"
  else
    echo "Destination not writable and sudo disabled; set DEST_DIR to a writable directory." >&2
    exit 1
  fi
fi

echo "Installed $BIN to $DEST_DIR/$BIN"

# PATH hint
case ":$PATH:" in
  *:"$DEST_DIR":*) ;;
  *) echo "Add to PATH: export PATH=\"$DEST_DIR:\$PATH\"";;
esac

