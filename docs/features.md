<!-- Generated from docs/features.tsv; do not edit by hand. -->

# Features

What re-psn does today and what is planned, one row per feature. Each status rests on the evidence next to it: a test in this repository, a recorded run in the docs, or the pull request that added it. *Not yet verified* means the code or config is in the repository but nothing here checks it. *Since* is the version that first had the feature; *main* means merged but not in a release yet.

| Status | Meaning | Features |
|---|---|---|
| available | merged, with a test or a recorded run | 18 |
| not yet verified | merged, but nothing in the repository checks it | 1 |
| partial | merged in part, or with known gaps | 1 |
| in progress | an open pull request or branch | 0 |
| planned | wanted and intended, not started | 8 |
| later | after the planned items, or after a decision | 2 |
| declined | will not be done | 0 |
| all | | 30 |

## Rooms and matchmaking

| ID | Feature | Status | Since | Evidence | Note |
|---|---|---|---|---|---|
| `rooms-basic` | **Rooms and search**: Create, search, join and leave rooms modelled on sceNpMatching2. A search sees only the rooms of its own title, with each room's external data. Room messages, owner handoff when the owner leaves, and member pushes with each peer's address and P2P port. | available | 0.1.0 | [`server/rooms.go`](../server/rooms.go), [`TestRooms`](../server/server_test.go) | Upstream work. Kick (`KICK_MEMBER`) is in the server but no test covers it. |
| `rooms-data` | **Room and member data**: External and internal room data (only the owner writes it), room flags that searchers see (`SET_ROOM_DATA` which 2), and member data sent with `CREATE_ROOM` and `JOIN_ROOM` and returned in every member entry. | available | main | [`server/rooms.go`](../server/rooms.go), [`TestRooms`](../server/server_test.go) | External and internal data are upstream work (0.1.0). Room flags and member data are newer, so clients built against 0.1.0 no longer match. |
| `rooms-codes` | **Invite-only rooms**: An 8-character code per room (`SET_ROOM_ACCESS`, `JOIN_BY_CODE`, `GET_ROOM_ACCESS`). The room leaves search and the status page, which counts unlisted rooms. A connection that sends too many wrong codes is closed. | available | main | [`server/codes_test.go`](../server/codes_test.go), [`TestInviteOnlyRoom`](../server/codes_test.go), [`TestCodeGuessesCloseTheConnection`](../server/codes_test.go), [`TestCodeStaysOutOfLogAndStatus`](../server/codes_test.go), [#7](https://github.com/undecuple/re-psn/pull/7) | Protocol 0.4: `HELLO_ACK` flags bit 1 says the server has them. Codes are never logged. |
| `rooms-private` | **Private rooms**: Matching2 room passwords, private slots and join by room ID. A relay-only variant that keeps members' addresses hidden from each other is also wanted. | planned |  | [`ROADMAP.md`](../ROADMAP.md) | Listed in ROADMAP.md under friends and invites; the relay-only variant is not in it yet. |

## Leaderboards

| ID | Feature | Status | Since | Evidence | Note |
|---|---|---|---|---|---|
| `scores-boards` | **Leaderboards**: Modelled on sceNpScore: each player's best score is kept, a tie goes to whoever scored first, boards are higher-is-better or lower-is-better, and ranks are read by range or by online ID. | available | 0.1.0 | [`server/scores.go`](../server/scores.go), [`TestScores`](../server/server_test.go) | Upstream work. Kept in memory: a restart empties every board. |

## Players

| ID | Feature | Status | Since | Evidence | Note |
|---|---|---|---|---|---|
| `players-names` | **One player per name**: Names are compared case-insensitively. A HELLO with a name held from another address gets ERROR 8 (name taken); from the same address it is the same player reconnecting, and the old connection is closed. | available | main | [`TestNameTakenFromAnotherAddress`](../server/server_test.go), [`TestSameAddressReconnectTakesOver`](../server/server_test.go) | No passwords: a name is not authenticated. |
| `players-friends` | **Friends, presence and invites**: sceNpBasic: a friends list (everyone on the same server), presence (online, title, room), invites and join-a-friend, player history and a block list. | planned |  | [`ROADMAP.md`](../ROADMAP.md) | Not in the server today. |
| `players-accounts` | **Optional sign-in**: Optional accounts next to the default of a name per player: sign-in through a generic OpenID Connect provider, or with Discord or GitHub. | later |  |  | Not started. The scope in ROADMAP.md says no accounts or passwords, so this needs that decision changed first. |

## Limits

| ID | Feature | Status | Since | Evidence | Note |
|---|---|---|---|---|---|
| `limits-requests` | **Request and connection limits**: `-max-rooms`, `-max-player-rooms`, `-max-conns-per-addr`, `-max-req-rate` (requests per second per connection) and a size cap on room data. A request over a limit is refused; a connection over the rate is closed. Each refusal is logged at warn level. | available | main | [`server/limits_test.go`](../server/limits_test.go), [`TestBlobLimit`](../server/limits_test.go), [`TestPlayerRoomLimit`](../server/limits_test.go), [`TestConnLimitPerAddr`](../server/limits_test.go), [`TestRequestRateLimit`](../server/limits_test.go), [`TestLogHelloAndRefusals`](../server/logging_test.go) | TCP only. UDP rate limits are planned (net-guardrails). |

## Logging

| ID | Feature | Status | Since | Evidence | Note |
|---|---|---|---|---|---|
| `logs-levels` | **Log levels and formats**: `-log-level` (debug, info, warn, error), `-v` as debug, `-log-format` text or json, and `-log-file` to append to a file (mode 0600). Every refusal is logged at warn level; tokens and query strings are never logged. | available | main | [`server/logging.go`](../server/logging.go), [`TestLogHelloAndRefusals`](../server/logging_test.go), [`TestLogLevelAndFormat`](../server/logging_test.go), [`TestLogStatusRefusalKeepsTokenOut`](../server/logging_test.go), [#4](https://github.com/undecuple/re-psn/pull/4) | Not checked: no test covers `-log-file`. The server does not rotate the file. |

## Status page

| ID | Feature | Status | Since | Evidence | Note |
|---|---|---|---|---|---|
| `status-page` | **Status page and /api/stats, with a token**: A read-only page and JSON on a separate listener, on localhost by default. `-http-token` requires a token (`Authorization: Bearer`, or `?token=`), and the server warns at start when the page is reachable from elsewhere without one. | available | main | [`server/api.go`](../server/api.go), [`TestStatusToken`](../server/api_test.go), [`TestReachable`](../server/api_test.go), [`docs/status.png`](status.png) | The page and /api/stats are upstream work (0.1.0); the token is the fork's. |

## Reverse proxy

| ID | Feature | Status | Since | Evidence | Note |
|---|---|---|---|---|---|
| `proxy-support` | **PROXY protocol and trusted proxies**: `-trusted-proxies` (CIDRs or addresses) and `-proxy-protocol` (a PROXY v1 or v2 header required on the game TCP port from a listed proxy), and forwarded headers for the status page log. The real client address then counts for the limits, the name rule, `public_ip` and the log. | available | main | [`server/proxy.go`](../server/proxy.go), [`TestReadProxyHeader`](../server/proxy_test.go), [`TestClientAddr`](../server/proxy_test.go), [`TestProxyHeaderFromUntrustedPeerIsJunk`](../server/proxy_test.go), [`TestTrustedPeerWithoutHeaderIsClosed`](../server/proxy_test.go), [`TestLimitUsesRealAddress`](../server/proxy_test.go), [#9](https://github.com/undecuple/re-psn/pull/9), [#12](https://github.com/undecuple/re-psn/pull/12) | Off by default. UDP is not proxied. See docs/running.md. |

## Deployment

| ID | Feature | Status | Since | Evidence | Note |
|---|---|---|---|---|---|
| `deploy-examples` | **Settings file, service unit and Traefik example**: `deploy/` holds `re-psn.env.example` (every setting, commented), a systemd unit and a Traefik config that sends PROXY v2. The release archives carry them as `examples/`. | available | main | [`deploy/re-psn.env.example`](../deploy/re-psn.env.example), [`deploy/re-psn.service`](../deploy/re-psn.service), [`deploy/traefik/dynamic.yml`](../deploy/traefik/dynamic.yml), [`scripts/check-deploy.sh`](../scripts/check-deploy.sh), [`docs/running.md`](running.md), [#10](https://github.com/undecuple/re-psn/pull/10) | `scripts/check-deploy.sh` checks the env file and the unit against the server's flags. The Traefik run (Traefik 3.1 under Podman) is recorded in docs/running.md; no run of the systemd unit is recorded. |
| `deploy-nginx-haproxy` | **nginx and HAProxy examples**: An nginx `stream` block with `proxy_protocol on;` (PROXY v1) and an HAProxy config with `send-proxy-v2`. | not yet verified | main | [`deploy/nginx/psnr.conf`](../deploy/nginx/psnr.conf), [`deploy/haproxy/haproxy.cfg`](../deploy/haproxy/haproxy.cfg) | Not checked: neither config has been run against the server (docs/running.md says so). |
| `deploy-caddy` | **Caddy example**: A Caddy config next to the other examples in `deploy/`. | planned |  |  | Not started. |
| `deploy-container` | **Docker and Podman**: A two-stage Dockerfile and a compose file in `docker/`, configured only by `RE_PSN_*` variables. The status page stays on localhost inside the container until a token is set. | available | main | [`docker/Dockerfile`](../docker/Dockerfile), [`docker/docker-compose.yml`](../docker/docker-compose.yml), [`docker/README.md`](../docker/README.md), [`lab/run.sh`](../lab/run.sh), [#11](https://github.com/undecuple/re-psn/pull/11) | A Podman 6.1 run is recorded in docker/README.md, and the NAT lab builds the image with Docker in CI. No image is published. Compose under Podman is not tested. |

## Settings

| ID | Feature | Status | Since | Evidence | Note |
|---|---|---|---|---|---|
| `settings-env` | **Flags and RE_PSN_* variables**: Every flag can be set as `RE_PSN_<FLAG>`; the old `PSNR_<FLAG>` names still work as aliases. A flag wins, then `RE_PSN_`, then `PSNR_`. An old name logs a warning that names the variables, never the value. `-tcp-addr` and `-udp-addr` give the two game sockets their own addresses. A bad value stops the server at start. | available | main | [`server/config.go`](../server/config.go), [`TestEnvAlias`](../server/config_test.go), [`TestEnvNoticeHidesValue`](../server/config_test.go), [`TestFlagBeatsEnv`](../server/config_test.go), [`TestEnvBad`](../server/config_test.go), [`TestListenAddrs`](../server/config_test.go), [#8](https://github.com/undecuple/re-psn/pull/8), [#17](https://github.com/undecuple/re-psn/pull/17) | No config file yet. |

## Network

| ID | Feature | Status | Since | Evidence | Note |
|---|---|---|---|---|---|
| `net-punch` | **Address discovery and hole punching**: A player's P2P socket sends the server a UDP PROBE. The server records the public endpoint and gives each viewer the address that works from where they are: the public one from another network, the LAN address from the same one. | available | main | [`server/udp.go`](../server/udp.go), [`TestProbeRecordsPublicEndpoint`](../server/udp_test.go), [`TestEndpointPerViewer`](../server/udp_test.go), [`lab/README.md`](../lab/README.md) | Upstream work. The NAT lab runs it in Docker in CI. |
| `net-relay` | **Relay**: `-relay`, off by default: streams into a player behind a router go through the server, and datagrams go through the server's UDP port when punching does not get through (as with a symmetric NAT). | available | main | [`server/relay.go`](../server/relay.go), [`TestUDPRelay`](../server/relay_test.go), [`TestStreamRelay`](../server/relay_test.go), [`TestRoutePushOnJoin`](../server/relay_test.go), [`lab/run.sh`](../lab/run.sh) | Upstream work. Uses the server's bandwidth; there is no per-player budget yet (net-guardrails). |
| `net-udp-guard` | **UDP source address check**: PROBE and RELAY packets are accepted only from the IP address of the player's TCP connection. A probe's `local_ip` is kept only if it is private, carrier-grade NAT, link-local or the probe's own source. `-udp-any-source` turns the check off. | available | main | [`TestProbeFromOtherAddressRefused`](../server/udp_test.go), [`TestUDPRelayFromOtherAddress`](../server/relay_test.go), [`TestProbeAnySource`](../server/udp_test.go), [`TestProbeDropsPublicLocalIP`](../server/udp_test.go), [`docker/README.md`](../docker/README.md), [#16](https://github.com/undecuple/re-psn/pull/16) | A Podman run is recorded in docker/README.md. |
| `net-guardrails` | **More UDP and relay guardrails**: Per-source UDP rate limits and a failed-token counter, relay budgets (datagram size, per-player rates, idle streams), a 64-bit token, the limits written down, counters of dropped packets on /api/stats, and a test for each, with a fuzz test for the UDP packets. | planned |  |  | Not started. The 64-bit token changes the wire format and waits for a protocol version bump. |
| `net-checks` | **Connection checks**: A check for players and hosts that says where a connection fails (server reachable, HELLO accepted, UDP probe answered, NAT behaviour, peer reachable, relay available) and shows round trip, loss and whether each peer is up. | planned |  |  | Not designed yet. |

## Client

| ID | Feature | Status | Since | Evidence | Note |
|---|---|---|---|---|---|
| `client-c` | **C client**: One file with no dependencies and no threads: `psnr_send` returns a request id, and `psnr_poll` hands out replies and pushes without blocking. It also builds the UDP probe and relay packets. | available | 0.1.0 | [`client/psnr.c`](../client/psnr.c), [`client/test_psnr.c`](../client/test_psnr.c), [`TestCClient`](../server/client_test.go) | Upstream work. `TestCClient` skips without a C compiler; CI runs it on Linux only. |
| `client-desktop` | **Desktop client**: A program for players and hosts: a terminal UI first, then a web UI served by the same binary, launch profiles per game, and a host mode. | planned |  |  | Not started. |

## Testing and CI

| ID | Feature | Status | Since | Evidence | Note |
|---|---|---|---|---|---|
| `test-fuzz` | **Fuzz tests**: Go fuzz targets for the frame parser, the request handlers and the PROXY header parser. Their seed inputs run with every `go test`. | available | main | [`server/fuzz_test.go`](../server/fuzz_test.go), [`FuzzReadPacket`](../server/fuzz_test.go), [`FuzzHandle`](../server/fuzz_test.go), [`FuzzReadProxyHeader`](../server/proxy_test.go), [#2](https://github.com/undecuple/re-psn/pull/2) | CI runs the seeds only; longer fuzzing runs are by hand. The UDP packets have no fuzz target yet. |
| `ci-checks` | **CI checks**: gofmt, go vet, `go test -race` on Linux (with the C client) and `go test` on Windows, govulncheck, gitleaks over the full history, the NAT lab in Docker, and CodeQL for Go and the workflows. Actions are pinned and permissions are read-only by default. | available | main | [`.github/workflows/ci.yml`](../.github/workflows/ci.yml), [`.github/workflows/codeql.yml`](../.github/workflows/codeql.yml), [`.github/dependabot.yml`](../.github/dependabot.yml), [#1](https://github.com/undecuple/re-psn/pull/1), [#6](https://github.com/undecuple/re-psn/pull/6) | Dependabot proposes version updates. |
| `ci-more` | **More CI checks**: Candidates: the C client outside Linux, the race detector on Windows, and a scan of the container image. | planned |  | [`.github/workflows/ci.yml`](../.github/workflows/ci.yml) | The list is not settled. The header of ci.yml says what CI does not check today. |

## Release

| ID | Feature | Status | Since | Evidence | Note |
|---|---|---|---|---|---|
| `release-archives` | **Release archives and checksums**: `scripts/release-build.sh` builds static binaries for windows/amd64, linux/amd64 and linux/arm64 into `re-psn-vX.Y.Z-<os>-<arch>` archives with a .sha256 file each and SHA256SUMS. A pushed version tag makes a draft pre-release; a second workflow checks a published one. | partial | main | [`scripts/release-build.sh`](../scripts/release-build.sh), [`.github/workflows/release.yml`](../.github/workflows/release.yml), [`.github/workflows/release-check.yml`](../.github/workflows/release-check.yml), [#5](https://github.com/undecuple/re-psn/pull/5) | Not released yet: there is no tag, so nothing has been published. The Windows and arm64 binaries are cross-built and not run. |
| `release-process` | **Release process**: Written steps for cutting a release (tag, notes, checking the draft, publishing) and signed checksums. | planned |  |  | Not written yet. |
| `release-signing` | **Windows code signing**: Sign `re-psn.exe` so Windows does not warn about an unsigned program. | later |  | [`.github/workflows/release.yml`](../.github/workflows/release.yml) | The exe is unsigned today. |

The order of the planned work is in [ROADMAP.md](../ROADMAP.md).
