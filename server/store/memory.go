package store

import (
	"context"
	"sort"
	"sync"
)

// memory keeps the boards in maps, behind one lock. Everything goes with the
// process: a restart empties every board.
type memory struct {
	mu     sync.Mutex
	boards map[Key]*board
}

type board struct {
	ascending bool // lower is better (lap times)
	entries   map[string]*Entry
}

// NewMemory returns an empty memory backend.
func NewMemory() Store { return &memory{boards: map[Key]*board{}} }

func (b *board) better(a, c *Entry) bool {
	if a.Score != c.Score {
		if b.ascending {
			return a.Score < c.Score
		}
		return a.Score > c.Score
	}
	return a.At.Before(c.At)
}

func (b *board) ranked() []*Entry {
	out := make([]*Entry, 0, len(b.entries))
	for _, e := range b.entries {
		out = append(out, e)
	}
	sort.Slice(out, func(i, j int) bool { return b.better(out[i], out[j]) })
	return out
}

func (b *board) rankOf(onlineID string) uint32 {
	for i, e := range b.ranked() {
		if e.OnlineID == onlineID {
			return uint32(i + 1)
		}
	}
	return 0
}

func (m *memory) Put(ctx context.Context, k Key, ascending bool, e Entry) (uint32, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	b := m.boards[k]
	if b == nil {
		b = &board{ascending: ascending, entries: map[string]*Entry{}}
		m.boards[k] = b
	}
	e.Comment = append([]byte(nil), e.Comment...)
	if old := b.entries[e.OnlineID]; old == nil || b.better(&e, old) {
		b.entries[e.OnlineID] = &e
	}
	return b.rankOf(e.OnlineID), nil
}

func (m *memory) Range(ctx context.Context, k Key, offset, count int) (int, []Entry, error) {
	if err := ctx.Err(); err != nil {
		return 0, nil, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	var all []*Entry
	if b := m.boards[k]; b != nil {
		all = b.ranked()
	}
	lo := min(max(offset, 0), len(all))
	hi := min(lo+max(count, 0), len(all))
	rows := make([]Entry, 0, hi-lo)
	for _, e := range all[lo:hi] {
		rows = append(rows, *e)
	}
	return len(all), rows, nil
}

func (m *memory) Lookup(ctx context.Context, k Key, ids []string) (int, []Entry, []uint32, error) {
	if err := ctx.Err(); err != nil {
		return 0, nil, nil, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	rows := make([]Entry, len(ids))
	ranks := make([]uint32, len(ids))
	b := m.boards[k]
	if b == nil {
		return 0, rows, ranks, nil
	}
	for i, who := range ids {
		if e := b.entries[who]; e != nil {
			rows[i], ranks[i] = *e, b.rankOf(who)
		}
	}
	return len(b.entries), rows, ranks, nil
}

func (m *memory) Boards(ctx context.Context) ([]Board, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]Board, 0, len(m.boards))
	for k, b := range m.boards {
		bi := Board{Key: k, Entries: len(b.entries)}
		if r := b.ranked(); len(r) > 0 {
			bi.Leader = *r[0]
		}
		out = append(out, bi)
	}
	return out, nil
}

func (m *memory) Name() string { return "memory" }

func (m *memory) Close() error { return nil }
