#!/usr/bin/env bash
# Guards against Apple's XProtect deleting the binary as adware.
#
# Rule macos_adload_g_bundle (MACOS.ADLOAD) matches any Mach-O under 15MB
# containing ALL of "_main.main", "/Library/Application Support/Google/Chrome/",
# "killall" and "cfprefs". A plain build of the macOS Chrome-integration code
# hit all four, and macOS moved the binary to the Trash on first run.
#
# "_main.main" is unavoidable in a Go binary, so the other three must stay out.
# Build the Chrome paths with filepath.Join instead of writing the literal, and
# do not shell out to killall.
#
# Usage: macos/xprotect-check.sh path/to/binary
set -euo pipefail

bin="${1:?usage: xprotect-check.sh BINARY}"
[ -f "$bin" ] || { echo "xprotect-check: no such file: $bin"; exit 1; }

# All four must be present for the rule to fire; any one missing is safe.
needles=(
  '_main.main'
  '/Library/Application Support/Google/Chrome/'
  'killall'
  'cfprefs'
)

hits=0
present=()
for n in "${needles[@]}"; do
  if strings -a "$bin" | grep -qF -- "$n"; then
    hits=$((hits + 1))
    present+=("$n")
  fi
done

if [ "$hits" -eq "${#needles[@]}" ]; then
  echo "xprotect-check: FAIL — $bin matches XProtect rule macos_adload_g_bundle."
  echo "  macOS will delete this binary as MACOS.ADLOAD malware on first run."
  echo "  Remove one of these strings (see cmd/baaz/platform_darwin.go):"
  printf '    %s\n' "${present[@]}"
  exit 1
fi

echo "xprotect-check: ok ($hits/${#needles[@]} signature strings present, needs all ${#needles[@]} to trip)"
