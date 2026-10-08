package scanner_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/dahnutz/procsift-ui/internal/scanner"
)

func TestRunAcceptsExitCodeOne(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("scanner helper uses linux shebang")
	}
	dir := t.TempDir()
	helper := filepath.Join(dir, "fake-procsift")
	sample := filepath.Join("..", "..", "testdata", "sample.json")
	script := "#!/bin/sh\n" +
		"cat \"" + sample + "\"\n" +
		"exit 1\n"
	if err := os.WriteFile(helper, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	res, err := scanner.Run(helper)
	if err != nil {
		t.Fatal(err)
	}
	if res.ExitCode != 1 {
		t.Fatalf("exit: %d", res.ExitCode)
	}
	if len(res.Report.Findings) != 4 {
		t.Fatalf("findings: %d", len(res.Report.Findings))
	}
}

func TestRunRejectsExitCodeTwo(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("scanner helper uses linux shebang")
	}
	dir := t.TempDir()
	helper := filepath.Join(dir, "fail-procsift")
	if err := os.WriteFile(helper, []byte("#!/bin/sh\necho error 1>&2\nexit 2\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := scanner.Run(helper); err == nil {
		t.Fatal("expected failure on exit 2")
	}
}

func TestRunUsesExplicitScanner(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("scanner helper uses linux shebang")
	}
	path, err := exec.LookPath("sh")
	if err != nil {
		t.Skip(err)
	}
	dir := t.TempDir()
	helper := filepath.Join(dir, "ok-procsift")
	sample := filepath.Join("..", "..", "testdata", "sample.json")
	script := "#!/bin/sh\n" + "cat \"" + sample + "\"\n"
	if err := os.WriteFile(helper, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	_ = path
	res, err := scanner.Run(helper)
	if err != nil {
		t.Fatal(err)
	}
	if res.Report.Host.Hostname != "example" {
		t.Fatalf("hostname: %s", res.Report.Host.Hostname)
	}
}

func TestRunRejectsUnexpectedExitCode(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("scanner helper uses linux shebang")
	}
	dir := t.TempDir()
	helper := filepath.Join(dir, "bad-exit-procsift")
	sample := filepath.Join("..", "..", "testdata", "sample.json")
	script := "#!/bin/sh\ncat \"" + sample + "\"\nexit 3\n"
	if err := os.WriteFile(helper, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := scanner.Run(helper); err == nil || !strings.Contains(err.Error(), "exit 3") {
		t.Fatalf("unexpected exit accepted: %v", err)
	}
}

func TestRunBoundsScannerOutput(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("scanner helper uses linux shebang")
	}
	dir := t.TempDir()
	helper := filepath.Join(dir, "large-procsift")
	if err := os.WriteFile(helper, []byte("#!/bin/sh\nhead -c 67108865 /dev/zero\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := scanner.Run(helper); err == nil || !strings.Contains(err.Error(), "64 MiB") {
		t.Fatalf("oversized output accepted: %v", err)
	}
}
