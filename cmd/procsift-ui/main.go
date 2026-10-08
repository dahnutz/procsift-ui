package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/dahnutz/procsift-ui/internal/offline"
	"github.com/dahnutz/procsift-ui/internal/report"
	"github.com/dahnutz/procsift-ui/internal/scanner"
	"github.com/dahnutz/procsift-ui/internal/terminal"
	"github.com/dahnutz/procsift-ui/internal/ui"
)

func main() {
	os.Exit(run())
}

func run() int {
	var filePath, folderPath, scannerPath string
	flag.StringVar(&filePath, "file", "", "open one saved ProcSift JSON report")
	flag.StringVar(&folderPath, "folder", "", "pick a JSON report from this directory")
	flag.StringVar(&scannerPath, "scanner", "", "path to procsift executable (default: PATH)")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: procsift-ui [--file PATH | --folder DIR] [--scanner PATH]\n\n")
		fmt.Fprintf(os.Stderr, "  procsift-ui                 scan now, then browse\n")
		fmt.Fprintf(os.Stderr, "  procsift-ui --file PATH     browse one saved report\n")
		fmt.Fprintf(os.Stderr, "  procsift-ui --folder DIR    choose a saved report\n")
		flag.PrintDefaults()
	}
	flag.Parse()
	if filePath != "" && folderPath != "" {
		fmt.Fprintln(os.Stderr, "procsift-ui: --file and --folder are mutually exclusive")
		return 2
	}
	if !terminal.IsTTY(int(os.Stdin.Fd())) || !terminal.IsTTY(int(os.Stdout.Fd())) {
		fmt.Fprintln(os.Stderr, "procsift-ui: interactive mode requires a terminal")
		return 2
	}

	scanExit := -1
	var idx *report.IndexedReport
	var sourcePath string

	switch {
	case filePath != "":
		rep, err := offline.ReadFile(filePath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "procsift-ui: %v\n", err)
			return 2
		}
		idx = report.Index(rep)
		sourcePath = filePath
	case folderPath != "":
		entries, err := offline.ListFolder(folderPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "procsift-ui: %v\n", err)
			return 2
		}
		term, err := terminal.Open()
		if err != nil {
			fmt.Fprintf(os.Stderr, "procsift-ui: %v\n", err)
			return 2
		}
		defer term.Close()
		res, err := ui.SelectFromFolder(term, entries)
		if err != nil {
			fmt.Fprintf(os.Stderr, "procsift-ui: %v\n", err)
			return 2
		}
		if res.Quit {
			return 0
		}
		idx = res.Idx
		sourcePath = res.SourcePath
		app := ui.NewApp(term, idx, sourcePath, scanExit)
		if err := app.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "procsift-ui: %v\n", err)
			return 2
		}
		return 0
	default:
		res, err := scanner.Run(scannerPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "procsift-ui: %v\n", err)
			return 2
		}
		idx = report.Index(res.Report)
		scanExit = res.ExitCode
	}

	term, err := terminal.Open()
	if err != nil {
		fmt.Fprintf(os.Stderr, "procsift-ui: %v\n", err)
		return 2
	}
	defer term.Close()
	app := ui.NewApp(term, idx, sourcePath, scanExit)
	if err := app.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "procsift-ui: %v\n", err)
		return 2
	}
	if scanExit >= 0 {
		return scanExit
	}
	return 0
}
