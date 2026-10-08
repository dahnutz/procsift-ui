package offline_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/dahnutz/procsift-ui/internal/offline"
)

func TestReadFileSample(t *testing.T) {
	path := filepath.Join("..", "..", "testdata", "sample.json")
	rep, err := offline.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Findings) != 4 {
		t.Fatalf("findings: %d", len(rep.Findings))
	}
}

func TestReadFileRejectsManifest(t *testing.T) {
	path := filepath.Join("..", "..", "testdata", "manifest.json")
	if _, err := offline.ReadFile(path); err == nil {
		t.Fatal("expected error for capture manifest")
	}
}

func TestListFolderMultipleReports(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"a.json", "b.json", "note.txt"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(`{"tool":"procsift","version":"0.5.1","findings":[]}`), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	entries, err := offline.ListFolder(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("entries: %d", len(entries))
	}
}
