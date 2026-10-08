package terminal

import (
	"bytes"
	"strings"
	"testing"
)

func TestPaintPositionsRowsWithoutNewlines(t *testing.T) {
	var buf bytes.Buffer
	Paint(&buf, []string{"one", "two"})
	out := buf.String()
	if strings.Contains(out, "\n") || strings.Contains(out, "\r") {
		t.Fatalf("frame contains a line break: %q", out)
	}
	if !strings.Contains(out, "\x1b[1;1Hone\x1b[K") {
		t.Fatalf("row 1 not addressed: %q", out)
	}
	if !strings.Contains(out, "\x1b[2;1Htwo\x1b[K") {
		t.Fatalf("row 2 not addressed: %q", out)
	}
}
