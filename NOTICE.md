# Third-party components and credits

re-psn is a fork of [sp00nznet/psnr](https://github.com/sp00nznet/psnr) (MIT), written by sp00nz. The upstream licence and copyright line are in
[LICENSE](LICENSE) and stay in this fork; the server, the C client and the protocol are upstream's work, and this fork's changes are listed in
[CHANGELOG.md](CHANGELOG.md).

Feature page generator adapted from Project Helghast (https://github.com/undecuple/project-helghast, MIT, Copyright (c) 2026 undecuple): tools/features.py.
Art generator (tools/art/make_art.py) adapted from Project Helghast (https://github.com/undecuple/project-helghast, MIT, Copyright (c) 2026 undecuple): tools/release/installer_art.py, tools/features.py.
Colour palette from gruvbox-material (https://github.com/sainnhe/gruvbox-material, MIT, Copyright (c) 2020 sainnhe).

| Component | Use | Licence |
|---|---|---|
| Go standard library | the server (`server/`) uses nothing else | BSD-3-Clause (Go) |
| `client/psnr.c`, `client/psnr.h` | the C client, one file with no dependencies, written for upstream psnr | MIT |
| [JetBrains Mono](https://github.com/JetBrains/JetBrainsMono) | the font in the images in `docs/img/`; fetched by `tools/art/make_art.py`, not committed or shipped | SIL Open Font License 1.1, JetBrains |
| [Pillow](https://python-pillow.org/) | runs `tools/art/make_art.py`; in CI only, not shipped | MIT-CMU |
| [govulncheck](https://pkg.go.dev/golang.org/x/vuln/cmd/govulncheck), [gitleaks](https://github.com/gitleaks/gitleaks) | run in CI only; not shipped | BSD-3-Clause, MIT |

PlayStation, PS3 and PSN are trademarks of Sony Interactive Entertainment. This project is not affiliated with or endorsed by
Sony or any game publisher. The names are used only to say what the server stands in for; no logos or other marks are used.

**No game content is distributed.** This repository contains no game files, keys or firmware.
