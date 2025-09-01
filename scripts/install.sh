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

# Use sudo only if needed and allowed; otherwise attempt user install
INSTALL_CMD="install -m 0755"
if [ ! -w "$DEST_DIR" ] && command -v sudo >/dev/null 2>&1 && [ "${NO_SUDO:-}" != "1" ]; then
  INSTALL_CMD="sudo $INSTALL_CMD"
fi

$INSTALL_CMD "$tmp/${BIN}/${BIN}"
if [ -n "${SUDO_USER:-}" ] && echo "$INSTALL_CMD" | grep -q sudo; then
  # When installing with sudo without explicit target, install(1) puts in CWD; ensure target path
  $INSTALL_CMD -t "$DEST_DIR" "$tmp/${BIN}/${BIN}"
else
  $INSTALL_CMD -t "$DEST_DIR" "$tmp/${BIN}/${BIN}" 2>/dev/null || true
fi

echo "Installed $BIN to $DEST_DIR/$BIN"

# PATH hint
case ":$PATH:" in
  *:"$DEST_DIR":*) ;;
  *) echo "Add to PATH: export PATH=\"$DEST_DIR:\$PATH\"";;
esac

