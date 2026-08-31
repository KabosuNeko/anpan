#!/usr/bin/env bash
set -euo pipefail

REPO="KabosuNeko/anpan"
BIN_NAME="anpan"
INSTALL_DIR="${ANPAN_INSTALL_DIR:-$HOME/.local/bin}"

detect_platform() {
  local os arch
  os="$(uname -s | tr '[:upper:]' '[:lower:]')"
  arch="$(uname -m)"

  case "$os" in
    linux) os="linux" ;;
    darwin) os="darwin" ;;
    *) echo "Unsupported OS: $os" >&2; exit 1 ;;
  esac

  case "$arch" in
    x86_64|amd64) arch="amd64" ;;
    arm64|aarch64) arch="arm64" ;;
    *) echo "Unsupported architecture: $arch" >&2; exit 1 ;;
  esac

  echo "anpan-${os}-${arch}.tar.gz"
}

get_latest_version() {
  if command -v curl >/dev/null 2>&1; then
    curl -fsSL "https://api.github.com/repos/$REPO/releases/latest" 2>/dev/null | grep '"tag_name"' | sed -E 's/.*"v?([^"]+)".*/\1/' || true
  elif command -v wget >/dev/null 2>&1; then
    wget -qO- "https://api.github.com/repos/$REPO/releases/latest" 2>/dev/null | grep '"tag_name"' | sed -E 's/.*"v?([^"]+)".*/\1/' || true
  fi
}

ASSET="$(detect_platform)"
VERSION="${ANPAN_VERSION:-}"
if [ -z "$VERSION" ]; then
  VERSION="$(get_latest_version || true)"
fi

if [ -z "$VERSION" ]; then
  DOWNLOAD_URL="https://github.com/$REPO/releases/latest/download/$ASSET"
else
  DOWNLOAD_URL="https://github.com/$REPO/releases/download/v${VERSION}/$ASSET"
fi

TMP_DIR="$(mktemp -d)"
cleanup() { rm -rf "$TMP_DIR"; }
trap cleanup EXIT

echo "→ Downloading $BIN_NAME ($ASSET) ..."
ARCHIVE="$TMP_DIR/$ASSET"
if command -v curl >/dev/null 2>&1; then
  if ! curl -fsSL "$DOWNLOAD_URL" -o "$ARCHIVE"; then
    echo "→ Retrying with latest release..." >&2
    DOWNLOAD_URL="https://github.com/$REPO/releases/latest/download/$ASSET"
    curl -fsSL "$DOWNLOAD_URL" -o "$ARCHIVE"
  fi
elif command -v wget >/dev/null 2>&1; then
  wget -qO "$ARCHIVE" "$DOWNLOAD_URL"
else
  echo "Error: curl or wget is required." >&2
  exit 1
fi

echo "→ Extracting $ASSET ..."
tar -xzf "$ARCHIVE" -C "$TMP_DIR"

mkdir -p "$INSTALL_DIR"
INSTALL_PATH="$INSTALL_DIR/$BIN_NAME"
mv "$TMP_DIR/$BIN_NAME" "$INSTALL_PATH"
chmod +x "$INSTALL_PATH"

echo "✓ Installed $BIN_NAME to $INSTALL_PATH"

if ! echo "$PATH" | tr ':' '\n' | grep -qx "$INSTALL_DIR"; then
  echo ""
  PROFILE=""
  if [ -f "$HOME/.zshrc" ]; then
    PROFILE="$HOME/.zshrc"
  elif [ -f "$HOME/.bashrc" ]; then
    PROFILE="$HOME/.bashrc"
  elif [ -f "$HOME/.profile" ]; then
    PROFILE="$HOME/.profile"
  fi

  if [ -n "$PROFILE" ] && ! grep -q "$INSTALL_DIR" "$PROFILE" 2>/dev/null; then
    echo "export PATH=\"$INSTALL_DIR:\$PATH\"" >> "$PROFILE"
    echo "✓ Added $INSTALL_DIR to $PROFILE"
    echo "  Restart your terminal or run: source $PROFILE"
  else
    echo "⚠  $INSTALL_DIR is not in your PATH"
    echo "   Add to your shell profile (.bashrc or .zshrc):"
    echo "   export PATH=\"\$HOME/.local/bin:\$PATH\""
  fi
fi

echo ""
echo "Run: $BIN_NAME --help"
"$INSTALL_PATH" --version 2>/dev/null || true
