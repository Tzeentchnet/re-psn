#!/bin/sh
# Records the transcripts in docs/art/transcripts/ from a real run on loopback, the same steps as the README's Example:
# the server with a debug log and the status page turned on, alice hosting a room, the status page and bob joining, then
# the C client's test.
# Each file holds the command lines as typed ("$ ...") and the output exactly as printed; nothing is edited afterwards.
#
#   sh tools/art/capture.sh        # from the repository root; needs go, gcc and curl, and ports 36100-36101 free
#
# Then run python3 tools/art/make_art.py to render the images from them.
set -eu
ROOT=$(cd "$(dirname "$0")/../.." && pwd)
OUT="$ROOT/docs/art/transcripts"
RUN=$(mktemp -d)
trap 'kill "$SP" 2>/dev/null || true; rm -rf "$RUN"' EXIT
SP=
export TZ=UTC
mkdir -p "$OUT"
(cd "$ROOT" && go build -o "$RUN/re-psn" ./server)
ln -s "$ROOT/client" "$RUN/client"
ln -s "$ROOT/lab" "$RUN/lab"
cd "$RUN"
gcc -std=gnu11 -o natcheck client/psnr.c lab/natcheck.c
gcc -std=gnu11 -o test_psnr client/psnr.c client/test_psnr.c

# rec FILE COMMAND...: append the command line and its output to FILE.
rec() { f=$1; shift; printf '$ %s\n' "$*" >> "$OUT/$f"; sh -c "$*" >> "$OUT/$f" 2>&1; }
for f in server alice bob client-test; do : > "$OUT/$f.txt"; done

printf '$ ./re-psn -v -http 127.0.0.1:36101\n' > "$OUT/server.txt"
./re-psn -v -http 127.0.0.1:36101 >> "$OUT/server.txt" 2>&1 &
SP=$!
sleep 1
printf '$ ./natcheck 127.0.0.1 alice host 3658\n' > "$OUT/alice.txt"
./natcheck 127.0.0.1 alice host 3658 >> "$OUT/alice.txt" 2>&1 &
AP=$!
sleep 1
rec bob.txt curl -s http://127.0.0.1:36101/api/stats
rec bob.txt ./natcheck 127.0.0.1 bob join 3659
wait "$AP"
rec client-test.txt ./test_psnr 127.0.0.1 36100
rec client-test.txt curl -s http://127.0.0.1:36101/api/stats
sleep 0.5
kill -TERM "$SP"
wait "$SP" || true
SP=
echo "capture: wrote $(ls "$OUT" | tr '\n' ' ')to docs/art/transcripts/"
