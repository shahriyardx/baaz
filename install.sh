#!/usr/bin/env bash
# baaz installer: downloads the latest release binary to ~/.local/bin.
# Usage: curl -fsSL https://github.com/shahriyardx/baaz/releases/latest/download/install.sh | bash
set -euo pipefail

REPO="shahriyardx/baaz"
BIN_DIR="${HOME}/.local/bin"

case "$(uname -m)" in
  x86_64)  ARCH="amd64" ;;
  aarch64) ARCH="arm64" ;;
  *) echo "baaz: unsupported architecture: $(uname -m)"; exit 1 ;;
esac

URL="https://github.com/${REPO}/releases/latest/download/baaz-linux-${ARCH}"
echo "Downloading baaz (${ARCH})..."
mkdir -p "$BIN_DIR"
curl -fL --progress-bar "$URL" -o "${BIN_DIR}/baaz"
chmod +x "${BIN_DIR}/baaz"

echo
echo "Installed: ${BIN_DIR}/baaz"
if ! command -v baaz >/dev/null 2>&1; then
  echo "NOTE: ${BIN_DIR} is not in your PATH."
  echo "Add this line to your ~/.bashrc or ~/.zshrc and open a new terminal:"
  echo "  export PATH=\"\$HOME/.local/bin:\$PATH\""
fi
echo
echo "Next steps:"
echo "  sudo ${BIN_DIR}/baaz install-chrome   # set up Chrome (extension + no save dialogs)"
echo "  ${BIN_DIR}/baaz install-bar           # Omarchy users: bar widget"
echo "Then restart Chrome. Done."
