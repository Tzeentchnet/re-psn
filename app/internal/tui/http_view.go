package tui

import (
	"fmt"
	"strings"
)

const maxHTTPDisplayRows = 100

func httpText(s string) string {
	r := []rune(clean(s))
	if len(r) > 128 {
		return string(r[:125]) + "..."
	}
	return string(r)
}

func (m Model) httpStats() *HTTPStats {
	if m.snap == nil || m.fetchErr != nil || m.snapQuery.StatusURL != m.set.StatusURL {
		return nil
	}
	return m.snap.HTTP
}

func (m Model) httpRoomLines(h *HTTPStats) []string {
	b := []string{m.st.strong.Render(fmt.Sprintf("%-10s %-14s %-8s %-10s %s", "ID", "Title", "Members", "Age", "Names"))}
	count, shown := 0, 0
	for _, r := range h.Rooms {
		if m.title != "" && r.Title != m.title {
			continue
		}
		count++
		if shown >= maxHTTPDisplayRows {
			continue
		}
		shown++
		b = append(b, m.st.text.Render(fmt.Sprintf("%-10d %-14s %-8s %-10s %s", r.ID, httpText(r.Title), fmt.Sprintf("%d/%d", len(r.Members), r.Max), httpText(r.Age), httpText(strings.Join(r.Members, ", ")))))
	}
	if count == 0 {
		b = append(b, m.st.text.Render("No public rooms match the title filter."))
	}
	return append(b, "", m.st.muted.Render(fmt.Sprintf("Showing %d of %d public rooms; full rooms are included. %d invite-only rooms are counted without names.", shown, count, h.Unlisted)))
}

func (m Model) httpBoardLines(h *HTTPStats) []string {
	b := []string{m.st.strong.Render(fmt.Sprintf("%-14s %-10s %-8s %-16s %s", "Title", "Board", "Entries", "Leader", "Best"))}
	count, shown := 0, 0
	for _, r := range h.Boards {
		if m.title != "" && r.Title != m.title {
			continue
		}
		if m.board != "" && fmt.Sprint(r.Board) != m.board {
			continue
		}
		count++
		if shown >= maxHTTPDisplayRows {
			continue
		}
		shown++
		leader, best := "-", "-"
		if r.Entries > 0 {
			leader, best = httpText(r.Leader), fmt.Sprint(r.Best)
		}
		b = append(b, m.st.text.Render(fmt.Sprintf("%-14s %-10d %-8d %-16s %s", httpText(r.Title), r.Board, r.Entries, leader, best)))
	}
	if count == 0 {
		b = append(b, m.st.text.Render("No leaderboards match the title and board filters."))
	}
	return append(b, "", m.st.muted.Render(fmt.Sprintf("Showing %d of %d board summaries. HTTP shares each board's leader, not the full ranking.", shown, count)))
}
