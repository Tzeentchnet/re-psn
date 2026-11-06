# re-psn desktop app

The start of the desktop app for players and hosts of a re-psn server. It is a Go module of its own,
`github.com/undecuple/re-psn/app`, next to the server; the server's module is not changed by it. It has no dependencies outside
the Go standard library yet. There is no terminal UI yet; that comes next.

## Contents

- [What is here](#what-is-here)
- [Build and test](#build-and-test)
- [The check command](#the-check-command)
- [Settings file](#settings-file)
- [What the client does not do yet](#what-the-client-does-not-do-yet)

## What is here

| Path | What is there |
|---|---|
| `internal/client/` | A Go client for the server's TCP protocol ([docs/api.md](../docs/api.md)), with no UI |
| `internal/client/conformance_test.go` | Builds the real server from `../server`, starts it on free loopback ports and drives the client against it |
| `internal/config/` | Reads and writes the per-user settings file, `re-psn.ini` |
| `cmd/re-psn-app/` | The program: `check` and `version` for now |

The client speaks only the game protocol. It does not need the server's HTTP status page.

## Build and test

Go 1.26 or newer:

```sh
cd app
go vet ./...
go test -race ./...
go build ./cmd/re-psn-app
```

The conformance test needs the `go` command to build the server; it skips, saying why, when the server cannot be built. It runs
the server with `-http ""`, so no status page is started.

## The check command

```sh
re-psn-app check [-timeout 5s] HOST:PORT
re-psn-app version
```

`check` connects, says HELLO under a throwaway name (`check-` and 10 random hex digits) for a made-up title, `RE-PSN-CHECK`, so it
never sees a game's rooms, and times one more request. It prints what the server answered and `OK`, or `FAIL` and what failed, and
exits 0 or 1:

```text
connect   127.0.0.1:47391 in 0.3 ms
hello     accepted as check-b9656030ba in 0.7 ms; the server sees this machine as 127.0.0.1
features  relay on, invite-only rooms on (flags 0x03)
version   not reported: the protocol has no version message
rtt       0.2 ms (one SEARCH_ROOMS request)
OK
```

## Settings file

`re-psn.ini` lives in `$XDG_CONFIG_HOME/re-psn/` when that variable holds an absolute path, else in `~/.config/re-psn/`; on Windows
in `%AppData%\re-psn\`. It is written atomically (a temporary file, then a rename) with mode 0600.

```ini
# Comments are whole lines starting with # or ;
[app]
mode = play

[player]
name = alice_01

[server]
server = play.example.org:36100
status_url = https://play.example.org/status
status_token_file = /etc/re-psn/status-token
```

| Section | Key | Value |
|---|---|---|
| `[app]` | `mode` | `play`, `host` or `both` |
| `[player]` | `name` | the online ID: 3 to 16 letters, digits, `-` or `_` |
| `[server]` | `server` | `host:port`, port 1 to 65535 |
| `[server]` | `status_url` | an `http://` or `https://` URL |
| `[server]` | `status_token_file` | a path, used exactly as written |

The rules:

- `key = value` lines go under a `[section]` line; spaces around the key and the value are dropped.
- A `#` or `;` after a value is part of the value, not a comment.
- An empty value means the key is unset. A known key set twice is an error.
- There are no includes, quotes, escapes or variables (`~` and `$HOME` are not expanded).
- Other keys, other sections and comment lines are kept and written back as they were.
- An invalid value is an error that names the section and key, such as `re-psn.ini: [app] mode: "watch": want play, host or both`.

## What the client does not do yet

The client implements HELLO, HEARTBEAT, CREATE_ROOM, SEARCH_ROOMS, JOIN_ROOM, LEAVE_ROOM, RECORD_SCORE, GET_RANKING and
GET_RANKING_BY_ID, and decodes the MEMBER_JOINED, MEMBER_LEFT and ROUTE pushes; other pushes arrive undecoded. Room data and
messages, kicks, invite-only rooms, relay streams, UDP probes and the relay's datagrams are not in it yet. The protocol has no
message for the server's version, uptime or connection counts, so the app cannot show them without the status page.
