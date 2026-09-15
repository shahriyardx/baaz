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
vol="Baaz"
[ -d "$app" ] || { echo "make-dmg: $app missing"; exit 1; }

# Stage exactly what the window should show: the app, and a shortcut to drop
# it on. Anything else here becomes clutter in the mounted volume — the
# instructions live in the background image instead of a Read Me nobody opens.
stage="$(mktemp -d)"
rw="$stage.rw.dmg"
trap 'hdiutil detach "/Volumes/$vol" -quiet -force 2>/dev/null || true; rm -rf "$stage" "$rw"' EXIT

cp -R "$app" "$stage/Baaz.app"
ln -s /Applications "$stage/Applications"
mkdir -p "$stage/.background"
cp "$here/dmg/background.tiff" "$stage/.background/background.tiff"

# The app's own icon, so the mounted volume shows the falcon on the desktop
# and in the sidebar rather than a generic white disk. Staged here only to
# reserve room in the image; it is written for real after the layout, below,
# because Finder deletes it in passing.
icns="$app/Contents/Resources/Baaz.icns"
[ -f "$icns" ] && cp "$icns" "$stage/.VolumeIcon.icns"

rm -f "$dmg" "$rw"
hdiutil detach "/Volumes/$vol" -quiet -force 2>/dev/null || true
hdiutil create -volname "$vol" -srcfolder "$stage" -fs HFS+ -format UDRW -quiet "$rw"

# Everything below has to happen on the mounted image rather than on the
# folder handed to hdiutil: the custom-icon attribute lives on the volume
# root, and the window layout has to be written against the volume that
# actually ships. Finder stores the background as an alias carrying the
# volume and file id it was made against, so a layout prepared on some other
# volume sets the size and the icon positions but silently loses the
# background — which is what the first release shipped.
#
# Mounted browsable, without -nobrowse, because Finder has to see the disk.
styled=no
if hdiutil attach "$rw" -quiet -noautoopen -mountpoint "/Volumes/$vol" 2>/dev/null; then
  if osascript "$here/dmg/layout.applescript" "$vol" >/dev/null 2>&1; then
    styled=yes
  elif [ -f "$here/dmg/DS_Store" ]; then
    # No Finder to drive — a headless runner, say. The prepared layout still
    # gives the right window size and icon positions; only the background is
    # lost, for the alias reason above.
    echo "make-dmg: warning: no Finder to lay the window out; using the prepared layout (no background)"
    cp "$here/dmg/DS_Store" "/Volumes/$vol/.DS_Store"
  else
    echo "make-dmg: warning: no Finder and no prepared layout; the window will be unstyled"
  fi

  # Strictly after the layout, and in this order. Opening a volume whose
  # custom-icon flag is not yet set makes Finder delete the .VolumeIcon.icns
  # it finds there, and rewriting the window settings clears the flag itself
  # — so doing either of these first is silently undone.
  if [ -f "$icns" ] && command -v SetFile >/dev/null 2>&1; then
    cp "$icns" "/Volumes/$vol/.VolumeIcon.icns"
    SetFile -a C "/Volumes/$vol" 2>/dev/null || echo "make-dmg: note: could not set the volume icon attribute"
  fi

  sync
  sleep 1
  hdiutil detach "/Volumes/$vol" -quiet || hdiutil detach "/Volumes/$vol" -quiet -force || true
else
  echo "make-dmg: warning: could not mount the image; the window will be unstyled"
fi

hdiutil convert "$rw" -format UDZO -imagekey zlib-level=9 -quiet -o "$dmg"
rm -f "$rw"

# Check the result rather than trusting it: mount the finished image and ask
# Finder whether the background actually resolved. This is the exact thing
# that was broken before, and it is invisible unless you look.
if [ "$styled" = yes ]; then
  "$here/dmg/verify-window.sh" "$dmg" || echo "make-dmg: warning: the built window did not verify"
fi

echo "built $dmg ($(du -h "$dmg" | cut -f1))"
