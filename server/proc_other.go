//go:build !linux

package main

// Off Linux the pidfile holds the pid and the executable only, and no holder
// passes checkHolder, so nothing is ever signalled.

const replaceSupported = false

func procStart(int) string { return "" }

func procInfo(pid int) procFacts { return procFacts{alive: alive(pid), uid: -1} }

func startedAt(string) string { return "at an unknown time" }
