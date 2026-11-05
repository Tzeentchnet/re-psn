# Security

re-psn (the `psnr` server) handles player data: online IDs, and each player's IP address and P2P
port. It hands that data to the other players in the same room, because they
need it to connect to each other. Anyone who joins your room learns your
address, the same as on real PSN P2P.

- **No authentication.** Anyone who can reach TCP 36100 can join, and can claim
  any online ID. Run it for people you'd let onto your network: a LAN, a
  tailnet, or friends.
- **The status page** (`-http`, `127.0.0.1:36101` by default) lists every
  connected player's online ID. Don't expose it publicly. If you bind it
  beyond localhost, set `-http-token`; the server warns when you don't.
- **The relay** (`-relay`, off by default) carries traffic between players
  in the same room. It can't tell what the traffic is, but it does see how
  much there is and between whom, and it uses the host's bandwidth.
- **Nothing is stored on disk.**

What this fork changes (see `docs/running.md` for the flags):

- Room blobs are capped at 4096 bytes, a player can be in 8 rooms at once, one address can hold 32 connections, and a
  connection that sends more than 200 requests a second is closed (`-max-player-rooms`, `-max-conns-per-addr`,
  `-max-req-rate`; 0 turns a limit off).
- `-http-token` protects the status page and `/api/stats`.

To report a vulnerability, use "Report a vulnerability" under this repository's Security tab (a private advisory).
Please don't use a public issue. A problem that is also in sp00nznet/psnr should be reported there as well.
