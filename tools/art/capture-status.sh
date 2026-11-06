#!/bin/sh
# Takes docs/status.png: a screenshot of the status page of a real server on loopback, in headless Chromium.
# The state on the page is real too: the C client's own test (client/test_psnr.c) leaves a leaderboard, and
# natcheck (lab/natcheck.c) as alice hosts a room and waits, as in the README Example. The page is the server's own HTML.
#
# This image is NOT part of tools/art/make_art.py or its --check: a browser screenshot is not byte-reproducible (fonts,
# Chromium version), so run this by hand when the page changes and commit the new PNG.
#
#   sh tools/art/capture-status.sh   # from the repository root; needs go, gcc, nix (for mermaid-cli's Chromium), python3
#                                    # with Pillow (optional, makes the PNG smaller), and ports 36100-36101 free
set -eu
ROOT=$(cd "$(dirname "$0")/../.." && pwd)
RUN=$(mktemp -d)
SP= AP=
trap 'kill "$AP" "$SP" 2>/dev/null || true; rm -rf "$RUN"' EXIT
(cd "$ROOT" && go build -o "$RUN/re-psn" ./server)
ln -s "$ROOT/client" "$RUN/client"
ln -s "$ROOT/lab" "$RUN/lab"
cd "$RUN"
gcc -std=gnu11 -o natcheck client/psnr.c lab/natcheck.c
gcc -std=gnu11 -o test_psnr client/psnr.c client/test_psnr.c
./re-psn -http 127.0.0.1:36101 >/dev/null 2>&1 &
SP=$!
sleep 1
./test_psnr 127.0.0.1 36100 >/dev/null
./natcheck 127.0.0.1 alice host 3658 >/dev/null 2>&1 &
AP=$!
sleep 1
curl -s http://127.0.0.1:36101/api/stats >/dev/null
# mermaid-cli carries puppeteer and a Chromium; find them from its launcher script.
mmdc=$(nix shell nixpkgs#mermaid-cli -c sh -c 'readlink -f "$(command -v mmdc)"')
chrome=$(sed -n "s/^export PUPPETEER_EXECUTABLE_PATH='\(.*\)'/\1/p" "$mmdc")
node=$(sed -n 's/^exec "\([^"]*node\)".*/\1/p' "$mmdc")
pup=$(dirname "$(dirname "$mmdc")")/lib/node_modules/@mermaid-js/mermaid-cli/node_modules/puppeteer
PUPPETEER_EXECUTABLE_PATH="$chrome" "$node" "$ROOT/tools/art/capture-status.js" "$pup" http://127.0.0.1:36101/ "$RUN/raw.png"
if python3 -c 'import PIL' 2>/dev/null; then
  python3 - "$RUN/raw.png" "$ROOT/docs/status.png" <<'PY'
import sys
from PIL import Image
im = Image.open(sys.argv[1]).convert("RGB")
im.quantize(32, method=Image.Quantize.MEDIANCUT, dither=Image.Dither.NONE).save(sys.argv[2], optimize=True)
PY
else
  cp "$RUN/raw.png" "$ROOT/docs/status.png"
fi
echo "capture-status: wrote docs/status.png"
