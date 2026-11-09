package main

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"testing"
	"time"
)

func healthResponse(s *Server, method, auth string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, "/healthz", nil)
	if auth != "" {
		req.Header.Set("Authorization", auth)
	}
	w := httptest.NewRecorder()
	s.HTTP().ServeHTTP(w, req)
	return w
}

func TestHealthAuthAndMethods(t *testing.T) {
	s := NewServer(10, false)
	for _, method := range []string{http.MethodGet, http.MethodHead} {
		w := healthResponse(s, method, "")
		if w.Code != http.StatusServiceUnavailable || w.Header().Get("Cache-Control") != "no-store" {
			t.Fatalf("unready %s: %d, %v", method, w.Code, w.Header())
		}
		if method == http.MethodHead && w.Body.Len() != 0 {
			t.Fatal("HEAD returned a body")
		}
	}
	s.lifeMu.Lock()
	s.tcpReady, s.udpReady = true, true
	s.lifeMu.Unlock()
	w := healthResponse(s, http.MethodGet, "")
	if w.Code != http.StatusOK || w.Body.String() != "ok\n" {
		t.Fatalf("ready response: %d %q", w.Code, w.Body.String())
	}
	if w = healthResponse(s, http.MethodHead, ""); w.Code != http.StatusOK || w.Body.Len() != 0 {
		t.Fatalf("ready HEAD: %d %q", w.Code, w.Body.String())
	}
	if w = healthResponse(s, http.MethodPost, ""); w.Code != http.StatusMethodNotAllowed || w.Header().Get("Allow") != "GET, HEAD" {
		t.Fatalf("POST: %d, %v", w.Code, w.Header())
	}
	s.httpToken = "fixture-health"
	for _, auth := range []string{"", "Bearer wrong"} {
		if got := healthResponse(s, http.MethodGet, auth).Code; got != http.StatusUnauthorized {
			t.Fatalf("unauthorized health: %d", got)
		}
	}
	if got := healthResponse(s, http.MethodGet, "Bearer fixture-health").Code; got != http.StatusOK {
		t.Fatalf("authorized health: %d", got)
	}
}

func healthGame(t *testing.T, s *Server) (net.Listener, net.PacketConn, <-chan error, <-chan error) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	pc, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		ln.Close()
		t.Fatal(err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		s.Shutdown(ctx)
		ln.Close()
		pc.Close()
	})
	tcpDone, udpDone := make(chan error, 1), make(chan error, 1)
	go func() { tcpDone <- s.Serve(ln) }()
	go func() { udpDone <- s.ServeUDP(pc) }()
	awaitHealth(t, s, true)
	return ln, pc, tcpDone, udpDone
}

func awaitHealth(t *testing.T, s *Server, want bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for s.ready() != want {
		if time.Now().After(deadline) {
			t.Fatalf("readiness did not become %v", want)
		}
		time.Sleep(time.Millisecond)
	}
}

func TestHealthServeLoops(t *testing.T) {
	for _, failed := range []string{"tcp", "udp"} {
		t.Run(failed, func(t *testing.T) {
			s := NewServer(10, false)
			ln, pc, tcpDone, udpDone := healthGame(t, s)
			if got := healthResponse(s, http.MethodGet, "").Code; got != http.StatusOK {
				t.Fatalf("serving: %d", got)
			}
			var done <-chan error
			if failed == "tcp" {
				ln.Close()
				done = tcpDone
			} else {
				pc.Close()
				done = udpDone
			}
			select {
			case err := <-done:
				if !errors.Is(err, net.ErrClosed) {
					t.Fatalf("serve exit: %v", err)
				}
			case <-time.After(2 * time.Second):
				t.Fatal("serve loop did not exit")
			}
			awaitHealth(t, s, false)
			if w := healthResponse(s, http.MethodGet, ""); w.Code != http.StatusServiceUnavailable || w.Body.String() != "not ready\n" {
				t.Fatalf("closed listener: %d %q", w.Code, w.Body.String())
			}
		})
	}
}

func TestShutdownClosesAcceptedConnections(t *testing.T) {
	for _, proxy := range []bool{false, true} {
		t.Run(map[bool]string{false: "pre-hello", true: "proxy-header"}[proxy], func(t *testing.T) {
			s := NewServer(10, false)
			if proxy {
				s.proxyProto = true
				s.trustedProxies = []netip.Prefix{netip.MustParsePrefix("127.0.0.1/32")}
			}
			ln, _, _, _ := healthGame(t, s)
			conn, err := net.Dial("tcp", ln.Addr().String())
			if err != nil {
				t.Fatal(err)
			}
			defer conn.Close()
			deadline := time.Now().Add(2 * time.Second)
			for {
				s.lifeMu.Lock()
				n := len(s.owned)
				s.lifeMu.Unlock()
				if n == 1 {
					break
				}
				if time.Now().After(deadline) {
					t.Fatal("connection was not accepted")
				}
				time.Sleep(time.Millisecond)
			}
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			if err := s.Shutdown(ctx); err != nil {
				t.Fatal(err)
			}
			conn.SetReadDeadline(time.Now().Add(time.Second))
			if _, err := conn.Read(make([]byte, 1)); !errors.Is(err, io.EOF) {
				t.Fatalf("connection did not close: %v", err)
			}
			if s.ready() {
				t.Fatal("ready after shutdown")
			}
			if err := s.Shutdown(ctx); err != nil {
				t.Fatalf("repeat shutdown: %v", err)
			}
		})
	}
}

func TestShutdownClosesPlayerAndRejectsRestart(t *testing.T) {
	s := NewServer(10, false)
	ln, _, _, _ := healthGame(t, s)
	c := dial(t, ln.Addr().String(), "FIXTURE", "player", 1234)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := s.Shutdown(ctx); err != nil {
		t.Fatal(err)
	}
	c.conn.SetReadDeadline(time.Now().Add(time.Second))
	if _, err := c.conn.Read(make([]byte, 1)); !errors.Is(err, io.EOF) {
		t.Fatalf("player did not disconnect: %v", err)
	}
	s.mu.Lock()
	n := len(s.clients)
	s.mu.Unlock()
	if n != 0 {
		t.Fatalf("%d clients remain after handler cleanup", n)
	}
	late, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Serve(late); !errors.Is(err, net.ErrClosed) {
		t.Fatalf("restart: %v", err)
	}
}

func TestShutdownClosesRelayLegs(t *testing.T) {
	for _, paired := range []bool{false, true} {
		t.Run(map[bool]string{false: "pending", true: "paired"}[paired], func(t *testing.T) {
			s := NewServer(10, false)
			s.relay = true
			ln, _, _, _ := healthGame(t, s)
			a, b := join(t, ln.Addr().String(), "alice"), join(t, ln.Addr().String(), "bob")
			meet(t, a, b)
			ac, err := net.Dial("tcp", ln.Addr().String())
			if err != nil {
				t.Fatal(err)
			}
			defer ac.Close()
			w := &writer{}
			w.u32(1)
			w.u32(a.id)
			w.u32(a.token)
			w.u32(b.id)
			w.u16(3658)
			if err := writePacket(ac, msgStreamConnect, w.b); err != nil {
				t.Fatal(err)
			}
			var id uint32
			b.c.conn.SetReadDeadline(time.Now().Add(2 * time.Second))
			for id == 0 {
				typ, body := b.c.read()
				if typ == msgStreamOffer {
					id = (&reader{b: body}).u32()
				}
			}
			var bc net.Conn
			if paired {
				w = &writer{}
				w.u32(b.id)
				w.u32(b.token)
				w.u32(id)
				var typ byte
				bc, typ = streamLeg(t, ln.Addr().String(), msgStreamAccept, w.b)
				if typ != msgStreamReady {
					t.Fatalf("accept: %x", typ)
				}
				ac.SetReadDeadline(time.Now().Add(2 * time.Second))
				if typ, _, err := readPacket(ac); err != nil || typ != msgStreamReady {
					t.Fatalf("connect: %x, %v", typ, err)
				}
			}
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			if err := s.Shutdown(ctx); err != nil {
				t.Fatal(err)
			}
			conns := []net.Conn{ac}
			if bc != nil {
				conns = append(conns, bc)
			}
			for _, conn := range conns {
				conn.SetReadDeadline(time.Now().Add(time.Second))
				if _, err := conn.Read(make([]byte, 1)); !errors.Is(err, io.EOF) {
					t.Fatalf("relay leg did not close: %v", err)
				}
			}
			s.lifeMu.Lock()
			n := len(s.owned)
			s.lifeMu.Unlock()
			if n != 0 {
				t.Fatalf("%d owned sockets remain after shutdown", n)
			}
		})
	}
}

// gatedAccept pauses after the kernel accepts a socket but before Serve can
// register it, exercising shutdown's late-admission boundary deterministically.
type gatedAccept struct {
	net.Listener
	accepted chan struct{}
	release  chan struct{}
}

func (g *gatedAccept) Accept() (net.Conn, error) {
	conn, err := g.Listener.Accept()
	if err == nil {
		close(g.accepted)
		<-g.release
	}
	return conn, err
}

func TestShutdownDeadlineAndLateAdmission(t *testing.T) {
	s := NewServer(10, false)
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	g := &gatedAccept{Listener: ln, accepted: make(chan struct{}), release: make(chan struct{})}
	defer func() {
		select {
		case <-g.release:
		default:
			close(g.release)
		}
	}()
	done := make(chan error, 1)
	go func() { done <- s.Serve(g) }()
	conn, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	select {
	case <-g.accepted:
	case <-time.After(2 * time.Second):
		t.Fatal("listener did not accept the connection")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	if err := s.Shutdown(ctx); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("blocked accept did not respect shutdown deadline: %v", err)
	}
	close(g.release)
	select {
	case err := <-done:
		if !errors.Is(err, net.ErrClosed) {
			t.Fatalf("late accept: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("late acceptance did not finish")
	}
	conn.SetReadDeadline(time.Now().Add(time.Second))
	if _, err := conn.Read(make([]byte, 1)); !errors.Is(err, io.EOF) {
		t.Fatalf("late connection did not close: %v", err)
	}
	s.mu.Lock()
	n := len(s.clients)
	s.mu.Unlock()
	if n != 0 || s.ready() {
		t.Fatalf("late admission: %d clients, ready %v", n, s.ready())
	}
}
