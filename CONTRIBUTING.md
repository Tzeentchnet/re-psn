# Contributing

This is a fork of [sp00nznet/psnr](https://github.com/sp00nznet/psnr). Bug reports, fixes and ideas are welcome here, through
the issue forms (bug report, feature request) or a pull request. Something that is also wrong in upstream may be worth
reporting there too.

## Ground rules

- **No game material, ever:** no disc or update files, decrypted or lifted game code, extracted assets, saves, keys or
  firmware, and no links to them. The server never carries game data; room blobs are capped at 4096 bytes.
- **No secrets or private logs:** remove user names and paths from anything you paste.
- Security problems: see [SECURITY.md](SECURITY.md), not an issue.

## Making a change

- One focused change per pull request, with tests in the same commit. Say what you ran and what you did not.
- A change to the wire format bumps the protocol version and updates [docs/api.md](docs/api.md) in the same commit.
- Commits: a plain subject line, then a short body saying what and why.
- The pull request template asks for Why, Change, Tests, Evidence and Overlap.
- Using an AI coding agent is fine; you remain responsible for what it submits, and the pull request should say so.

## Testing

What CI runs, and what to run first:

```
gofmt -l server
go vet ./...
go test -race ./...
```

On Linux the tests also build `client/test_psnr.c` and run it against the server (they skip without a C compiler).
`nix develop` gives a shell with Go and gcc. CI also runs `govulncheck`, a secret scan and the NAT lab (`sh lab/run.sh`, needs Docker).
