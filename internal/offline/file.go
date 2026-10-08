package offline

import (
	"fmt"
	"io"
	"os"
	"syscall"

	"github.com/dahnutz/procsift-ui/internal/report"
)

// ReadFile loads one regular ProcSift JSON report from path.
func ReadFile(path string) (report.Report, error) {
	f, err := os.OpenFile(path, os.O_RDONLY|syscall.O_NONBLOCK, 0)
	if err != nil {
		return report.Report{}, fmt.Errorf("open report: %w", err)
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return report.Report{}, fmt.Errorf("stat report: %w", err)
	}
	if !info.Mode().IsRegular() {
		return report.Report{}, fmt.Errorf("report %s is not a regular file", path)
	}
	if info.Size() > report.MaxBytes {
		return report.Report{}, fmt.Errorf("report %s exceeds the 64 MiB input limit", path)
	}
	rep, err := report.ParseReader(f)
	if err != nil {
		return report.Report{}, err
	}
	return rep, nil
}

// ReadFileBytes is like ReadFile but accepts an io.Reader at caller's risk.
func ReadFileBytes(path string, r io.Reader) (report.Report, error) {
	rep, err := report.ParseReader(r)
	if err != nil {
		return report.Report{}, fmt.Errorf("%s: %w", path, err)
	}
	return rep, nil
}
