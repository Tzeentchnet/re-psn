package main

import (
	"net"
	"testing"
)

func availableTestPair(t *testing.T) (net.Listener, net.PacketConn) {
	t.Helper()
	ln, pc, err := listenTestTCPUDP(randomTestPort, 100)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ln.Close(); pc.Close() })
	return ln, pc
}

func TestTCPUDPListenRetriesOccupiedProtocols(t *testing.T) {
	for _, busy := range []string{"tcp", "udp"} {
		t.Run(busy, func(t *testing.T) {
			blockedTCP, blockedUDP := availableTestPair(t)
			blockedAddr := blockedTCP.Addr().String()
			blockedPort := portOf(blockedAddr)
			if busy == "tcp" {
				blockedUDP.Close()
			} else {
				blockedTCP.Close()
			}
			freeTCP, freeUDP := availableTestPair(t)
			freePort := portOf(freeTCP.Addr().String())
			freeTCP.Close()
			freeUDP.Close()
			calls := 0
			ln, pc, err := listenTestTCPUDP(func() int {
				calls++
				if calls == 1 {
					return blockedPort
				}
				return freePort
			}, 2)
			if err != nil {
				t.Fatal(err)
			}
			defer ln.Close()
			defer pc.Close()
			if calls != 2 || portOf(ln.Addr().String()) != freePort || ln.Addr().String() != pc.LocalAddr().String() {
				t.Fatalf("calls %d, TCP %s, UDP %s", calls, ln.Addr(), pc.LocalAddr())
			}
			if busy == "udp" {
				// The first attempt acquired TCP before UDP failed; that TCP
				// descriptor must be gone even though UDP is still occupied.
				probe, err := net.Listen("tcp", blockedAddr)
				if err != nil {
					t.Fatalf("failed candidate kept TCP bound: %v", err)
				}
				probe.Close()
			}
		})
	}
}

func TestTCPUDPListenExhaustionReleasesPartialPair(t *testing.T) {
	blockedTCP, blockedUDP := availableTestPair(t)
	addr := blockedTCP.Addr().String()
	blockedTCP.Close()
	calls := 0
	ln, pc, err := listenTestTCPUDP(func() int {
		calls++
		return portOf(addr)
	}, 3)
	if err == nil || ln != nil || pc != nil || calls != 3 {
		t.Fatalf("exhaustion: TCP %v, UDP %v, calls %d, error %v", ln, pc, calls, err)
	}
	probe, err := net.Listen("tcp", addr)
	if err != nil {
		t.Fatalf("exhausted attempts kept TCP bound: %v", err)
	}
	probe.Close()
	blockedUDP.Close()
}
