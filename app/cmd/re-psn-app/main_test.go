package main

import (
	"bytes"
	"errors"
	"io"
	"net"
	"strings"
	"testing"
)

func restoreStartupHooks(t *testing.T) {
	t.Helper()
	tty, tui, desktop := isTerminal, startTUI, launchDesktop
	t.Cleanup(func() { isTerminal, startTUI, launchDesktop = tty, tui, desktop })
	launchDesktop = func() error { return errors.New("run in a terminal") }
}

func TestRun(t *testing.T) {
	restoreStartupHooks(t)
	isTerminal = func() bool { return false }
	startTUI = func(io.Writer) int { t.Error("the UI started with no terminal"); return 0 }
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	closed := ln.Addr().String()
	ln.Close() // nothing listens there now

	cases := []struct {
		args []string
		code int
		out  string
	}{
		{[]string{"version"}, 0, "re-psn-app " + version + "\n"},
		{nil, 2, ""}, // no terminal: a note on stderr
		{[]string{"--no-tui"}, 2, ""},
		{[]string{"frobnicate"}, 2, ""},
		{[]string{"check"}, 2, ""},
		{[]string{"check", "-timeout", "2s", closed}, 1, "FAIL dial " + closed},
	}
	for _, c := range cases {
		var out, errOut bytes.Buffer
		if code := run(c.args, &out, &errOut); code != c.code || !strings.HasPrefix(out.String(), c.out) {
			t.Errorf("run(%q) = %d, stdout %q (stderr %q); want %d, stdout starting %q", c.args, code, out.String(), errOut.String(), c.code, c.out)
		}
	}
}

func TestRunStartsTheUIOnlyInATerminal(t *testing.T) {
	restoreStartupHooks(t)
	started := 0
	startTUI = func(io.Writer) int { started++; return 0 }
	for _, tty := range []bool{false, true} {
		isTerminal = func() bool { return tty }
		var out, errOut bytes.Buffer
		code := run(nil, &out, &errOut)
		switch {
		case tty && (code != 0 || started != 1):
			t.Errorf("in a terminal: exit %d, UI started %d times; want 0 and once", code, started)
		case !tty && (code != 2 || started != 0 || !strings.Contains(errOut.String(), "not a terminal")):
			t.Errorf("no terminal: exit %d, UI started %d times, stderr %q", code, started, errOut.String())
		}
		if strings.Count(errOut.String(), "\n") > 1 {
			t.Errorf("want a one-line note, got %q", errOut.String())
		}
	}
}

func TestRunDesktopStartup(t *testing.T) {
	restoreStartupHooks(t)
	isTerminal = func() bool { return false }
	startTUI = func(io.Writer) int { t.Fatal("started TUI without a terminal"); return 0 }
	for _, args := range [][]string{nil, {"--"}} {
		for _, failed := range []bool{false, true} {
			calls := 0
			launchDesktop = func() error {
				calls++
				if failed {
					return errors.New("synthetic failure")
				}
				return nil
			}
			var out, errOut bytes.Buffer
			code := run(args, &out, &errOut)
			want := 0
			if failed {
				want = 2
			}
			if code != want || calls != 1 || out.Len() != 0 || strings.Count(errOut.String(), "\n") != map[bool]int{false: 0, true: 1}[failed] {
				t.Fatalf("desktop run(%q) = %d, calls %d, output %q/%q", args, code, calls, out.String(), errOut.String())
			}
		}
	}
	launchDesktop = func() error { t.Fatal("noninteractive command opened a terminal"); return nil }
	for _, args := range [][]string{{"version"}, {"--help"}, {"--no-tui"}, {"unknown"}, {"check"}} {
		run(args, io.Discard, io.Discard)
	}
}
