package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/dahnutz/procsift-ui/internal/exporter"
	"github.com/dahnutz/procsift-ui/internal/offline"
	"github.com/dahnutz/procsift-ui/internal/report"
	"github.com/dahnutz/procsift-ui/internal/scanner"
	"github.com/dahnutz/procsift-ui/internal/terminal"
)

// ViewTab selects the main browse tab.
type ViewTab int

const (
	TabFindings ViewTab = iota
	TabProcesses
	TabCoverage
	TabScannerSelf
)

type screen int

const (
	screenBrowse screen = iota
	screenFinding
	screenPID
	screenPrompt
	screenFolder
)

// App runs the interactive report browser.
type App struct {
	Term       *terminal.Term
	Idx        *report.IndexedReport
	SourcePath string
	ScanExit   int // -1 when offline

	tab             ViewTab
	screen          screen
	cursor          int
	scroll          int
	filter          string
	promptKind      string
	promptInput     string
	promptCursor    int
	picker          []offline.Entry
	pickerCursor    int
	pickerErr       string
	detailScroll    int
	pageRows        int
	listIndex       int
	pid             int
	statusMsg       string
	quit            bool
	liveJSON        []byte
	liveRun         func() (scanner.Result, error)
	saveDefaultPath string
	detailItems     []int
	detailPos       int
	detailFromList  bool
}

// NewApp creates an app ready to run.
func NewApp(t *terminal.Term, idx *report.IndexedReport, sourcePath string, scanExit int) *App {
	return &App{
		Term:       t,
		Idx:        idx,
		SourcePath: sourcePath,
		ScanExit:   scanExit,
		tab:        TabFindings,
		screen:     screenBrowse,
	}
}

// EnableLive supplies the current scanner output and a way to take another scan.
func (a *App) EnableLive(rawJSON []byte, run func() (scanner.Result, error)) {
	a.liveJSON = rawJSON
	a.liveRun = run
}

// Run starts the main event loop.
func (a *App) Run() error {
	if a.Term == nil {
		return fmt.Errorf("terminal is not open")
	}
	for !a.quit {
		a.draw()
		ev, err := a.Term.ReadEvent()
		if err != nil {
			return err
		}
		if ev.Key == terminal.KeyResize {
			continue
		}
		a.handleKey(ev)
	}
	return nil
}

func (a *App) handleKey(ev terminal.KeyEvent) {
	switch a.screen {
	case screenBrowse:
		a.handleBrowseKey(ev)
	case screenFinding, screenPID:
		a.handleDetailKey(ev)
	case screenPrompt:
		a.handlePromptKey(ev)
	}
}

func (a *App) handleBrowseKey(ev terminal.KeyEvent) {
	a.statusMsg = ""
	switch ev.Key {
	case terminal.KeyQuit:
		a.quit = true
	case terminal.KeyUp:
		a.moveList(-1)
	case terminal.KeyDown:
		a.moveList(1)
	case terminal.KeyPageUp:
		a.moveList(-a.pageStep())
	case terminal.KeyPageDown:
		a.moveList(a.pageStep())
	case terminal.KeyTab:
		a.tab = ViewTab((int(a.tab) + 1) % 4)
		a.cursor = 0
		a.scroll = 0
		a.filter = ""
	case terminal.KeyEnter:
		a.openSelection()
	case terminal.KeyPID:
		a.promptKind = "pid"
		a.promptInput = ""
		a.promptCursor = 0
		a.screen = screenPrompt
	case terminal.KeyFilter:
		a.promptKind = "filter"
		a.promptInput = a.filter
		a.promptCursor = len(a.promptInput)
		a.screen = screenPrompt
	case terminal.KeyRune:
		switch ev.Rune {
		case 'r':
			if a.liveRun != nil {
				a.refresh()
			}
		case 's':
			if a.liveRun != nil {
				a.promptKind = "save"
				a.promptInput = ""
				a.promptCursor = 0
				a.saveDefaultPath = suggestedSavePath(".", time.Now())
				a.screen = screenPrompt
			}
		case '\t':
			a.tab = ViewTab((int(a.tab) + 1) % 4)
			a.cursor = 0
			a.scroll = 0
		}
	}
}

func (a *App) refresh() {
	a.statusMsg = "Refreshing scan..."
	a.draw()
	res, err := a.liveRun()
	if err != nil {
		a.statusMsg = fmt.Sprintf("Refresh failed: %v", err)
		return
	}
	a.Idx = report.Index(res.Report)
	a.liveJSON = res.RawJSON
	a.ScanExit = res.ExitCode
	a.cursor, a.scroll = 0, 0
	a.statusMsg = "Scan refreshed"
}

func (a *App) handleDetailKey(ev terminal.KeyEvent) {
	switch ev.Key {
	case terminal.KeyQuit:
		a.quit = true
	case terminal.KeyEsc:
		a.screen = screenBrowse
		a.detailScroll = 0
	case terminal.KeyUp:
		if a.detailScroll > 0 {
			a.detailScroll--
		}
	case terminal.KeyDown:
		a.detailScroll++
	case terminal.KeyPageUp:
		a.detailScroll -= a.pageStep()
		if a.detailScroll < 0 {
			a.detailScroll = 0
		}
	case terminal.KeyPageDown:
		a.detailScroll += a.pageStep()
	case terminal.KeyPID:
		a.moveDetail(-1)
	case terminal.KeyRune:
		if ev.Rune == 'n' {
			a.moveDetail(1)
		}
	}
}

func (a *App) moveDetail(delta int) {
	pos := a.detailPos + delta
	if pos < 0 || pos >= len(a.detailItems) {
		return
	}
	a.detailPos = pos
	a.detailScroll = 0
	if a.screen == screenFinding {
		a.listIndex = a.detailItems[pos]
	} else if a.screen == screenPID {
		a.pid = a.detailItems[pos]
	}
	if a.detailFromList {
		a.cursor = pos
	}
}

func (a *App) pageStep() int {
	n := a.pageRows
	if n < 2 {
		return 1
	}
	return n - 1
}

func (a *App) moveList(delta int) {
	n := a.listLen()
	if n <= 0 {
		return
	}
	a.cursor += delta
	if a.cursor < 0 {
		a.cursor = 0
	}
	if a.cursor >= n {
		a.cursor = n - 1
	}
}

func (a *App) movePicker(delta int) {
	n := len(a.picker)
	if n <= 0 {
		return
	}
	a.pickerCursor += delta
	if a.pickerCursor < 0 {
		a.pickerCursor = 0
	}
	if a.pickerCursor >= n {
		a.pickerCursor = n - 1
	}
}

func (a *App) handlePromptKey(ev terminal.KeyEvent) {
	if a.promptCursor > len(a.promptInput) {
		a.promptCursor = len(a.promptInput)
	}
	switch ev.Key {
	case terminal.KeyEsc:
		a.screen = screenBrowse
	case terminal.KeyEnter:
		a.submitPrompt()
	case terminal.KeyBackspace:
		if a.promptCursor > 0 {
			_, size := utf8.DecodeLastRuneInString(a.promptInput[:a.promptCursor])
			a.promptInput = a.promptInput[:a.promptCursor-size] + a.promptInput[a.promptCursor:]
			a.promptCursor -= size
		}
	case terminal.KeyDelete:
		if a.promptCursor < len(a.promptInput) {
			_, size := utf8.DecodeRuneInString(a.promptInput[a.promptCursor:])
			a.promptInput = a.promptInput[:a.promptCursor] + a.promptInput[a.promptCursor+size:]
		}
	case terminal.KeyLeft:
		if a.promptCursor > 0 {
			_, size := utf8.DecodeLastRuneInString(a.promptInput[:a.promptCursor])
			a.promptCursor -= size
		}
	case terminal.KeyRight:
		if a.promptCursor < len(a.promptInput) {
			_, size := utf8.DecodeRuneInString(a.promptInput[a.promptCursor:])
			a.promptCursor += size
		}
	default:
		// Navigation keys are text inside a prompt.
		if ev.Rune >= 32 && ev.Rune != 127 {
			a.promptInput = a.promptInput[:a.promptCursor] + string(ev.Rune) + a.promptInput[a.promptCursor:]
			a.promptCursor += utf8.RuneLen(ev.Rune)
			return
		}
		if ev.Key == terminal.KeyQuit {
			a.quit = true
		}
	}
}

func (a *App) submitPrompt() {
	switch a.promptKind {
	case "filter":
		a.filter = strings.TrimSpace(a.promptInput)
		a.cursor = 0
		a.scroll = 0
		a.screen = screenBrowse
	case "pid":
		pid, err := terminal.ParsePID(a.promptInput)
		if err != nil {
			a.statusMsg = err.Error()
			a.screen = screenBrowse
			return
		}
		if _, ok := a.Idx.ProcessByPID[pid]; !ok {
			a.statusMsg = fmt.Sprintf("PID %d not in report", pid)
			a.screen = screenBrowse
			return
		}
		a.pid = pid
		a.detailItems = a.detailItems[:0]
		a.detailPos = 0
		a.detailFromList = false
		for i, p := range a.Idx.SortedProcesses {
			a.detailItems = append(a.detailItems, p.PID)
			if p.PID == pid {
				a.detailPos = i
			}
		}
		a.detailScroll = 0
		a.screen = screenPID
		a.statusMsg = ""
	case "save":
		path := strings.TrimSpace(a.promptInput)
		if path == "" {
			path = a.saveDefaultPath
		}
		format := "json"
		switch strings.ToLower(filepath.Ext(path)) {
		case "":
			path += ".json"
		case ".json":
		case ".txt":
			format = "txt"
		default:
			a.statusMsg = "Use .json or .txt, or leave off the extension"
			return
		}
		if err := exporter.Save(path, format, a.liveJSON, a.Idx.Report); err != nil {
			a.statusMsg = fmt.Sprintf("Save failed: %v", err)
			return
		}
		a.statusMsg = "Saved " + filepath.Clean(path)
		a.screen = screenBrowse
	}
}

func suggestedSavePath(dir string, now time.Time) string {
	base := "procsift-" + now.Local().Format("20060102-150405")
	for n := 1; ; n++ {
		name := base
		if n > 1 {
			name = fmt.Sprintf("%s-%d", base, n)
		}
		path := filepath.Join(dir, name+".json")
		if _, err := os.Lstat(path); err != nil {
			return path
		}
	}
}

func (a *App) openSelection() {
	switch a.tab {
	case TabFindings, TabScannerSelf:
		list := a.currentFindings()
		if len(list) == 0 {
			return
		}
		if a.cursor >= len(list) {
			return
		}
		a.listIndex = a.globalFindingIndex(list[a.cursor])
		a.detailItems = a.detailItems[:0]
		for _, f := range list {
			a.detailItems = append(a.detailItems, a.globalFindingIndex(f))
		}
		a.detailPos = a.cursor
		a.detailFromList = true
		a.detailScroll = 0
		a.screen = screenFinding
	case TabProcesses:
		list := a.filteredProcesses()
		if a.cursor >= len(list) {
			return
		}
		a.pid = list[a.cursor].PID
		a.detailItems = a.detailItems[:0]
		for _, p := range list {
			a.detailItems = append(a.detailItems, p.PID)
		}
		a.detailPos = a.cursor
		a.detailFromList = true
		a.detailScroll = 0
		a.screen = screenPID
	}
}

func (a *App) globalFindingIndex(f report.Finding) int {
	for i, g := range a.Idx.AllFindings {
		if g.Rule == f.Rule && g.PID == f.PID && g.Subject == f.Subject && g.Summary == f.Summary {
			return i
		}
	}
	return 0
}

func (a *App) listLen() int {
	switch a.tab {
	case TabFindings, TabScannerSelf:
		return len(a.currentFindings())
	case TabProcesses:
		return len(a.filteredProcesses())
	case TabCoverage:
		n := len(a.Idx.Report.Coverage) + len(a.Idx.Report.Limits) + len(a.Idx.Report.Notes)
		return n
	}
	return 0
}

func (a *App) currentFindings() []report.Finding {
	var base []report.Finding
	if a.tab == TabScannerSelf {
		base = a.Idx.ScannerFindings
	} else {
		base = a.Idx.HostFindings
	}
	if a.filter == "" {
		return base
	}
	out := make([]report.Finding, 0, len(base))
	for _, f := range base {
		if matchesFilter(f, a.filter) {
			out = append(out, f)
		}
	}
	return out
}

func (a *App) filteredProcesses() []report.Process {
	if a.filter == "" {
		return a.Idx.SortedProcesses
	}
	out := make([]report.Process, 0)
	for _, p := range a.Idx.SortedProcesses {
		if matchesProcessFilter(p, a.filter) {
			out = append(out, p)
		}
	}
	return out
}

func (a *App) draw() {
	if a.Term == nil {
		return
	}
	w, h := a.Term.Size()
	terminal.Paint(a.Term.Out(), a.renderRows(w, h))
}

func (a *App) coverageLines() []string {
	var lines []string
	for _, c := range a.Idx.Report.Coverage {
		detail := c.Detail
		if detail != "" {
			lines = append(lines, fmt.Sprintf("[%s] %s — %s", c.State, c.Source, detail))
		} else {
			lines = append(lines, fmt.Sprintf("[%s] %s", c.State, c.Source))
		}
	}
	for _, l := range a.Idx.Report.Limits {
		lines = append(lines, "limit: "+l)
	}
	for _, n := range a.Idx.Report.Notes {
		lines = append(lines, "note: "+n)
	}
	if len(lines) == 0 {
		lines = append(lines, "(no coverage issues recorded)")
	}
	return lines
}

// FolderResult holds the outcome of folder selection.
type FolderResult struct {
	Idx        *report.IndexedReport
	SourcePath string
	Quit       bool
}

// SelectFromFolder runs the picker and returns the chosen report.
func SelectFromFolder(t *terminal.Term, entries []offline.Entry) (FolderResult, error) {
	app := &App{Term: t, picker: entries, screen: screenFolder}
	for {
		if app.screen != screenFolder {
			return FolderResult{Idx: app.Idx, SourcePath: app.SourcePath}, nil
		}
		app.draw()
		ev, err := t.ReadEvent()
		if err != nil {
			return FolderResult{}, err
		}
		switch ev.Key {
		case terminal.KeyResize:
			continue
		case terminal.KeyQuit, terminal.KeyEsc:
			return FolderResult{Quit: true}, nil
		case terminal.KeyUp:
			app.movePicker(-1)
		case terminal.KeyDown:
			app.movePicker(1)
		case terminal.KeyPageUp:
			app.movePicker(-app.pageStep())
		case terminal.KeyPageDown:
			app.movePicker(app.pageStep())
		case terminal.KeyEnter:
			if len(app.picker) == 0 {
				app.pickerErr = "no JSON reports in folder"
				continue
			}
			path := app.picker[app.pickerCursor].Path
			rep, err := offline.ReadFile(path)
			if err != nil {
				app.pickerErr = err.Error()
				continue
			}
			return FolderResult{Idx: report.Index(rep), SourcePath: path}, nil
		}
	}
}

func unknownIfEmpty(s string) string {
	if s == "" {
		return "unknown"
	}
	return clean(s)
}

func unknownUint(v uint64) string {
	if v == 0 {
		return "unknown"
	}
	return fmt.Sprintf("%d", v)
}

func unknownBool(v, known bool) string {
	if !known {
		return "unknown"
	}
	if v {
		return "true"
	}
	return "false"
}
