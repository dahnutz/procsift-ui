package report_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/dahnutz/procsift-ui/internal/report"
)

func TestSortFindingsBySeverityAndAttention(t *testing.T) {
	b, err := os.ReadFile(filepath.Join("..", "..", "testdata", "sample.json"))
	if err != nil {
		t.Fatal(err)
	}
	rep, err := report.ParseJSON(b)
	if err != nil {
		t.Fatal(err)
	}
	idx := report.Index(rep)
	if len(idx.HostFindings) != 3 {
		t.Fatalf("host findings: %d", len(idx.HostFindings))
	}
	if idx.HostFindings[0].Rule != "PROC-DELETED-EXE" {
		t.Fatalf("first host finding: %s", idx.HostFindings[0].Rule)
	}
	if len(idx.ScannerFindings) != 1 || idx.ScannerFindings[0].Origin != "scanner_self" {
		t.Fatalf("scanner findings: %+v", idx.ScannerFindings)
	}
}

func TestPIDLinking(t *testing.T) {
	b, err := os.ReadFile(filepath.Join("..", "..", "testdata", "sample.json"))
	if err != nil {
		t.Fatal(err)
	}
	rep, err := report.ParseJSON(b)
	if err != nil {
		t.Fatal(err)
	}
	idx := report.Index(rep)
	p, ok := idx.ProcessByPID[4242]
	if !ok || p.Name != "worker" {
		t.Fatalf("process 4242: ok=%v name=%s", ok, p.Name)
	}
	socks := idx.SocketsByPID[4242]
	if len(socks) != 1 || socks[0].Proto != "tcp" {
		t.Fatalf("sockets: %+v", socks)
	}
	findings := idx.FindingsByPID[4242]
	if len(findings) != 1 {
		t.Fatalf("findings for 4242: %d", len(findings))
	}
}
