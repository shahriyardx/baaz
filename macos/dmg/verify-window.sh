#!/usr/bin/env bash
# Mounts a built DMG and checks the window Finder will actually show.
#
# Worth doing because the interesting failure is invisible: Finder stores the
# background as an alias, and an alias resolves by file id first. A layout
# built against a different volume therefore applies its window size and icon
# positions perfectly while quietly resolving the background to the wrong
# node — a styled window with no background, which no build log would mention.
#
# Usage: macos/dmg/verify-window.sh path/to/Baaz.dmg
set -euo pipefail

dmg="${1:?usage: verify-window.sh path/to/Baaz.dmg}"
vol="Baaz"
mnt="/Volumes/$vol"

hdiutil detach "$mnt" -quiet -force 2>/dev/null || true
hdiutil attach "$dmg" -quiet -noautoopen -mountpoint "$mnt"
trap 'hdiutil detach "$mnt" -quiet -force 2>/dev/null || true' EXIT

fail=0
check() { # label, actual, expected
  if [ "$2" = "$3" ]; then
    printf "  ok    %-20s %s\n" "$1" "$2"
  else
    printf "  FAIL  %-20s %s (expected %s)\n" "$1" "$2" "$3"
    fail=1
  fi
}

# The alias embeds the file's name, so its presence in the .DS_Store is a
# reliable sign the background was recorded. Finder's own
# `background picture ... exists` is not: it answers false even for a volume
# that is visibly drawing one.
ds="$mnt/.DS_Store"
check "layout present" "$([ -f "$ds" ] && echo yes || echo no)" "yes"
check "background ref" "$(grep -qa 'background.tiff' "$ds" 2>/dev/null && echo yes || echo no)" "yes"
check "background file" "$([ -f "$mnt/.background/background.tiff" ] && echo yes || echo no)" "yes"
check "volume icon" "$(GetFileInfo "$mnt" 2>/dev/null | grep -i attributes | grep -q 'C' && echo yes || echo no)" "yes"
check "app present" "$([ -d "$mnt/Baaz.app" ] && echo yes || echo no)" "yes"
check "applications link" "$([ -L "$mnt/Applications" ] && echo yes || echo no)" "yes"

# Window geometry, read back through Finder. Every probe is tolerant: a
# failing osascript inside a command substitution would otherwise take the
# whole script down under `set -e`, and a machine with no Finder session
# should skip these rather than fail the build.
ask() { osascript -e "tell application \"Finder\" to tell disk \"$vol\" to return $1" 2>/dev/null || true; }

# Lists (bounds, positions) need the delimiter set, or `as text` runs the
# numbers together into one unreadable string.
asklist() {
  osascript -e 'set AppleScript'"'"'s text item delimiters to ","' \
            -e "tell application \"Finder\" to tell disk \"$vol\" to return ($1) as text" 2>/dev/null || true
}

if osascript -e "tell application \"Finder\" to open disk \"$vol\"" >/dev/null 2>&1; then
  sleep 2
  check "window bounds" "$(asklist 'bounds of container window')" "200,140,840,540"
  check "icon view" "$(ask 'current view of container window as string')" "icon view"
  check "toolbar hidden" "$(ask 'toolbar visible of container window')" "false"
  check "icon size" "$(ask 'icon size of icon view options of container window')" "112"
  check "app position" "$(asklist 'position of item "Baaz.app" of container window')" "160,205"
  check "applications pos" "$(asklist 'position of item "Applications" of container window')" "480,205"
  osascript -e "tell application \"Finder\" to close every window" >/dev/null 2>&1 || true
else
  echo "  --    window geometry     skipped (no Finder session)"
fi

exit $fail
