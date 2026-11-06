package tui

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/undecuple/re-psn/app/internal/client"
)

// Query is what one refresh of the Status screen asks the server.
type Query struct {
	Server string // host:port
	Title  string // the title's communication ID; empty means only HELLO
	Board  uint32
	Boards bool // read Board's ranking too
}

// Snapshot is what the server answered.
type Snapshot struct {
	Ack     client.HelloAck
	Name    string        // the throwaway online ID the app said HELLO with
	RTT     time.Duration // the HELLO round trip
	Rooms   client.RoomList
	Ranking client.Ranking
	RankErr error // reading the board failed; the rooms are still good
}

// Fetcher reads a Snapshot. The real one speaks the game protocol; tests use a fake.
type Fetcher interface {
	Fetch(ctx context.Context, q Query) (Snapshot, error)
}

// probeTitle is the communication ID used when no title is set: no real title has it, so it sees no game's rooms.
const probeTitle = "RE-PSN-CHECK"

// MaxRooms and MaxRows bound one refresh.
const (
	MaxRooms = 50
	MaxRows  = 20
)

// Protocol is the Fetcher that connects to the server, says HELLO under a throwaway name and asks with
// SEARCH_ROOMS and GET_RANKING. It never uses the HTTP status page.
type Protocol struct{}

// Fetch opens one connection per call and closes it before it returns.
func (Protocol) Fetch(ctx context.Context, q Query) (Snapshot, error) {
	var s Snapshot
	c, err := client.Dial(ctx, q.Server)
	if err != nil {
		return s, err
	}
	defer c.Close()
	if s.Name, err = throwawayName(); err != nil {
		return s, err
	}
	title := q.Title
	if title == "" {
		title = probeTitle
	}
	start := time.Now()
	if s.Ack, err = c.Hello(ctx, client.Hello{CommID: title, OnlineID: s.Name}); err != nil {
		return s, err
	}
	s.RTT = time.Since(start)
	if q.Title == "" {
		return s, nil
	}
	if s.Rooms, err = c.SearchRooms(ctx, 0, MaxRooms); err != nil {
		return s, err
	}
	if q.Boards {
		s.Ranking, s.RankErr = c.Ranking(ctx, q.Board, 1, MaxRows)
	}
	return s, nil
}

// throwawayName is "view-" and 10 random hex digits (15 bytes), so it never takes the player's own name.
func throwawayName() (string, error) {
	var b [5]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", fmt.Errorf("random name: %w", err)
	}
	return "view-" + hex.EncodeToString(b[:]), nil
}
