# Running the server

## Run it

```
cd server
go build -o psnr .        # psnr.exe on Windows
./psnr
```

```
2026/09/29 02:17:35 psnr 0.1.0: clients on [::]:36100
2026/09/29 02:17:35 status page on http://127.0.0.1:36101/
```

With Docker:

```
docker compose up -d
```

## Flags

| Flag | Default | What it does |
|---|---|---|
| `-addr` | `:36100` | TCP and UDP address for game clients |
| `-tcp-addr`, `-udp-addr` | `-addr` | Give TCP or UDP its own address (a different port, or one interface). Unset, each takes `-addr`. |
| `-http` | `127.0.0.1:36101` | Status page and `/api/stats`. Set it to `""` to turn it off. |
| `-http-token` | none | Require this token on the status page, as `Authorization: Bearer <token>` or `?token=<token>`. Set it whenever `-http` is not on localhost; the server logs a warning if it is not. |
| `-max-rooms` | `1000` | Room limit across all titles |
| `-relay` | off | Carry traffic between players who can't reach each other directly: streams into a player behind a router, and datagrams when a router won't let punched traffic through. Uses this server's bandwidth. |
| `-log-level` | `info` | `debug` (room and relay events), `info` (connects, hellos, disconnects), `warn` (every refusal) or `error`. |
| `-log-format` | `text` | `text` or `json`, one line per event. |
| `-log-file` | none | Also append the log to this file (created with mode 0600). |
| `-v` | off | Same as `-log-level debug`. |

## Environment variables

Every flag can be set as `PSNR_<FLAG>`, upper case, dashes as underscores: `-max-rooms` is `PSNR_MAX_ROOMS`, `-tcp-addr`
is `PSNR_TCP_ADDR`, `-v` is `PSNR_V`. A flag on the command line wins over the variable. A variable that is set but empty
counts as set, so `PSNR_HTTP=` turns the status page off. A number or true/false that does not parse stops the server at
start with a message naming the variable. Booleans take `true` or `false` (also `1`, `0`).

The Docker image has no fixed flags: it is configured only by these variables. The status page is not reachable from the
host until you set `PSNR_HTTP` (for example `0.0.0.0:36101` inside the container) and `PSNR_HTTP_TOKEN`; the compose file
shows the lines. A token set this way is visible to `docker inspect` and to other users on the host (process list, for a
native run), so treat the token as protecting a status page on a trusted machine, not as a secret that survives a shared host.
If you change a port, change it in the compose `ports` list as well, and in your firewall rule.

## Logging

The log goes to standard error, and to `-log-file` if set. It holds online IDs and client addresses, because a refusal
is about them; it never holds the status page token, the UDP probe token or a request query string. What is logged:

| Level | Events |
|---|---|
| info | server start, each HELLO (client, title, online ID, address), a reconnect that replaces a connection, each disconnect, a failed read |
| warn | a connection refused (too many from one address), a connection closed for its request rate, a HELLO refused (name taken), a room request refused (room limit, blob over limit), an unknown request type, a status page request refused (path only) |
| debug | connects, requests before HELLO, room created, joined and closed, relay streams, UDP probe results |

Rotate the file with logrotate or your service manager (`copytruncate`, or restart after moving it); the server does not rotate it.

## Ports

| Port | Proto | Open to |
|---|---|---|
| 36100 | TCP and UDP | every player (the only port the server needs reachable). UDP carries the players' address probes. |
| 36101 | TCP | you. It has no auth and lists every player's online ID. Keep it on localhost or a LAN, not the internet. |
| each player's P2P port (3658 unless `PS3_NET_P2P_PORT` says otherwise) | UDP and TCP | the other players. The game opens this port itself; titles use both (Simpsons Arcade sends its game setup over TCP). |

Players on the same LAN or tailnet need nothing forwarded. Across the internet,
the server's machine must accept TCP and UDP 36100, and each player must accept the
other players' traffic on their P2P port. The server passes out addresses and
relays nothing.

## Firewall

**The server (TCP and UDP 36100).** On Windows, `Setup.cmd` offers to add the
rules. By hand, from an admin terminal:

```
netsh advfirewall firewall add rule name=psnr dir=in action=allow protocol=TCP localport=36100 program="C:\path\to\psnr\server\psnr.exe"
netsh advfirewall firewall add rule name=psnr dir=in action=allow protocol=UDP localport=36100 program="C:\path\to\psnr\server\psnr.exe"
```

On Linux, `setup.sh` prints the commands if a firewall is on:

```
sudo ufw allow 36100                                        # ufw: TCP and UDP
sudo firewall-cmd --permanent --add-port=36100/tcp --add-port=36100/udp && sudo firewall-cmd --reload   # firewalld
```

**Each player (the P2P port, UDP and TCP).** On Windows, the first time the
game goes online Windows asks whether to allow it; allow it on the network
type you play on (private, or public too). Or add the rule ahead of time:

```
netsh advfirewall firewall add rule name="psnr game" dir=in action=allow program="C:\path\to\game.exe"
```

On Linux: `sudo ufw allow 3658` (both protocols).

A tailnet needs neither rule for players on it.

## State, backup, upgrade

- **State:** everything lives in memory. Restarting the server closes every room
  and empties every leaderboard.
- **Backup:** there is nothing to back up.
- **Upgrade:** stop the server, rebuild it, start it again. Titles reconnect when
  they next go online.
