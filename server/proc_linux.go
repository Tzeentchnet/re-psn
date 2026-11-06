package main

import (
	"os"
	"strconv"
	"strings"
	"syscall"
	"time"
)

const replaceSupported = true

// procStart returns field 22 of /proc/<pid>/stat, the start time in clock
// ticks since boot, or "" when it cannot be read. With the pid it names one
// process: a reused pid has another start time.
func procStart(pid int) string {
	b, err := os.ReadFile("/proc/" + strconv.Itoa(pid) + "/stat")
	if err != nil {
		return ""
	}
	return statStart(string(b))
}

// statStart picks field 22 out of a /proc/<pid>/stat line. Field 2, the
// command name in parentheses, can hold spaces and parentheses itself, so
// counting starts after the last ')'.
func statStart(stat string) string {
	i := strings.LastIndexByte(stat, ')')
	if i < 0 {
		return ""
	}
	f := strings.Fields(stat[i+1:]) // f[0] is field 3
	if len(f) < 20 {
		return ""
	}
	return f[19]
}

func procInfo(pid int) procFacts {
	p := procFacts{alive: alive(pid), uid: -1}
	dir := "/proc/" + strconv.Itoa(pid)
	if fi, err := os.Stat(dir); err == nil {
		if st, ok := fi.Sys().(*syscall.Stat_t); ok {
			p.uid = int(st.Uid)
		}
	}
	p.start = procStart(pid)
	p.exe, _ = os.Readlink(dir + "/exe")
	return p
}

// startedAt turns a start time in ticks into a wall clock time, for messages.
// The kernel reports it in USER_HZ, which is 100 on Linux.
func startedAt(start string) string {
	ticks, err := strconv.ParseInt(start, 10, 64)
	b, berr := os.ReadFile("/proc/stat")
	if err != nil || berr != nil {
		return "at an unknown time"
	}
	for _, line := range strings.Split(string(b), "\n") {
		if v, ok := strings.CutPrefix(line, "btime "); ok {
			boot, err := strconv.ParseInt(strings.TrimSpace(v), 10, 64)
			if err != nil {
				break
			}
			return time.Unix(boot+ticks/100, 0).Format(time.DateTime)
		}
	}
	return "at an unknown time"
}
