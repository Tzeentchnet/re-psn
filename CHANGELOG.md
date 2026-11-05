# Changelog

All notable changes to this project are documented here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and the project uses
[Semantic Versioning](https://semver.org/).

## [Unreleased]

### Added
- A features list: `docs/features.tsv` holds one row per feature with its status and evidence, and `tools/features.py` generates
  `docs/features.md` from it. CI runs `python3 tools/features.py --check`, which fails on a stale page, a malformed row, or
  evidence that does not exist.
- Release archives include an `examples/` folder: `re-psn.env.example` with every setting, a systemd unit, and Traefik, nginx and HAProxy
  configs that pass the real client address. `scripts/check-deploy.sh` checks the env file against the server's flags and runs in
  `scripts/release-build.sh`. Only the Traefik config has been run.
- Reverse proxy support: `-trusted-proxies` and `-proxy-protocol` (PROXY protocol v1/v2 on the game TCP port), and forwarded
  headers for the status page log, both off by default. The real client address then counts for `-max-conns-per-addr`, the
  name take-over rule, `public_ip` and the log. UDP is not proxied. See `docs/running.md`.
- Every flag can be set as a `PSNR_<FLAG>` environment variable (a flag wins), and `-tcp-addr` / `-udp-addr` give the two
  game sockets their own addresses or ports. A bad value stops the server at start. See `docs/running.md`.
- Invite-only rooms (protocol 0.4): `SET_ROOM_ACCESS`, `JOIN_BY_CODE` and `GET_ROOM_ACCESS` give a room an 8-character code and take it
  out of search and the status page (which counts "unlisted rooms"). Wrong codes are rate-limited. HELLO_ACK flags bit 1 says the server
  has them. See `docs/api.md`.
- Logging with levels: `-log-level` (debug, info, warn, error), `-log-format` (text or json) and `-log-file`. Every refusal
  (connection or request limit, name taken, status page token) is now logged at warn level; `-v` is the same as
  `-log-level debug`. Tokens and query strings are never logged. See `docs/running.md`.
- One player per name on a server, compared case-insensitively. A HELLO with a
  name someone else holds gets ERROR 8 (name taken); from the same IP address
  it is the same player reconnecting, and the old connection is closed. The C
  client reports the refusal through `psnr_connect_error()`.
- Firewall: `Setup.cmd` offers to add a Windows Firewall rule for the server
  (TCP 36100, one UAC prompt); `setup.sh` prints the ufw/firewalld command when
  a firewall is on. `docs/running.md` covers the server's and players' rules.

- NAT traversal. The server listens on UDP 36100 too. A player's P2P socket
  sends it a PROBE (with a token from HELLO_ACK), and the server records the
  public endpoint the probe came from. Member entries now give each viewer the
  address that works from where they are: that public endpoint from another
  network, the member's LAN address from the same one. Clients built against
  0.2 still work; without a probe nothing changes. The C client builds and
  reads the packets (`psnr_probe_packet`, `psnr_punch_packet`,
  `psnr_probe_reply`, `psnr_is_control`); the caller sends them from its own
  P2P socket. Protocol in `docs/api.md`, "UDP".
- The relay (`-relay`, off by default), for players who can't reach each other
  directly. Streams into a player behind a router go through the server
  (STREAM_CONNECT / STREAM_OFFER / STREAM_ACCEPT / STREAM_READY), and ROUTE
  pushes say which members need it. Datagrams go through the server's UDP
  port (RELAY) when punching doesn't get through, as with a symmetric NAT.
  HELLO_ACK gains a flags byte saying whether the relay is on. The C client
  wraps and unwraps relayed datagrams and opens and accepts relayed streams.
  In the lab, the symmetric-NAT scenario now passes, and every scenario also
  runs a host-to-joiner stream.
- `lab/`: a NAT lab in Docker. A psnr server on a pretend internet, headless
  players behind NATing routers, and three scenarios: two homes (passes), a
  symmetric NAT (fails until psnr has a relay), and two players in one home
  (passes, over LAN addresses). CI runs it.

### Fixed
- `docs/running.md` said the P2P port is UDP only; titles use TCP on it too.

### Changed
- `ROADMAP.md` is rewritten around the feature ids (Done, Now, Next, Later, Not planned), keeping upstream's scope section; the
  features list gains planned and later rows for storage, high availability, metrics, the apps' themes and the docs, and
  `tools/features.py --check` also fails when a roadmap item sits under a section that does not match its status.
- `deploy/README.txt` is now `deploy/README.md` (`examples/README.md` in the release archives); the docs share one layout, and the deploy and Docker READMEs gain Mermaid diagrams.
- The docs use the fork's name, re-psn, and credit sp00nz as the author of the original psnr. `client/psnr.c` keeps its name.
- Settings are read from `RE_PSN_<FLAG>` variables. The old `PSNR_<FLAG>` names still work as aliases: a flag wins, then
  `RE_PSN_<FLAG>`, then `PSNR_<FLAG>`. An old name logs a warning at start naming the old and new variable (never the value), and
  if both are set and differ the new one is used, with a warning. `deploy/psnr.env.example` is now `deploy/re-psn.env.example`,
  with the old names in a comment block; `scripts/check-deploy.sh` checks both lists and refuses a live `deploy/re-psn.env` or
  `deploy/psnr.env`. The compose file, Docker README and proxy examples use the new names.
- The server binary is `re-psn` (`re-psn.exe` on Windows): release archives are `re-psn-vX.Y.Z-<os>-<arch>` (.tar.gz, .zip) with
  `re-psn-*.sha256` files and SHA256SUMS, and the release workflows look for those names. No `psnr` copy is shipped. The image, the
  compose service and the `Setup.cmd` / `setup.sh` launchers (`Start re-psn.cmd`, `start-re-psn.sh`) and Windows firewall rule
  (`re-psn`; the old `psnr` rule is removed) use the new name. `deploy/psnr.service` is now `deploy/re-psn.service`: it starts
  `/usr/local/bin/re-psn` and reads `/etc/psnr/psnr.env`, then `/etc/re-psn/re-psn.env` (both optional, the later one wins); the
  state directory stays `/var/lib/psnr`. `scripts/check-deploy.sh` checks the unit's binary and both env paths. The Go module path,
  the C API, the UDP magic and the wire format are unchanged.
- The architecture and NAT lab diagrams are Mermaid (GitHub renders them), and `docs/architecture.md` gains a diagram of the optional
  reverse proxy path and diagrams of a relayed connection (a VPS example) and of how a relayed stream is set up. The text there that said
  there is no hole punching and no relay is corrected.
- PROXY protocol v2 headers are now checked for family and transport: unassigned values, and a datagram or unspecified transport on the
  TCP game port, close the connection (a LOCAL header still carries no client and is consumed to its declared length). The nginx example
  is labelled v1 (`proxy_protocol on;` sends v1), and `docs/running.md` says Cloudflare Spectrum's PROXY protocol is an Enterprise feature.
- The Dockerfile, compose file and `Dockerfile.dockerignore` moved into `docker/`, with a README. Build from the repository root with
  `docker build -f docker/Dockerfile .`, or run `docker compose -f docker/docker-compose.yml up -d`. `lab/run.sh` and Dependabot follow.
- The Docker image no longer forces `-http 0.0.0.0:36101` without a token: the status page stays on localhost inside the
  container until `PSNR_HTTP` and `PSNR_HTTP_TOKEN` are set (the compose file shows how).
- Protocol additions that Simpsons Arcade's Matching2 use needs. Clients
  built against 0.1.0 no longer match.
  - CREATE_ROOM and JOIN_ROOM carry the member's own data, and every member
    entry returns it.
  - SET_ROOM_DATA `which` 2 sets the room's flags, which searchers see.
  - ROOM_MSG pushes say which member the message was sent to.

### Security
- UDP PROBE and RELAY packets are accepted only from the IP address of the player's TCP connection, so a stolen or guessed
  token, or a forged source address, can no longer point a player's endpoint at a third party. A probe's `local_ip` is kept
  only if it is private, carrier-grade NAT, link-local or the probe's own source. Players whose UDP leaves from another address
  (some multi-WAN or carrier NAT setups) are refused; `-udp-any-source` (`PSNR_UDP_ANY_SOURCE`) turns the check off. See
  `docs/running.md`.

## [0.1.0] - 2026-09-29

### Added
- Server (`server/`), speaking a big-endian TCP protocol with request ids so
  several asynchronous NP calls can be in flight at once.
- Rooms modelled on sceNpMatching2:
  - create, search, join, leave and kick;
  - external and internal room data, and room messages;
  - owner handoff when the owner leaves;
  - member pushes that carry each peer's address and P2P port.
- Leaderboards modelled on sceNpScore:
  - each player's best score kept, ties to whoever scored first;
  - higher-is-better and lower-is-better boards;
  - ranking by range and by online ID.
- A read-only status page and `/api/stats` JSON on a separate localhost
  listener.
- C client (`client/`) with no threads: `psnr_send` returns a request id, and
  `psnr_poll` hands out replies and pushes without blocking.
- Tests:
  - the Go end-to-end tests run over real sockets;
  - `TestCClient` builds the C client test and runs it against the server.
- Docker image and compose file. `Setup.cmd` / `setup.sh` quick start.
