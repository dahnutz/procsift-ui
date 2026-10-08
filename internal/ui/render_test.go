package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/dahnutz/procsift-ui/internal/report"
	"github.com/dahnutz/procsift-ui/internal/terminal"
)

func TestMatchesFilter(t *testing.T) {
	b, err := os.ReadFile(filepath.Join("..", "..", "testdata", "sample.json"))
	if err != nil {
		t.Fatal(err)
	}
	rep, err := report.ParseJSON(b)
	if err != nil {
		t.Fatal(err)
	}
	idx := report.Index(rep)
	f := idx.HostFindings[0]
	if !matchesFilter(f, "deleted") {
		t.Fatal("expected deleted match")
	}
	if matchesFilter(f, "nomatchxyz") {
		t.Fatal("unexpected match")
	}
}

func TestClipKeepsRuneBoundary(t *testing.T) {
	got := clip("éééé", 2)
	if got != "é…" {
		t.Fatalf("clip = %q", got)
	}
	if !utf8.ValidString(got) {
		t.Fatalf("invalid utf-8 %q", got)
	}
	if colWidth(got) > 2 {
		t.Fatalf("width %d", colWidth(got))
	}
}

func TestWrapTextStaysInWidth(t *testing.T) {
	parts := wrapText(strings.Repeat("alpha ", 40), 12)
	if len(parts) < 2 {
		t.Fatalf("expected wrap, got %#v", parts)
	}
	for _, p := range parts {
		if colWidth(p) > 12 {
			t.Fatalf("segment %q width %d", p, colWidth(p))
		}
	}
}

func TestFindingColumnsFit(t *testing.T) {
	for width := 1; width <= 160; width++ {
		cols := fitFindingCols(width)
		if width >= 2 && findingBudget(cols) > width {
			t.Fatalf("width %d budget %d cols %+v", width, findingBudget(cols), cols)
		}
		line := findingLine(report.Finding{
			Severity: "medium", PID: 4242, Rule: "PROC-DELETED-EXE",
			Process: "worker", Summary: strings.Repeat("summary ", 30),
		}, width%2 == 0, cols, width)
		if visibleWidth(line) > width {
			t.Fatalf("width %d visible %d %q", width, visibleWidth(line), line)
		}
	}
}

func TestRenderRowsStayInsideTerminal(t *testing.T) {
	app := sampleApp(t)
	for _, size := range [][2]int{{80, 24}, {40, 12}, {20, 6}} {
		rows := app.renderRows(size[0], size[1])
		if len(rows) != size[1] {
			t.Fatalf("%v: got %d rows", size, len(rows))
		}
		for i, row := range rows {
			if strings.ContainsAny(row, "\n\r") {
				t.Fatalf("%v row %d contains a newline: %q", size, i, row)
			}
			if visibleWidth(row) > size[0] {
				t.Fatalf("%v row %d width %d: %q", size, i, visibleWidth(row), row)
			}
		}
	}
	rows := app.renderRows(80, 24)
	joined := strings.Join(rows, "\n")
	plain := oneLine(joined)
	for _, want := range []string{"ProcSift", "example", "incomplete", "PROC-DELETED-EXE", "4242", "worker"} {
		if !strings.Contains(plain, want) {
			t.Fatalf("missing %q\n%s", want, plain)
		}
	}
	if !strings.Contains(rows[0], "high 1") || !strings.Contains(rows[0], "med 2") {
		t.Fatalf("header counts: %s", oneLine(rows[0]))
	}
}

func TestFindingColumnsAlign(t *testing.T) {
	app := sampleApp(t)
	rows := app.renderRows(100, 24)
	var rules []int
	var pids []int
	for _, row := range rows {
		plain := oneLine(row)
		for _, rule := range []string{"PROC-DELETED-EXE", "PROC-UNOWNED-LIB", "FILE-PKG-MISMATCH"} {
			if i := strings.Index(plain, rule); i >= 0 {
				rules = append(rules, i)
			}
		}
		for _, pid := range []string{"4242", "1830"} {
			if i := strings.Index(plain, pid); i >= 0 {
				pids = append(pids, i)
			}
		}
	}
	if len(rules) != 3 {
		t.Fatalf("rules found at %v", rules)
	}
	if rules[0] != rules[1] || rules[1] != rules[2] {
		t.Fatalf("rule columns differ: %v", rules)
	}
	if len(pids) != 2 || pids[0] != pids[1] {
		t.Fatalf("pid columns differ: %v", pids)
	}
}

func TestRenderSanitizesReportText(t *testing.T) {
	app := sampleApp(t)
	app.Idx.HostFindings[0].Summary = "before\x1b[2Jafter\nnext"
	app.Idx.HostFindings[0].Rule = "PROC-DELETED-EXE"
	rows := app.renderRows(80, 24)
	for _, row := range rows {
		if strings.Contains(row, "\x1b[2J") || strings.Contains(row, "\n") {
			t.Fatalf("report control leaked: %q", row)
		}
	}
	if !strings.Contains(oneLine(strings.Join(rows, " ")), "before") {
		t.Fatal("summary text dropped")
	}
}

func TestDetailWrapsLongFields(t *testing.T) {
	app := sampleApp(t)
	p := app.Idx.ProcessByPID[4242]
	p.Cmdline = strings.Repeat("A", 240) + "\n\x1b[31mEND"
	app.Idx.ProcessByPID[4242] = p
	app.screen = screenPID
	app.pid = 4242
	const width = 48
	rows := app.renderRows(width, 30)
	var got strings.Builder
	for _, row := range rows {
		if strings.ContainsAny(row, "\n\r") || visibleWidth(row) > width {
			t.Fatalf("bad row %q width %d", row, visibleWidth(row))
		}
		if strings.Contains(row, "\x1b[31m") {
			t.Fatalf("color from report kept: %q", row)
		}
		got.WriteString(oneLine(row))
	}
	text := got.String()
	if strings.Count(text, "A") != 240 || !strings.Contains(text, "END") {
		t.Fatalf("wrapped cmdline lost text (%d A's)", strings.Count(text, "A"))
	}
	if !strings.Contains(text, "boot-abc") || !strings.Contains(text, "abc123") || !strings.Contains(text, "4444") {
		t.Fatalf("pid detail missing fields: %s", text)
	}
}

func TestScannerTabUsesScannerCounts(t *testing.T) {
	app := sampleApp(t)
	app.tab = TabScannerSelf
	rows := app.renderRows(80, 24)
	plain := oneLine(strings.Join(rows, "\n"))
	if !strings.Contains(plain, "PROC-MEMFD-EXE") {
		t.Fatalf("scanner finding missing: %s", plain)
	}
	if !strings.Contains(rows[0], "med 0") {
		t.Fatalf("scanner counts: %s", oneLine(rows[0]))
	}
}

func TestPromptTypesNavigationLetters(t *testing.T) {
	a := &App{screen: screenPrompt, promptKind: "filter"}
	a.handlePromptKey(terminal.KeyEvent{Key: terminal.KeyDown, Rune: 'j'})
	a.handlePromptKey(terminal.KeyEvent{Key: terminal.KeyUp, Rune: 'k'})
	a.handlePromptKey(terminal.KeyEvent{Key: terminal.KeyQuit, Rune: 'q'})
	a.handlePromptKey(terminal.KeyEvent{Key: terminal.KeyPID, Rune: 'p'})
	a.handlePromptKey(terminal.KeyEvent{Key: terminal.KeyFilter, Rune: '/'})
	if a.promptInput != "jkqp/" {
		t.Fatalf("prompt %q", a.promptInput)
	}
	if a.quit {
		t.Fatal("letter q quit the prompt")
	}
	a.handlePromptKey(terminal.KeyEvent{Key: terminal.KeyQuit})
	if !a.quit {
		t.Fatal("ctrl-c should quit")
	}
}

func TestDetailQuitAndBack(t *testing.T) {
	a := &App{screen: screenFinding}
	a.handleDetailKey(terminal.KeyEvent{Key: terminal.KeyEsc})
	if a.screen != screenBrowse || a.quit {
		t.Fatalf("esc screen %v quit %v", a.screen, a.quit)
	}
	a.screen = screenFinding
	a.handleDetailKey(terminal.KeyEvent{Key: terminal.KeyQuit, Rune: 'q'})
	if !a.quit {
		t.Fatal("q should quit from a detail view")
	}
}

func TestPageKeysScrollByScreen(t *testing.T) {
	app := sampleApp(t)
	for i := 0; i < 40; i++ {
		app.Idx.HostFindings = append(app.Idx.HostFindings, report.Finding{
			Rule: "EXTRA", Severity: "info", Summary: "row",
		})
	}
	_ = app.renderRows(80, 24)
	step := app.pageStep()
	if step < 2 {
		t.Fatalf("page step %d", step)
	}
	app.handleBrowseKey(terminal.KeyEvent{Key: terminal.KeyPageDown})
	if app.cursor != step {
		t.Fatalf("cursor %d, step %d", app.cursor, step)
	}
	app.handleBrowseKey(terminal.KeyEvent{Key: terminal.KeyPageDown})
	if app.cursor != step*2 {
		t.Fatalf("cursor %d", app.cursor)
	}
	app.handleBrowseKey(terminal.KeyEvent{Key: terminal.KeyPageUp})
	if app.cursor != step {
		t.Fatalf("cursor after page up %d", app.cursor)
	}
	app.detailScroll = 30
	app.handleDetailKey(terminal.KeyEvent{Key: terminal.KeyPageUp})
	if app.detailScroll != 30-step {
		t.Fatalf("detail scroll %d", app.detailScroll)
	}
	app.detailScroll = 2
	app.handleDetailKey(terminal.KeyEvent{Key: terminal.KeyPageUp})
	if app.detailScroll != 0 {
		t.Fatalf("detail scroll %d", app.detailScroll)
	}
}

func sampleApp(t *testing.T) *App {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("..", "..", "testdata", "sample.json"))
	if err != nil {
		t.Fatal(err)
	}
	rep, err := report.ParseJSON(b)
	if err != nil {
		t.Fatal(err)
	}
	return NewApp(nil, report.Index(rep), "", -1)
}
