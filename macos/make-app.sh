#!/usr/bin/env bash
# Wraps the SwiftPM executable in Baaz.app.
#
# SwiftPM cannot emit an app bundle, and the menu bar app needs one: only a
# bundle carries the Info.plist whose LSUIElement keeps the app out of the
# Dock and the app switcher.
#
# Usage: macos/make-app.sh [OUTPUT_DIR] [VERSION]
set -euo pipefail

here="$(cd "$(dirname "$0")" && pwd)"
root="$(dirname "$here")"
out="${1:-$root/build}"
version="${2:-0.0.0-dev}"

app="$out/Baaz.app"
macos_dir="$app/Contents/MacOS"
res_dir="$app/Contents/Resources"

# Universal, so one bundle serves Apple Silicon and Intel. The CI runner is
# arm64 and a thin build would leave Intel users with an app that cannot run.
arch_flags="--arch arm64 --arch x86_64"

echo "building BaazMenuBar (release, universal)..."
swift build --package-path "$here/menubar" -c release $arch_flags

bin="$(swift build --package-path "$here/menubar" -c release $arch_flags --show-bin-path)/BaazMenuBar"
[ -x "$bin" ] || { echo "make-app: $bin missing"; exit 1; }

rm -rf "$app"
mkdir -p "$macos_dir" "$res_dir"
cp "$bin" "$macos_dir/BaazMenuBar"

# The CLI ships inside the bundle. It is the daemon, the Chrome
# native-messaging host and the command line tool, so dragging the app to
# Applications has to be enough to get it — there is no second download.
echo "building the baaz CLI (universal)..."
for a in amd64 arm64; do
  CGO_ENABLED=0 GOOS=darwin GOARCH="$a" go build -trimpath     -ldflags "-s -w -X main.version=${version}"     -o "$out/baaz-darwin-$a" "$root/cmd/baaz"
done
lipo -create -output "$res_dir/baaz" "$out/baaz-darwin-amd64" "$out/baaz-darwin-arm64"
rm -f "$out/baaz-darwin-amd64" "$out/baaz-darwin-arm64"
chmod 755 "$res_dir/baaz"

# XProtect deletes Go binaries matching its adware signature; the copy inside
# the bundle is just as exposed as a standalone one.
"$here/xprotect-check.sh" "$res_dir/baaz"

cat > "$app/Contents/Info.plist" <<PLIST
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>CFBundleName</key><string>Baaz</string>
	<key>CFBundleDisplayName</key><string>Baaz</string>
	<key>CFBundleExecutable</key><string>BaazMenuBar</string>
	<key>CFBundleIdentifier</key><string>com.shahriyar.baaz.menubar</string>
	<key>CFBundlePackageType</key><string>APPL</string>
	<key>CFBundleShortVersionString</key><string>${version}</string>
	<key>CFBundleVersion</key><string>${version}</string>
	<key>CFBundleIconFile</key><string>Baaz</string>
	<key>LSMinimumSystemVersion</key><string>13.0</string>
	<key>NSHighResolutionCapable</key><true/>
	<!-- A normal app: Dock tile, app switcher, main window. The menu bar
	     item is in addition to that, not instead of it. -->
</dict>
</plist>
PLIST

# Icon: reuse the extension artwork. Only Finder ever shows it (LSUIElement
# hides the Dock tile), so the largest PNG on hand is enough.
tmp="$(mktemp -d)"
iconset="$tmp/Baaz.iconset"
mkdir -p "$iconset"
for size in 16 32 128; do
  src="$root/extension/icons/icon${size}.png"
  [ -f "$src" ] || continue
  cp "$src" "$iconset/icon_${size}x${size}.png"
done
if [ -f "$root/extension/icons/icon128.png" ]; then
  sips -z 256 256 "$root/extension/icons/icon128.png" \
    --out "$iconset/icon_128x128@2x.png" >/dev/null 2>&1 || true
fi
iconutil -c icns "$iconset" -o "$res_dir/Baaz.icns" 2>/dev/null \
  || echo "make-app: no icns produced (harmless — the menu bar app has no Dock icon)"
rm -rf "$tmp"

# Ad-hoc signature. Unsigned bundles are killed on arm64; this is not
# notarization, it just makes the binary loadable on this machine.
codesign --force --deep --sign - "$app" >/dev/null 2>&1 \
  || echo "make-app: ad-hoc codesign failed (app may not launch)"

echo "built $app"
