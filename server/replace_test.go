package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestParsePidFile(t *testing.T) {
	for _, tt := range []struct {
		name, in string
		want     pidFile
		ok       bool
	}{
		{"full", "pid=4242\nstart=123\nexe=/usr/bin/re-psn\naddr=[::]:36100\n", pidFile{4242, "123", "/usr/bin/re-psn", "[::]:36100"}, true},
		{"pid only", "pid=7", pidFile{pid: 7}, true},
		{"unknown lines", "x=1\npid=7\n\n", pidFile{pid: 7}, true},
		{"no pid", "start=1\n", pidFile{}, false},
		{"bad pid", "pid=abc\n", pidFile{}, false},
		{"zero pid", "pid=0\n", pidFile{}, false},
		{"negative pid", "pid=-1\n", pidFile{}, false},
		{"empty", "", pidFile{}, false},
	} {
		got, err := parsePidFile([]byte(tt.in))
		if (err == nil) != tt.ok || got != tt.want {
			t.Errorf("%s: got %+v, %v", tt.name, got, err)
		}
	}
	f := pidFile{1, "2", "/a b/re-psn", "127.0.0.1:1"}
	if got, err := parsePidFile([]byte(f.String())); err != nil || got != f {
		t.Errorf("round trip: %+v, %v", got, err)
	}
}

func TestCheckHolder(t *testing.T) {
	m := self{1000, "/tmp/.mount_re-psnAbC/usr/bin/re-psn"}
	f := pidFile{pid: 4242, start: "555", exe: "/tmp/.mount_re-psnXyZ/usr/bin/re-psn"}
	ok := procFacts{alive: true, uid: 1000, start: "555", exe: "/tmp/.mount_re-psnXyZ/usr/bin/re-psn"}
	with := func(change func(*procFacts)) procFacts { p := ok; change(&p); return p }
	for _, tt := range []struct {
		name string
		f    pidFile
		p    procFacts
		m    self
		why  string // "" accepted
	}{
		{"earlier AppImage", f, ok, m, ""},
		{"other name starting re-psn", f, with(func(p *procFacts) { p.exe = "/usr/local/bin/re-psn-0.2.0" }), m, ""},
		{"same name as ours", f, with(func(p *procFacts) { p.exe = "/opt/x/server.test" }), self{1000, "/tmp/y/server.test"}, ""},
		{"binary replaced on disk", f, with(func(p *procFacts) { p.exe = "/usr/local/bin/re-psn (deleted)" }), m, ""},
		{"exe unreadable, pidfile names re-psn", f, with(func(p *procFacts) { p.exe = "" }), m, ""},
		{"exe unreadable, pidfile names other", pidFile{4242, "555", "/usr/bin/python3", ""}, with(func(p *procFacts) { p.exe = "" }), m, `its program is "python3"`},
		{"exe unreadable, no exe line", pidFile{pid: 4242, start: "555"}, with(func(p *procFacts) { p.exe = "" }), m, "its program is"},
		{"not running", f, with(func(p *procFacts) { p.alive = false }), m, "not running"},
		{"other user", f, with(func(p *procFacts) { p.uid = 0 }), m, "another user"},
		{"pid reused", f, with(func(p *procFacts) { p.start = "999" }), m, "start time"},
		{"empty start in file", pidFile{pid: 4242, exe: f.exe}, ok, m, "start time"},
		{"unknown uid (not Linux)", f, with(func(p *procFacts) { p.uid = -1 }), m, "cannot be checked"},
		{"unknown start", f, with(func(p *procFacts) { p.start = "" }), m, "cannot be checked"},
		{"another program", f, with(func(p *procFacts) { p.exe = "/usr/bin/python3.12" }), m, `its program is "python3.12"`},
		{"name only contains re-psn", f, with(func(p *procFacts) { p.exe = "/usr/bin/not-re-psn" }), m, "its program is"},
	} {
		err := checkHolder(tt.f, tt.p, tt.m)
		if tt.why == "" && err != nil || tt.why != "" && (err == nil || !strings.Contains(err.Error(), tt.why)) {
			t.Errorf("%s: got %v, want %q", tt.name, err, tt.why)
		}
	}
}

func discard() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

func TestPidFileWrite(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the pidfile directory checks are Unix file modes")
	}
	t.Setenv("XDG_RUNTIME_DIR", t.TempDir())
	path := writePidFile(discard(), 40001, "127.0.0.1:40001")
	dir, err := pidDir()
	if err != nil || path != filepath.Join(dir, "40001.pid") {
		t.Fatalf("path %q, dir %q, %v", path, dir, err)
	}
	for p, want := range map[string]os.FileMode{dir: 0o700, path: 0o600} {
		if fi, err := os.Stat(p); err != nil || fi.Mode().Perm() != want {
			t.Errorf("%s: mode %v, %v", p, fi.Mode().Perm(), err)
		}
	}
	f, err := readPidFile(path)
	exe, _ := os.Executable()
	if err != nil || f.pid != os.Getpid() || f.exe != exe || f.addr != "127.0.0.1:40001" {
		t.Fatalf("%+v, %v", f, err)
	}
	if runtime.GOOS == "linux" && (f.start == "" || f.start != procStart(os.Getpid())) {
		t.Errorf("start %q", f.start)
	}
	if checkHolder(f, procInfo(f.pid), me()) != nil && runtime.GOOS == "linux" {
		t.Errorf("our own pidfile does not pass: %v", checkHolder(f, procInfo(f.pid), me()))
	}
	removePidFile(path)
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("not removed: %v", err)
	}
}

// A file that names a process that is gone, or a reused pid, is replaced.
func TestPidFileStale(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the pidfile directory checks are Unix file modes")
	}
	t.Setenv("XDG_RUNTIME_DIR", t.TempDir())
	dir, err := pidDir()
	if err != nil {
		t.Fatal(err)
	}
	gone := exec.Command("true")
	if err := gone.Run(); err != nil {
		t.Skip("no true command:", err)
	}
	exe, _ := os.Executable()
	for name, stale := range map[string]pidFile{
		"gone":       {gone.Process.Pid, "1", exe, ""},
		"pid reused": {os.Getppid(), "1", exe, ""},
	} {
		path := pidPath(dir, 40002)
		if err := os.WriteFile(path, []byte(stale.String()), 0o600); err != nil {
			t.Fatal(err)
		}
		if got := writePidFile(discard(), 40002, "x"); got != path {
			t.Errorf("%s: not replaced (%q)", name, got)
		}
		if f, err := readPidFile(path); err != nil || f.pid != os.Getpid() {
			t.Errorf("%s: %+v, %v", name, f, err)
		}
	}
	// Removing leaves a file that names another process.
	other := pidFile{pid: os.Getppid()}
	path := pidPath(dir, 40003)
	os.WriteFile(path, []byte(other.String()), 0o600)
	removePidFile(path)
	if _, err := os.Stat(path); err != nil {
		t.Errorf("removed another process's file: %v", err)
	}
}

func TestPidDirRefused(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the pidfile directory checks are Unix file modes")
	}
	for name, setup := range map[string]func(dir string) error{
		"symlink": func(dir string) error { return os.Symlink(t.TempDir(), dir) },
		"open":    func(dir string) error { os.Mkdir(dir, 0o700); return os.Chmod(dir, 0o755) },
		"not dir": func(dir string) error { return os.WriteFile(dir, nil, 0o600) },
		"fine":    nil,
	} {
		x := t.TempDir()
		t.Setenv("XDG_RUNTIME_DIR", x)
		if setup == nil {
			if _, err := pidDir(); err != nil {
				t.Errorf("%s: %v", name, err)
			}
			continue
		}
		if err := setup(filepath.Join(x, "re-psn")); err != nil {
			t.Fatal(err)
		}
		if _, err := pidDir(); err == nil {
			t.Errorf("%s: accepted", name)
		}
		if path := writePidFile(discard(), 40004, "x"); path != "" {
			t.Errorf("%s: wrote %s", name, path)
		}
	}
}

// TestHelperProcess is not a test: the process tests below run the test
// binary again with RE_PSN_TEST_HELPER set, to play another process.
//
//	server    runs main with the arguments after "--"
//	stubborn  binds the port and writes a pidfile like the server, but ignores SIGTERM
//	listen    binds the TCP port only, with no pidfile (a SIGTERM would end it)
func TestHelperProcess(t *testing.T) {
	mode := os.Getenv("RE_PSN_TEST_HELPER")
	if mode == "" {
		return
	}
	var args []string
	for i, a := range os.Args {
		if a == "--" {
			args = os.Args[i+1:]
			break
		}
	}
	switch mode {
	case "server":
		if w, err := time.ParseDuration(os.Getenv("RE_PSN_TEST_WAIT")); err == nil {
			replaceWait = w
		}
		os.Args = append([]string{"re-psn"}, args...)
		flag.CommandLine = flag.NewFlagSet("re-psn", flag.ExitOnError)
		main()
	case "stubborn", "listen":
		if mode == "stubborn" {
			signal.Ignore(syscall.SIGTERM)
		}
		ln, err := net.Listen("tcp", args[0])
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
		if mode == "stubborn" {
			writePidFile(discard(), portOf(args[0]), ln.Addr().String())
		}
		fmt.Println("ready")
		select {}
	}
	os.Exit(0)
}

type proc struct {
	t    *testing.T
	cmd  *exec.Cmd
	out  syncBuf
	done chan error
}

func spawn(t *testing.T, mode, xdg string, env []string, args ...string) *proc {
	p := &proc{t: t, done: make(chan error, 1)}
	p.cmd = exec.Command(os.Args[0], append([]string{"-test.run=^TestHelperProcess$", "--"}, args...)...)
	p.cmd.Env = append(os.Environ(), append([]string{"RE_PSN_TEST_HELPER=" + mode, "XDG_RUNTIME_DIR=" + xdg}, env...)...)
	p.cmd.Stdout, p.cmd.Stderr = &p.out, &p.out
	if err := p.cmd.Start(); err != nil {
		t.Fatal(err)
	}
	go func() { p.done <- p.cmd.Wait() }()
	t.Cleanup(func() {
		select {
		case <-p.done:
		default:
			p.cmd.Process.Kill()
			<-p.done
		}
	})
	return p
}

// wait waits for the process to exit and returns its exit code.
func (p *proc) wait() int {
	select {
	case err := <-p.done:
		p.done <- err // keep it for running() and Cleanup
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			return ee.ExitCode()
		} else if err != nil {
			p.t.Fatal(err)
		}
		return 0
	case <-time.After(20 * time.Second):
		p.t.Fatalf("pid %d did not exit", p.cmd.Process.Pid)
		return -1
	}
}

func (p *proc) running() bool {
	select {
	case err := <-p.done:
		p.done <- err
		return false
	default:
		return true
	}
}

// waitFor polls until ok or a deadline.
func waitFor(t *testing.T, what string, ok func() bool) {
	for deadline := time.Now().Add(10 * time.Second); !ok(); time.Sleep(20 * time.Millisecond) {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %s", what)
		}
	}
}

// freeAddr returns 127.0.0.1:<port> with the port free for TCP and UDP.
func freeAddr(t *testing.T) string {
	for i := 0; i < 20; i++ {
		ln, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatal(err)
		}
		addr := ln.Addr().String()
		pc, err := net.ListenPacket("udp", addr)
		ln.Close()
		if err == nil {
			pc.Close()
			return addr
		}
	}
	t.Fatal("no free port")
	return ""
}

func pidfileNames(t *testing.T, xdg, addr string, pid int) func() bool {
	return func() bool {
		f, err := readPidFile(filepath.Join(xdg, "re-psn", strconv.Itoa(portOf(addr))+".pid"))
		return err == nil && f.pid == pid
	}
}

func needReplace(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows has no SIGTERM, and -replace is not supported there")
	}
	if !replaceSupported {
		t.Skip("-replace is Linux only")
	}
}

// An earlier instance gets a hint without -replace, and SIGTERM with it; the
// new one then binds, and stops cleanly on SIGTERM itself.
func TestReplaceEarlierInstance(t *testing.T) {
	needReplace(t)
	xdg, addr := t.TempDir(), freeAddr(t)
	srv := []string{"-addr", addr, "-http="}
	old := spawn(t, "server", xdg, nil, srv...)
	waitFor(t, "the first instance's pidfile", pidfileNames(t, xdg, addr, old.cmd.Process.Pid))

	plain := spawn(t, "server", xdg, nil, srv...)
	if code := plain.wait(); code != 1 || !strings.Contains(plain.out.String(), "stop it, or run with -replace") {
		t.Fatalf("without -replace: exit %d\n%s", code, plain.out.String())
	}
	if !old.running() {
		t.Fatal("the first instance stopped without -replace")
	}

	repl := spawn(t, "server", xdg, nil, append(srv, "-replace")...)
	if code := old.wait(); code != 0 || !strings.Contains(old.out.String(), "msg=stopping") {
		t.Fatalf("first instance: exit %d\n%s", code, old.out.String())
	}
	waitFor(t, "the new instance's pidfile", pidfileNames(t, xdg, addr, repl.cmd.Process.Pid))
	repl.cmd.Process.Signal(syscall.SIGTERM)
	out := func() string { return repl.out.String() }
	if code := repl.wait(); code != 0 || !strings.Contains(out(), "replacing the earlier instance") ||
		!strings.Contains(out(), "msg=started") || !strings.Contains(out(), "msg=stopping") || strings.Contains(out(), "fatal") {
		t.Fatalf("new instance: exit %d\n%s", code, out())
	}
	if _, err := os.Stat(filepath.Join(xdg, "re-psn", strconv.Itoa(portOf(addr))+".pid")); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("pidfile left after a graceful stop: %v", err)
	}
}

// A holder that is not an earlier re-psn of this user is never signalled.
func TestReplaceRefuses(t *testing.T) {
	needReplace(t)
	xdg := t.TempDir()
	run := func(addr string, env ...string) (int, string) {
		p := spawn(t, "server", xdg, env, "-addr", addr, "-http=", "-replace")
		return p.wait(), p.out.String()
	}
	ready := func(p *proc) {
		waitFor(t, "the holder", func() bool { return strings.Contains(p.out.String(), "ready") || !p.running() })
	}
	forge := func(addr string, f pidFile) {
		os.MkdirAll(filepath.Join(xdg, "re-psn"), 0o700)
		if err := os.WriteFile(filepath.Join(xdg, "re-psn", strconv.Itoa(portOf(addr))+".pid"), []byte(f.String()), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	t.Run("no pidfile", func(t *testing.T) {
		addr := freeAddr(t)
		h := spawn(t, "listen", xdg, nil, addr)
		ready(h)
		code, out := run(addr)
		if code != 1 || !strings.Contains(out, "no re-psn pidfile names its holder") || !h.running() {
			t.Fatalf("exit %d, holder running %v\n%s", code, h.running(), out)
		}
	})

	t.Run("pid reused", func(t *testing.T) {
		addr := freeAddr(t)
		h := spawn(t, "listen", xdg, nil, addr)
		ready(h)
		forge(addr, pidFile{h.cmd.Process.Pid, "1", os.Args[0], addr})
		code, out := run(addr)
		want := fmt.Sprintf("in use by pid %d (started", h.cmd.Process.Pid)
		if code != 1 || !strings.Contains(out, want) || !strings.Contains(out, "not a re-psn instance of this user") || !h.running() {
			t.Fatalf("exit %d, holder running %v\n%s", code, h.running(), out)
		}
	})

	t.Run("another program", func(t *testing.T) {
		py, err := exec.LookPath("python3")
		if err != nil {
			t.Skip("no python3 to play a non-re-psn listener")
		}
		addr := freeAddr(t)
		host, port, _ := net.SplitHostPort(addr)
		h := &proc{t: t, done: make(chan error, 1)}
		h.cmd = exec.Command(py, "-c", "import socket,sys,time\ns=socket.socket()\ns.bind((sys.argv[1],int(sys.argv[2])))\n"+
			"s.listen()\nprint('ready',flush=True)\ntime.sleep(60)", host, port)
		h.cmd.Stdout = &h.out
		if err := h.cmd.Start(); err != nil {
			t.Fatal(err)
		}
		go func() { h.done <- h.cmd.Wait() }()
		t.Cleanup(func() { h.cmd.Process.Kill(); <-h.done })
		ready(h)
		pid := h.cmd.Process.Pid
		forge(addr, pidFile{pid, procStart(pid), py, addr}) // everything right but the program
		code, out := run(addr)
		if code != 1 || !strings.Contains(out, "not a re-psn instance of this user: its program is") || !h.running() {
			t.Fatalf("exit %d, holder running %v\n%s", code, h.running(), out)
		}
	})

	t.Run("does not stop", func(t *testing.T) {
		addr := freeAddr(t)
		h := spawn(t, "stubborn", xdg, nil, addr)
		ready(h)
		code, out := run(addr, "RE_PSN_TEST_WAIT=500ms")
		if code != 1 || !strings.Contains(out, "replacing the earlier instance") || !strings.Contains(out, "did not stop in 500ms") || !h.running() {
			t.Fatalf("exit %d, holder running %v\n%s", code, h.running(), out)
		}
	})
}
