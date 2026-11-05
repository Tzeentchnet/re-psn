# Releasing

How a maintainer cuts a re-psn pre-release: where the version lives, the bump, the tag, the draft release and what is
checked on the way. Pre-releases come often and on no schedule, whenever there is something worth trying.

> [!IMPORTANT]
> Not run yet: there is no tag and nothing has been published. The release workflow has not run on a tag, the dry run
> (a manual run of the workflow) has not been done, and the workflow still builds its notes itself instead of with
> `scripts/release-notes.sh` (see [Not checked yet](#not-checked-yet)).

## Contents

- [Versions and tags](#versions-and-tags)
- [Cut a pre-release](#cut-a-pre-release)
- [Before the first tag: the dry run](#before-the-first-tag-the-dry-run)
- [Checked automatically](#checked-automatically)
- [Not checked yet](#not-checked-yet)

## Versions and tags

- [`VERSION`](../VERSION) holds the version, `X.Y.Z`. The default of `var version` in `server/main.go`, the newest numbered
  heading in [CHANGELOG.md](../CHANGELOG.md) and the `<!-- version -->...<!-- /version -->` span in the
  [README](../README.md) status line state the same number; `TestVersionAgrees` fails when they differ. Release builds
  also set the version with `-ldflags -X main.version=...` from the tag.
- A tag is `v` + `VERSION` at the tagged commit, for example `v0.2.0`. Tags are never moved, deleted or reused. A bad
  release is followed by the next version.
- Every release is a pre-release while the version is below 1.0.0: the workflow creates it with the pre-release flag,
  the notes start with a pre-release notice, and `release-check` fails a 0.x release that is not marked pre-release.
- The notes end with `Next release: when ready`. There is no release day and no skipped cycle to announce.
- `VERSION` starts at 0.1.0, the number the code and CHANGELOG.md already state (the original psnr's first version). No
  `v0.1.0` tag exists. The first pre-release of re-psn takes a new number with the first bump; 0.2.0 fits, since the
  Unreleased entries include protocol changes that clients built against 0.1.0 do not match.

## Cut a pre-release

1. Start from a current `main` with CI green. Everything to ship is merged and listed under `## [Unreleased]` in
   CHANGELOG.md.
2. On a new branch with a clean working tree, look at the bump, then make it:

   ```sh
   scripts/bump-version.sh --dry-run 0.2.0   # prints the diff, writes nothing
   scripts/bump-version.sh 0.2.0
   ```

   It changes exactly four files: `VERSION`, the `var version` default in `server/main.go`, `CHANGELOG.md` (the
   Unreleased entries move under `## [0.2.0] - <today, UTC>`, a fresh empty Unreleased stays above, and an
   `[Unreleased]: .../compare/...` link moves on if the file has one) and the version span in `README.md`. It refuses a
   version that is not `X.Y.Z` or not newer than `VERSION`, an empty Unreleased section, a CHANGELOG whose newest version
   is not `VERSION`, and a dirty working tree. `--date YYYY-MM-DD` sets the heading's date.
3. Review: `git diff` shows only those version places, the span and the CHANGELOG move. Then run the checks and read the
   notes:

   ```sh
   scripts/check-release.sh
   scripts/release-notes.sh 0.2.0
   ```

4. Commit the four files by name as one commit, with nothing else in it, open a pull request, and merge it once CI is
   green and it is reviewed. The script makes no commit, tag or push.
5. Tag the merge commit on `main`, and only with the maintainer's word:

   ```sh
   git tag -a v0.2.0 -m v0.2.0 <merge commit>
   git push origin v0.2.0
   ```

6. The tag starts [`.github/workflows/release.yml`](../.github/workflows/release.yml): vet and tests, the archives for
   windows/amd64, linux/amd64 and linux/arm64 with a `.sha256` file each and `SHA256SUMS` (LF, no BOM), and a **draft**
   pre-release titled with the tag. The job fails if a release for the tag exists already.
7. Check the draft: the notes, the assets (three archives, their `.sha256` files, `SHA256SUMS`), and a download checked
   with `sha256sum -c SHA256SUMS`. A bad draft is deleted before it is published; then re-run the workflow for the same
   tag, or, if the tagged commit is wrong, bump to the next version.
8. Publishing is the maintainer's. Publishing starts
   [`.github/workflows/release-check.yml`](../.github/workflows/release-check.yml), which reads the release back.
9. Afterwards: CI is green on the tagged commit, and a fixed issue gets a comment naming the release.

> [!NOTE]
> One bump is one commit, made by the maintainer. A fix found after the tag goes into the next pre-release, never into a
> moved tag.

## Before the first tag: the dry run

Run the release workflow by hand (Actions, release, Run workflow). A manual run builds the archives for `v0.0.0` and keeps
them as a run artifact; it creates no release. Before the first tag, check that artifact by hand: the sizes, the checksums
against `SHA256SUMS`, and that the Linux archive unpacks in an empty directory and the server inside starts without Go or
build tools. The Windows and arm64 archives are cross-built; say in the notes which ones were not run.

`scripts/release-build.sh v0.0.0` builds the same archives locally into `dist/` (ignored by git).

## Checked automatically

| Check | Where |
|---|---|
| `VERSION`, `server/main.go`, the newest CHANGELOG.md version and the README span agree (`TestVersionAgrees`) | `go test ./...` in CI on pushes to `main` and pull requests; `scripts/check-release.sh`; `scripts/release-build.sh` |
| The bump and notes scripts on a fixture tree (`TestBumpVersion`, `TestReleaseNotes`) | the same; skipped on Windows, where the scripts do not run |
| The notes for the current version build from the real CHANGELOG.md | `scripts/check-release.sh` |
| Tag shape `vMAJOR.MINOR.PATCH` | `scripts/release-build.sh`, both release workflows |
| The deploy examples match the server's flags | `scripts/check-deploy.sh`, run by `scripts/release-build.sh` |
| `SHA256SUMS` verifies | `scripts/release-build.sh`, `release.yml` (build and draft jobs), `release-check.yml` |
| No release exists for the tag yet | `release.yml` |
| Published, titled with the tag, pre-release while 0.x, exactly the expected assets, checksums | `release-check.yml` |

## Not checked yet

- That the tag equals `v` + `VERSION` at the tagged commit. Nothing compares them yet.
- The notes `release.yml` writes come from its own copy of the Unreleased section, which a bump leaves empty, and end with
  a different closing line. Until the workflow uses `scripts/release-notes.sh`, read the draft's notes and fix them by hand
  in the draft before publishing.
- That the Windows and arm64 binaries run; they are cross-built. The exe and `SHA256SUMS` are unsigned.
- Sample log output in the README and `docs/running.md` (`version=0.1.0`) is an example and is not changed by the bump.
