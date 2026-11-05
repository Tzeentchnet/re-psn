package main

import "testing"

func TestEnvName(t *testing.T) {
	if got := envName("max-rooms"); got != "PSNR_MAX_ROOMS" {
		t.Fatal(got)
	}
	if got := envName("v"); got != "PSNR_V" {
		t.Fatal(got)
	}
}

func TestEnvValues(t *testing.T) {
	t.Setenv("PSNR_MAX_ROOMS", " 5 ")
	t.Setenv("PSNR_RELAY", "true")
	t.Setenv("PSNR_HTTP", "")
	if n, err := envInt("max-rooms", 1000); err != nil || n != 5 {
		t.Fatalf("int: %d %v", n, err)
	}
	if b, err := envBool("relay", false); err != nil || !b {
		t.Fatalf("bool: %v %v", b, err)
	}
	if got := envStr("http", "127.0.0.1:36101"); got != "" { // set but empty turns it off
		t.Fatalf("empty variable gave %q", got)
	}
	if got := envStr("log-file", "x"); got != "x" { // not set: default
		t.Fatalf("unset variable gave %q", got)
	}
}

func TestEnvBad(t *testing.T) {
	t.Setenv("PSNR_MAX_ROOMS", "lots")
	t.Setenv("PSNR_RELAY", "maybe")
	if _, err := envInt("max-rooms", 1); err == nil {
		t.Fatal("bad number accepted")
	}
	if _, err := envBool("relay", false); err == nil {
		t.Fatal("bad bool accepted")
	}
}

func TestListenAddrs(t *testing.T) {
	for _, c := range []struct{ addr, tcp, udp, wantT, wantU string }{
		{":36100", "", "", ":36100", ":36100"},
		{":36100", ":1", "", ":1", ":36100"},
		{":36100", "", ":2", ":36100", ":2"},
		{":36100", ":1", ":2", ":1", ":2"},
	} {
		if gt, gu := listenAddrs(c.addr, c.tcp, c.udp); gt != c.wantT || gu != c.wantU {
			t.Fatalf("%+v: got %s %s", c, gt, gu)
		}
	}
}
