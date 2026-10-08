package sanitize_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dahnutz/procsift-ui/internal/report"
	"github.com/dahnutz/procsift-ui/internal/sanitize"
)

func TestEscapeReportStrings(t *testing.T) {
	b, err := os.ReadFile(filepath.Join("..", "..", "testdata", "escape.json"))
	if err != nil {
		t.Fatal(err)
	}
	rep, err := report.ParseJSON(b)
	if err != nil {
		t.Fatal(err)
	}
	f := rep.Findings[0]
	for _, s := range []string{f.Process, f.Summary, f.Detail} {
		out := sanitize.ForDisplay(s)
		if strings.Contains(out, "\x1b") || strings.Contains(out, "\x07") {
			t.Fatalf("unsafe output: %q", out)
		}
	}
}
