package scanner

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/dahnutz/procsift-ui/internal/report"
)

const runTimeout = 30 * time.Minute

const maxScannerStderr = 64 << 10

type limitedCapture struct {
	bytes.Buffer
	max      int
	abort    bool
	exceeded bool
}

func (c *limitedCapture) Write(p []byte) (int, error) {
	remaining := c.max - c.Len()
	if len(p) <= remaining {
		return c.Buffer.Write(p)
	}
	c.exceeded = true
	if remaining > 0 {
		_, _ = c.Buffer.Write(p[:remaining])
	}
	if c.abort {
		return remaining, fmt.Errorf("scanner JSON exceeds the 64 MiB input limit")
	}
	return len(p), nil
}

// Result holds a parsed live scan and the scanner exit code.
type Result struct {
	Report   report.Report
	ExitCode int
	Stderr   string
}

// Run invokes procsift -json -q and parses stdout.
func Run(scannerPath string) (Result, error) {
	path, err := resolve(scannerPath)
	if err != nil {
		return Result{}, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), runTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, path, "-json", "-q")
	stdout := limitedCapture{max: report.MaxBytes, abort: true}
	stderr := limitedCapture{max: maxScannerStderr}
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	runErr := cmd.Run()
	if stdout.exceeded {
		return Result{}, fmt.Errorf("scanner JSON exceeds the 64 MiB input limit")
	}
	if runErr != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return Result{}, fmt.Errorf("scanner timed out after %s", runTimeout)
		}
		exit := 2
		if ee, ok := runErr.(*exec.ExitError); ok {
			exit = ee.ExitCode()
		}
		if exit != 1 || stdout.Len() == 0 {
			msg := stderr.String()
			if msg == "" {
				msg = runErr.Error()
			}
			return Result{}, fmt.Errorf("scanner failed (exit %d): %s", exit, msg)
		}
		rep, perr := report.ParseJSON(stdout.Bytes())
		if perr != nil {
			return Result{}, fmt.Errorf("scanner exit %d but JSON invalid: %w; stderr: %s", exit, perr, stderr.String())
		}
		return Result{Report: rep, ExitCode: exit, Stderr: stderr.String()}, nil
	}
	rep, err := report.ParseJSON(stdout.Bytes())
	if err != nil {
		return Result{}, fmt.Errorf("scanner stdout is not a valid report: %w", err)
	}
	return Result{Report: rep, ExitCode: 0, Stderr: stderr.String()}, nil
}

func resolve(explicit string) (string, error) {
	if explicit != "" {
		info, err := os.Stat(explicit)
		if err != nil {
			return "", fmt.Errorf("scanner %q: %w", explicit, err)
		}
		if info.IsDir() {
			return "", fmt.Errorf("scanner %q is a directory", explicit)
		}
		return explicit, nil
	}
	path, err := exec.LookPath("procsift")
	if err != nil {
		return "", fmt.Errorf("procsift not found in PATH (use --scanner PATH)")
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return path, nil
	}
	return abs, nil
}
