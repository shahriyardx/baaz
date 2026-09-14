#!/usr/bin/env bash
# Renders the Chrome Web Store screenshots.
#
# The scenes are HTML so the popup and the in-page button are the real thing
# rather than a drawing of it — scene1 loads popup.html itself. WebKit renders
# at 2x, so each is downscaled to the 1280x800 the store requires.
#
# Deliberately no third-party page content: a listing is rejected for showing
# anything whose rights are not yours, so the video is a plain placeholder.
#
# Usage: extension/store-assets/render.sh [OUTPUT_DIR]
set -euo pipefail

here="$(cd "$(dirname "$0")" && pwd)"
out="${1:-$(dirname "$(dirname "$here")")/build/store-screenshots}"

# scene1 embeds the live popup; refresh it from the real source each time.
python3 - "$here" <<'PY'
import sys, pathlib
here = pathlib.Path(sys.argv[1])
src = (here.parent / "popup.html").read_text()
src = src.replace('<script src="popup.js"></script>', '''<script>
document.getElementById("version").textContent="v" + "PLACEHOLDER";
document.getElementById("enabled").checked=true;
document.getElementById("dot").className="dot busy";
document.getElementById("statusText").textContent="connected";
document.getElementById("activeCount").textContent="3";
document.getElementById("live").classList.add("show");
</script>''')
import json
ver = json.loads((here.parent / "manifest.json").read_text())["version"]
src = src.replace("PLACEHOLDER", ver)
(here / "popup-live.html").write_text(src)
print(f"  popup-live.html rebuilt at v{ver}")
PY

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT
cp "$here"/*.html "$here"/*.css "$here"/shoot.swift "$tmp/"
( cd "$tmp" && swiftc -O -o shoot shoot.swift && ./shoot )

mkdir -p "$out"
names=(1-faster-downloads 2-video-one-click 3-privacy)
for i in 0 1 2; do
  sips -z 800 1280 "$tmp/baaz-scene$((i+1)).png" --out "$out/${names[$i]}.png" >/dev/null
done

# The listing needs the 128x128 icon uploaded separately from the package, so
# it belongs with the rest of the upload rather than being dug out of the
# source tree.
icon="$(dirname "$here")/icons/icon128.png"
if [ -f "$icon" ]; then
  cp "$icon" "$out/store-icon-128.png"
else
  echo "render: WARNING - icons/icon128.png missing; the listing needs it"
fi

echo "wrote $(ls "$out" | wc -l | tr -d ' ') files to $out"
