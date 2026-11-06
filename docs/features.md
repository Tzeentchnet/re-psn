<!-- Generated from docs/features.tsv; do not edit by hand. -->

# Features

What re-psn does today and what is planned, one row per feature. Each status rests on the evidence next to it: a test in this repository, a recorded run in the docs, or the pull request that added it. *Not yet verified* means the code or config is in the repository but nothing here checks it. *Since* is the version that first had the feature; *main* means merged but not in a release yet.

| Status | Meaning | Features |
|---|---|---|
| available | merged, with a test or a recorded run | 26 |
| not yet verified | merged, but nothing in the repository checks it | 1 |
| partial | merged in part, or with known gaps | 3 |
| in progress | an open pull request or branch | 0 |
| planned | wanted and intended, not started | 16 |
| later | after the planned items, or after a decision | 8 |
| declined | will not be done | 1 |
| all | | 55 |

## Rooms and matchmaking

| ID | Feature | Status | Since | Evidence | Note |
|---|---|---|---|---|---|
| `rooms-basic` | **Rooms and search**: Create, search, join and leave rooms modelled on sceNpMatching2. A search sees only the rooms of its own title, with each room's external data. Room messages, owner handoff when the owner leaves, and member pushes with each peer's address and P2P port. | available | 0.1.0 | [`server/rooms.go`](../server/rooms.go), [`TestRooms`](../server/server_test.go) | Upstream work. Kick (`KICK_MEMBER`) is in the server but no test covers it. |
| `rooms-data` | **Room and member data**: External and internal room data (only the owner writes it), room flags that searchers see (`SET_ROOM_DATA` which 2), and member data sent with `CREATE_ROOM` and `JOIN_ROOM` and returned in every member entry. | available | 0.2.0 | [`server/rooms.go`](../server/rooms.go), [`TestRooms`](../server/server_test.go) | External and internal data are upstream work (0.1.0). Room flags and member data are newer, so clients built against 0.1.0 no longer match. |
| `rooms-codes` | **Invite-only rooms**: An 8-character code per room (`SET_ROOM_ACCESS`, `JOIN_BY_CODE`, `GET_ROOM_ACCESS`). The room leaves search and the status page, which counts unlisted rooms. A connection that sends too many wrong codes is closed. | available | 0.2.0 | [`server/codes_test.go`](../server/codes_test.go), [`TestInviteOnlyRoom`](../server/codes_test.go), [`TestCodeGuessesCloseTheConnection`](../server/codes_test.go), [`TestCodeStaysOutOfLogAndStatus`](../server/codes_test.go), [#7](https://github.com/undecuple/re-psn/pull/7) | Protocol 0.4: `HELLO_ACK` flags bit 1 says the server has them. Codes are never logged. |
| `rooms-private` | **Private rooms**: Matching2 room passwords, private slots and join by room ID. A relay-only variant that keeps members' addresses hidden from each other is also wanted. | planned |  | [`ROADMAP.md`](../ROADMAP.md) | The relay-only variant is not designed yet. |

## Leaderboards

| ID | Feature | Status | Since | Evidence | Note |
|---|---|---|---|---|---|
| `scores-boards` | **Leaderboards**: Modelled on sceNpScore: each player's best score is kept, a tie goes to whoever scored first, boards are higher-is-better or lower-is-better, and ranks are read by range or by online ID. | available | 0.1.0 | [`server/scores.go`](../server/scores.go), [`TestScores`](../server/server_test.go) | Upstream work. Kept in memory: a restart empties every board. Opt-in storage is planned (store-sqlite). |

## Players

| ID | Feature | Status | Since | Evidence | Note |
|---|---|---|---|---|---|
| `players-names` | **One player per name**: Names are compared case-insensitively. A HELLO with a name held from another address gets ERROR 8 (name taken); from the same address it is the same player reconnecting, and the old connection is closed. | available | 0.2.0 | [`TestNameTakenFromAnotherAddress`](../server/server_test.go), [`TestSameAddressReconnectTakesOver`](../server/server_test.go) | No passwords: a name is not authenticated. |
| `players-friends` | **Friends, presence and invites**: sceNpBasic: a friends list (everyone on the same server), presence (online, title, room), invites and join-a-friend, player history and a block list. | planned |  | [`ROADMAP.md`](../ROADMAP.md) | Not in the server today. |
| `players-accounts` | **Optional sign-in**: Optional accounts next to the default of a name per player: a credential per server kept on the player's machine, sign-in through a generic OpenID Connect provider, or with Discord or GitHub. | later |  |  | Not started. Needs the opt-in database (store-sqlite) first, and the scope in ROADMAP.md says no accounts or passwords, so that needs changing too. The default would stay a name per player. |

## Limits

| ID | Feature | Status | Since | Evidence | Note |
|---|---|---|---|---|---|
| `limits-requests` | **Request and connection limits**: `-max-rooms`, `-max-player-rooms`, `-max-conns-per-addr`, `-max-req-rate` (requests per second per connection) and a size cap on room data. A request over a limit is refused; a connection over the rate is closed. Each refusal is logged at warn level. | available | 0.2.0 | [`server/limits_test.go`](../server/limits_test.go), [`TestBlobLimit`](../server/limits_test.go), [`TestPlayerRoomLimit`](../server/limits_test.go), [`TestConnLimitPerAddr`](../server/limits_test.go), [`TestRequestRateLimit`](../server/limits_test.go), [`TestLogHelloAndRefusals`](../server/logging_test.go) | TCP only. UDP rate limits are planned (net-guardrails). |

## Logging

| ID | Feature | Status | Since | Evidence | Note |
|---|---|---|---|---|---|
| `logs-levels` | **Log levels and formats**: `-log-level` (debug, info, warn, error), `-v` as debug, `-log-format` text or json, and `-log-file` to append to a file (mode 0600). Every refusal is logged at warn level; tokens and query strings are never logged. | available | 0.2.0 | [`server/logging.go`](../server/logging.go), [`TestLogHelloAndRefusals`](../server/logging_test.go), [`TestLogLevelAndFormat`](../server/logging_test.go), [`TestLogStatusRefusalKeepsTokenOut`](../server/logging_test.go), [#4](https://github.com/undecuple/re-psn/pull/4) | Not checked: no test covers `-log-file`. The server does not rotate the file. |

## Status page

| ID | Feature | Status | Since | Evidence | Note |
|---|---|---|---|---|---|
| `status-page` | **Status page and /api/stats, with a token**: A read-only page and JSON on a separate listener, off unless `-http` names an address (`127.0.0.1:36101` keeps it on localhost). `-http-token` requires a token (`Authorization: Bearer`, or `?token=`), and the server warns at start when the page is reachable from elsewhere without one. | available | 0.2.0 | [`server/api.go`](../server/api.go), [`TestStatusToken`](../server/api_test.go), [`TestReachable`](../server/api_test.go), [`TestStatusPageOff`](../server/statuspage_test.go), [`TestStatusPageOn`](../server/statuspage_test.go), [`docs/status.png`](status.png) | The page and /api/stats are upstream work (0.1.0); the token is the fork's. |

## Metrics

| ID | Feature | Status | Since | Evidence | Note |
|---|---|---|---|---|---|
| `metrics-endpoint` | **Metrics endpoint**: `/metrics` in the Prometheus text format, off by default, with the status page's token rules: connections, rooms, refusals, UDP probes and relay traffic. No names, addresses, room codes or tokens in any label. | later |  |  | Not started. Needs the counters from net-guardrails first. |
| `metrics-dashboard` | **Grafana dashboard and alerts**: Example Prometheus scrape config, alert rules and a Grafana dashboard in `deploy/`, written from the real metric names. | later |  |  | After metrics-endpoint. |

## Reverse proxy

| ID | Feature | Status | Since | Evidence | Note |
|---|---|---|---|---|---|
| `proxy-support` | **PROXY protocol and trusted proxies**: `-trusted-proxies` (CIDRs or addresses) and `-proxy-protocol` (a PROXY v1 or v2 header required on the game TCP port from a listed proxy), and forwarded headers for the status page log. The real client address then counts for the limits, the name rule, `public_ip` and the log. | available | 0.2.0 | [`server/proxy.go`](../server/proxy.go), [`TestReadProxyHeader`](../server/proxy_test.go), [`TestClientAddr`](../server/proxy_test.go), [`TestProxyHeaderFromUntrustedPeerIsJunk`](../server/proxy_test.go), [`TestTrustedPeerWithoutHeaderIsClosed`](../server/proxy_test.go), [`TestLimitUsesRealAddress`](../server/proxy_test.go), [#9](https://github.com/undecuple/re-psn/pull/9), [#12](https://github.com/undecuple/re-psn/pull/12) | Off by default. UDP is not proxied. See docs/running.md. |

## Deployment

| ID | Feature | Status | Since | Evidence | Note |
|---|---|---|---|---|---|
| `deploy-examples` | **Settings file, service unit and Traefik example**: `deploy/` holds `re-psn.env.example` (every setting, commented), a systemd unit and a Traefik config that sends PROXY v2. The release archives carry them as `examples/`. | available | 0.2.0 | [`deploy/re-psn.env.example`](../deploy/re-psn.env.example), [`deploy/re-psn.service`](../deploy/re-psn.service), [`deploy/traefik/dynamic.yml`](../deploy/traefik/dynamic.yml), [`scripts/check-deploy.sh`](../scripts/check-deploy.sh), [`docs/running.md`](running.md), [#10](https://github.com/undecuple/re-psn/pull/10) | `scripts/check-deploy.sh` checks the env file and the unit against the server's flags. The Traefik run (Traefik 3.1 under Podman) is recorded in docs/running.md; no run of the systemd unit is recorded. |
| `deploy-nginx-haproxy` | **nginx and HAProxy examples**: An nginx `stream` block with `proxy_protocol on;` (PROXY v1) and an HAProxy config with `send-proxy-v2`. | not yet verified | 0.2.0 | [`deploy/nginx/psnr.conf`](../deploy/nginx/psnr.conf), [`deploy/haproxy/haproxy.cfg`](../deploy/haproxy/haproxy.cfg) | Not checked: neither config has been run against the server (docs/running.md says so). |
| `deploy-caddy` | **Caddy example**: A Caddy config next to the other examples in `deploy/`. | planned |  |  | Not started. |
| `deploy-container` | **Docker and Podman**: A two-stage Dockerfile and a compose file in `docker/`, configured only by `RE_PSN_*` variables. The status page is off until `RE_PSN_HTTP` is set. | available | 0.2.0 | [`docker/Dockerfile`](../docker/Dockerfile), [`docker/docker-compose.yml`](../docker/docker-compose.yml), [`docker/docker-compose.host.yml`](../docker/docker-compose.host.yml), [`docker/docker-compose.test.yml`](../docker/docker-compose.test.yml), [`docker/README.md`](../docker/README.md), [`lab/run.sh`](../lab/run.sh), [#11](https://github.com/undecuple/re-psn/pull/11) | A Podman 6.1 run is recorded in docker/README.md, and the NAT lab builds the image with Docker in CI. The v0.2.0 image is pushed to ghcr.io (amd64 and arm64, unsigned; arm64 not run). The server runs as a non-root user and the bases are pinned by digest. Opt-in host-networking compose files, one hardened for a shared test host, are for Linux only. Compose under Podman is not tested. |

## Settings

| ID | Feature | Status | Since | Evidence | Note |
|---|---|---|---|---|---|
| `settings-env` | **Flags and RE_PSN_* variables**: Every flag can be set as `RE_PSN_<FLAG>`; the old `PSNR_<FLAG>` names still work as aliases. A flag wins, then `RE_PSN_`, then `PSNR_`. An old name logs a warning that names the variables, never the value. `-tcp-addr` and `-udp-addr` give the two game sockets their own addresses. A bad value stops the server at start. | available | 0.2.0 | [`server/config.go`](../server/config.go), [`TestEnvAlias`](../server/config_test.go), [`TestEnvNoticeHidesValue`](../server/config_test.go), [`TestFlagBeatsEnv`](../server/config_test.go), [`TestEnvBad`](../server/config_test.go), [`TestListenAddrs`](../server/config_test.go), [#8](https://github.com/undecuple/re-psn/pull/8), [#17](https://github.com/undecuple/re-psn/pull/17) | No config file yet. |

## Storage

| ID | Feature | Status | Since | Evidence | Note |
|---|---|---|---|---|---|
| `store-interface` | **Storage interface**: A storage layer with an in-memory backend, the leaderboards wired through it, and a `-db` (`RE_PSN_DB`) setting. Off by default, so nothing changes unless a host turns it on. | available | 0.2.0 | [`server/store/store.go`](../server/store/store.go), [`server/store/storetest/storetest.go`](../server/store/storetest/storetest.go), [`TestMemoryContract`](../server/store/store_test.go), [`TestDBSetting`](../server/storage_test.go), [`TestDBNotLogged`](../server/storage_test.go), [`TestScoreStoreFailure`](../server/storage_test.go), [`TestScoresOutsideRoomLock`](../server/storage_test.go), [#22](https://github.com/undecuple/re-psn/pull/22) | Only `off` and `memory` exist, so scores are still kept in memory and lost on restart; other schemes stop the start. First step of opt-in storage (store-sqlite is next). |
| `store-sqlite` | **SQLite storage (opt-in)**: An SQLite backend with a pure-Go driver, so the release archives stay static, and migrations built into the binary. With it on, scores survive a restart. The connection string is never logged. | planned |  |  | Not started. The server's first third-party Go module. Player names and bans would follow with accounts. |
| `store-backup` | **Backup and restore**: Documented backup and restore for each database: a backup command or a copy procedure for SQLite; a dump command with timer and cron examples for PostgreSQL and MariaDB. How long backups are kept is up to the host. | planned |  |  | Not started. Comes with store-sqlite. |
| `store-postgres` | **PostgreSQL storage**: A PostgreSQL backend for hosts who bring their own database; the server does not run or manage it. | later |  |  | After store-sqlite. |
| `store-mariadb` | **MariaDB storage**: A MariaDB backend for hosts who bring their own database; the server does not run or manage it. | later |  |  | After store-sqlite. |

## High availability

| ID | Feature | Status | Since | Evidence | Note |
|---|---|---|---|---|---|
| `ha-restart` | **Health check and clean restarts**: A `/healthz` endpoint on the status listener (200 when the server accepts connections, no player data), a graceful shutdown that closes connections so players reconnect at once, and a page on restart settings and what players see. | planned |  |  | Not started. The systemd unit and the compose file already restart the server; live rooms are lost on a restart. |
| `ha-failover` | **Active/passive failover**: Docs and example files for a standby server: a floating address (keepalived) or a proxy's health checks, both servers on the same database, one active at a time. No election or replication inside the server. | planned |  |  | Not started. After ha-restart; the shared database needs store-postgres or store-mariadb. Live rooms are lost on failover. |
| `ha-multi` | **Several servers**: Several servers behind one address, each room wholly on one server, with a routing layer that sends a new room or a join to the right one. | later |  |  | On the back burner: revisited after ha-restart and ha-failover exist and one server is not enough. |
| `ha-handover` | **Live room handover**: Rooms that survive the loss of a server. | declined |  |  | Needs replicated live state; out of scope for a server of this size. |

## Network

| ID | Feature | Status | Since | Evidence | Note |
|---|---|---|---|---|---|
| `net-punch` | **Address discovery and hole punching**: A player's P2P socket sends the server a UDP PROBE. The server records the public endpoint and gives each viewer the address that works from where they are: the public one from another network, the LAN address from the same one. | available | 0.2.0 | [`server/udp.go`](../server/udp.go), [`TestProbeRecordsPublicEndpoint`](../server/udp_test.go), [`TestEndpointPerViewer`](../server/udp_test.go), [`lab/README.md`](../lab/README.md) | Upstream work. The NAT lab runs it in Docker in CI. |
| `net-relay` | **Relay**: `-relay`, off by default: streams into a player behind a router go through the server, and datagrams go through the server's UDP port when punching does not get through (as with a symmetric NAT). | available | 0.2.0 | [`server/relay.go`](../server/relay.go), [`TestUDPRelay`](../server/relay_test.go), [`TestStreamRelay`](../server/relay_test.go), [`TestRoutePushOnJoin`](../server/relay_test.go), [`lab/run.sh`](../lab/run.sh) | Upstream work. Uses the server's bandwidth; there is no per-player budget yet (net-guardrails). |
| `net-udp-guard` | **UDP source address check**: PROBE and RELAY packets are accepted only from the IP address of the player's TCP connection. A probe's `local_ip` is kept only if it is private, carrier-grade NAT, link-local or the probe's own source. `-udp-any-source` turns the check off. | available | 0.2.0 | [`TestProbeFromOtherAddressRefused`](../server/udp_test.go), [`TestUDPRelayFromOtherAddress`](../server/relay_test.go), [`TestProbeAnySource`](../server/udp_test.go), [`TestProbeDropsPublicLocalIP`](../server/udp_test.go), [`docker/README.md`](../docker/README.md), [#16](https://github.com/undecuple/re-psn/pull/16) | A Podman run is recorded in docker/README.md. |
| `net-guardrails` | **More UDP and relay guardrails**: Per-source UDP rate limits and a failed-token counter, relay budgets (datagram size, per-player rates, idle streams), a 64-bit token, the limits written down, counters of dropped packets on /api/stats, and a test for each, with a fuzz test for the UDP packets. | planned |  |  | Not started. The 64-bit token changes the wire format and waits for a protocol version bump. |
| `net-checks` | **Connection checks**: A check for players and hosts that says where a connection fails (server reachable, HELLO accepted, UDP probe answered, NAT behaviour, peer reachable, relay available) and shows round trip, loss and whether each peer is up. | planned |  |  | Not designed yet. |
| `net-upnp` | **UPnP**: Open a player's P2P port on their router automatically. | later |  |  | Not started. |

## Client

| ID | Feature | Status | Since | Evidence | Note |
|---|---|---|---|---|---|
| `client-c` | **C client**: One file with no dependencies and no threads: `psnr_send` returns a request id, and `psnr_poll` hands out replies and pushes without blocking. It also builds the UDP probe and relay packets. | available | 0.1.0 | [`client/psnr.c`](../client/psnr.c), [`client/test_psnr.c`](../client/test_psnr.c), [`TestCClient`](../server/client_test.go) | Upstream work. `TestCClient` skips without a C compiler; CI runs it on Linux only. |
| `app-client` | **Go client package**: `app/internal/client`: a Go client for the TCP protocol, for the desktop app. HELLO, heartbeat, create, search, join and leave rooms, record and read scores, and the member and route pushes, with typed messages and errors. `re-psn-app check HOST:PORT` uses it. | partial | main | [`app/internal/client`](../app/internal/client), [`TestConformanceRoomsAndScores`](../app/internal/client/conformance_test.go), [`TestConformanceRelayRoute`](../app/internal/client/conformance_test.go), [`TestConformanceErrors`](../app/internal/client/conformance_test.go) | Room data and messages, kicks, invite-only rooms, relay streams and UDP are not in it yet. |
| `client-desktop` | **Desktop client, terminal UI**: A terminal program for players and hosts: launch profiles per game, a host mode, and a check of the connection. | planned |  |  | Not started. One executable per platform, with nothing else to install. |
| `client-gui` | **Desktop client, graphical UI**: A local web UI served by the same program, and a graphical app on Windows, with the same screens as the terminal UI. | planned |  |  | Not started. After client-desktop. |
| `app-themes` | **Colour themes**: Colour presets in the desktop apps: gruvbox-material by default, then gruvbox, catppuccin, kanagawa and melange. A preset is a data file shared by the apps and the README art. | planned |  |  | Not started. A preset ships only after its project's licence is read and credited in NOTICE.md. The terminal UI falls back to the terminal's own 16 colours and respects NO_COLOR. |
| `app-appearance` | **Appearance mode**: Dark, Light or System (the default): presets carry a dark and a light variant where their project has both, and System follows the OS or the terminal background and switches while the app runs. | planned |  |  | Not started. Whether the status page follows the browser's setting too is not decided. |
| `app-colours` | **Custom colours**: Edit named colour slots (background, surface, text, muted, accent, good, warn, bad) as hex values, with a contrast warning below 4.5:1, import and export, saved per user and per mode. | planned |  |  | Not started. A bad value falls back to the default preset. |

## Testing and CI

| ID | Feature | Status | Since | Evidence | Note |
|---|---|---|---|---|---|
| `test-fuzz` | **Fuzz tests**: Go fuzz targets for the frame parser, the request handlers and the PROXY header parser. Their seed inputs run with every `go test`. | available | 0.2.0 | [`server/fuzz_test.go`](../server/fuzz_test.go), [`FuzzReadPacket`](../server/fuzz_test.go), [`FuzzHandle`](../server/fuzz_test.go), [`FuzzReadProxyHeader`](../server/proxy_test.go), [#2](https://github.com/undecuple/re-psn/pull/2) | CI runs the seeds only; longer fuzzing runs are by hand. The UDP packets have no fuzz target yet. |
| `ci-checks` | **CI checks**: gofmt, go vet, `go test -race` on Linux (with the C client) and `go test` on Windows, govulncheck, gitleaks over the full history, the NAT lab in Docker, and CodeQL for Go and the workflows. Actions are pinned and permissions are read-only by default. | available | 0.2.0 | [`.github/workflows/ci.yml`](../.github/workflows/ci.yml), [`.github/workflows/codeql.yml`](../.github/workflows/codeql.yml), [`.github/dependabot.yml`](../.github/dependabot.yml), [#1](https://github.com/undecuple/re-psn/pull/1), [#6](https://github.com/undecuple/re-psn/pull/6) | Dependabot proposes version updates. |
| `ci-more` | **More CI checks**: Candidates: the C client outside Linux, the race detector on Windows, and a scan of the container image. | planned |  | [`.github/workflows/ci.yml`](../.github/workflows/ci.yml) | The list is not settled. The header of ci.yml says what CI does not check today. |

## Release

| ID | Feature | Status | Since | Evidence | Note |
|---|---|---|---|---|---|
| `release-archives` | **Release archives and checksums**: `scripts/release-build.sh` builds static binaries for windows/amd64, linux/amd64 and linux/arm64 into `re-psn-vX.Y.Z-<os>-<arch>` archives and one SHA256SUMS listing them. A pushed version tag makes a draft pre-release; a second workflow checks a published one. | partial | 0.2.0 | [`scripts/release-build.sh`](../scripts/release-build.sh), [`.github/workflows/release.yml`](../.github/workflows/release.yml), [`.github/workflows/release-check.yml`](../.github/workflows/release-check.yml), [#5](https://github.com/undecuple/re-psn/pull/5) | Published as the v0.2.0 pre-release (2026-11-06): the release workflow and the published-release check both passed. The Windows and arm64 binaries are cross-built and not run. |
| `release-appimage` | **Linux AppImage**: `scripts/build-appimage.sh` packs the linux/amd64 server binary and its examples as `re-psn-vX.Y.Z-x86_64.AppImage`, with appimagetool 1.9.1 and the type-2 runtime 20251108, both downloaded by the release workflow and checked against pinned SHA-256 hashes. A smoke test compares its `-h` and version with the binary's, and it is listed in SHA256SUMS. | partial | 0.2.0 | [`scripts/build-appimage.sh`](../scripts/build-appimage.sh), [`scripts/appimage/AppRun`](../scripts/appimage/AppRun), [`.github/workflows/release.yml`](../.github/workflows/release.yml), [`TestBuildAppImageRefuses`](../server/release_scripts_test.go), [`TestAppImagePinsAgree`](../server/release_scripts_test.go), [`docs/releasing.md`](releasing.md) | Published in the v0.2.0 pre-release (2026-11-06); the release workflow built it on a runner and passed its smoke test. It was also built and run on one x86_64 Linux machine, with FUSE and with `--appimage-extract-and-run`. x86_64 only, unsigned. |
| `release-process` | **Release process**: Written steps for cutting a release (tag, notes from CHANGELOG.md, checking the draft, publishing), a check that the version in the code and in CHANGELOG.md agree, and a dry run of the release workflow before the first tag. | available | 0.2.0 | [`VERSION`](../VERSION), [`scripts/bump-version.sh`](../scripts/bump-version.sh), [`scripts/release-notes.sh`](../scripts/release-notes.sh), [`docs/releasing.md`](releasing.md), [`TestVersionAgrees`](../server/version_test.go) | VERSION, the version check (TestVersionAgrees), scripts/bump-version.sh and scripts/release-notes.sh are built and docs/releasing.md has the steps; the release workflow builds its notes with the notes script and checks that a tag is v + VERSION, and the first tag, v0.2.0 (2026-11-06), ran the whole path: the dry run, the tag step and the draft job passed, and the draft was published by hand after its assets were read back. No fixed release day: while things are worked out, pre-releases (v0.x, marked pre-release) come whenever there is something worth trying. |
| `release-signing` | **Signed releases**: Sign `re-psn.exe` so Windows does not warn about an unsigned program, and sign the checksums. | later |  | [`.github/workflows/release.yml`](../.github/workflows/release.yml) | The exe and SHA256SUMS are unsigned today. |

## Documentation

| ID | Feature | Status | Since | Evidence | Note |
|---|---|---|---|---|---|
| `docs-features` | **Features list**: `docs/features.tsv` holds one row per feature with its status and evidence, and `tools/features.py` generates this page. CI fails when the page is stale, a row is malformed, or evidence is missing. | available | 0.2.0 | [`docs/features.tsv`](features.tsv), [`tools/features.py`](../tools/features.py), [`tools/test_features.py`](../tools/test_features.py), [`.github/workflows/ci.yml`](../.github/workflows/ci.yml), [#19](https://github.com/undecuple/re-psn/pull/19) |  |
| `docs-diagrams` | **Mermaid diagrams**: The architecture, reverse proxy, relay and NAT lab diagrams are Mermaid, which GitHub renders. | available | 0.2.0 | [`docs/architecture.md`](architecture.md), [`lab/README.md`](../lab/README.md), [#13](https://github.com/undecuple/re-psn/pull/13) | No generated images to keep in step. |
| `docs-markdown` | **Docs in Markdown**: Every doc in Markdown with one layout, including the deploy and Docker READMEs with diagrams. | available | 0.2.0 | [`deploy/README.md`](../deploy/README.md), [`docker/README.md`](../docker/README.md), [`docs/running.md`](running.md), [#20](https://github.com/undecuple/re-psn/pull/20) | Links were checked once with a throwaway script; a CI link check is planned (repo-check). |
| `docs-connecting` | **Ways to connect**: A page on how players reach a server: same LAN, port forwarding, a public server with `-relay`, a VPN overlay (Tailscale, ZeroTier) or a reverse proxy, with what each needs, its limits, diagrams and links to the overlays' official documentation. | available | 0.2.0 | [`docs/connecting.md`](connecting.md) | Not tested: re-psn has not been run over Tailscale, ZeroTier or any other overlay. Overlay addressing was imitated with network namespaces only. |
| `docs-readme` | **README rewrite**: A README that says what re-psn is, with a quick start, the available features linking to this page, installation, configuration and credit to upstream. | available | 0.2.0 | [`README.md`](../README.md) | Example output comes from real local runs; the settings table was checked against the server flags. The banner comes from the art generator (docs-art). |
| `docs-art` | **Art generator**: `tools/art/make_art.py` draws the README banner, a social preview image and terminal images in the gruvbox-material palette (`docs/art/palette.json`), with JetBrains Mono fetched from its tagged release and pinned by SHA-256. Terminal images are drawn only from transcripts of real runs (`docs/art/transcripts/`, recorded by `tools/art/capture.sh`). CI renders again and fails when a byte differs. | available | 0.2.0 | [`tools/art/make_art.py`](../tools/art/make_art.py), [`tools/art/test_make_art.py`](../tools/art/test_make_art.py), [`tools/art/art.json`](../tools/art/art.json), [`tools/art/capture.sh`](../tools/art/capture.sh), [`docs/art/palette.json`](art/palette.json), [`docs/img/banner.png`](img/banner.png), [`docs/img/social.png`](img/social.png), [`.github/workflows/ci.yml`](../.github/workflows/ci.yml) | The README shows the banner; the social image is uploaded by hand in the repository settings; its example stays text, and the terminal images are kept for the site. Images are dark only; the light colours are in the palette for later. Fonts are not committed. |
| `docs-website` | **Project website**: A static page built from the features list, the version, the settings table and the changelog by `tools/site.py`: overview, features with status tags, roadmap, settings, install, connect, docs and credits. No script, no third-party loads, and the colours of the README art. `pages.yml` deploys `site/` to GitHub Pages. | partial | main | [`tools/site.py`](../tools/site.py), [`tools/test_site.py`](../tools/test_site.py), [`site/index.html`](../site/index.html), [`site/style.css`](../site/style.css), [`.github/workflows/pages.yml`](../.github/workflows/pages.yml) | Partial: the page is built and checked but not published yet; Pages is enabled; the deploy job waits for the repository variable PAGES_ENABLED. No terminal-style interface yet. |
| `repo-check` | **Repository layout check**: A CI check that the root and top-level folders hold only what is expected, that relative links and anchors in the Markdown files resolve, and that paths named in the workflows exist. | planned |  |  | Not started. After docs-markdown. |

The order of the planned work is in [ROADMAP.md](../ROADMAP.md).
