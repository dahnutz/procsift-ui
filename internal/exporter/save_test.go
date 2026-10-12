package exporter

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dahnutz/procsift-ui/internal/report"
)

func TestSaveTextAndRefuseOverwrite(t *testing.T) {
	rep := report.Report{
		Version: "0.5.1", HostResult: "fail",
		Host:      report.HostInfo{Hostname: "example"},
		Findings:  []report.Finding{{Rule: "PROC-DELETED-EXE", Severity: "high", PID: 42, Summary: "deleted executable", Evidence: []string{"/tmp/x"}}},
		Processes: []report.Process{{PID: 42, Name: "worker", Exe: "/tmp/x"}},
		Coverage:  []report.ReadIssue{{State: "denied", Source: "/proc/42/maps"}},
	}
	path := filepath.Join(t.TempDir(), "report.txt")
	if err := Save(path, "txt", nil, rep); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"example", "PROC-DELETED-EXE", "deleted executable", "PID 42", "/proc/42/maps"} {
		if !strings.Contains(string(b), want) {
			t.Fatalf("text export missing %q", want)
		}
	}
	if err := Save(path, "json", []byte("replacement"), rep); err == nil {
		t.Fatal("overwrote existing file")
	}
	again, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(again) != string(b) {
		t.Fatal("existing file changed")
	}
}

func TestSaveJSONNeedsOriginalBytes(t *testing.T) {
	path := filepath.Join(t.TempDir(), "report.json")
	if err := Save(path, "json", nil, report.Report{}); err == nil {
		t.Fatal("empty JSON accepted")
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("file exists after failed save: %v", err)
	}
}
