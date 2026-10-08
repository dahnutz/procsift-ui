package terminal

import (
	"fmt"
	"io"
	"strings"
)

// Paint draws a full frame with cursor addressing. Raw mode disables
// newline-to-CRLF translation, so a line feed alone would walk each row
// toward the right edge. Each row is placed explicitly and erased to the
// end of the line. Callers clip rows to the terminal width; this write
// contains no newlines, so a row cannot wrap the frame off the top.
func Paint(w io.Writer, rows []string) {
	var b strings.Builder
	b.Grow(32 + len(rows)*64)
	b.WriteString("\x1b[0m\x1b[?25l\x1b[?7l")
	for i, row := range rows {
		fmt.Fprintf(&b, "\x1b[%d;1H", i+1)
		b.WriteString(row)
		b.WriteString("\x1b[K")
	}
	_, _ = io.WriteString(w, b.String())
}
