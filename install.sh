#!/usr/bin/env bash
# baaz installer for Linux: downloads the latest release and finishes setup.
# Usage: curl -fsSL https://github.com/shahriyardx/baaz/releases/latest/download/install.sh | bash
#
# Run this as yourself, NOT with sudo. It asks for your password once, up
# front, and uses it only for the step that needs root (the Chrome extension
# and the no-save-prompt policy). Everything else is installed into your own
# account, which is where it has to live to work.
set -euo pipefail

REPO="shahriyardx/baaz"
BIN_DIR="${HOME}/.local/bin"
BIN="${BIN_DIR}/baaz"

if [ "$(id -u)" -eq 0 ]; then
  echo "baaz: run this as your normal user, not with sudo."
  echo "      It asks for your password when it needs it; running the whole"
  echo "      script as root would install into root's account instead."
  exit 1
fi

# Linux only. macOS installs from the disk image, which carries the app, the
# CLI and the whole setup — running this there would leave a half-install
# with no app, no menu bar item and no automatic updates.
case "$(uname -s)" in
  Linux) OS="linux" ;;
  Darwin)
    cat <<'MSG'
baaz: this installer is for Linux.

On macOS, download the disk image instead:

  https://github.com/shahriyardx/baaz/releases/latest/download/Baaz.dmg

Drag Baaz onto Applications and open it. It sets up everything, keeps
itself updated, and puts a falcon in your menu bar.
MSG
    exit 1
    ;;
  *) echo "baaz: unsupported OS: $(uname -s)"; exit 1 ;;
esac

case "$(uname -m)" in
  x86_64)         ARCH="amd64" ;;
  aarch64|arm64)  ARCH="arm64" ;;
  *) echo "baaz: unsupported architecture: $(uname -m)"; exit 1 ;;
esac

# ---------- sudo up front ----------
# `curl | bash` leaves stdin on the pipe, so point sudo at the terminal.
# HAVE_SUDO stays 0 when there is no terminal (CI, a pipe with no tty) or the
# user declines: the download still works, and the root step is printed at the
# end for them to run by hand.
HAVE_SUDO=0
SUDO_KEEPALIVE_PID=""
if command -v sudo >/dev/null 2>&1; then
  echo "baaz needs your password once, to install the Chrome extension"
  echo "and turn off Chrome's \"ask where to save\" dialog."
  echo
  if [ -r /dev/tty ] && sudo -v < /dev/tty; then
    HAVE_SUDO=1
  elif sudo -v; then
    HAVE_SUDO=1
  else
    echo "baaz: continuing without root; the Chrome step is printed at the end."
  fi
fi

if [ "$HAVE_SUDO" -eq 1 ]; then
  # Refresh the timestamp while the downloads run so the password is asked
  # only the once. Stops when this script exits.
  ( while kill -0 "$$" 2>/dev/null; do sudo -n true 2>/dev/null; sleep 50; done ) &
  SUDO_KEEPALIVE_PID=$!
  # shellcheck disable=SC2064
  trap "kill $SUDO_KEEPALIVE_PID 2>/dev/null || true" EXIT
fi

# ---------- the binary ----------
URL="https://github.com/${REPO}/releases/latest/download/baaz-linux-${ARCH}"
echo "Downloading baaz (linux/${ARCH})..."
mkdir -p "$BIN_DIR"
# Replace rather than overwrite in place. macOS caches a malware verdict per
# inode, so a binary that was ever blocked at this path stays blocked even
# after new bytes are written into the same file.
rm -f "$BIN"
curl -fL --progress-bar "$URL" -o "$BIN"
chmod +x "$BIN"
echo "Installed: $BIN"

# ---------- Chrome (needs root) ----------
echo
if [ "$HAVE_SUDO" -eq 1 ]; then
  echo "Setting up Chrome..."
  sudo "$BIN" install-chrome
else
  echo "Skipped the Chrome setup. To finish it, run:"
  echo "  sudo $BIN install-chrome"
fi

# ---------- yt-dlp + ffmpeg (optional, for video downloads) ----------
# Only offered, never silent: these come from the system package manager and
# ffmpeg is a large download. Skipping leaves everything except video
# downloads working.
missing=""
command -v yt-dlp >/dev/null 2>&1 || missing="yt-dlp"
# yt-dlp shells out to ffmpeg to join separate video and audio tracks.
command -v ffmpeg >/dev/null 2>&1 || missing="${missing:+$missing }ffmpeg"

if [ -n "$missing" ]; then
  install_cmd=""
  if command -v pacman >/dev/null 2>&1; then
    install_cmd="sudo pacman -S --needed --noconfirm $missing"
  elif command -v apt-get >/dev/null 2>&1; then
    install_cmd="sudo apt-get install -y $missing"
  elif command -v dnf >/dev/null 2>&1; then
    install_cmd="sudo dnf install -y $missing"
  elif command -v zypper >/dev/null 2>&1; then
    install_cmd="sudo zypper install -y $missing"
  fi

  echo
  echo "Video downloads (YouTube, Facebook, TikTok…) need: $missing"
  if [ -z "$install_cmd" ]; then
    echo "Install $missing with your package manager to enable them."
  # Opening /dev/tty is the real test — `[ -r /dev/tty ]` passes even with no
  # controlling terminal, and a failed read would otherwise look like the user
  # pressing Enter and start an install nobody agreed to.
  elif { exec 3</dev/tty; } 2>/dev/null; then
    printf "Install now with '%s'? [Y/n] " "$install_cmd"
    if read -r reply <&3; then
      exec 3<&-
      case "$reply" in
        [Nn]*) echo "Skipped. Run it later:  $install_cmd" ;;
        *)
          # A package manager failure must not abort the rest of the install.
          if sh -c "$install_cmd"; then
            echo "Installed $missing."
          else
            echo "NOTE: that failed. Run it by hand when convenient:"
            echo "  $install_cmd"
          fi
          ;;
      esac
    else
      exec 3<&-
      echo
      echo "Run this to enable them:  $install_cmd"
    fi
  else
    echo "Run this to enable them:  $install_cmd"
  fi
fi

# ---------- the bar widget ----------
echo
if command -v omarchy >/dev/null 2>&1; then
  "$BIN" install-bar || echo "NOTE: run '$BIN install-bar' by hand to add the bar widget."
fi

echo
if ! command -v baaz >/dev/null 2>&1; then
  echo "NOTE: ${BIN_DIR} is not in your PATH."
  echo "Add this line to your ~/.bashrc or ~/.zshrc and open a new terminal:"
  echo "  export PATH=\"\$HOME/.local/bin:\$PATH\""
  echo
fi
echo "Last step: install the extension from the Chrome Web Store,"
echo "  https://chromewebstore.google.com/detail/nidklljbjhpljgdeebcpbbnbcijbbcdl"
echo "then restart Chrome."
