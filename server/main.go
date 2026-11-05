// psnr: a stand-in for PSN's matchmaking and leaderboard services, for
// statically recompiled PS3 titles. The runtime's sceNp modules talk to it
// over the protocol in docs/api.md; peers then talk to each other directly.
// See docs/architecture.md.
package main

import (
	"flag"
	"io"
	"log"
	"log/slog"
	"net"
	"net/http"
	"os"
)

var version = "0.1.0"

func main() {
	addr := flag.String("addr", ":36100", "TCP and UDP address for game clients")
	httpAddr := flag.String("http", "127.0.0.1:36101", "HTTP status page and /api/stats (empty to disable)")
	httpToken := flag.String("http-token", "", "require this token (Authorization: Bearer, or ?token=) on the status page; set it when -http is not on localhost")
	maxRooms := flag.Int("max-rooms", 1000, "room limit across all titles")
	playerRooms := flag.Int("max-player-rooms", 8, "rooms one player may be in at once (0 for no limit)")
	perAddr := flag.Int("max-conns-per-addr", 32, "open connections per remote address (0 for no limit)")
	reqRate := flag.Int("max-req-rate", 200, "requests per second per connection before it is closed (0 for no limit)")
	verbose := flag.Bool("v", false, "same as -log-level debug")
	logLevel := flag.String("log-level", "info", "log level: debug (room events), info, warn or error")
	logFormat := flag.String("log-format", "text", "log format: text or json")
	logFile := flag.String("log-file", "", "also append the log to this file (mode 0600)")
	relay := flag.Bool("relay", false, "carry traffic between players who can't reach each other directly (uses this server's bandwidth)")
	flag.Parse()

	level, format, err := parseLogFlags(*logLevel, *logFormat)
	if err != nil {
		log.Fatal(err)
	}
	if *verbose {
		level = slog.LevelDebug
	}
	var out io.Writer = os.Stderr
	if *logFile != "" {
		f, err := os.OpenFile(*logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
		if err != nil {
			log.Fatal(err)
		}
		out = io.MultiWriter(os.Stderr, f)
	}
	logger := newLogger(out, level, format)
	fatal := func(err error) {
		logger.Error("fatal", "err", err)
		os.Exit(1)
	}

	s := NewServer(*maxRooms, *verbose)
	s.log = logger
	s.relay = *relay
	s.maxPlayerRooms, s.maxPerAddr, s.maxReqRate = *playerRooms, *perAddr, *reqRate
	ln, err := net.Listen("tcp", *addr)
	if err != nil {
		fatal(err)
	}
	pc, err := net.ListenPacket("udp", *addr)
	if err != nil {
		fatal(err)
	}
	go func() { fatal(s.ServeUDP(pc)) }()
	logger.Info("started", "version", version, "clients", ln.Addr().String(), "relay", *relay)
	if *httpAddr != "" {
		s.httpToken = *httpToken
		logger.Info("status page", "addr", *httpAddr, "token", *httpToken != "")
		if reachable(*httpAddr) && *httpToken == "" {
			logger.Warn("the status page lists every online ID and has no token; set -http-token or bind -http to localhost")
		}
		go func() { fatal(http.ListenAndServe(*httpAddr, s.HTTP())) }()
	}
	fatal(s.Serve(ln))
}
