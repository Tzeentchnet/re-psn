//go:build unix

package main

import (
	"errors"
	"os"
	"syscall"
)

// privateDir refuses a pidfile directory that another user owns or can open.
func privateDir(fi os.FileInfo) error {
	st, ok := fi.Sys().(*syscall.Stat_t)
	if !ok || int(st.Uid) != os.Getuid() {
		return errors.New("is not owned by this user")
	}
	if fi.Mode().Perm()&0o077 != 0 {
		return errors.New("is open to other users")
	}
	return nil
}

// alive says whether pid exists (EPERM: it does, as someone else's).
func alive(pid int) bool {
	err := syscall.Kill(pid, 0)
	return err == nil || errors.Is(err, syscall.EPERM)
}

func terminate(pid int) error { return syscall.Kill(pid, syscall.SIGTERM) }
