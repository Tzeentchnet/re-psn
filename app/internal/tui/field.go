package tui

import (
	"strings"
	"unicode/utf8"

	tea "charm.land/bubbletea/v2"
)

// field is a one-line text input: typed and pasted text is appended, backspace removes the last character, ctrl+u
// clears it. Control characters are never taken in.
type field struct {
	value string
	limit int // in bytes; 0 means 256
}

func (f *field) max() int {
	if f.limit > 0 {
		return f.limit
	}
	return 256
}

// insert appends printable text, up to the limit.
func (f *field) insert(s string) {
	for _, r := range s {
		if r < 0x20 || r == 0x7f || r == utf8.RuneError {
			continue
		}
		if len(f.value)+utf8.RuneLen(r) > f.max() {
			return
		}
		f.value += string(r)
	}
}

// key applies an editing key and reports whether it was one.
func (f *field) key(k tea.KeyPressMsg) bool {
	switch k.String() {
	case "backspace", "ctrl+h":
		if _, n := utf8.DecodeLastRuneInString(f.value); n > 0 {
			f.value = f.value[:len(f.value)-n]
		}
		return true
	case "ctrl+u":
		f.value = ""
		return true
	}
	if k.Text != "" && k.Mod&^tea.ModShift == 0 {
		f.insert(k.Text)
		return true
	}
	return false
}

func (f field) trimmed() string { return strings.TrimSpace(f.value) }
