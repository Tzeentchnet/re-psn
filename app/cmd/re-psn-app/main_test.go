package main

import (
	"bytes"
	"io"
	"net"
	"strings"
	"testing"
)

func TestRun(t *testing.T) {
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
