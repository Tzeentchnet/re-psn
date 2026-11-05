package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sp00nznet/psnr/server/store"
)

// secret stands in for a database password: it must never reach an error,
// the log or the status page.
const secret = "hunter2-s3cret"

var dsns = map[string]string{ // scheme -> a connection string holding secret
	"sqlite":     "sqlite:/var/lib/" + secret + "/re-psn.db",
	"postgres":   "postgres://re:" + secret + "@db.internal:5432/re?sslmode=require",
	"postgresql": "postgresql://re:" + secret + "@db.internal/re",
	"mysql":      "mysql://re:" + secret + "@tcp(db.internal:3306)/re",
}

// leaks reports what of the setting, past its scheme, appears in s.
func leaks(s string) string {
	for _, bad := range []string{secret, "hunter2", "db.internal", "re:", "/var/lib"} {
		if strings.Contains(s, bad) {
			return bad
		}
	}
	return ""
}

// clearDBEnv unsets the storage variables for one test.
func clearDBEnv(t *testing.T) {
	for _, n := range []string{"RE_PSN_DB", "PSNR_DB", "RE_PSN_DB_FILE", "PSNR_DB_FILE"} {
		t.Setenv(n, "") // restored after the test
		os.Unsetenv(n)
	}
	envNotices = nil
	t.Cleanup(func() { envNotices = nil })
}

func writeFile(t *testing.T, body string) string {
	p := filepath.Join(t.TempDir(), "db")
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestDBSetting(t *testing.T) {
	type flags struct{ db, file string } // command-line values; "" is not given
	cases := []struct {
		name   string
		env    map[string]string
		flags  flags
		errHas []string // "" error: none expected
	}{
		{"default", nil, flags{}, nil},
		{"off", map[string]string{"RE_PSN_DB": "off"}, flags{}, nil},
		{"memory", map[string]string{"RE_PSN_DB": "memory"}, flags{}, nil},
		{"memory flag", nil, flags{db: "memory"}, nil},
		{"alias", map[string]string{"PSNR_DB": "memory"}, flags{}, nil},
		{"unknown", map[string]string{"RE_PSN_DB": "redis://re:" + secret + "@db.internal"}, flags{}, []string{"RE_PSN_DB:", "not a known storage setting"}},
		{"empty", map[string]string{"RE_PSN_DB": ""}, flags{}, []string{"RE_PSN_DB:", "not a known storage setting"}},
		{"unknown alias", map[string]string{"PSNR_DB": secret}, flags{}, []string{"PSNR_DB:", "not a known storage setting"}},
		{"unknown flag", nil, flags{db: secret}, []string{"-db:", "not a known"}},
		{"flag wins", map[string]string{"RE_PSN_DB": dsns["mysql"]}, flags{db: "memory"}, nil},
		{"both", map[string]string{"RE_PSN_DB": "memory", "RE_PSN_DB_FILE": "/nonexistent"}, flags{}, []string{"RE_PSN_DB and RE_PSN_DB_FILE are both set"}},
		{"both flags", nil, flags{db: "memory", file: "/nonexistent"}, []string{"-db and -db-file are both set"}},
		{"file missing", map[string]string{"RE_PSN_DB_FILE": "/nonexistent/re-psn-db"}, flags{}, []string{"RE_PSN_DB_FILE:", `could not read the file "/nonexistent/re-psn-db"`}},
		{"file alias missing", map[string]string{"PSNR_DB_FILE": "/nonexistent/x"}, flags{}, []string{"PSNR_DB_FILE:", "could not read"}},
	}
	for scheme, dsn := range dsns {
		cases = append(cases,
			struct {
				name   string
				env    map[string]string
				flags  flags
				errHas []string
			}{scheme, map[string]string{"RE_PSN_DB": dsn}, flags{}, []string{"RE_PSN_DB:", "the " + scheme + " backend is not built yet"}},
			struct {
				name   string
				env    map[string]string
				flags  flags
				errHas []string
			}{scheme + " alias", map[string]string{"PSNR_DB": dsn}, flags{}, []string{"PSNR_DB:", "the " + scheme + " backend is not built yet"}},
			struct {
				name   string
				env    map[string]string
				flags  flags
				errHas []string
			}{scheme + " flag", nil, flags{db: dsn}, []string{"-db:", "the " + scheme + " backend is not built yet"}},
		)
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			clearDBEnv(t)
			for k, v := range c.env {
				t.Setenv(k, v)
			}
			st, err := openStore(lookupSetting("db", c.flags.db, c.flags.db != ""), lookupSetting("db-file", c.flags.file, c.flags.file != ""))
			if c.errHas == nil {
				if err != nil || st.Name() != "memory" {
					t.Fatalf("got %v", err)
				}
				return
			}
			if err == nil {
				t.Fatal("no error")
			}
			for _, want := range c.errHas {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("error %q lacks %q", err, want)
				}
			}
			if bad := leaks(err.Error()); bad != "" {
				t.Errorf("error %q holds %q from the setting", err, bad)
			}
		})
	}
}

func TestDBFile(t *testing.T) {
	clearDBEnv(t)
	t.Setenv("RE_PSN_DB_FILE", writeFile(t, "memory\n"))
	if st, err := openStore(lookupSetting("db", "", false), lookupSetting("db-file", "", false)); err != nil || st.Name() != "memory" {
		t.Fatalf("memory from a file: %v", err)
	}
	for scheme, dsn := range dsns {
		p := writeFile(t, dsn+"\n")
		_, err := openStore(lookupSetting("db", "", false), lookupSetting("db-file", p, true))
		if err == nil || !strings.Contains(err.Error(), "-db-file (the file's contents): the "+scheme+" backend is not built yet") {
			t.Errorf("%s from a file: %v", scheme, err)
		}
		if err != nil && leaks(err.Error()) != "" {
			t.Errorf("error %q holds the file's contents", err)
		}
	}
	// The OS words for a directory differ (Linux "is a directory", Windows "Incorrect function."),
	// so only require an error that names the variable and the path.
	dir := t.TempDir()
	_, err := openStore(lookupSetting("db", "", false), lookupSetting("db-file", dir, true))
	if err == nil || !strings.Contains(err.Error(), "RE_PSN_DB_FILE") && !strings.Contains(err.Error(), "-db-file") || !strings.Contains(err.Error(), "could not read the file") || !strings.Contains(err.Error(), fmt.Sprintf("%q", dir)) {
		t.Errorf("a directory: %v", err)
	}
	clearDBEnv(t)
	t.Setenv("PSNR_DB_FILE", writeFile(t, "off"))
	if _, err := openStore(lookupSetting("db", "", false), lookupSetting("db-file", "", false)); err != nil {
		t.Fatalf("old name: %v", err)
	}
	if len(envNotices) != 1 || envNotices[0].old != "PSNR_DB_FILE" || envNotices[0].new != "RE_PSN_DB_FILE" {
		t.Fatalf("notices %+v", envNotices)
	}
}

// TestMainProcess is not a test on its own: TestDBNotLogged runs the test
// binary again with RE_PSN_TEST_MAIN=1 to start the real main and read its log.
func TestMainProcess(t *testing.T) {
	if os.Getenv("RE_PSN_TEST_MAIN") != "1" {
		t.Skip("run by TestDBNotLogged")
	}
	os.Args = append([]string{"re-psn"}, strings.Split(os.Getenv("RE_PSN_TEST_ARGS"), "\x1f")...)
	if os.Getenv("RE_PSN_TEST_ARGS") == "" {
		os.Args = os.Args[:1]
	}
	flag.CommandLine = flag.NewFlagSet("re-psn", flag.ExitOnError)
	main()
}

// TestDBNotLogged starts the server with each connection string, from the
// variable, the old variable, the flag and a file, at debug level and in both
// log formats, and checks it stops with a message naming the scheme and
// nothing after it.
func TestDBNotLogged(t *testing.T) {
	if testing.Short() {
		t.Skip("starts the server in a subprocess")
	}
	run := func(t *testing.T, env []string, args ...string) string {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestMainProcess$")
		for _, e := range os.Environ() {
			if !strings.HasPrefix(e, "RE_PSN_") && !strings.HasPrefix(e, "PSNR_") {
				cmd.Env = append(cmd.Env, e)
			}
		}
		cmd.Env = append(cmd.Env, "RE_PSN_TEST_MAIN=1", "RE_PSN_ADDR=127.0.0.1:0", "RE_PSN_HTTP=", "RE_PSN_LOG_LEVEL=debug")
		cmd.Env = append(cmd.Env, "RE_PSN_TEST_ARGS="+strings.Join(args, "\x1f"))
		cmd.Env = append(cmd.Env, env...)
		out, err := cmd.CombinedOutput()
		var ee *exec.ExitError
		if !errors.As(err, &ee) || ee.ExitCode() != 1 {
			t.Fatalf("server did not stop with status 1: %v\n%s", err, out)
		}
		return string(out)
	}
	for scheme, dsn := range dsns {
		file := writeFile(t, dsn)
		for _, c := range []struct {
			name string
			env  []string
			args []string
		}{
			{"RE_PSN_DB", []string{"RE_PSN_DB=" + dsn}, nil},
			{"PSNR_DB", []string{"PSNR_DB=" + dsn}, nil},
			{"-db", nil, []string{"-db", dsn}},
			{"RE_PSN_DB_FILE", []string{"RE_PSN_DB_FILE=" + file}, nil},
			{"json", []string{"RE_PSN_DB=" + dsn, "RE_PSN_LOG_FORMAT=json"}, nil},
		} {
			t.Run(scheme+"/"+c.name, func(t *testing.T) {
				out := run(t, c.env, c.args...)
				if !strings.Contains(out, "the "+scheme+" backend is not built yet") {
					t.Errorf("log lacks the reason:\n%s", out)
				}
				if bad := leaks(out); bad != "" {
					t.Errorf("log holds %q from the connection string:\n%s", bad, out)
				}
			})
		}
	}
}

// failing is a store whose every call fails, as a database that is down.
type failing struct{}

var errDown = errors.New("database is down")

func (failing) Put(context.Context, store.Key, bool, store.Entry) (uint32, error) { return 0, errDown }
func (failing) Range(context.Context, store.Key, int, int) (int, []store.Entry, error) {
	return 0, nil, errDown
}
func (failing) Lookup(context.Context, store.Key, []string) (int, []store.Entry, []uint32, error) {
	return 0, nil, nil, errDown
}
func (failing) Boards(context.Context) ([]store.Board, error) { return nil, errDown }
func (failing) Name() string                                  { return "failing" }
func (failing) Close() error                                  { return nil }

// TestScoreStoreFailure: a failing store answers each score request with
// ERROR 6, logs it, and leaves the rest of the server working.
func TestScoreStoreFailure(t *testing.T) {
	s, addr, buf := logged(t)
	s.scores = failing{}
	a := dial(t, addr, "NPWR00002_00", "alice", 0)
	rec := &writer{}
	rec.u32(1)
	rec.u8(0)
	rec.u64(100)
	rec.blob([]byte("gg"))
	rng := &writer{}
	rng.u32(1)
	rng.u32(1)
	rng.u16(10)
	byID := &writer{}
	byID.u32(1)
	byID.u16(1)
	byID.str("alice", 16)
	for _, req := range []struct {
		t    byte
		body []byte
	}{{msgRecordScore, rec.b}, {msgGetRanking, rng.b}, {msgGetRankingID, byID.b}} {
		rt, r := a.call(req.t, req.body)
		if code := r.u32(); rt != msgError || code != 6 { // errStoreFailed, as docs/api.md lists it
			t.Fatalf("0x%02X: got 0x%02X code %d, want ERROR 6", req.t, rt, code)
		}
	}
	// a bad request is still a bad request, without reaching the store
	if rt, r := a.call(msgGetRanking, []byte{0, 0, 0, 1, 0, 0, 0, 0, 0, 1}); rt != msgError || r.u32() != errBadRequest {
		t.Fatal("start 0 not refused as a bad request")
	}
	// rooms still work, and so does the status page
	if rt, _ := a.call(msgCreateRoom, createBody(nil)); rt != msgRoomJoined {
		t.Fatalf("create room: 0x%02X", rt)
	}
	if st := s.snapshot(); len(st.Boards) != 0 || len(st.Rooms) != 1 {
		t.Fatalf("snapshot: %+v", st)
	}
	time.Sleep(50 * time.Millisecond)
	out := buf.String()
	for _, want := range []string{`msg="request failed"`, `request="record score"`, `request="get ranking"`, `request="get ranking by id"`, `reason="score store"`, `msg="status page"`} {
		if !strings.Contains(out, want) {
			t.Errorf("log lacks %q:\n%s", want, out)
		}
	}
}

// TestScoresOutsideRoomLock: a score request does not wait for s.mu beyond
// reading who sent it, and a store that blocks does not hold up rooms.
func TestScoresOutsideRoomLock(t *testing.T) {
	s, addr, _ := logged(t)
	gate := make(chan struct{})
	s.scores = &blocking{Store: store.NewMemory(), gate: gate}
	a := dial(t, addr, "NPWR00002_00", "alice", 0)
	b := dial(t, addr, "NPWR00002_00", "bob", 0)
	rec := &writer{}
	rec.u32(1)
	rec.u8(0)
	rec.u64(100)
	rec.blob(nil)
	done := make(chan byte)
	go func() {
		c := &tc{t: t, conn: a.conn, req: a.req}
		rt, _ := c.call(msgRecordScore, rec.b)
		done <- rt
	}()
	time.Sleep(50 * time.Millisecond) // alice's Put is now waiting on the gate
	if rt, _ := b.call(msgCreateRoom, createBody(nil)); rt != msgRoomJoined {
		t.Fatalf("create room while a score waits: 0x%02X", rt)
	}
	close(gate)
	if rt := <-done; rt != msgScoreRecorded {
		t.Fatalf("record: 0x%02X", rt)
	}
}

type blocking struct {
	store.Store
	gate chan struct{}
}

func (b *blocking) Put(ctx context.Context, k store.Key, asc bool, e store.Entry) (uint32, error) {
	<-b.gate
	return b.Store.Put(ctx, k, asc, e)
}
