package main

// Every flag can also be set from the environment as RE_PSN_<FLAG>, with the
// dashes as underscores (-max-rooms is RE_PSN_MAX_ROOMS). The old name,
// PSNR_<FLAG>, still works when the new one is not set. A flag on the command
// line wins over both. A variable that does not parse stops the server at
// start instead of being ignored.

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
)

func envName(flag string) string {
	return "RE_PSN_" + strings.ToUpper(strings.ReplaceAll(flag, "-", "_"))
}

// oldEnvName is the name before the rename, still read as an alias.
func oldEnvName(flag string) string {
	return "PSNR_" + strings.ToUpper(strings.ReplaceAll(flag, "-", "_"))
}

// envNotice is a warning about an old variable name. It holds names only,
// never a value: some of these variables carry tokens.
type envNotice struct{ msg, old, new string }

// envNotices collects the notices while the flags are read, before the logger
// exists; logEnvNotices writes them once it does.
var envNotices []envNotice

// lookupEnv returns the variable for flag and the name it came from:
// RE_PSN_<FLAG> if set (even empty), else PSNR_<FLAG>.
func lookupEnv(flag string) (string, string, bool) {
	name, old := envName(flag), oldEnvName(flag)
	v, ok := os.LookupEnv(name)
	ov, oldOK := os.LookupEnv(old)
	switch {
	case ok && oldOK && v != ov:
		envNotices = append(envNotices, envNotice{"both the new and the old variable are set and differ; using the new one", old, name})
	case !ok && oldOK:
		envNotices = append(envNotices, envNotice{"old variable name; it still works, rename it when convenient", old, name})
		return ov, old, true
	}
	return v, name, ok
}

func logEnvNotices(l *slog.Logger) {
	for _, n := range envNotices {
		l.Warn(n.msg, "old", n.old, "new", n.new)
	}
	envNotices = nil
}

// envStr returns the variable for flag, or def when it is not set.
func envStr(flag, def string) string {
	if v, _, ok := lookupEnv(flag); ok {
		return v
	}
	return def
}

func envInt(flag string, def int) (int, error) {
	v, name, ok := lookupEnv(flag)
	if !ok {
		return def, nil
	}
	n, err := strconv.Atoi(strings.TrimSpace(v))
	if err != nil {
		return 0, fmt.Errorf("%s: not a number", name)
	}
	return n, nil
}

func envBool(flag string, def bool) (bool, error) {
	v, name, ok := lookupEnv(flag)
	if !ok {
		return def, nil
	}
	b, err := strconv.ParseBool(strings.TrimSpace(v))
	if err != nil {
		return false, fmt.Errorf("%s: not true or false", name)
	}
	return b, nil
}

// listenAddrs picks the TCP and UDP addresses: -tcp-addr and -udp-addr when
// set, else -addr for both.
func listenAddrs(addr, tcp, udp string) (string, string) {
	if tcp == "" {
		tcp = addr
	}
	if udp == "" {
		udp = addr
	}
	return tcp, udp
}
