#!/bin/sh
# Release checks that need no network: VERSION, the default in server/main.go, the newest CHANGELOG.md version and the
# README version spans agree (TestVersionAgrees), the bump and notes scripts pass their tests, and the notes for the
# current version build from the real CHANGELOG.md. Run by release-build.sh and runnable on its own: scripts/check-release.sh
set -eu
root="$(cd "$(dirname "$0")/.." && pwd)"
(cd "$root/server" && go test -count=1 -run '^Test(VersionAgrees.*|ReleaseNotes|BumpVersion)$' .)
version="$(tr -d '\r\n' < "$root/VERSION")"
"$root/scripts/release-notes.sh" "$version" > /dev/null
echo "release checks passed for $version"
