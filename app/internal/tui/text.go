package tui

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// clean makes text that came from the server (or from an error that quotes it) safe to draw: invalid UTF-8 and every
// control character (C0, DEL, C1, which includes ESC and NUL) is dropped, so a server cannot move the cursor, clear
// the screen or set the window title through a name.
func clean(s string) string {
	if !utf8.ValidString(s) {
		s = strings.ToValidUTF8(s, "")
	}
	return strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, s)
}
