# Third-party components and credits

re-psn is a fork of [sp00nznet/psnr](https://github.com/sp00nznet/psnr) (MIT), written by sp00nz. The upstream licence and copyright line are in
[LICENSE](LICENSE) and stay in this fork; the server, the C client and the protocol are upstream's work, and this fork's changes are listed in
[CHANGELOG.md](CHANGELOG.md).

| Component | Use | Licence |
|---|---|---|
| Go standard library | the server (`server/`) uses nothing else | BSD-3-Clause (Go) |
| `client/psnr.c`, `client/psnr.h` | the C client, one file with no dependencies, written for upstream psnr | MIT |
| [govulncheck](https://pkg.go.dev/golang.org/x/vuln/cmd/govulncheck), [gitleaks](https://github.com/gitleaks/gitleaks) | run in CI only; not shipped | BSD-3-Clause, MIT |

PlayStation, PS3 and PSN are trademarks of Sony Interactive Entertainment. This project is not affiliated with or endorsed by
Sony or any game publisher. The names are used only to say what the server stands in for; no logos or other marks are used.

**No game content is distributed.** This repository contains no game files, keys or firmware.
