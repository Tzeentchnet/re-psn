package main

// Every flag can also be set from the environment as PSNR_<FLAG>, with the
// dashes as underscores (-max-rooms is PSNR_MAX_ROOMS). A flag on the command
// line wins over the variable. A variable that does not parse stops the
// server at start instead of being ignored.

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

func envName(flag string) string {
	return "PSNR_" + strings.ToUpper(strings.ReplaceAll(flag, "-", "_"))
}

// envStr returns the variable for flag, or def when it is not set.
func envStr(flag, def string) string {
	if v, ok := os.LookupEnv(envName(flag)); ok {
		return v
	}
	return def
}

func envInt(flag string, def int) (int, error) {
	v, ok := os.LookupEnv(envName(flag))
	if !ok {
		return def, nil
	}
	n, err := strconv.Atoi(strings.TrimSpace(v))
	if err != nil {
		return 0, fmt.Errorf("%s: not a number", envName(flag))
	}
	return n, nil
}

func envBool(flag string, def bool) (bool, error) {
	v, ok := os.LookupEnv(envName(flag))
	if !ok {
		return def, nil
	}
	b, err := strconv.ParseBool(strings.TrimSpace(v))
	if err != nil {
		return false, fmt.Errorf("%s: not true or false", envName(flag))
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
