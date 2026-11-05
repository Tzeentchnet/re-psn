package main

import (
	"bytes"
	"net"
	"net/http"
	"net/netip"
	"testing"
	"time"
)

func v2(cmd, fam byte, body []byte) []byte {
	b := append([]byte{}, proxyV2Sig...)
	b = append(b, 0x20|cmd, fam, byte(len(body)>>8), byte(len(body)))
	return append(b, body...)
}

func v2tcp4(src [4]byte) []byte {
	return v2(1, 0x11, append(append(src[:], 10, 0, 0, 1), 0x30, 0x39, 0x8d, 0xa4))
}

func TestReadProxyHeader(t *testing.T) {
	v6 := append(netip.MustParseAddr("2001:db8::7").AsSlice(), make([]byte, 16+4)...)
	for _, c := range []struct {
		name string
		in   []byte
		want string // "" with ok false
		ok   bool
		err  bool
	}{
		{"v1 tcp4", []byte("PROXY TCP4 203.0.113.9 10.0.0.1 51000 36100\r\n"), "203.0.113.9", true, false},
		{"v1 tcp6", []byte("PROXY TCP6 2001:db8::1 2001:db8::2 51000 36100\r\n"), "2001:db8::1", true, false},
		{"v1 unknown", []byte("PROXY UNKNOWN\r\n"), "", false, false},
		{"v1 family mismatch", []byte("PROXY TCP4 2001:db8::1 10.0.0.1 1 2\r\n"), "", false, true},
		{"v1 no crlf", []byte("PROXY TCP4 1.2.3.4 10.0.0.1 1 2\n"), "", false, true},
		{"v1 too long", append([]byte("PROXY TCP4 "), bytes.Repeat([]byte("1"), 200)...), "", false, true},
		{"v2 tcp4", v2tcp4([4]byte{198, 51, 100, 4}), "198.51.100.4", true, false},
		{"v2 tcp6", v2(1, 0x21, v6), "2001:db8::7", true, false},
		{"v2 local", v2(0, 0, nil), "", false, false},
		{"v2 unspec", v2(1, 0, nil), "", false, false},
		{"v2 short block", v2(1, 0x11, []byte{1, 2, 3}), "", false, true},
		{"v2 too big", append(v2(1, 0x11, nil)[:14], 0x10, 0x00), "", false, true},
		{"not proxy", []byte("\x00\x05\x01hello world, not a header"), "", false, true},
		{"truncated", []byte("PROXY TCP4 1.2"), "", false, true},
	} {
		a, ok, err := readProxyHeader(bytes.NewReader(c.in))
		if (err != nil) != c.err || ok != c.ok || (ok && a.String() != c.want) {
			t.Errorf("%s: got %v %v %v", c.name, a, ok, err)
		}
	}
}

// The header is read exactly: the bytes after it are the first frame.
func TestReadProxyHeaderStopsAtEnd(t *testing.T) {
	r := bytes.NewReader(append(v2tcp4([4]byte{1, 2, 3, 4}), 0xAA, 0xBB))
	if _, ok, err := readProxyHeader(r); !ok || err != nil || r.Len() != 2 {
		t.Fatalf("left %d bytes, ok %v err %v", r.Len(), ok, err)
	}
	r = bytes.NewReader([]byte("PROXY TCP4 1.2.3.4 5.6.7.8 1 2\r\nrest"))
	if _, ok, err := readProxyHeader(r); !ok || err != nil || r.Len() != 4 {
		t.Fatalf("v1 left %d bytes, ok %v err %v", r.Len(), ok, err)
	}
}

func TestParseTrusted(t *testing.T) {
	p, err := parseTrusted(" 10.0.0.0/8, 192.0.2.1 ,2001:db8::/32,")
	if err != nil || len(p) != 3 {
		t.Fatalf("%v %v", p, err)
	}
	s := &Server{trustedProxies: p}
	for a, want := range map[string]bool{"10.9.9.9": true, "192.0.2.1": true, "192.0.2.2": false, "2001:db8::5": true, "::ffff:10.1.1.1": true, "8.8.8.8": false} {
		if got := s.trusted(netip.MustParseAddr(a)); got != want {
			t.Errorf("trusted(%s) = %v", a, got)
		}
	}
	if _, err := parseTrusted("10.0.0.0/8, nonsense"); err == nil {
		t.Fatal("junk accepted")
	}
	if p, err := parseTrusted(""); err != nil || len(p) != 0 {
		t.Fatal("empty list should be empty")
	}
}

func TestClientAddr(t *testing.T) {
	trusted, _ := parseTrusted("10.0.0.0/8")
	s := &Server{trustedProxies: trusted}
	req := func(peer string, h map[string]string) *http.Request {
		r := &http.Request{RemoteAddr: peer, Header: http.Header{}}
		for k, v := range h {
			r.Header.Add(k, v)
		}
		return r
	}
	for _, c := range []struct {
		name string
		r    *http.Request
		want string
	}{
		{"untrusted peer, forwarded ignored", req("203.0.113.5:1000", map[string]string{"X-Forwarded-For": "1.1.1.1", "CF-Connecting-IP": "2.2.2.2"}), "203.0.113.5"},
		{"trusted peer, no header", req("10.0.0.2:1000", nil), "10.0.0.2"},
		{"trusted peer, CF header", req("10.0.0.2:1000", map[string]string{"CF-Connecting-IP": "198.51.100.7"}), "198.51.100.7"},
		{"trusted peer, XFF right to left", req("10.0.0.2:1000", map[string]string{"X-Forwarded-For": "6.6.6.6, 198.51.100.8, 10.0.0.9"}), "198.51.100.8"},
		{"client-supplied left entries are not believed", req("10.0.0.2:1000", map[string]string{"X-Forwarded-For": "6.6.6.6, 198.51.100.8"}), "198.51.100.8"},
		{"all hops trusted", req("10.0.0.2:1000", map[string]string{"X-Forwarded-For": "10.0.0.5, 10.0.0.6"}), "10.0.0.2"},
		{"junk entry stops the walk", req("10.0.0.2:1000", map[string]string{"X-Forwarded-For": "6.6.6.6, junk, 10.0.0.9"}), "10.0.0.2"},
	} {
		if got := s.clientAddr(c.r); got != c.want {
			t.Errorf("%s: got %s want %s", c.name, got, c.want)
		}
	}
}

// startProxied starts a server whose trusted proxies are the given list.
func startProxied(t *testing.T, trusted string, perAddr int) string {
	s := NewServer(10, false)
	var err error
	if s.trustedProxies, err = parseTrusted(trusted); err != nil {
		t.Fatal(err)
	}
	s.proxyProto, s.maxPerAddr = true, perAddr
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ln.Close() })
	go s.Serve(ln)
	return ln.Addr().String()
}

func helloAfter(t *testing.T, addr string, prefix []byte, name string) (*tc, byte, *reader) {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	conn.Write(prefix)
	c := &tc{t: t, conn: conn}
	rt, r := hello(c, "NPWR00001_00", name)
	return c, rt, r
}

func TestProxyHeaderGivesRealAddress(t *testing.T) {
	addr := startProxied(t, "127.0.0.0/8", 0)
	_, rt, r := helloAfter(t, addr, v2tcp4([4]byte{198, 51, 100, 4}), "alice")
	if rt != msgHelloAck {
		t.Fatalf("hello: 0x%02X", rt)
	}
	r.u32() // client id
	if ip := r.take(4); !bytes.Equal(ip, []byte{198, 51, 100, 4}) {
		t.Fatalf("public_ip %v, want the address in the PROXY header", ip)
	}
}

// Control for the test above: a peer that is not trusted gets its header
// ignored. Its bytes are read as a frame header (a type and a length of about
// 2.5 KB), so the HELLO sent after them is swallowed as that frame's payload
// and never answered; no address from the header is ever used.
func TestProxyHeaderFromUntrustedPeerIsJunk(t *testing.T) {
	addr := startProxied(t, "10.0.0.0/8", 0)
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	conn.Write(v2tcp4([4]byte{198, 51, 100, 4}))
	w := &writer{}
	w.str("NPWR00001_00", 12)
	w.str("alice", 16)
	w.u16(1)
	writePacket(conn, msgHello, append([]byte{0, 0, 0, 1}, w.b...))
	conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
	if rt, _, err := readPacket(conn); err == nil {
		t.Fatalf("answered a peer that is not trusted: 0x%02X", rt)
	}
}

func TestTrustedPeerWithoutHeaderIsClosed(t *testing.T) {
	addr := startProxied(t, "127.0.0.0/8", 0)
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	c := &tc{t: t, conn: conn}
	w := &writer{}
	w.str("NPWR00001_00", 12)
	w.str("alice", 16)
	w.u16(1)
	writePacket(conn, msgHello, append([]byte{0, 0, 0, 1}, w.b...))
	_ = c
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	if _, _, err := readPacket(conn); err == nil {
		t.Fatal("served a trusted peer that sent no PROXY header")
	}
}

// The per-address limit counts the address in the header, not the proxy's.
func TestLimitUsesRealAddress(t *testing.T) {
	addr := startProxied(t, "127.0.0.0/8", 1)
	a, rt, _ := helloAfter(t, addr, v2tcp4([4]byte{198, 51, 100, 4}), "alice")
	if rt != msgHelloAck {
		t.Fatalf("first: 0x%02X", rt)
	}
	_ = a
	// another real address through the same proxy is served
	if _, rt, _ := helloAfter(t, addr, v2tcp4([4]byte{198, 51, 100, 5}), "bob"); rt != msgHelloAck {
		t.Fatalf("a second real address was refused: 0x%02X", rt)
	}
	// the same real address again is over the limit of 1
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	conn.Write(v2tcp4([4]byte{198, 51, 100, 4}))
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	var one [1]byte
	if _, err := conn.Read(one[:]); err == nil {
		t.Fatal("read data from a connection that should be closed")
	} else if ne, ok := err.(net.Error); ok && ne.Timeout() {
		t.Fatal("the connection over the limit stayed open")
	}
}

func FuzzReadProxyHeader(f *testing.F) {
	f.Add([]byte("PROXY TCP4 1.2.3.4 5.6.7.8 1 2\r\n"))
	f.Add([]byte("PROXY UNKNOWN\r\n"))
	f.Add(v2tcp4([4]byte{1, 2, 3, 4}))
	f.Add(v2(0, 0, nil))
	f.Fuzz(func(t *testing.T, b []byte) {
		r := bytes.NewReader(b)
		readProxyHeader(r) // must return, never panic or read past the input
	})
}
