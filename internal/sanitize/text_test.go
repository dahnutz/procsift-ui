package sanitize_test

import (
	"strings"
	"testing"

	"github.com/dahnutz/procsift-ui/internal/sanitize"
)

func TestForDisplayStripsEscapesAndControls(t *testing.T) {
	in := "\x1b[31mred\x1b[0m bell\x07 ok"
	out := sanitize.ForDisplay(in)
	if strings.Contains(out, "\x1b") {
		t.Fatalf("escape retained: %q", out)
	}
	if strings.Contains(out, "\x07") {
		t.Fatalf("bell retained: %q", out)
	}
	if !strings.Contains(out, "red") || !strings.Contains(out, "ok") {
		t.Fatalf("text lost: %q", out)
	}
}
