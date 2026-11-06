package main

import (
	"errors"
	"os"
)

// privateDir: the Windows temp directory is already per user.
func privateDir(os.FileInfo) error { return nil }

func alive(int) bool { return false }

func terminate(int) error { return errors.New("not supported on Windows") }
