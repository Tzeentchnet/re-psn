#!/usr/bin/env bash
# Build the release archives into dist/: static server binaries (no cgo) for the three targets, a .sha256 file per
# archive and one SHA256SUMS (LF, no BOM). Used by .github/workflows/release.yml and runnable locally.
# usage: scripts/release-build.sh vX.Y.Z
set -euo pipefail
tag="${1:?usage: scripts/release-build.sh vX.Y.Z}"
case "$tag" in v[0-9]*.[0-9]*.[0-9]*) ;; *) echo "tag '$tag' is not vMAJOR.MINOR.PATCH" >&2; exit 1 ;; esac
case "$tag" in *[!0-9v.]*) echo "tag '$tag' has odd characters" >&2; exit 1 ;; esac
version="${tag#v}"
root="$(cd "$(dirname "$0")/.." && pwd)"
dist="$root/dist"
rm -rf "$dist" && mkdir -p "$dist"
stage="$(mktemp -d)"
trap 'rm -rf "$stage"' EXIT

"$root/scripts/check-deploy.sh"

for target in windows/amd64 linux/amd64 linux/arm64; do
  os="${target%/*}"; arch="${target#*/}"
  exe=re-psn; [ "$os" = windows ] && exe=re-psn.exe
  name="re-psn-$tag-$os-$arch"
  d="$stage/$name"; mkdir -p "$d"
  (cd "$root" && CGO_ENABLED=0 GOOS="$os" GOARCH="$arch" \
    go build -trimpath -ldflags "-s -w -X main.version=$version" -o "$d/$exe" ./server)
  cp "$root/LICENSE" "$root/NOTICE.md" "$d/"
  cp -r "$root/deploy" "$d/examples"
  if [ "$os" = windows ]; then
    (cd "$stage" && zip -qrX "$dist/$name.zip" "$name")
  else
    tar --sort=name --mtime=@0 --owner=0 --group=0 --numeric-owner -C "$stage" -czf "$dist/$name.tar.gz" "$name"
  fi
done

cd "$dist"
for f in re-psn-*; do sha256sum "$f" > "$f.sha256"; done
sha256sum re-psn-*.zip re-psn-*.tar.gz > SHA256SUMS
sha256sum -c SHA256SUMS
