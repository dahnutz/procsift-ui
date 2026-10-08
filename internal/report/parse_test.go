package report_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dahnutz/procsift-ui/internal/report"
)

func TestParseSampleReport(t *testing.T) {
	b, err := os.ReadFile(filepath.Join("..", "..", "testdata", "sample.json"))
	if err != nil {
		t.Fatal(err)
	}
	rep, err := report.ParseJSON(b)
	if err != nil {
		t.Fatal(err)
	}
	if rep.Tool != "procsift" || rep.Version != "0.5.1" {
		t.Fatalf("unexpected meta: %+v", rep)
	}
	if len(rep.Findings) != 4 {
		t.Fatalf("findings: got %d", len(rep.Findings))
	}
}

func TestParseRejectsWrongTool(t *testing.T) {
	b, err := os.ReadFile(filepath.Join("..", "..", "testdata", "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := report.ParseJSON(b); err == nil {
		t.Fatal("expected reject for capture manifest")
	}
}

func TestParseRejectsOversized(t *testing.T) {
	data := strings.Repeat(" ", report.MaxBytes+1)
	if _, err := report.ParseJSON([]byte(data)); err == nil || !strings.Contains(err.Error(), "64 MiB") {
		t.Fatalf("oversized: %v", err)
	}
}

func TestParseRejectsUnsupportedVersion(t *testing.T) {
	raw := `{"tool":"procsift","version":"0.4.0","findings":[]}`
	if _, err := report.ParseJSON([]byte(raw)); err == nil {
		t.Fatal("expected version reject")
	}
}

func TestParseAcceptsWazuhBranchVersion(t *testing.T) {
	b, err := os.ReadFile(filepath.Join("..", "..", "testdata", "sample.json"))
	if err != nil {
		t.Fatal(err)
	}
	b = []byte(strings.Replace(string(b), `"version": "0.5.1"`, `"version": "0.6.0-dev"`, 1))
	rep, err := report.ParseJSON(b)
	if err != nil {
		t.Fatal(err)
	}
	if rep.Version != "0.6.0-dev" || len(rep.Findings) != 4 {
		t.Fatalf("unexpected 0.6 report: %+v", rep)
	}
}

func TestParseDoesNotClaimFinal06Compatibility(t *testing.T) {
	raw := `{"tool":"procsift","version":"0.6.0","findings":[]}`
	if _, err := report.ParseJSON([]byte(raw)); err == nil {
		t.Fatal("unreleased final version accepted without a compatibility check")
	}
}
