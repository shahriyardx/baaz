#!/usr/bin/env bash
# Builds Baaz.dmg: the whole macOS install, as one drag.
#
# The app carries the baaz CLI (daemon + Chrome native-messaging host) and
# wires everything up on first launch, so there is nothing to run in a
# terminal afterwards.
#
# The window the user sees — size, background, icon positions — comes from
# macos/dmg/DS_Store, which is generated separately by macos/dmg/make-layout.sh
# and committed. Laying the window out needs a logged-in Finder, and CI has
# none; copying a prepared .DS_Store in needs nothing at all.
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
# it on. Anything else here becomes clutter in the mounted volume — the
# instructions live in the background image instead of a Read Me nobody opens.
stage="$(mktemp -d)"
trap 'hdiutil detach "$stage.mnt" -quiet -force 2>/dev/null || true; rm -rf "$stage" "$stage.mnt" "$stage.rw.dmg"' EXIT
cp -R "$app" "$stage/Baaz.app"
ln -s /Applications "$stage/Applications"

mkdir -p "$stage/.background"
cp "$here/dmg/background.tiff" "$stage/.background/background.tiff"

# Give the mounted volume the app's own icon, so it shows as the falcon on
# the desktop and in the Finder sidebar rather than a generic white disk.
# The icon file is staged here; the attribute that makes the system use it
# has to be set on the mounted volume, below.
icns="$app/Contents/Resources/Baaz.icns"
[ -f "$icns" ] && cp "$icns" "$stage/.VolumeIcon.icns"
if [ -f "$here/dmg/DS_Store" ]; then
  cp "$here/dmg/DS_Store" "$stage/.DS_Store"
else
  # Without it the volume still installs correctly, it just opens as a plain
  # Finder window. Worth saying out loud rather than shipping it silently.
  echo "make-dmg: warning: macos/dmg/DS_Store missing — window will be unstyled"
fi

rm -f "$dmg"
rw="$stage.rw.dmg"
hdiutil create -volname "Baaz" -srcfolder "$stage" -fs HFS+ -format UDRW -quiet "$rw"

# Mark the volume as having a custom icon. This needs the volume mounted —
# the attribute lives on the volume root, and setting it on the folder that
# was handed to hdiutil does not survive imaging. Mounting needs no window
# server, so this is fine on a headless runner; it is still best-effort,
# because a cosmetic icon is not worth failing a release over.
mnt="$stage.mnt"
mkdir -p "$mnt"
if hdiutil attach "$rw" -quiet -nobrowse -noautoopen -mountpoint "$mnt" 2>/dev/null; then
  if [ -f "$mnt/.VolumeIcon.icns" ] && command -v SetFile >/dev/null 2>&1; then
    SetFile -a C "$mnt" 2>/dev/null || echo "make-dmg: note: could not set the volume icon attribute"
  fi
  hdiutil detach "$mnt" -quiet || hdiutil detach "$mnt" -quiet -force || true
else
  echo "make-dmg: note: could not mount the image; volume icon skipped"
fi

hdiutil convert "$rw" -format UDZO -imagekey zlib-level=9 -quiet -o "$dmg"
rm -f "$rw"

echo "built $dmg ($(du -h "$dmg" | cut -f1))"
