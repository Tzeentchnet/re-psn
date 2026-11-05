<!-- Why, Change, Tests, Evidence, Overlap. What changes and why, what you ran and what you did not. -->

## Why

## Change

## Tests

## Evidence

## Overlap

- [ ] No game files, keys, credentials, saves or private logs are added
- [ ] No local paths or user names in the diff or the text
- [ ] `gofmt -l server`, `go vet ./...` and `go test -race ./...` pass (CI runs them too)
- [ ] A wire-format change bumps the protocol version and `docs/api.md` in the same commit
- [ ] Docs updated where behaviour or setup changes
