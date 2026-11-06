package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"testing"
	"time"
)

// TestStatusPageDefaultOff: the -http flag has no default address, so the
// status page and /api/stats are off unless asked for.
func TestStatusPageDefaultOff(t *testing.T) {
	t.Setenv("RE_PSN_HTTP", "")
	os.Unsetenv("RE_PSN_HTTP")
	os.Unsetenv("PSNR_HTTP")
	if got := envStr("http", ""); got != "" {
		t.Fatalf("http with no variable set gave %q", got)
	}
	usage := runUsage(t)
	if !strings.Contains(usage, "off unless set") {
		t.Errorf("usage of -http does not say it is off by default:\n%s", usage)
	}
}

func runUsage(t *testing.T) string {
	t.Helper()
	if testing.Short() {
		t.Skip("starts the server in a subprocess")
	}
	cmd := exec.Command(os.Args[0], "-test.run=^TestMainProcess$")
	cmd.Env = append(cleanEnv(), "RE_PSN_TEST_MAIN=1", "RE_PSN_TEST_ARGS=-h")
	out, _ := cmd.CombinedOutput()
	return string(out)
}

func cleanEnv() []string {
	var env []string
	for _, e := range os.Environ() {
		if !strings.HasPrefix(e, "RE_PSN_") && !strings.HasPrefix(e, "PSNR_") {
			env = append(env, e)
		}
	}
	return env
}

// startMain runs the real main in a subprocess with the given extra
// environment and arguments, waits for its "started" line, and returns the
// process and every log line read until then.
func startMain(t *testing.T, env []string, args ...string) (*exec.Cmd, string) {
	t.Helper()
	if testing.Short() {
		t.Skip("starts the server in a subprocess")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestMainProcess$")
	cmd.Env = append(cleanEnv(), "RE_PSN_TEST_MAIN=1", "RE_PSN_ADDR=127.0.0.1:0", "RE_PSN_LOG_LEVEL=info")
	cmd.Env = append(cmd.Env, "RE_PSN_TEST_ARGS="+strings.Join(args, "\x1f"))
	cmd.Env = append(cmd.Env, env...)
	pr, pw := io.Pipe()
	cmd.Stdout, cmd.Stderr = pw, pw
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { cmd.Process.Kill(); cmd.Wait(); pw.Close() })
	lines := make(chan string)
	go func() {
		sc := bufio.NewScanner(pr)
		for sc.Scan() {
			lines <- sc.Text()
		}
		close(lines)
	}()
	var log strings.Builder
	deadline := time.After(20 * time.Second)
	started := false
	var settle <-chan time.Time
	for {
		select {
		case l, ok := <-lines:
			if !ok {
				t.Fatalf("server stopped:\n%s", log.String())
			}
			log.WriteString(l + "\n")
			if !started && strings.Contains(l, "msg=started") {
				// The status page line, if any, follows "started" at once.
				started, settle = true, time.After(500*time.Millisecond)
			}
		case <-settle:
			go func() {
				for range lines {
				}
			}()
			return cmd, log.String()
		case <-deadline:
			t.Fatalf("server did not start:\n%s", log.String())
		}
	}
}

// listeningTCP counts the listening TCP sockets the process pid owns, from
// /proc (Linux only).
func listeningTCP(t *testing.T, pid int) int {
	t.Helper()
	fds, err := os.ReadDir(fmt.Sprintf("/proc/%d/fd", pid))
	if err != nil {
		t.Fatal(err)
	}
	inodes := map[string]bool{}
	for _, fd := range fds {
		link, err := os.Readlink(fmt.Sprintf("/proc/%d/fd/%s", pid, fd.Name()))
		if err == nil && strings.HasPrefix(link, "socket:[") {
			inodes[strings.TrimSuffix(strings.TrimPrefix(link, "socket:["), "]")] = true
		}
	}
	n := 0
	for _, f := range []string{"tcp", "tcp6"} {
		b, err := os.ReadFile(fmt.Sprintf("/proc/%d/net/%s", pid, f))
		if err != nil {
			continue
		}
		for _, row := range strings.Split(string(b), "\n")[1:] {
			c := strings.Fields(row)
			if len(c) > 9 && c[3] == "0A" && inodes[c[9]] {
				n++
			}
		}
	}
	return n
}

func freePort(t *testing.T) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	return ln.Addr().String()
}

// TestStatusPageOff: with no -http the server logs no status page line,
// opens no listener besides the game's, and the old default address does not
// answer /api/stats.
func TestStatusPageOff(t *testing.T) {
	old := "127.0.0.1:36101"
	oldFree := false
	if ln, err := net.Listen("tcp", old); err == nil {
		ln.Close()
		oldFree = true
	}
	cmd, log := startMain(t, nil)
	if strings.Contains(log, "status page") {
		t.Errorf("status page logged with -http unset:\n%s", log)
	}
	if runtime.GOOS == "linux" {
		if n := listeningTCP(t, cmd.Process.Pid); n != 1 {
			t.Errorf("%d listening TCP sockets, want only the game's", n)
		}
	}
	if oldFree {
		c := http.Client{Timeout: 2 * time.Second}
		if resp, err := c.Get("http://" + old + "/api/stats"); err == nil {
			resp.Body.Close()
			t.Errorf("/api/stats answered on %s with -http unset: %s", old, resp.Status)
		}
	}
}

// TestStatusPageOn: -http (or RE_PSN_HTTP) turns the page and /api/stats on
// at that address, as before, and -http-token still guards them.
func TestStatusPageOn(t *testing.T) {
	for _, c := range []struct {
		name  string
		env   func(addr string) []string
		args  func(addr string) []string
		token string
	}{
		{"flag", nil, func(a string) []string { return []string{"-http", a} }, ""},
		{"variable", func(a string) []string { return []string{"RE_PSN_HTTP=" + a} }, nil, ""},
		{"token", nil, func(a string) []string { return []string{"-http", a, "-http-token", "s3cret-test"} }, "s3cret-test"},
	} {
		t.Run(c.name, func(t *testing.T) {
			addr := freePort(t)
			var env, args []string
			if c.env != nil {
				env = c.env(addr)
			}
			if c.args != nil {
				args = c.args(addr)
			}
			cmd, log := startMain(t, env, args...)
			if !strings.Contains(log, `msg="status page"`) {
				t.Errorf("no status page line:\n%s", log)
			}
			if runtime.GOOS == "linux" {
				if n := listeningTCP(t, cmd.Process.Pid); n != 2 {
					t.Errorf("%d listening TCP sockets, want the game's and the status page's", n)
				}
			}
			cl := http.Client{Timeout: 5 * time.Second}
			url := "http://" + addr + "/api/stats"
			if c.token != "" {
				resp, err := cl.Get(url)
				if err != nil {
					t.Fatal(err)
				}
				resp.Body.Close()
				if resp.StatusCode != http.StatusUnauthorized {
					t.Errorf("no token: %s, want 401", resp.Status)
				}
				url += "?token=" + c.token
			}
			resp, err := cl.Get(url)
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()
			var v map[string]any
			if resp.StatusCode != http.StatusOK || json.NewDecoder(resp.Body).Decode(&v) != nil {
				t.Fatalf("/api/stats: %s", resp.Status)
			}
		})
	}
}
