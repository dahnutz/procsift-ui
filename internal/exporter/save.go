package exporter

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/dahnutz/procsift-ui/internal/report"
	"github.com/dahnutz/procsift-ui/internal/sanitize"
)

// Save writes the current live snapshot to a new file. It never overwrites a file.
func Save(path, format string, rawJSON []byte, rep report.Report) error {
	if format != "json" && format != "txt" {
		return fmt.Errorf("unsupported format %q", format)
	}
	if format == "json" && len(rawJSON) == 0 {
		return fmt.Errorf("current scan has no JSON to save")
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	completed := false
	defer func() {
		_ = f.Close()
		if !completed {
			_ = os.Remove(path)
		}
	}()
	if format == "json" {
		_, err = io.Copy(f, bytes.NewReader(rawJSON))
	} else {
		err = writeText(f, rep)
	}
	if err != nil {
		return err
	}
	if err = f.Sync(); err != nil {
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	completed = true
	return nil
}

func line(s string) string {
	s = sanitize.ForDisplay(s)
	s = strings.ReplaceAll(s, "\n", " ")
	return strings.ReplaceAll(s, "\t", " ")
}

func writeText(w io.Writer, rep report.Report) error {
	b := bufio.NewWriter(w)
	put := func(format string, args ...any) {
		_, _ = fmt.Fprintf(b, format, args...)
	}
	put("ProcSift %s report\n", line(rep.Version))
	put("Time: %s\nHost: %s\nResult: %s\nExit reason: %s\n", rep.Time.Format("2006-01-02T15:04:05Z07:00"), line(rep.Host.Hostname), line(rep.HostResult), line(rep.HostExitReason))
	put("Kernel: %s\nBoot ID: %s\n", line(rep.Host.Kernel), line(rep.Host.BootID))
	put("OS release: %s\nEUID: %d\nPID max: %d\nTasks seen: %d\n", line(rep.Host.OSRelease), rep.Host.EUID, rep.Host.PIDMax, rep.Host.TasksSeen)
	put("PID sweep: %t (probed %d, complete %t)\nTaint: %d\n", rep.Host.SweepUsed, rep.Host.SweepProbed, rep.Host.SweepComplete, rep.Host.Taint)
	put("Duration: %d ms\nMinimum severity: %s\n", rep.DurationMS, line(rep.MinSeverity))
	put("Host findings: high %d, medium %d, low %d, info %d\n", rep.HostCounts.High, rep.HostCounts.Medium, rep.HostCounts.Low, rep.HostCounts.Info)
	put("Scanner self: high %d, medium %d, low %d, info %d\n", rep.ScannerCounts.High, rep.ScannerCounts.Medium, rep.ScannerCounts.Low, rep.ScannerCounts.Info)
	put("\nAttention (%d)\n", len(rep.Attention))
	for _, a := range rep.Attention {
		put("\n[%s] score %d  PID %d  %s  %s\n", line(a.Level), a.Score, a.PID, line(a.Process), line(a.Subject))
		for _, why := range a.Why {
			put("   Why: %s\n", line(why))
		}
		for _, alt := range a.Alternatives {
			put("   Alternative: %s\n", line(alt))
		}
	}
	put("\nFindings (%d)\n", len(rep.Findings))
	for i, f := range rep.Findings {
		put("\n%d. [%s] %s", i+1, line(f.Severity), line(f.Rule))
		if f.PID > 0 {
			put(" PID %d", f.PID)
		}
		put("\n   %s\n", line(f.Summary))
		if f.Process != "" {
			put("   Process: %s\n", line(f.Process))
		}
		if f.Subject != "" {
			put("   Subject: %s\n", line(f.Subject))
		}
		if f.Origin != "" {
			put("   Origin: %s\n", line(f.Origin))
		}
		if f.Category != "" {
			put("   Category: %s\n", line(f.Category))
		}
		if f.Detail != "" {
			put("   Detail: %s\n", line(f.Detail))
		}
		for _, evidence := range f.Evidence {
			put("   Evidence: %s\n", line(evidence))
		}
	}
	put("\nProcesses (%d)\n", len(rep.Processes))
	for _, p := range rep.Processes {
		put("PID %d  PPID %d  UID %d  %s\n", p.PID, p.PPID, p.UID, line(p.Name))
		put("   Exe: %s\n   Command: %s\n", line(p.Exe), line(p.Cmdline))
		put("   State: %s  RSS: %d KiB  Threads: %d  Listed: %t\n", line(p.State), p.RSSKiB, p.Threads, p.Listed)
		put("   Start ticks: %d  Exe device: %s  Exe inode: %d\n", p.StartTicks, line(p.ExeDev), p.ExeInode)
		if p.Cwd != "" {
			put("   Cwd: %s\n", line(p.Cwd))
		}
		if p.StartupStatus != "" {
			put("   Startup: %s\n", line(p.StartupStatus))
		}
		if p.ExeHash != "" {
			put("   Exe hash (%s): %s\n", line(p.ExeHashScope), line(p.ExeHash))
		}
	}
	put("\nSockets (%d)\n", len(rep.Sockets))
	for _, s := range rep.Sockets {
		put("PID %d  %s  %s -> %s  %s  inode %d\n", s.PID, line(s.Proto), line(s.Local), line(s.Remote), line(s.State), s.Inode)
	}
	put("\nFile checks (%d)\n", len(rep.FileChecks))
	for _, f := range rep.FileChecks {
		put("%s  %s  %s  source %s\n", line(f.Path), line(f.State), line(f.Package), line(f.Source))
		if f.SHA256 != "" {
			put("   SHA256: %s\n", line(f.SHA256))
		}
		if f.MD5 != "" {
			put("   MD5: %s\n", line(f.MD5))
		}
	}
	put("\nCoverage (%d)\n", len(rep.Coverage))
	for _, c := range rep.Coverage {
		put("[%s] %s: %s\n", line(c.State), line(c.Source), line(c.Detail))
	}
	for _, l := range rep.Limits {
		put("Limit: %s\n", line(l))
	}
	for _, n := range rep.Notes {
		put("Note: %s\n", line(n))
	}
	return b.Flush()
}
