package main

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

// roomOf creates a room (max 4) as c and returns its id.
func roomOf(t *testing.T, c *tc) uint64 {
	rt, r := c.call(msgCreateRoom, createBody(nil))
	if rt != msgRoomJoined {
		t.Fatalf("create: 0x%02X", rt)
	}
	return r.u64()
}

// res is a reply: its type and the payload after the request id.
type res struct {
	rt byte
	r  *reader
}

func call(c *tc, t byte, body []byte) res { rt, r := c.call(t, body); return res{rt, r} }

func (x res) errCode(t *testing.T) uint32 {
	if x.rt != msgError {
		t.Fatalf("got 0x%02X, want an error", x.rt)
	}
	return x.r.u32()
}

// access reads a ROOM_ACCESS reply: mode and code.
func (x res) access(t *testing.T) (byte, string) {
	if x.rt != msgRoomAccess {
		t.Fatalf("got 0x%02X, want ROOM_ACCESS", x.rt)
	}
	return x.r.u8(), x.r.str(codeLen)
}

func setAccess(c *tc, room uint64, mode byte) res {
	w := &writer{}
	w.u64(room)
	w.u8(mode)
	return call(c, msgSetRoomAccess, w.b)
}

func getAccess(c *tc, room uint64) res {
	w := &writer{}
	w.u64(room)
	return call(c, msgGetRoomAccess, w.b)
}

func joinCode(c *tc, code string) res {
	w := &writer{}
	w.str(code, codeLen)
	w.blob(nil)
	return call(c, msgJoinByCode, w.b)
}

func joinID(c *tc, room uint64) res {
	w := &writer{}
	w.u64(room)
	w.blob(nil)
	return call(c, msgJoinRoom, w.b)
}

func searchTotal(c *tc) int {
	w := &writer{}
	w.u16(0)
	w.u16(10)
	_, r := c.call(msgSearchRooms, w.b)
	return int(r.u16())
}

func TestInviteOnlyRoom(t *testing.T) {
	addr := start(t)
	a := dial(t, addr, "NPWR00001_00", "alice", 3658)
	b := dial(t, addr, "NPWR00001_00", "bob", 3659)
	other := dial(t, addr, "NPWR99999_00", "carol", 3658)
	room := roomOf(t, a)

	if searchTotal(b) != 1 { // control: a public room is listed
		t.Fatal("public room not listed")
	}
	mode, code := (setAccess(a, room, accessCode)).access(t)
	if mode != accessCode || len(code) != codeLen {
		t.Fatalf("mode %d code %q", mode, code)
	}
	if again := func() string { _, c := (setAccess(a, room, accessCode)).access(t); return c }(); again != code {
		t.Fatalf("second request gave %q, want the same code %q", again, code)
	}
	if searchTotal(b) != 0 {
		t.Fatal("invite-only room is in search")
	}
	if e := (joinID(b, room)).errCode(t); e != errNotFound {
		t.Fatalf("join by room ID: error %d, want %d", e, errNotFound)
	}
	if e := (joinCode(other, code)).errCode(t); e != errNotFound {
		t.Fatalf("other title: error %d", e)
	}

	// typed sloppily: lower case, O for 0, I for 1
	typed := strings.NewReplacer("0", "O", "1", "I").Replace(strings.ToLower(code))
	if x := joinCode(b, typed); x.rt != msgRoomJoined {
		t.Fatalf("join by code %q: 0x%02X", typed, x.rt)
	}
	a.next(msgMemberJoined) // the owner is told
	// a member can read the code
	if _, got := (getAccess(b, room)).access(t); got != code {
		t.Fatalf("member sees %q, want %q", got, code)
	}
	if getAccess(other, room).errCode(t) != errNotInRoom {
		t.Fatal("a non-member read the code")
	}

	// bob leaves (the room holds two), then it is public again: listed, and the old code stops working
	lw := &writer{}
	lw.u64(room)
	b.call(msgLeaveRoom, lw.b)
	if m, c := (setAccess(a, room, accessPublic)).access(t); m != accessPublic || strings.Trim(c, "\x00") != "" {
		t.Fatalf("public: mode %d code %q", m, c)
	}
	d := dial(t, addr, "NPWR00001_00", "dave", 3660)
	if e := (joinCode(d, code)).errCode(t); e != errNotFound {
		t.Fatalf("old code: error %d", e)
	}
	if searchTotal(d) != 1 {
		t.Fatal("room not listed again")
	}
}

func TestRoomAccessErrors(t *testing.T) {
	addr := start(t)
	a := dial(t, addr, "NPWR00001_00", "alice", 3658)
	b := dial(t, addr, "NPWR00001_00", "bob", 3659)
	room := roomOf(t, a)
	joinID(b, room)

	if e := (setAccess(b, room, accessCode)).errCode(t); e != errNotOwner {
		t.Errorf("non-owner: error %d, want %d", e, errNotOwner)
	}
	if e := (setAccess(a, room, 2)).errCode(t); e != errBadRequest {
		t.Errorf("mode 2 (not built yet): error %d, want %d", e, errBadRequest)
	}
	if e := (setAccess(a, room+99, accessCode)).errCode(t); e != errNotInRoom {
		t.Errorf("no such room: error %d", e)
	}
	if e := (joinCode(b, "ZZZZZZZZ")).errCode(t); e != errNotFound {
		t.Errorf("wrong code: error %d, want %d", e, errNotFound)
	}
	if e := (joinCode(b, "ABCDEFGU")).errCode(t); e != errBadRequest { // U is not in the alphabet
		t.Errorf("malformed code: error %d, want %d", e, errBadRequest)
	}
}

func TestCodeGuessesCloseTheConnection(t *testing.T) {
	addr := start(t)
	a := dial(t, addr, "NPWR00001_00", "alice", 3658)
	roomOf(t, a)
	b := dial(t, addr, "NPWR00001_00", "bob", 3659)
	for i := 0; i < maxCodeMisses-1; i++ { // under the limit: still answered
		if e := (joinCode(b, "ZZZZZZZZ")).errCode(t); e != errNotFound {
			t.Fatalf("miss %d: error %d", i+1, e)
		}
	}
	if searchTotal(b) != 1 { // control: the connection still works
		t.Fatal("connection unusable under the limit")
	}
	w := &writer{}
	w.u32(99)
	w.str("ZZZZZZZZ", codeLen)
	w.blob(nil)
	writePacket(b.conn, msgJoinByCode, w.b) // the fifth miss
	b.conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	for {
		if _, _, err := readPacket(b.conn); err != nil {
			return // closed
		}
	}
}

func TestCodeLeavesWithTheRoom(t *testing.T) {
	addr := start(t)
	a := dial(t, addr, "NPWR00001_00", "alice", 3658)
	b := dial(t, addr, "NPWR00001_00", "bob", 3659)
	room := roomOf(t, a)
	_, code := (setAccess(a, room, accessCode)).access(t)
	w := &writer{}
	w.u64(room)
	a.call(msgLeaveRoom, w.b) // the last member leaves: the room closes
	if e := (joinCode(b, code)).errCode(t); e != errNotFound {
		t.Fatalf("code of a closed room: error %d", e)
	}
}

func TestCodeShape(t *testing.T) {
	s := NewServer(10, false)
	seen := map[string]bool{}
	for i := 0; i < 500; i++ {
		c, err := s.newCode()
		if err != nil || len(c) != codeLen || seen[c] {
			t.Fatalf("bad or repeated code %q", c)
		}
		for _, ch := range c {
			if !strings.ContainsRune(codeAlphabet, ch) {
				t.Fatalf("code %q has %q", c, ch)
			}
		}
		seen[c] = true
		s.byCode[c] = &room{}
	}
	for in, want := range map[string]string{"abcdefgh": "ABCDEFGH", "OIL01234": "01101234", "ZZZZZZZZ": "ZZZZZZZZ"} {
		if got, ok := normalizeCode(in); !ok || got != want {
			t.Errorf("normalizeCode(%q) = %q, %v", in, got, ok)
		}
	}
	for _, in := range []string{"", "ABCDEFG", "ABCDEFGHJ", "ABCDEFGU", "ABCD-EFG", "ABCDEF G"} {
		if _, ok := normalizeCode(in); ok {
			t.Errorf("normalizeCode(%q) accepted", in)
		}
	}
}

// The code is a bearer secret: not in the log, not on the status page, not in /api/stats.
func TestCodeStaysOutOfLogAndStatus(t *testing.T) {
	s, addr, buf := logged(t)
	a := dial(t, addr, "NPWR00001_00", "alice", 3658)
	b := dial(t, addr, "NPWR00001_00", "bob", 3659)
	room := roomOf(t, a)
	_, code := (setAccess(a, room, accessCode)).access(t)
	joinCode(b, code)
	joinCode(b, "ZZZZZZZZ") // a miss is logged, without the code
	time.Sleep(50 * time.Millisecond)

	if strings.Contains(buf.String(), code) || strings.Contains(strings.ToLower(buf.String()), strings.ToLower(code)) {
		t.Errorf("code in the log:\n%s", buf.String())
	}
	if !strings.Contains(buf.String(), "room code refused") {
		t.Errorf("miss not logged:\n%s", buf.String())
	}
	st := s.snapshot()
	if len(st.Rooms) != 0 || st.Unlisted != 1 {
		t.Errorf("status: %d listed, %d unlisted", len(st.Rooms), st.Unlisted)
	}
	j, _ := json.Marshal(st)
	if strings.Contains(string(j), code) {
		t.Errorf("code in /api/stats: %s", j)
	}
	var sb strings.Builder
	if err := page.Execute(&sb, st); err != nil || strings.Contains(sb.String(), code) || strings.Contains(sb.String(), "alice") {
		t.Errorf("status page: err %v, shows the code or the members:\n%s", err, sb.String())
	}
}
