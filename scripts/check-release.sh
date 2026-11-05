#!/bin/sh
# Release checks that need no network: with --tag, that the tag is v + VERSION (checked first, so a wrong tag fails before
# anything else runs); VERSION, the default in server/main.go, the newest CHANGELOG.md version and the README version spans
# agree (TestVersionAgrees); the bump and notes scripts pass their tests; and the notes for the current version build from
# the real CHANGELOG.md. Run by release-build.sh (without --tag) and by release.yml on a tag (with --tag).
# usage: scripts/check-release.sh [--tag vX.Y.Z]
set -eu
root="$(cd "$(dirname "$0")/.." && pwd)"
usage() { echo "usage: scripts/check-release.sh [--tag vX.Y.Z]" >&2; exit 2; }
tag=''
case "${1-}" in
  '') [ $# -eq 0 ] || usage ;;
  --tag) [ $# -eq 2 ] && [ -n "$2" ] || usage; tag="$2" ;;
  *) usage ;;
esac
version="$(tr -d '\r\n' < "$root/VERSION")"
if [ -n "$tag" ] && [ "$tag" != "v$version" ]; then
  echo "check-release: tag $tag is not v + VERSION (v$version)" >&2
  exit 1
fi
(cd "$root/server" && go test -count=1 -run '^Test(VersionAgrees.*|ReleaseNotes|BumpVersion|CheckReleaseTag)$' .)
"$root/scripts/release-notes.sh" "$version" > /dev/null
echo "release checks passed for $version${tag:+ (tag $tag)}"
