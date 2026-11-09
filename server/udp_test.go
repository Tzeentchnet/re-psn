package main

import (
	"encoding/binary"
	"fmt"
	"math/rand"
	"net"
	"strconv"
	"strings"
	"testing"
	"time"
)

// startTCPUDP serves TCP and UDP on the same port, as main does.
func startTCPUDP(t *testing.T) (*Server, string) { return startTCPUDPWith(t, nil) }

// startTCPUDPWith is startTCPUDP with setup applied before serving starts.
func startTCPUDPWith(t *testing.T, setup func(*Server)) (*Server, string) {
	s := NewServer(10, false)
	if setup != nil {
		setup(s)
	}
	ln, pc, err := listenTestTCPUDP(randomTestPort, 100)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ln.Close(); pc.Close() })
	go s.Serve(ln)
	go s.ServeUDP(pc)
	return s, ln.Addr().String()
}

func randomTestPort() int { return 1024 + rand.Intn(65536-1024) }

// Choose independent candidates across the nonprivileged range. Asking either
// protocol for port zero can keep returning adjacent ports inside a range that
// Windows excludes for the other protocol. Bind both explicitly and release a
// partial pair before trying the next candidate.
func listenTestTCPUDP(nextPort func() int, attempts int) (net.Listener, net.PacketConn, error) {
	var last error
	for try := 0; try < attempts; try++ {
		addr := net.JoinHostPort("127.0.0.1", strconv.Itoa(nextPort()))
		ln, err := net.Listen("tcp", addr)
		if err != nil {
			last = err
			continue
		}
		pc, err := net.ListenPacket("udp", addr)
		if err == nil {
			return ln, pc, nil
		}
		ln.Close()
		last = err
	}
	return nil, nil, fmt.Errorf("bind test TCP and UDP after %d candidates: %w", attempts, last)
}

// helloToken says HELLO and returns the user id and probe token.
func helloToken(t *testing.T, c *tc, name string, port uint16) (uint32, uint32) {
	w := &writer{}
	w.str("NPWR00001_00", 12)
	w.str(name, 16)
	w.u16(port)
	rt, r := c.call(msgHello, w.b)
	if rt != msgHelloAck {
		t.Fatalf("hello: 0x%02X", rt)
	}
	id := r.u32()
	r.take(4)
	return id, r.u32()
}

func probe(t *testing.T, addr string, id, token uint32, local [4]byte) (*net.UDPConn, []byte) {
	return probeVia(t, nil, addr, id, token, local)
}

// probeVia is probe from a socket bound to laddr (nil: any).
func probeVia(t *testing.T, laddr *net.UDPAddr, addr string, id, token uint32, local [4]byte) (*net.UDPConn, []byte) {
	ua, _ := net.ResolveUDPAddr("udp", addr)
	u, err := net.DialUDP("udp", laddr, ua)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { u.Close() })
	p := append([]byte(udpMagic), udpProbe)
	p = binary.BigEndian.AppendUint32(p, id)
	p = binary.BigEndian.AppendUint32(p, token)
	p = append(p, local[:]...)
	u.Write(p)
	u.SetReadDeadline(time.Now().Add(time.Second))
	buf := make([]byte, 64)
	n, err := u.Read(buf)
	if err != nil {
		return u, nil
	}
	return u, buf[:n]
}

// A probe with the right token is answered with the address it came from, and
// the server keeps it; a wrong token is ignored.
func TestProbeRecordsPublicEndpoint(t *testing.T) {
	s, addr := startTCPUDP(t)
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	c := &tc{t: t, conn: conn}
	id, token := helloToken(t, c, "homer", 3658)

	if _, reply := probe(t, addr, id, token+1, [4]byte{192, 168, 1, 5}); reply != nil {
		t.Fatal("a probe with the wrong token was answered")
	}
	u, reply := probe(t, addr, id, token, [4]byte{192, 168, 1, 5})
	if len(reply) != 11 || string(reply[:4]) != udpMagic || reply[4] != udpProbeReply {
		t.Fatalf("probe reply %x", reply)
	}
	port := uint16(u.LocalAddr().(*net.UDPAddr).Port)
	if got := binary.BigEndian.Uint16(reply[9:]); got != port {
		t.Fatalf("reply says port %d, socket is on %d", got, port)
	}
	s.mu.Lock()
	cl := s.clients[id]
	s.mu.Unlock()
	if cl.udpPort != port || cl.udpIP != [4]byte{127, 0, 0, 1} || cl.localIP != [4]byte{192, 168, 1, 5} {
		t.Fatalf("recorded %v:%d local %v", cl.udpIP, cl.udpPort, cl.localIP)
	}
}

// Which address a member is given depends on who is looking.
func TestEndpointPerViewer(t *testing.T) {
	home := &client{ip: [4]byte{203, 0, 113, 7}, p2pPort: 3658,
		udpIP: [4]byte{203, 0, 113, 7}, udpPort: 40001, localIP: [4]byte{192, 168, 1, 5}}
	sameHouse := &client{ip: [4]byte{203, 0, 113, 7}}
	elsewhere := &client{ip: [4]byte{198, 51, 100, 9}}

	if ip, port := endpoint(home, elsewhere); ip != home.udpIP || port != 40001 {
		t.Fatalf("from another network: %v:%d, want the public mapping", ip, port)
	}
	if ip, port := endpoint(home, sameHouse); ip != home.localIP || port != 3658 {
		t.Fatalf("from the same network: %v:%d, want the LAN address", ip, port)
	}
	unprobed := &client{ip: [4]byte{127, 0, 0, 1}, p2pPort: 3659}
	if ip, port := endpoint(unprobed, elsewhere); ip != unprobed.ip || port != 3659 {
		t.Fatalf("no probe: %v:%d, want what TCP shows", ip, port)
	}
}

// otherLoopback is 127.0.0.2, a source address other than the 127.0.0.1 the
// test's TCP connections come from. The test is skipped where it can't be bound.
func otherLoopback(t *testing.T) *net.UDPAddr {
	a := &net.UDPAddr{IP: net.IPv4(127, 0, 0, 2)}
	u, err := net.ListenUDP("udp", a)
	if err != nil {
		t.Skipf("can't bind 127.0.0.2: %v", err)
	}
	u.Close()
	return a
}

func helloOn(t *testing.T, addr, name string) (uint32, uint32) {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	return helloToken(t, &tc{t: t, conn: conn}, name, 3658)
}

func recorded(s *Server, id uint32) (ip [4]byte, port uint16) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c := s.clients[id]
	return c.udpIP, c.udpPort
}

// A probe with the right token but from another address than the player's TCP
// connection is not answered and moves nothing; the same player's probe from
// its own address is then accepted. The refusal is logged once.
func TestProbeFromOtherAddressRefused(t *testing.T) {
	other := otherLoopback(t)
	buf := &syncBuf{}
	s, addr := startTCPUDPWith(t, func(s *Server) { s.log = newLogger(buf, levelFor(true), "text") })
	id, token := helloOn(t, addr, "homer")

	if _, reply := probeVia(t, other, addr, id, token, [4]byte{192, 168, 1, 5}); reply != nil {
		t.Fatalf("a probe from 127.0.0.2 was answered: %x", reply)
	}
	probeVia(t, other, addr, id, token, [4]byte{192, 168, 1, 5}) // a second one, not logged again
	if ip, port := recorded(s, id); port != 0 || ip != ([4]byte{}) {
		t.Fatalf("refused probe recorded %v:%d", ip, port)
	}
	if n := strings.Count(buf.String(), `msg="UDP probe refused"`); n != 1 {
		t.Fatalf("refusal logged %d times, want once:\n%s", n, buf.String())
	}

	// Control: the same player, same token, from its TCP address.
	u, reply := probe(t, addr, id, token, [4]byte{192, 168, 1, 5})
	if len(reply) != 11 || reply[4] != udpProbeReply {
		t.Fatalf("probe from the TCP address: reply %x", reply)
	}
	if ip, port := recorded(s, id); ip != [4]byte{127, 0, 0, 1} || port != uint16(u.LocalAddr().(*net.UDPAddr).Port) {
		t.Fatalf("recorded %v:%d", ip, port)
	}
}

// With -udp-any-source the same mismatched probe is accepted.
func TestProbeAnySource(t *testing.T) {
	other := otherLoopback(t)
	s, addr := startTCPUDPWith(t, func(s *Server) { s.udpAnySource = true })
	id, token := helloOn(t, addr, "homer")

	u, reply := probeVia(t, other, addr, id, token, [4]byte{192, 168, 1, 5})
	if len(reply) != 11 || reply[4] != udpProbeReply {
		t.Fatalf("reply %x", reply)
	}
	if string(reply[5:9]) != string([]byte{127, 0, 0, 2}) {
		t.Fatalf("reply names %v, want 127.0.0.2", reply[5:9])
	}
	if ip, port := recorded(s, id); ip != [4]byte{127, 0, 0, 2} || port != uint16(u.LocalAddr().(*net.UDPAddr).Port) {
		t.Fatalf("recorded %v:%d", ip, port)
	}
	if _, reply := probeVia(t, other, addr, id, token+1, [4]byte{}); reply != nil {
		t.Fatal("the token is still checked with -udp-any-source")
	}
}

func TestUsableLocalIP(t *testing.T) {
	src := [4]byte{203, 0, 113, 7}
	for _, c := range []struct {
		local, want [4]byte
	}{
		{[4]byte{10, 1, 2, 3}, [4]byte{10, 1, 2, 3}},
		{[4]byte{172, 16, 0, 1}, [4]byte{172, 16, 0, 1}},
		{[4]byte{172, 31, 255, 254}, [4]byte{172, 31, 255, 254}},
		{[4]byte{172, 32, 0, 1}, [4]byte{}}, // just past 172.16/12
		{[4]byte{172, 15, 0, 1}, [4]byte{}},
		{[4]byte{192, 168, 1, 5}, [4]byte{192, 168, 1, 5}},
		{[4]byte{100, 64, 0, 1}, [4]byte{100, 64, 0, 1}}, // carrier-grade NAT
		{[4]byte{100, 127, 255, 1}, [4]byte{100, 127, 255, 1}},
		{[4]byte{100, 128, 0, 1}, [4]byte{}},
		{[4]byte{169, 254, 3, 4}, [4]byte{169, 254, 3, 4}},
		{[4]byte{198, 51, 100, 9}, [4]byte{}}, // someone else's public address
		{[4]byte{8, 8, 8, 8}, [4]byte{}},
		{src, src},                         // a host with a public address and no router
		{[4]byte{}, [4]byte{}},             // not given
		{[4]byte{127, 0, 0, 1}, [4]byte{}}, // loopback is refused unless it is the source
		{[4]byte{255, 255, 255, 255}, [4]byte{}},
	} {
		if got := usableLocalIP(c.local, src); got != c.want {
			t.Errorf("usableLocalIP(%v, %v) = %v, want %v", c.local, src, got, c.want)
		}
	}
	lo := [4]byte{127, 0, 0, 1}
	if got := usableLocalIP(lo, lo); got != lo {
		t.Errorf("loopback equal to the source: %v", got)
	}
}

// A refused local address is stored as none, so a same-network viewer falls
// back to the TCP address instead of being sent somewhere else.
func TestProbeDropsPublicLocalIP(t *testing.T) {
	s, addr := startTCPUDP(t)
	id, token := helloOn(t, addr, "homer")
	if _, reply := probe(t, addr, id, token, [4]byte{198, 51, 100, 9}); reply == nil {
		t.Fatal("probe not answered")
	}
	s.mu.Lock()
	local := s.clients[id].localIP
	s.mu.Unlock()
	if local != ([4]byte{}) {
		t.Fatalf("kept local address %v", local)
	}
}
