package main

import (
	"os"
	"testing"
)

func TestStatStart(t *testing.T) {
	tail := " S 1 2 3 4 5 6 7 8 9 10 11 12 13 14 15 16 17 18 778899 20 21"
	for _, tt := range []struct{ in, want string }{
		{"4242 (re-psn)" + tail, "778899"},
		{"4242 (a) b (c))" + tail, "778899"}, // a command name with spaces and parentheses
		{"4242 (re-psn) S 1 2", ""},
		{"no parenthesis", ""},
	} {
		if got := statStart(tt.in); got != tt.want {
			t.Errorf("%q: got %q, want %q", tt.in, got, tt.want)
		}
	}
	if procStart(os.Getpid()) == "" {
		t.Error("no start time for this process")
	}
	if got := startedAt(procStart(os.Getpid())); got == "at an unknown time" {
		t.Error(got)
	}
}
