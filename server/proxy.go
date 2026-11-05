package main

// Running behind a reverse proxy. Off by default. A peer whose address is in
// -trusted-proxies may tell the server who the real client is: with a PROXY
// protocol header on the game port (-proxy-protocol), and with forwarded
// headers on the status page. Everyone else is taken at their socket
// address, and a header or forwarded address from them is ignored.
// UDP is not proxied; the probe keeps using the packet's source address.

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	"strings"
	"time"
)

const proxyHeaderTimeout = 2 * time.Second

// maxProxyV2 caps the v2 payload: addresses for IPv4 and IPv6 are 12 and 36
// bytes, the rest is TLVs we skip. 536 is the v2 spec's own limit for them.
const maxProxyV2 = 536

var proxyV2Sig = []byte("\r\n\r\n\x00\r\nQUIT\n")

// parseTrusted reads a comma-separated list of CIDRs or single addresses.
func parseTrusted(list string) ([]netip.Prefix, error) {
	var out []netip.Prefix
	for _, f := range strings.Split(list, ",") {
		f = strings.TrimSpace(f)
		if f == "" {
			continue
		}
		if p, err := netip.ParsePrefix(f); err == nil {
			out = append(out, p.Masked())
			continue
		}
		a, err := netip.ParseAddr(f)
		if err != nil {
			return nil, fmt.Errorf("trusted proxies: %q is not an address or CIDR", f)
		}
		out = append(out, netip.PrefixFrom(a, a.BitLen()))
	}
	return out, nil
}

func (s *Server) trusted(a netip.Addr) bool {
	a = a.Unmap()
	for _, p := range s.trustedProxies {
		if p.Contains(a) {
			return true
		}
	}
	return false
}

// connAddr is the address of the other end of conn.
func connAddr(conn net.Conn) netip.Addr {
	if a, ok := conn.RemoteAddr().(*net.TCPAddr); ok {
		if ip, ok := netip.AddrFromSlice(a.IP); ok {
			return ip.Unmap()
		}
	}
	return netip.Addr{}
}

// readProxyHeader reads one PROXY protocol v1 or v2 header from r, no further
// than its end (always 16 bytes plus the declared length for v2, TLVs skipped). ok is false for a valid header that
// carries no client (a v2 LOCAL command, as health checks send, v1 UNKNOWN, an UNSPEC or UNIX family). admit then keeps the proxy's own address for that connection; it is
// only an error to send no valid header at all.
func readProxyHeader(r io.Reader) (addr netip.Addr, ok bool, err error) {
	head := make([]byte, 16)
	if _, err = io.ReadFull(r, head[:5]); err != nil {
		return
	}
	if bytes.Equal(head[:5], []byte("PROXY")) {
		return readProxyV1(r)
	}
	if _, err = io.ReadFull(r, head[5:]); err != nil {
		return
	}
	if !bytes.Equal(head[:12], proxyV2Sig) {
		return addr, false, errors.New("not a PROXY header")
	}
	ver, cmd := head[12]>>4, head[12]&0x0f
	n := int(binary.BigEndian.Uint16(head[14:16]))
	if ver != 2 || cmd > 1 || n > maxProxyV2 {
		return addr, false, errors.New("bad PROXY v2 header")
	}
	body := make([]byte, n)
	if _, err = io.ReadFull(r, body); err != nil {
		return
	}
	if cmd == 0 { // LOCAL: the sender's own check; the family and transport bytes are to be ignored, the length is not
		return addr, false, nil
	}
	fam, transport := head[13]>>4, head[13]&0x0f
	if fam > 3 || transport > 2 { // unassigned values
		return addr, false, errors.New("bad PROXY v2 family or transport")
	}
	if fam == 0 || fam == 3 { // UNSPEC, or UNIX: no client address to take
		return addr, false, nil
	}
	if transport != 1 { // this is the TCP game port: only a stream is meaningful (a datagram or unspecified transport is not)
		return addr, false, errors.New("PROXY v2 transport is not a stream")
	}
	switch fam {
	case 1: // IPv4: src dst sport dport
		if n < 12 {
			return addr, false, errors.New("short PROXY v2 address block")
		}
		return netip.AddrFrom4([4]byte(body[:4])), true, nil
	default: // 2: IPv6
		if n < 36 {
			return addr, false, errors.New("short PROXY v2 address block")
		}
		return netip.AddrFrom16([16]byte(body[:16])).Unmap(), true, nil
	}
}

// readProxyV1 reads the rest of "PROXY <proto> <src> <dst> <sport> <dport>\r\n"
// after the five bytes already read. The whole line is at most 107 bytes.
func readProxyV1(r io.Reader) (addr netip.Addr, ok bool, err error) {
	line := []byte("PROXY")
	one := make([]byte, 1)
	for len(line) < 107 {
		if _, err = io.ReadFull(r, one); err != nil {
			return
		}
		line = append(line, one[0])
		if one[0] == '\n' {
			break
		}
	}
	if !bytes.HasSuffix(line, []byte("\r\n")) {
		return addr, false, errors.New("bad PROXY v1 line")
	}
	f := strings.Fields(string(line))
	if len(f) < 2 {
		return addr, false, errors.New("bad PROXY v1 line")
	}
	if f[1] == "UNKNOWN" {
		return addr, false, nil
	}
	if len(f) != 6 || (f[1] != "TCP4" && f[1] != "TCP6") {
		return addr, false, errors.New("bad PROXY v1 line")
	}
	a, err := netip.ParseAddr(f[2])
	if err != nil || (f[1] == "TCP4") != a.Is4() {
		return netip.Addr{}, false, errors.New("bad PROXY v1 address")
	}
	return a.Unmap(), true, nil
}

// clientAddr is the address a status page request counts as coming from, for
// the log: the socket peer, unless that peer is a trusted proxy that names
// another. CF-Connecting-IP wins; else X-Forwarded-For is walked from the
// right, past other trusted proxies.
func (s *Server) clientAddr(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	peer, err := netip.ParseAddr(host)
	if err != nil || !s.trusted(peer) {
		return host
	}
	if a, err := netip.ParseAddr(strings.TrimSpace(r.Header.Get("CF-Connecting-IP"))); err == nil {
		return a.Unmap().String()
	}
	hops := strings.Split(strings.Join(r.Header.Values("X-Forwarded-For"), ","), ",")
	for i := len(hops) - 1; i >= 0; i-- {
		a, err := netip.ParseAddr(strings.TrimSpace(hops[i]))
		if err != nil {
			break // junk: stop at the last address we can believe
		}
		if !s.trusted(a) {
			return a.Unmap().String()
		}
	}
	return host
}
