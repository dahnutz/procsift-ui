package ui

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/dahnutz/procsift-ui/internal/report"
	"github.com/dahnutz/procsift-ui/internal/scanner"
	"github.com/dahnutz/procsift-ui/internal/terminal"
)

func key(r rune) terminal.KeyEvent { return terminal.KeyEvent{Key: terminal.KeyRune, Rune: r} }

func TestFindingDetailNextPreviousUsesFilteredOrder(t *testing.T) {
	a := sampleApp(t)
	a.filter = "PROC-"
	a.openSelection()
	if a.screen != screenFinding || a.Idx.AllFindings[a.listIndex].Rule != "PROC-DELETED-EXE" {
		t.Fatal("first finding not opened")
	}
	a.detailScroll = 3
	a.handleDetailKey(key('n'))
	if a.Idx.AllFindings[a.listIndex].Rule != "PROC-UNOWNED-LIB" || a.detailScroll != 0 || a.cursor != 1 {
		t.Fatalf("next finding: index %d cursor %d scroll %d", a.listIndex, a.cursor, a.detailScroll)
	}
	a.handleDetailKey(key('n'))
	if a.cursor != 1 {
		t.Fatal("advanced past filtered list")
	}
	a.handleDetailKey(terminal.KeyEvent{Key: terminal.KeyPID, Rune: 'p'})
	if a.Idx.AllFindings[a.listIndex].Rule != "PROC-DELETED-EXE" || a.cursor != 0 {
		t.Fatal("previous finding did not return to first item")
	}
}

func TestProcessDetailNavigationFromListAndPID(t *testing.T) {
	a := sampleApp(t)
	a.tab = TabProcesses
	a.openSelection()
	if a.pid != 1830 {
		t.Fatalf("first PID %d", a.pid)
	}
	a.handleDetailKey(key('n'))
	if a.pid != 4242 || a.cursor != 1 {
		t.Fatalf("next PID %d cursor %d", a.pid, a.cursor)
	}
	a.handleDetailKey(terminal.KeyEvent{Key: terminal.KeyEsc})
	a.tab = TabFindings
	a.promptKind, a.promptInput, a.screen = "pid", "4242", screenPrompt
	a.submitPrompt()
	a.handleDetailKey(terminal.KeyEvent{Key: terminal.KeyPID, Rune: 'p'})
	if a.pid != 1830 || a.cursor != 1 {
		t.Fatalf("prompt PID navigation %d cursor %d", a.pid, a.cursor)
	}
}

func TestLiveRefreshAndSaveCurrentSnapshot(t *testing.T) {
	a := sampleApp(t)
	original := a.Idx.Report.Host.Hostname
	raw := []byte(`{"tool":"procsift","version":"0.5.1","host":{"hostname":"new-host"},"findings":[]}`)
	rep, err := report.ParseJSON(raw)
	if err != nil {
		t.Fatal(err)
	}
	calls := 0
	a.EnableLive([]byte(`{"old":true}`), func() (scanner.Result, error) {
		calls++
		if calls == 1 {
			return scanner.Result{}, os.ErrPermission
		}
		return scanner.Result{Report: rep, RawJSON: raw, ExitCode: 1}, nil
	})
	a.handleBrowseKey(key('r'))
	if a.Idx.Report.Host.Hostname != original || a.ScanExit != -1 || !strings.Contains(a.statusMsg, "Refresh failed") {
		t.Fatal("failed refresh replaced the snapshot")
	}
	a.handleBrowseKey(key('r'))
	if a.Idx.Report.Host.Hostname != "new-host" || a.ScanExit != 1 || calls != 2 {
		t.Fatal("successful refresh did not replace the snapshot")
	}
	a.handleBrowseKey(key('s'))
	if a.screen != screenPrompt || a.promptKind != "save" || a.promptInput != "" {
		t.Fatal("save prompt not opened")
	}
	path := filepath.Join(t.TempDir(), "scan.json")
	a.promptInput = path
	a.submitPrompt()
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, raw) {
		t.Fatal("saved JSON differs from current scanner output")
	}
	if a.screen != screenBrowse {
		t.Fatal("did not return to browse after save")
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0600 {
		t.Fatalf("saved mode %o", info.Mode().Perm())
	}
	a.handleBrowseKey(key('s'))
	textPath := filepath.Join(t.TempDir(), "scan.txt")
	a.promptInput = textPath
	a.submitPrompt()
	text, err := os.ReadFile(textPath)
	if err != nil || !strings.Contains(string(text), "new-host") {
		t.Fatalf("text export: %v", err)
	}
	a.handleBrowseKey(key('s'))
	plainPath := filepath.Join(t.TempDir(), "scan-no-extension")
	a.promptInput = plainPath
	a.submitPrompt()
	if _, err := os.Stat(plainPath + ".json"); err != nil {
		t.Fatalf("default JSON extension: %v", err)
	}
}

func TestSaveSuggestionDoesNotOverwrite(t *testing.T) {
	dir := t.TempDir()
	when := time.Date(2026, 10, 12, 12, 0, 0, 0, time.Local)
	first := suggestedSavePath(dir, when)
	if err := os.WriteFile(first, []byte("first"), 0600); err != nil {
		t.Fatal(err)
	}
	second := suggestedSavePath(dir, when)
	if second == first || !strings.HasSuffix(second, "-2.json") {
		t.Fatalf("second suggestion %q after %q", second, first)
	}
}

func TestPromptArrowEditing(t *testing.T) {
	a := &App{screen: screenPrompt, promptKind: "save", promptInput: "report.txt", promptCursor: len("report.txt")}
	for i := 0; i < 4; i++ {
		a.handlePromptKey(terminal.KeyEvent{Key: terminal.KeyLeft})
	}
	a.handlePromptKey(key('2'))
	if a.promptInput != "report2.txt" || a.screen != screenPrompt {
		t.Fatalf("edit result %q screen %d", a.promptInput, a.screen)
	}
	a.handlePromptKey(terminal.KeyEvent{Key: terminal.KeyBackspace})
	if a.promptInput != "report.txt" {
		t.Fatalf("backspace result %q", a.promptInput)
	}
	a.handlePromptKey(terminal.KeyEvent{Key: terminal.KeyRight})
	a.handlePromptKey(terminal.KeyEvent{Key: terminal.KeyDelete})
	if a.promptInput != "report.xt" {
		t.Fatalf("delete result %q", a.promptInput)
	}
}

func TestOfflineHasNoRefreshOrSave(t *testing.T) {
	a := sampleApp(t)
	a.handleBrowseKey(key('r'))
	a.handleBrowseKey(key('s'))
	if a.screen != screenBrowse || a.promptKind != "" {
		t.Fatal("offline mode opened a live action")
	}
}
