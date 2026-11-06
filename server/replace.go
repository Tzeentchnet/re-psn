package main

// Pidfile and -replace. After the game listeners bind, the server writes
// <port>.pid in a per-user directory ($XDG_RUNTIME_DIR/re-psn, else
// <tmp>/re-psn-<uid>), and removes it on a graceful stop. When the port is
// busy, the file says whether the holder is an earlier re-psn of this user:
// without -replace that becomes a hint, with it the server sends the holder
// SIGTERM and waits for the port. A holder is only signalled when every check
// in checkHolder passes; it is never sent SIGKILL. -replace is Linux only.

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// How long -replace waits for the earlier instance to let go of the port.
var (
	replaceWait = 10 * time.Second
	replacePoll = 100 * time.Millisecond
)

// pidFile is what <port>.pid holds.
type pidFile struct {
	pid   int
	start string // field 22 of /proc/<pid>/stat; empty off Linux
	exe   string // os.Executable of the writer
	addr  string // its TCP listen address
}

func (f pidFile) String() string {
	return fmt.Sprintf("pid=%d\nstart=%s\nexe=%s\naddr=%s\n", f.pid, f.start, f.exe, f.addr)
}

func parsePidFile(b []byte) (pidFile, error) {
	var f pidFile
	for _, line := range strings.Split(string(b), "\n") {
		k, v, _ := strings.Cut(strings.TrimSpace(line), "=")
		switch k {
		case "pid":
			n, err := strconv.Atoi(v)
			if err != nil || n <= 0 {
				return pidFile{}, errors.New("pidfile: bad pid line")
			}
			f.pid = n
		case "start":
			f.start = v
		case "exe":
			f.exe = v
		case "addr":
			f.addr = v
		}
	}
	if f.pid == 0 {
		return pidFile{}, errors.New("pidfile: no pid line")
	}
	return f, nil
}

// pidDir returns the pidfile directory, creating it with mode 0700. It is
// refused when it is a symlink, not owned by this user or open to others.
func pidDir() (string, error) {
	dir := filepath.Join(os.TempDir(), "re-psn")
	if uid := os.Getuid(); uid >= 0 {
		dir += "-" + strconv.Itoa(uid)
	}
	if x := os.Getenv("XDG_RUNTIME_DIR"); x != "" {
		dir = filepath.Join(x, "re-psn")
	}
	if err := os.Mkdir(dir, 0o700); err != nil && !errors.Is(err, fs.ErrExist) {
		return "", err
	}
	fi, err := os.Lstat(dir)
	if err != nil {
		return "", err
	}
	if !fi.IsDir() {
		return "", fmt.Errorf("%s is not a directory (or is a symlink)", dir)
	}
	if err := privateDir(fi); err != nil {
		return "", fmt.Errorf("%s %v", dir, err)
	}
	return dir, nil
}

func pidPath(dir string, port int) string {
	return filepath.Join(dir, strconv.Itoa(port)+".pid")
}

func readPidFile(path string) (pidFile, error) {
	fi, err := os.Lstat(path)
	if err != nil {
		return pidFile{}, err
	}
	if !fi.Mode().IsRegular() {
		return pidFile{}, fmt.Errorf("%s is not a regular file", path)
	}
	fd, err := os.Open(path)
	if err != nil {
		return pidFile{}, err
	}
	defer fd.Close()
	b, err := io.ReadAll(io.LimitReader(fd, 4096))
	if err != nil {
		return pidFile{}, err
	}
	return parsePidFile(b)
}

// self is what a holder is compared with: this process's uid and executable.
type self struct {
	uid int
	exe string
}

func me() self {
	exe, _ := os.Executable()
	return self{os.Getuid(), exe}
}

// procFacts is what the system says about a pid (procInfo reads it).
type procFacts struct {
	alive bool
	uid   int    // owner of /proc/<pid>; -1 when unknown
	start string // field 22 of /proc/<pid>/stat; empty when unknown
	exe   string // target of /proc/<pid>/exe; empty when unreadable
}

// checkHolder says whether the process a pidfile names is an earlier re-psn
// of this user, and if not, why. It only compares; procInfo does the reading.
func checkHolder(f pidFile, p procFacts, m self) error {
	switch {
	case !p.alive:
		return errors.New("it is not running")
	case p.uid < 0 || p.start == "":
		return errors.New("it cannot be checked on this system")
	case p.uid != m.uid:
		return errors.New("it belongs to another user")
	case p.start != f.start:
		return errors.New("its start time is not the one in the pidfile")
	}
	exe := p.exe
	if exe == "" {
		exe = f.exe
	}
	name := strings.TrimSuffix(filepath.Base(exe), " (deleted)") // the binary was replaced on disk
	if exe == "" || (name != filepath.Base(m.exe) && !strings.HasPrefix(name, "re-psn")) {
		return fmt.Errorf("its program is %q", name)
	}
	return nil
}

// writePidFile writes <port>.pid for this process. A file left by a process
// that is gone (or a reused pid) is replaced; one naming a live re-psn on
// another address of the same port is left alone. It returns the path to
// remove on stop, or "" when nothing was written.
func writePidFile(logger *slog.Logger, port int, addr string) string {
	dir, err := pidDir()
	if err != nil {
		logger.Warn("no pidfile", "err", err)
		return ""
	}
	path := pidPath(dir, port)
	if old, err := readPidFile(path); err == nil && old.pid != os.Getpid() && checkHolder(old, procInfo(old.pid), me()) == nil {
		logger.Warn("no pidfile: another re-psn has one for this port", "pid", old.pid, "addr", old.addr)
		return ""
	}
	exe, _ := os.Executable()
	f := pidFile{os.Getpid(), procStart(os.Getpid()), exe, addr}
	tmp := fmt.Sprintf("%s.%d.tmp", path, os.Getpid())
	err = os.WriteFile(tmp, []byte(f.String()), 0o600)
	if err == nil {
		err = os.Rename(tmp, path)
	}
	if err != nil {
		os.Remove(tmp)
		logger.Warn("no pidfile", "err", err)
		return ""
	}
	return path
}

// removePidFile removes path if it still names this process.
func removePidFile(path string) {
	if path == "" {
		return
	}
	if f, err := readPidFile(path); err == nil && f.pid == os.Getpid() {
		os.Remove(path)
	}
}

func addrInUse(err error) bool { return errors.Is(err, syscall.EADDRINUSE) }

func portOf(addr string) int {
	_, p, _ := net.SplitHostPort(addr)
	n, _ := strconv.Atoi(p)
	return n
}

// listenGame binds the game TCP and UDP addresses. When one is busy it looks
// for an earlier re-psn holding it: with replace it stops that instance and
// waits for the port, without it the error gets a hint.
func listenGame(logger *slog.Logger, tcpA, udpA string, replace bool) (net.Listener, net.PacketConn, error) {
	var ln net.Listener
	err := busyRetry(logger, portOf(tcpA), replace, func() (err error) {
		ln, err = net.Listen("tcp", tcpA)
		return err
	})
	if err != nil {
		return nil, nil, err
	}
	var pc net.PacketConn
	err = busyRetry(logger, portOf(udpA), replace, func() (err error) {
		pc, err = net.ListenPacket("udp", udpA)
		return err
	})
	if err != nil {
		ln.Close()
		return nil, nil, err
	}
	return ln, pc, nil
}

func busyRetry(logger *slog.Logger, port int, replace bool, listen func() error) error {
	err := listen()
	if err == nil || !addrInUse(err) {
		return err
	}
	var f pidFile
	var why error
	if dir, derr := pidDir(); derr != nil {
		why = derr
	} else if f, why = readPidFile(pidPath(dir, port)); why == nil {
		why = checkHolder(f, procInfo(f.pid), me())
	}
	switch {
	case !replace && why == nil:
		return fmt.Errorf("%w: port %d is in use by an earlier re-psn (pid %d, started %s); stop it, or run with -replace",
			err, port, f.pid, startedAt(f.start))
	case !replace:
		return err
	case f.pid == 0:
		return fmt.Errorf("%w: port %d is in use and no re-psn pidfile names its holder (%v); not stopping anything", err, port, why)
	case why != nil:
		return fmt.Errorf("%w: port %d is in use by pid %d (started %s), which is not a re-psn instance of this user: %v",
			err, port, f.pid, startedAt(f.start), why)
	}
	logger.Info("replacing the earlier instance", "pid", f.pid, "port", port)
	if err := terminate(f.pid); err != nil {
		return fmt.Errorf("port %d: could not signal the earlier re-psn (pid %d): %w", port, f.pid, err)
	}
	for deadline := time.Now().Add(replaceWait); time.Now().Before(deadline); {
		time.Sleep(replacePoll)
		if err = listen(); err == nil || !addrInUse(err) {
			return err
		}
	}
	return fmt.Errorf("%w: port %d is in use by an earlier re-psn (pid %d, started %s), which did not stop in %v",
		err, port, f.pid, startedAt(f.start), replaceWait)
}
