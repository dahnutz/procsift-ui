package sanitize

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// ForDisplay strips terminal control sequences and unsafe runes from untrusted text.
func ForDisplay(s string) string {
	if s == "" {
		return s
	}
	var b strings.Builder
	b.Grow(len(s))
	i := 0
	for i < len(s) {
		if s[i] == '\x1b' {
			j := i + 1
			if j < len(s) && s[j] == '[' {
				j++
				for j < len(s) {
					c := s[j]
					if c >= '@' && c <= '~' {
						j++
						break
					}
					j++
				}
			} else if j < len(s) && s[j] == ']' {
				j++
				for j < len(s) {
					if s[j] == '\x07' {
						j++
						break
					}
					j++
				}
			} else if j < len(s) {
				j++
			}
			i = j
			continue
		}
		r, size := utf8.DecodeRuneInString(s[i:])
		if r == '\t' || r == '\n' {
			b.WriteRune(r)
		} else if r == utf8.RuneError && size == 1 {
			b.WriteRune('\uFFFD')
		} else if unicode.IsControl(r) {
			b.WriteRune('\uFFFD')
		} else {
			b.WriteRune(r)
		}
		i += size
	}
	return b.String()
}
