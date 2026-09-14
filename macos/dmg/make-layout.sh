#!/usr/bin/env bash
# Regenerates macos/dmg/DS_Store — the Finder window layout baked into the
# DMG: window size, icon-view settings, background picture, icon positions.
#
# This needs a logged-in Finder, so it is NOT part of the build. Run it on a
# Mac when the layout changes and commit the result; make-dmg.sh then just
# copies the file in, which works fine on a headless CI runner.
#
# Usage: macos/dmg/make-layout.sh        (needs build/Baaz.app to exist)
set -euo pipefail

here="$(cd "$(dirname "$0")" && pwd)"
root="$(dirname "$(dirname "$here")")"
app="$root/build/Baaz.app"
[ -d "$app" ] || { echo "make-layout: $app missing — run macos/make-app.sh first"; exit 1; }
[ -f "$here/background.tiff" ] || { echo "make-layout: background.tiff missing"; exit 1; }

vol="Baaz"
scratch="$(mktemp -d)"
dmg="$scratch/layout.dmg"
stage="$scratch/stage"
trap 'hdiutil detach "/Volumes/$vol" -quiet -force 2>/dev/null || true; rm -rf "$scratch"' EXIT

mkdir -p "$stage/.background"
cp "$here/background.tiff" "$stage/.background/background.tiff"
cp -R "$app" "$stage/Baaz.app"
ln -s /Applications "$stage/Applications"

hdiutil detach "/Volumes/$vol" -quiet -force 2>/dev/null || true
hdiutil create -volname "$vol" -srcfolder "$stage" -fs HFS+ -format UDRW -quiet "$dmg"
hdiutil attach "$dmg" -quiet -nobrowse -noautoopen -mountpoint "/Volumes/$vol"

# Window is 640x400 of content; the icon slots match render-background.swift.
osascript <<'APPLESCRIPT'
tell application "Finder"
  tell disk "Baaz"
    open
    set current view of container window to icon view
    set toolbar visible of container window to false
    set statusbar visible of container window to false
    set the bounds of container window to {200, 140, 840, 540}
    set opts to the icon view options of container window
    set arrangement of opts to not arranged
    set icon size of opts to 112
    set text size of opts to 12
    set label position of opts to bottom
    set background picture of opts to file ".background:background.tiff"
    set position of item "Baaz.app" of container window to {160, 205}
    set position of item "Applications" of container window to {480, 205}
    update without registering applications
    delay 2
    close
  end tell
end tell
APPLESCRIPT

sync
sleep 1
[ -f "/Volumes/$vol/.DS_Store" ] || { echo "make-layout: Finder wrote no .DS_Store"; exit 1; }
cp "/Volumes/$vol/.DS_Store" "$here/DS_Store"
hdiutil detach "/Volumes/$vol" -quiet
echo "wrote $here/DS_Store ($(stat -f%z "$here/DS_Store") bytes)"
