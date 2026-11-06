package main

import (
	"bytes"
	"net"
	"strings"
	"testing"
)

func TestRun(t *testing.T) {
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
		{nil, 2, ""},
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
