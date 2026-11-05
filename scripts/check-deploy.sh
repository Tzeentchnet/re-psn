#!/usr/bin/env bash
# Check that deploy/re-psn.env.example and the server's flags agree: every flag has a RE_PSN_<FLAG> line (commented is fine), every
# line names a real flag, and the deprecated-names comment block lists the old PSNR_<FLAG> name of every flag and nothing else.
# Also that no live deploy/re-psn.env or deploy/psnr.env is in the tree, and that the example files exist and name no real host
# or secret. Run by release-build.sh and runnable on its own: scripts/check-deploy.sh
set -euo pipefail
root="$(cd "$(dirname "$0")/.." && pwd)"
cd "$root/server"
flags="$(go run . -h 2>&1 | sed -n 's/^  -\([a-z0-9-]*\).*/\1/p' | sort -u)"
[ -n "$flags" ] || { echo "no flags found in the server's usage" >&2; exit 1; }
example="$root/deploy/re-psn.env.example"
[ -s "$example" ] || { echo "deploy/re-psn.env.example is missing or empty" >&2; exit 1; }
suffixes="$(for f in $flags; do [ "$f" = h ] || [ "$f" = help ] || echo "$f" | tr a-z- A-Z_; done | sort -u)"
vars="$(sed -n 's/^#\?\(RE_PSN_[A-Z0-9_]*\)=.*/\1/p' "$example" | sort -u)"
want="$(echo "$suffixes" | sed 's/^/RE_PSN_/')"
if [ "$vars" != "$want" ]; then
  echo "deploy/re-psn.env.example and the server's flags differ:" >&2
  diff <(echo "$want") <(echo "$vars") >&2 || true
  exit 1
fi
if grep -n '^#\?PSNR_[A-Z0-9_]*=' "$example" >&2; then
  echo "deploy/re-psn.env.example has a PSNR_ setting line: old names belong in the deprecated-names comment only" >&2; exit 1
fi
old="$(sed -n '/^# --- deprecated names ---$/,/^# --- end of deprecated names ---$/p' "$example" | grep '^#   ' | grep -o 'PSNR_[A-Z0-9_]\+' | sort -u || true)"
wantold="$(echo "$suffixes" | sed 's/^/PSNR_/')"
if [ "$old" != "$wantold" ]; then
  echo "the deprecated names in deploy/re-psn.env.example and the server's flags differ:" >&2
  diff <(echo "$wantold") <(echo "$old") >&2 || true
  exit 1
fi
for live in re-psn.env psnr.env; do
  [ ! -e "$root/deploy/$live" ] || { echo "deploy/$live is a live settings file: only re-psn.env.example ships" >&2; exit 1; }
done
for f in re-psn.env.example psnr.service README.txt traefik/traefik.yml traefik/dynamic.yml nginx/psnr.conf haproxy/haproxy.cfg; do
  [ -s "$root/deploy/$f" ] || { echo "deploy/$f is missing or empty" >&2; exit 1; }
done
if grep -rEn '([0-9]{1,3}\.){3}[0-9]{1,3}' "$root/deploy" | grep -Ev '127\.0\.0\.1|0\.0\.0\.0' ; then
  echo "deploy/ names an address other than localhost" >&2; exit 1
fi
if grep -rEin '(token|secret|password)=[^ ]' "$root/deploy"; then
  echo "deploy/ holds a value for a token or secret" >&2; exit 1
fi
echo "deploy examples ok: $(echo "$want" | wc -l) settings, $(echo "$old" | wc -l) deprecated names"
