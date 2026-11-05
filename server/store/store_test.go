package store_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/sp00nznet/psnr/server/store"
	"github.com/sp00nznet/psnr/server/store/storetest"
)

func TestMemoryContract(t *testing.T) {
	storetest.Run(t, func(*testing.T) store.Store { return store.NewMemory() })
}

// secret stands in for a password in a connection string: no error may hold
// it, nor anything else after the scheme.
const secret = "hunter2-s3cret"

func TestOpen(t *testing.T) {
	for _, in := range []string{"off", "memory", " Memory\n", "OFF"} {
		s, err := store.Open(in)
		if err != nil || s.Name() != "memory" {
			t.Errorf("Open(%q): %v", in, err)
		}
	}
	for in, backend := range map[string]string{
		"sqlite:/var/lib/psnr/" + secret + ".db":         "sqlite",
		"SQLite:" + secret:                               "sqlite",
		"postgres://re:" + secret + "@db.example/re":     "postgres",
		"postgresql://re:" + secret + "@db.example/re":   "postgresql",
		"mysql://re:" + secret + "@tcp(db.example)/re":   "mysql",
		"  mysql://re:" + secret + "@db.example/re  \n ": "mysql",
	} {
		_, err := store.Open(in)
		var nb *store.NotBuiltError
		if !errors.As(err, &nb) || nb.Backend != backend {
			t.Errorf("%s: got %v, want not built", backend, err)
			continue
		}
		checkClean(t, err.Error())
	}
	for _, in := range []string{"", "on", "sqlite", "postgres:" + secret, "redis://x:" + secret + "@h", secret} {
		_, err := store.Open(in)
		if !errors.Is(err, store.ErrUnknown) {
			t.Errorf("unknown setting: got %v", err)
			continue
		}
		checkClean(t, err.Error())
	}
}

func checkClean(t *testing.T, msg string) {
	t.Helper()
	for _, bad := range []string{secret, "hunter2", "re:", "@", "example", "/var/lib"} {
		if strings.Contains(msg, bad) {
			t.Errorf("error %q holds %q from the setting", msg, bad)
		}
	}
}
