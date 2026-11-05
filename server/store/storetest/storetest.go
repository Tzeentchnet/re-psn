// Package storetest is the contract every store backend passes: the same
// table of puts and reads, run against a fresh store each time. A backend's
// test calls Run with a function that opens an empty store.
package storetest

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/sp00nznet/psnr/server/store"
)

// t0 is the base time of the tables; step n is t0 plus n seconds.
var t0 = time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)

// put is one Put in a table.
type put struct {
	board     uint32
	ascending bool
	who       string
	score     int64
	at        int // seconds after t0
	rank      uint32
}

// row is what a read should return for one entry.
type row struct {
	who   string
	score int64
	at    int
}

var title = "NPWR00002_00"

func key(b uint32) store.Key { return store.Key{Title: title, Board: b} }

func entry(p put) store.Entry {
	return store.Entry{OnlineID: p.who, Score: p.score, Comment: []byte("c-" + p.who), At: t0.Add(time.Duration(p.at) * time.Second)}
}

var cases = []struct {
	name   string
	puts   []put
	board  uint32
	offset int
	count  int
	total  int
	want   []row
}{
	{"empty board", nil, 1, 0, 10, 0, nil},
	{"higher is better", []put{
		{1, false, "alice", 100, 0, 1},
		{1, false, "bob", 200, 1, 1},
		{1, false, "carol", 150, 2, 2},
	}, 1, 0, 10, 3, []row{{"bob", 200, 1}, {"carol", 150, 2}, {"alice", 100, 0}}},
	{"lower is better", []put{
		{2, true, "alice", 61000, 0, 1},
		{2, true, "bob", 59000, 1, 1},
	}, 2, 0, 10, 2, []row{{"bob", 59000, 1}, {"alice", 61000, 0}}},
	{"best stands", []put{
		{1, false, "alice", 100, 0, 1},
		{1, false, "bob", 200, 1, 1},
		{1, false, "alice", 50, 2, 2}, // worse: 100 and its time stand
		{1, false, "bob", 300, 3, 1},  // better: replaces
	}, 1, 0, 10, 2, []row{{"bob", 300, 3}, {"alice", 100, 0}}},
	{"best stands, lower is better", []put{
		{1, true, "alice", 100, 0, 1},
		{1, true, "alice", 120, 1, 1},
		{1, true, "alice", 90, 2, 1},
	}, 1, 0, 10, 1, []row{{"alice", 90, 2}}},
	{"order fixed by the first put", []put{
		{1, false, "alice", 100, 0, 1},
		{1, true, "bob", 50, 1, 2}, // asks for ascending; the board stays higher-is-better
	}, 1, 0, 10, 2, []row{{"alice", 100, 0}, {"bob", 50, 1}}},
	{"tie: the earlier one ranks higher", []put{
		{1, false, "bob", 100, 5, 1},
		{1, false, "alice", 100, 3, 1}, // put later, but recorded earlier
		{1, false, "carol", 100, 9, 3},
	}, 1, 0, 10, 3, []row{{"alice", 100, 3}, {"bob", 100, 5}, {"carol", 100, 9}}},
	{"an equal score does not replace", []put{
		{1, false, "alice", 100, 0, 1},
		{1, false, "bob", 100, 1, 2},
		{1, false, "bob", 100, 2, 2},
	}, 1, 0, 10, 2, []row{{"alice", 100, 0}, {"bob", 100, 1}}},
	{"range from the middle", fivePuts, 1, 1, 2, 5, []row{{"p2", 40, 2}, {"p3", 30, 3}}},
	{"range runs off the end", fivePuts, 1, 3, 10, 5, []row{{"p4", 20, 4}, {"p5", 10, 5}}},
	{"range past the end", fivePuts, 1, 5, 10, 5, nil},
	{"range far past the end", fivePuts, 1, 1 << 30, 10, 5, nil},
	{"count 0", fivePuts, 1, 0, 0, 5, nil},
	{"boards are separate", []put{
		{1, false, "alice", 100, 0, 1},
		{2, false, "bob", 200, 1, 1},
		{3, true, "carol", 300, 2, 1},
	}, 2, 0, 10, 1, []row{{"bob", 200, 1}}},
}

var fivePuts = []put{
	{1, false, "p1", 50, 1, 1},
	{1, false, "p2", 40, 2, 2},
	{1, false, "p3", 30, 3, 3},
	{1, false, "p4", 20, 4, 4},
	{1, false, "p5", 10, 5, 5},
}

// Run checks a backend. open must return an empty store; Run closes it.
func Run(t *testing.T, open func(t *testing.T) store.Store) {
	ctx := context.Background()
	fresh := func(t *testing.T) store.Store {
		s := open(t)
		t.Cleanup(func() {
			if err := s.Close(); err != nil {
				t.Errorf("close: %v", err)
			}
		})
		return s
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s := fresh(t)
			for i, p := range c.puts {
				rank, err := s.Put(ctx, key(p.board), p.ascending, entry(p))
				if err != nil {
					t.Fatalf("put %d: %v", i, err)
				}
				if rank != p.rank {
					t.Fatalf("put %d (%s %d): rank %d, want %d", i, p.who, p.score, rank, p.rank)
				}
			}
			total, rows, err := s.Range(ctx, key(c.board), c.offset, c.count)
			if err != nil {
				t.Fatal(err)
			}
			if total != c.total || len(rows) != len(c.want) {
				t.Fatalf("total %d rows %d, want %d and %d: %+v", total, len(rows), c.total, len(c.want), rows)
			}
			for i, w := range c.want {
				checkRow(t, rows[i], w)
			}
		})
	}

	t.Run("lookup", func(t *testing.T) {
		s := fresh(t)
		total, rows, ranks, err := s.Lookup(ctx, key(1), []string{"alice", "nobody"})
		if err != nil || total != 0 || len(rows) != 2 || ranks[0] != 0 || ranks[1] != 0 {
			t.Fatalf("missing board: total %d rows %v ranks %v err %v", total, rows, ranks, err)
		}
		for _, p := range fivePuts {
			if _, err := s.Put(ctx, key(1), false, entry(p)); err != nil {
				t.Fatal(err)
			}
		}
		ids := []string{"p3", "nobody", "p1", "p3", "P1", ""}
		total, rows, ranks, err = s.Lookup(ctx, key(1), ids)
		if err != nil {
			t.Fatal(err)
		}
		wantRanks := []uint32{3, 0, 1, 3, 0, 0} // names match exactly
		if total != 5 || len(rows) != len(ids) || len(ranks) != len(ids) {
			t.Fatalf("total %d rows %d ranks %d", total, len(rows), len(ranks))
		}
		for i := range ids {
			if ranks[i] != wantRanks[i] {
				t.Fatalf("%q: rank %d, want %d", ids[i], ranks[i], wantRanks[i])
			}
			if ranks[i] == 0 {
				if rows[i].OnlineID != "" || rows[i].Score != 0 || len(rows[i].Comment) != 0 || !rows[i].At.IsZero() {
					t.Fatalf("%q: no score, but entry %+v", ids[i], rows[i])
				}
				continue
			}
			checkRow(t, rows[i], row{ids[i], fivePuts[ranks[i]-1].score, fivePuts[ranks[i]-1].at})
		}
	})

	t.Run("comment is kept and copied", func(t *testing.T) {
		s := fresh(t)
		c := []byte("gg")
		if _, err := s.Put(ctx, key(1), false, store.Entry{OnlineID: "alice", Score: 1, Comment: c, At: t0}); err != nil {
			t.Fatal(err)
		}
		c[0] = 'x' // the caller's buffer, changed after the Put
		_, rows, err := s.Range(ctx, key(1), 0, 1)
		if err != nil || len(rows) != 1 || string(rows[0].Comment) != "gg" {
			t.Fatalf("rows %+v err %v", rows, err)
		}
	})

	t.Run("many boards", func(t *testing.T) {
		s := fresh(t)
		titles := []string{"NPWR00001_00", "NPWR00002_00", "NPWR00003_00"}
		for _, tt := range titles {
			for b := uint32(0); b < 20; b++ {
				for i := 0; i < 3; i++ {
					e := store.Entry{OnlineID: fmt.Sprintf("p%d", i), Score: int64(b)*10 + int64(i), At: t0}
					if _, err := s.Put(ctx, store.Key{Title: tt, Board: b}, b%2 == 1, e); err != nil {
						t.Fatal(err)
					}
				}
			}
		}
		boards, err := s.Boards(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if len(boards) != len(titles)*20 {
			t.Fatalf("%d boards, want %d", len(boards), len(titles)*20)
		}
		seen := map[store.Key]bool{}
		for _, b := range boards {
			if seen[b.Key] {
				t.Fatalf("board %v listed twice", b.Key)
			}
			seen[b.Key] = true
			want := "p2" // higher is better on even boards
			if b.Key.Board%2 == 1 {
				want = "p0"
			}
			if b.Entries != 3 || b.Leader.OnlineID != want {
				t.Fatalf("board %v: %d entries, leader %q, want 3 and %q", b.Key, b.Entries, b.Leader.OnlineID, want)
			}
		}
		if total, _, _ := s.Range(ctx, store.Key{Title: "NPWR00009_00", Board: 0}, 0, 10); total != 0 {
			t.Fatalf("another title's board 0 has %d entries", total)
		}
	})

	t.Run("no boards", func(t *testing.T) {
		boards, err := fresh(t).Boards(ctx)
		if err != nil || len(boards) != 0 {
			t.Fatalf("boards %v err %v", boards, err)
		}
	})

	t.Run("cancelled context", func(t *testing.T) {
		s := fresh(t)
		cctx, cancel := context.WithCancel(ctx)
		cancel()
		if _, err := s.Put(cctx, key(1), false, entry(put{who: "alice"})); err == nil {
			t.Fatal("put with a cancelled context succeeded")
		}
		if _, _, err := s.Range(cctx, key(1), 0, 1); err == nil {
			t.Fatal("range with a cancelled context succeeded")
		}
		if _, _, _, err := s.Lookup(cctx, key(1), []string{"alice"}); err == nil {
			t.Fatal("lookup with a cancelled context succeeded")
		}
		if _, err := s.Boards(cctx); err == nil {
			t.Fatal("boards with a cancelled context succeeded")
		}
	})

	t.Run("concurrent writers", func(t *testing.T) {
		s := fresh(t)
		const writers, each = 16, 50
		var wg sync.WaitGroup
		for w := 0; w < writers; w++ {
			wg.Add(1)
			go func(w int) {
				defer wg.Done()
				who := fmt.Sprintf("w%02d", w)
				for i := 0; i < each; i++ {
					e := store.Entry{OnlineID: who, Score: int64(w*each + i), At: t0.Add(time.Duration(i) * time.Millisecond)}
					if _, err := s.Put(ctx, key(uint32(i%3)), false, e); err != nil {
						t.Error(err)
						return
					}
					s.Range(ctx, key(uint32(i%3)), 0, 5)
					s.Lookup(ctx, key(uint32(i%3)), []string{who})
					s.Boards(ctx)
				}
			}(w)
		}
		wg.Wait()
		for b := uint32(0); b < 3; b++ {
			total, rows, err := s.Range(ctx, key(b), 0, writers)
			if err != nil || total != writers || len(rows) != writers {
				t.Fatalf("board %d: total %d rows %d err %v", b, total, len(rows), err)
			}
			for i := 1; i < len(rows); i++ {
				if rows[i-1].Score < rows[i].Score {
					t.Fatalf("board %d out of order at %d: %d then %d", b, i, rows[i-1].Score, rows[i].Score)
				}
			}
			// each writer's best on board b is its last score i with i%3 == b
			if top, want := rows[0].Score, int64((writers-1)*each+lastOf(each, b)); top != want {
				t.Fatalf("board %d: top %d, want %d", b, top, want)
			}
		}
	})
}

// lastOf is the largest i < n with i%3 == b.
func lastOf(n int, b uint32) int {
	for i := n - 1; i >= 0; i-- {
		if i%3 == int(b) {
			return i
		}
	}
	return -1
}

func checkRow(t *testing.T, got store.Entry, w row) {
	t.Helper()
	if got.OnlineID != w.who || got.Score != w.score || !got.At.Equal(t0.Add(time.Duration(w.at)*time.Second)) ||
		string(got.Comment) != "c-"+w.who {
		t.Fatalf("got %s %d %s %q, want %s %d at +%ds", got.OnlineID, got.Score, got.At, got.Comment, w.who, w.score, w.at)
	}
}
