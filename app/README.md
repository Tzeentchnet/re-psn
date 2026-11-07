# re-psn desktop app

The start of the desktop app for players and hosts of a re-psn server. It is a Go module of its own,
`github.com/undecuple/re-psn/app`, next to the server; the server's module is not changed by it and stays standard-library only.
The app has a terminal UI built on [Bubble Tea](https://github.com/charmbracelet/bubbletea); its dependencies are listed under
[Dependencies](#dependencies).

## Contents

- [What is here](#what-is-here)
- [Build and test](#build-and-test)
- [The terminal UI](#the-terminal-ui)
- [Starting from the desktop](#starting-from-the-desktop)
- [The check command](#the-check-command)
- [Settings file](#settings-file)
- [Dependencies](#dependencies)
- [What the client does not do yet](#what-the-client-does-not-do-yet)

## What is here

| Path | What is there |
|---|---|
| `internal/client/` | A Go client for the server's TCP protocol ([docs/api.md](../docs/api.md)), with no UI |
| `internal/client/conformance_test.go` | Builds the real server from `../server`, starts it on free loopback ports and drives the client against it |
| `internal/config/` | Reads and writes the per-user settings file, `re-psn.ini` |
| `internal/tui/` | The terminal UI: a Bubble Tea model whose Update and View are tested with messages and a fake server, no terminal |
| `cmd/re-psn-app/` | The program: the terminal UI with no arguments, `check` and `version` |

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

## The terminal UI

```sh
re-psn-app            # start the terminal UI
re-psn-app --no-tui   # do not: print a note and exit 2
```

The UI needs a terminal on stdin and stdout. On a Linux desktop it can open an installed terminal when started without
one, for example by a file manager. See [Starting from the desktop](#starting-from-the-desktop).

**First start.** With no settings file yet, the app asks what it is for on this machine: **Play** (join servers), **Host** (run a
server; the Host screen comes later, Status works now) or **Both**. Then the online ID (not for Host) and the server as
`host:port`, each checked with the settings file's own rules. Enter saves `re-psn.ini`; Esc quits without saving, as does `q` on
the first question; backspace in an empty field goes back one step.

**Home.** A menu (Overview, Rooms, Leaderboards, Settings, Help), the chosen item's content beside it, and a line of keys at the
bottom. Under 70 columns the menu and the content are shown one at a time.

- **Overview:** mode, name, server and the connection in words: connected or not, the HELLO round trip, relay and invite-only
  rooms on or off, the address the server sees.
- **Rooms** and **Leaderboards** are the two tabs of the Status screen. Rooms belong to one title, so the screen asks for the
  title's communication ID (such as `NPWR00860_00`) first. Rooms shows ID, owner, members as current/max and the room flags.
  Leaderboards shows one board, whose id you type: rank, player, score and when it was recorded.
- **Settings:** mode, name, server, status URL and theme. Enter changes the selected row; `s` saves to `re-psn.ini` (mode 0600).
  A refused value says which rule it broke. The status URL is only stored; nothing uses it yet.
- **Help:** the keys and what the screens can show.

| Key | What it does |
|---|---|
| ↑ / ↓, `k` / `j`, `1`-`5` | move in the menu or a list; the digits jump to a menu item |
| Enter | open the item; change a setting; finish typing |
| Esc | back to the menu; cancel typing |
| Tab | switch between Rooms and Leaderboards |
| `e`, `b` | type the title, the board id |
| `r` | ask the server again |
| `s` | save the settings |
| `t` | dark or light theme (saved with the settings) |
| `q`, Ctrl+C | quit (Ctrl+C also while typing) |

**Where the data comes from.** Only the game protocol, through `internal/client`: each refresh connects to the configured server,
says HELLO under a throwaway name (`view-` and 10 random hex digits) for the title on screen, sends SEARCH_ROOMS (up to 50 rooms)
and, with a board id, GET_RANKING (the top 20), and closes the connection. Every refresh has a 5-second timeout. A failure is
shown as text on the screen, with the error, and the app keeps running. The HTTP status page is never used.

**What the protocol cannot show.** The server's status page shows some things the game protocol has no message for. The UI says
"not available over the protocol" in those places:

- the server's version, uptime and connection counts (Overview and the Status header);
- a room's age (the Age column of Rooms);
- which boards exist, for one title or across titles (so you type the title and a board id);
- full rooms and invite-only rooms, which SEARCH_ROOMS leaves out.

**Theme.** Dark by default, light with `t`; the colours are the gruvbox-material values the website uses
([docs/art/palette.json](../docs/art/palette.json)). In light mode the accents that are too faint as text are replaced by the
strong text colour in bold. No state is shown by colour alone. With `NO_COLOR` set the UI uses no colour, only bold and reverse.

## Starting from the desktop

The same Go terminal app builds for Windows and Linux. Windows uses the normal console-subsystem executable: it can
open a console when double-clicked, following the Windows console configuration. A physical Windows terminal has not
been tested locally; CI tests and cross-builds provide separate evidence.

On Linux, a start with no terminal and with `DISPLAY` or `WAYLAND_DISPLAY` set opens a supported terminal. It tries
`TERMINAL` first, then `x-terminal-emulator`, `kitty` and `xterm`. `TERMINAL` is one executable name or path, including a
path containing spaces; command strings, shell expansion and extra options are not parsed. A custom terminal must
support the xterm-style `-e PROGRAM ARG...` convention. Kitty is invoked with its own command syntax and no startup session.

Arguments and the environment are passed directly to the terminal. An internal recursion guard stops a second relaunch
if that terminal fails to provide a TTY. The app waits for the selected terminal and does not launch another after a
runtime error. Without a display or supported terminal, or on an error, it prints one line and exits 2. Starting
`check`, `version`, `--help` or `--no-tui` never opens a terminal. No server or browser is started by this behavior.

The repository provides a Linux desktop entry at `share/applications/re-psn.desktop` with `Terminal=true`, plus
`share/icons/hicolor/scalable/apps/re-psn.svg`, copied from the site's favicon artwork. These are installation inputs;
they do not install the executable, register a desktop entry automatically or add release assets. Install the app on
the graphical session's PATH before installing the entry, and install the SVG under the matching icon name `re-psn`.

Tests exercise selection, literal arguments, failure/recursion paths and a synthetic terminal/child process. These are
not a physical desktop, Steam Deck game-mode or emulator compatibility run. macOS still requires an existing terminal.

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
theme = dark

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
| `[app]` | `theme` | `dark` or `light`; unset means dark |
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

## Dependencies

The server module has none. The app module has these, pinned in `go.mod` and `go.sum` (`go mod verify` checks them, CI builds
with `-mod=readonly`, and `govulncheck` runs on the module):

| Module | Version | Why | Licence |
|---|---|---|---|
| [charm.land/bubbletea/v2](https://github.com/charmbracelet/bubbletea) | v2.1.1 | the terminal UI's event loop and renderer | MIT |
| [charm.land/lipgloss/v2](https://github.com/charmbracelet/lipgloss) | v2.0.6 | colours, frames and layout | MIT |
| [github.com/charmbracelet/x/term](https://github.com/charmbracelet/x) | v0.2.2 | the "is this a terminal" check (already needed by the two above) | MIT |

The program also links the modules those three need: `github.com/charmbracelet/colorprofile`, `ultraviolet`, `x/ansi`, `x/termios`
and `x/windows` (MIT), `github.com/clipperhouse/displaywidth` and `uax29/v2` (MIT), `github.com/lucasb-eyer/go-colorful` (MIT),
`github.com/mattn/go-runewidth` (MIT), `github.com/muesli/cancelreader` (MIT), `github.com/rivo/uniseg` (MIT),
`github.com/xo/terminfo` (MIT), and `golang.org/x/sync` and `golang.org/x/sys` (BSD-3-Clause). The module graph (`go list -m all`)
also has modules that only their tests or their own dependencies' tests use (`aymanbagabas/go-udiff`, `bits-and-blooms/bitset`,
`charmbracelet/x/exp/golden`, `clipperhouse/stringish`, `golang.org/x/exp`); they are not built into the program. None needs cgo. Bubbles is not used.

## What the client does not do yet

The client implements HELLO, HEARTBEAT, CREATE_ROOM, SEARCH_ROOMS, JOIN_ROOM, LEAVE_ROOM, RECORD_SCORE, GET_RANKING and
GET_RANKING_BY_ID, and decodes the MEMBER_JOINED, MEMBER_LEFT and ROUTE pushes; other pushes arrive undecoded. Room data and
messages, kicks, invite-only rooms, relay streams, UDP probes and the relay's datagrams are not in it yet. The protocol has no
message for the server's version, uptime or connection counts, so the app cannot show them without the status page.
