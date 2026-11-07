package main

// UDP on the same port as TCP, for players behind NAT. A player's router
// gives their P2P socket a public port the server can't see over TCP, so the
// client's P2P socket sends a PROBE here and the server records where it came
// from: that public endpoint is what other players on other networks are
// given. Probes carry the token from HELLO_ACK, so nobody can move another
// player's endpoint, and (unless -udp-any-source) must come from the same IP
// address as the player's TCP connection, so a guessed or stolen token, or a
// spoofed source, cannot point other players' traffic at a third party. Peers
// also send PUNCH packets straight to each other to open their routers;
// receivers drop anything that starts with the magic.
//
// All packets: "PSNR" | type u8 | ...
//   PROBE        client -> server  user_id u32 | token u32 | local_ip [4]
//   PROBE_REPLY  server -> client  public_ip [4] | public_port u16
//   PUNCH        peer -> peer      user_id u32

import (
	"encoding/binary"
	"net"
)

const (
	udpMagic      = "PSNR"
	udpProbe      = 0x01
	udpPunch      = 0x02
	udpProbeReply = 0x81
)

func (s *Server) ServeUDP(pc net.PacketConn) error {
	s.lifeMu.Lock()
	if s.stopping || s.udp != nil {
		s.lifeMu.Unlock()
		pc.Close()
		return net.ErrClosed
	}
	s.udp, s.udpReady = pc, true
	s.handlers.Add(1)
	s.lifeMu.Unlock()
	defer func() {
		s.lifeMu.Lock()
		s.udpReady = false
		s.lifeMu.Unlock()
		s.handlers.Done()
	}()
	buf := make([]byte, 65536)
	for {
		n, addr, err := pc.ReadFrom(buf)
		if err != nil {
			return err
		}
		ua, ok := addr.(*net.UDPAddr)
		if ok && n >= 5 && string(buf[:4]) == udpMagic && buf[4] == udpRelay {
			s.relayUDP(pc, buf[:n], ua)
			continue
		}
		if !ok || n < 17 || string(buf[:4]) != udpMagic || buf[4] != udpProbe {
			continue
		}
		ip := ua.IP.To4()
		if ip == nil {
			continue
		}
		id, token := binary.BigEndian.Uint32(buf[5:]), binary.BigEndian.Uint32(buf[9:])

		local := usableLocalIP([4]byte(buf[13:17]), [4]byte(ip))
		s.mu.Lock()
		c := s.clients[id]
		ok = c != nil && c.commID != "" && c.token == token
		mismatch := ok && !s.udpAnySource && c.ip != [4]byte(ip)
		if ok && !mismatch {
			copy(c.udpIP[:], ip)
			c.udpPort = uint16(ua.Port)
			c.localIP = local
		}
		warn := mismatch && !c.udpWarned
		if warn {
			c.udpWarned = true
		}
		s.mu.Unlock()
		if mismatch {
			// Not this player's address: a spoofed source, a stolen token, or a network whose UDP leaves from another address (-udp-any-source).
			if warn {
				s.log.Warn("UDP probe refused", "client", id, "reason", "source address differs from the TCP connection's", "udp", ip.String(), "tcp", ipString(c.ip), "opt_out", "-udp-any-source")
			}
			continue
		}
		if !ok {
			continue
		}
		s.log.Debug("P2P socket seen", "client", id, "addr", ua.String(), "local", ipString(local))

		reply := append([]byte(udpMagic), udpProbeReply)
		reply = append(reply, ip...)
		reply = binary.BigEndian.AppendUint16(reply, uint16(ua.Port))
		pc.WriteTo(reply, addr)
	}
}

// usableLocalIP is the LAN address a player may announce in a probe. It is handed to players on the same public address, who then send UDP to it, so
// it has to be an address that is local to a network: private, link-local or carrier-grade NAT space. It may also equal the probe's own source
// address (a host with a public address and no router). Anything else, such as another public address, is dropped (treated as not given).
func usableLocalIP(local, source [4]byte) [4]byte {
	if local == source {
		return local
	}
	switch {
	case local[0] == 10,
		local[0] == 172 && local[1]&0xf0 == 16,
		local[0] == 192 && local[1] == 168,
		local[0] == 100 && local[1]&0xc0 == 64,
		local[0] == 169 && local[1] == 254:
		return local
	}
	return [4]byte{}
}

// endpoint is where viewer should send m's P2P traffic. From another network
// (a different public address), m's probed public endpoint; on the same
// network, m's local address, since most routers can't loop traffic from
// their LAN back to their own public address. Without a probe it falls back
// to what TCP shows: m's address and the P2P port m declared.
func endpoint(m, viewer *client) ([4]byte, uint16) {
	if m.udpPort != 0 && m.ip != viewer.ip {
		return m.udpIP, m.udpPort
	}
	if m.localIP != ([4]byte{}) {
		return m.localIP, m.p2pPort
	}
	return m.ip, m.p2pPort
}
