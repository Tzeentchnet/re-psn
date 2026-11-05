package main

import (
	"bytes"
	"flag"
	"log/slog"
	"strings"
	"testing"
)

func TestEnvName(t *testing.T) {
	if got := envName("max-rooms"); got != "RE_PSN_MAX_ROOMS" {
		t.Fatal(got)
	}
	if got := envName("v"); got != "RE_PSN_V" {
		t.Fatal(got)
	}
	if got := oldEnvName("max-rooms"); got != "PSNR_MAX_ROOMS" {
		t.Fatal(got)
	}
}

func TestEnvValues(t *testing.T) {
	t.Cleanup(func() { envNotices = nil })
	t.Setenv("RE_PSN_MAX_ROOMS", " 5 ")
	t.Setenv("RE_PSN_RELAY", "true")
	t.Setenv("RE_PSN_HTTP", "")
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
	t.Cleanup(func() { envNotices = nil })
	t.Setenv("RE_PSN_MAX_ROOMS", "lots")
	t.Setenv("PSNR_RELAY", "maybe")
	if _, err := envInt("max-rooms", 1); err == nil || !strings.Contains(err.Error(), "RE_PSN_MAX_ROOMS") {
		t.Fatalf("bad number: %v", err)
	}
	if _, err := envBool("relay", false); err == nil || !strings.Contains(err.Error(), "PSNR_RELAY") {
		t.Fatalf("bad bool: %v", err)
	}
	// A bad new name is reported even when the old one parses.
	t.Setenv("PSNR_MAX_ROOMS", "5")
	if _, err := envInt("max-rooms", 1); err == nil || !strings.Contains(err.Error(), "RE_PSN_MAX_ROOMS") {
		t.Fatalf("bad new over good old: %v", err)
	}
}

// TestEnvAlias covers the old PSNR_ names next to the new RE_PSN_ ones.
func TestEnvAlias(t *testing.T) {
	for _, c := range []struct {
		name           string
		setNew, setOld bool
		newV, oldV     string
		want           string
		notice         string // "" for none
	}{
		{"new only", true, false, "a", "", "a", ""},
		{"old only", false, true, "", "b", "b", "old variable name"},
		{"both same", true, true, "c", "c", "c", ""},
		{"both differ", true, true, "d", "e", "d", "differ"},
		{"new empty", true, false, "", "", "", ""},
		{"old empty", false, true, "", "", "", "old variable name"},
		{"new empty, old set", true, true, "", "f", "", "differ"},
		{"neither", false, false, "", "", "def", ""},
	} {
		t.Run(c.name, func(t *testing.T) {
			envNotices = nil
			t.Cleanup(func() { envNotices = nil })
			if c.setNew {
				t.Setenv("RE_PSN_LOG_FILE", c.newV)
			}
			if c.setOld {
				t.Setenv("PSNR_LOG_FILE", c.oldV)
			}
			if got := envStr("log-file", "def"); got != c.want {
				t.Fatalf("got %q, want %q", got, c.want)
			}
			if c.notice == "" {
				if len(envNotices) != 0 {
					t.Fatalf("unexpected notice %+v", envNotices)
				}
				return
			}
			if len(envNotices) != 1 || !strings.Contains(envNotices[0].msg, c.notice) ||
				envNotices[0].old != "PSNR_LOG_FILE" || envNotices[0].new != "RE_PSN_LOG_FILE" {
				t.Fatalf("notices %+v", envNotices)
			}
		})
	}
}

// TestEnvNoticeHidesValue: the notices name the variables but never print what they hold.
func TestEnvNoticeHidesValue(t *testing.T) {
	envNotices = nil
	t.Cleanup(func() { envNotices = nil })
	const secret = "s3cret-token-value"
	t.Setenv("PSNR_HTTP_TOKEN", secret)
	t.Setenv("RE_PSN_LOG_FILE", secret+"-new")
	t.Setenv("PSNR_LOG_FILE", secret+"-old")
	envStr("http-token", "")
	envStr("log-file", "")
	for _, format := range []string{"text", "json"} {
		saved := envNotices
		var buf bytes.Buffer
		logEnvNotices(newLogger(&buf, slog.LevelInfo, format))
		envNotices = saved
		out := buf.String()
		if strings.Contains(out, "s3cret") {
			t.Fatalf("%s log shows the value: %s", format, out)
		}
		for _, want := range []string{"PSNR_HTTP_TOKEN", "RE_PSN_HTTP_TOKEN", "PSNR_LOG_FILE", "RE_PSN_LOG_FILE", "WARN"} {
			if !strings.Contains(out, want) {
				t.Fatalf("%s log lacks %s: %s", format, want, out)
			}
		}
	}
}

// TestFlagBeatsEnv: a flag on the command line wins over both names.
func TestFlagBeatsEnv(t *testing.T) {
	t.Cleanup(func() { envNotices = nil })
	t.Setenv("RE_PSN_MAX_ROOMS", "5")
	t.Setenv("PSNR_MAX_ROOMS", "6")
	t.Setenv("RE_PSN_LOG_LEVEL", "warn")
	t.Setenv("PSNR_LOG_LEVEL", "error")
	n, err := envInt("max-rooms", 1000)
	if err != nil {
		t.Fatal(err)
	}
	fs := flag.NewFlagSet("t", flag.ContinueOnError)
	rooms := fs.Int("max-rooms", n, "")
	level := fs.String("log-level", envStr("log-level", "info"), "")
	if err := fs.Parse([]string{"-max-rooms", "7", "-log-level", "debug"}); err != nil {
		t.Fatal(err)
	}
	if *rooms != 7 || *level != "debug" {
		t.Fatalf("got %d %s", *rooms, *level)
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
