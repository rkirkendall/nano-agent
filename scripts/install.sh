#!/usr/bin/env bash
set -euo pipefail

REPO="rkirkendall/nano-agent"
BIN="nano-agent"
DEST_DIR="${DEST_DIR:-/usr/local/bin}"

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
sudo install -m 0755 "$tmp/${BIN}/${BIN}" "$DEST_DIR/$BIN"
echo "Installed $BIN to $DEST_DIR/$BIN"

