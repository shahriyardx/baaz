#!/usr/bin/env bash
# Packs the extension for a Chrome Web Store upload.
#
# The one difference from what baaz ships itself is the "key" field. Locally
# it pins the extension ID so unpacked loads, the packed CRX and the native
# messaging manifest all agree on one ID. The Web Store ignores it and
# assigns an ID of its own, and an upload carrying a key is rejected — so it
# is stripped here rather than removed from the manifest, which would break
# every non-store install.
#
# After the first upload, copy the public key the dashboard shows back into
# extension/manifest.json as "key" and set defaultExtID in
# cmd/baaz/platform_*.go to the ID the dashboard assigns. Every install route
# then shares the published ID.
#
# Usage: extension/pack-store.sh [OUTPUT_ZIP]
set -euo pipefail

here="$(cd "$(dirname "$0")" && pwd)"
out="${1:-$(dirname "$here")/build/baaz-extension-store.zip}"

stage="$(mktemp -d)"
trap 'rm -rf "$stage"' EXIT
mkdir -p "$stage/extension"

# A whitelist, not a blacklist. The extension directory also holds the
# publishing notes and the screenshot tooling, and a blacklist quietly ships
# whatever gets added next — an earlier build put store-assets/, including a
# .swift file, inside the upload.
for item in manifest.json background.js content.js popup.html popup.js icons; do
  [ -e "$here/$item" ] || { echo "pack-store: missing $item"; exit 1; }
  cp -R "$here/$item" "$stage/extension/"
done
find "$stage/extension" -name '.DS_Store' -delete

python3 - "$stage/extension/manifest.json" <<'PY'
import json, sys, collections
p = sys.argv[1]
with open(p) as f:
    m = json.load(f, object_pairs_hook=collections.OrderedDict)
removed = m.pop("key", None)
with open(p, "w") as f:
    json.dump(m, f, indent=2)
    f.write("\n")
print("  stripped 'key'" if removed else "  no 'key' to strip")
print("  name:", m["name"], "| version:", m["version"])
PY

mkdir -p "$(dirname "$out")"
rm -f "$out"
( cd "$stage/extension" && zip -qr "$out" . -x '.*' )
echo "built $out ($(du -h "$out" | cut -f1))"
echo
echo "Upload it at https://chrome.google.com/webstore/devconsole"
