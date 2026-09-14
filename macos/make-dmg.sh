#!/usr/bin/env bash
# Builds Baaz.dmg: the whole macOS install, as one drag.
#
# The app carries the baaz CLI (daemon + Chrome native-messaging host) and
# wires everything up on first launch, so there is nothing to run in a
# terminal afterwards.
#
# Usage: macos/make-dmg.sh [OUTPUT_DIR] [VERSION]
set -euo pipefail

here="$(cd "$(dirname "$0")" && pwd)"
root="$(dirname "$here")"
out="${1:-$root/build}"
version="${2:-0.0.0-dev}"

"$here/make-app.sh" "$out" "$version"

app="$out/Baaz.app"
dmg="$out/Baaz.dmg"
[ -d "$app" ] || { echo "make-dmg: $app missing"; exit 1; }

# Stage exactly what the window should show: the app, and a shortcut to drop
# it on. Anything else here becomes clutter in the mounted volume.
stage="$(mktemp -d)"
trap 'rm -rf "$stage"' EXIT
cp -R "$app" "$stage/Baaz.app"
ln -s /Applications "$stage/Applications"

# A README the user sees before installing, covering the one manual step.
cat > "$stage/Read Me.txt" <<'TXT'
baaz — fast downloads for Chrome

1. Drag Baaz.app onto the Applications folder.
2. Open it. A falcon appears in the menu bar near the clock.
   (First launch: right-click the app and choose Open — the app is not
   notarized, so a plain double-click is refused once.)

Opening it sets up everything except the Chrome extension, which Chrome
does not let an app install. The app will show you how: in Chrome open
chrome://extensions, turn on Developer mode, click "Load unpacked", and
pick the baaz-extension folder in your Downloads.

The "baaz" command is installed to ~/.local/bin as well.
TXT

rm -f "$dmg"
hdiutil create \
  -volname "baaz" \
  -srcfolder "$stage" \
  -fs HFS+ \
  -format UDZO \
  -imagekey zlib-level=9 \
  -quiet \
  "$dmg"

echo "built $dmg ($(du -h "$dmg" | cut -f1))"
