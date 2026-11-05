package main

// Logging: log/slog, text or JSON, one level for the whole server. The log
// holds online IDs and client addresses (they are what a refusal is about);
// it never holds the status page token, the UDP probe token or a request
// query string.

import (
	"fmt"
	"io"
	"log/slog"
)

func levelFor(verbose bool) slog.Level {
	if verbose {
		return slog.LevelDebug
	}
	return slog.LevelInfo
}

func newLogger(w io.Writer, level slog.Level, format string) *slog.Logger {
	opts := &slog.HandlerOptions{Level: level}
	if format == "json" {
		return slog.New(slog.NewJSONHandler(w, opts))
	}
	return slog.New(slog.NewTextHandler(w, opts))
}

// parseLogFlags turns the -log-level and -log-format flag values into a level
// and a format, or an error naming what is not allowed.
func parseLogFlags(level, format string) (slog.Level, string, error) {
	var l slog.Level
	if err := l.UnmarshalText([]byte(level)); err != nil {
		return 0, "", fmt.Errorf("-log-level %q: want debug, info, warn or error", level)
	}
	if format != "text" && format != "json" {
		return 0, "", fmt.Errorf("-log-format %q: want text or json", format)
	}
	return l, format, nil
}
