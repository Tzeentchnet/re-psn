#!/usr/bin/env bash
# Check that deploy/psnr.env.example and the server's flags agree: every flag has a PSNR_<FLAG> line (commented is fine) and every
# line names a real flag. Also that the example files exist and name no real host or secret. Run by release-build.sh
# and runnable on its own: scripts/check-deploy.sh
set -euo pipefail
root="$(cd "$(dirname "$0")/.." && pwd)"
cd "$root/server"
flags="$(go run . -h 2>&1 | sed -n 's/^  -\([a-z0-9-]*\).*/\1/p' | sort -u)"
[ -n "$flags" ] || { echo "no flags found in the server's usage" >&2; exit 1; }
vars="$(sed -n 's/^#\?\(PSNR_[A-Z0-9_]*\)=.*/\1/p' "$root/deploy/psnr.env.example" | sort -u)"
want="$(for f in $flags; do [ "$f" = h ] || [ "$f" = help ] || echo "PSNR_$(echo "$f" | tr a-z- A-Z_)"; done | sort -u)"
if [ "$vars" != "$want" ]; then
  echo "deploy/psnr.env.example and the server's flags differ:" >&2
  diff <(echo "$want") <(echo "$vars") >&2 || true
  exit 1
fi
[ ! -e "$root/deploy/psnr.env" ] || { echo "deploy/psnr.env is a live settings file: only psnr.env.example ships" >&2; exit 1; }
for f in psnr.env.example psnr.service README.txt traefik/traefik.yml traefik/dynamic.yml nginx/psnr.conf haproxy/haproxy.cfg; do
  [ -s "$root/deploy/$f" ] || { echo "deploy/$f is missing or empty" >&2; exit 1; }
done
if grep -rEn '([0-9]{1,3}\.){3}[0-9]{1,3}' "$root/deploy" | grep -Ev '127\.0\.0\.1|0\.0\.0\.0' ; then
  echo "deploy/ names an address other than localhost" >&2; exit 1
fi
if grep -rEin '(token|secret|password)=[^ ]' "$root/deploy"; then
  echo "deploy/ holds a value for a token or secret" >&2; exit 1
fi
echo "deploy examples ok: $(echo "$want" | wc -l) settings"
