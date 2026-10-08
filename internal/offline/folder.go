package offline

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const maxFolderEntries = 1000

// Entry is one selectable JSON report in a folder.
type Entry struct {
	Path    string
	ModTime int64
	Size    int64
}

// ListFolder returns regular .json files directly in dir, newest first.
func ListFolder(dir string) ([]Entry, error) {
	info, err := os.Stat(dir)
	if err != nil {
		return nil, fmt.Errorf("folder: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("folder %s is not a directory", dir)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read folder: %w", err)
	}
	var out []Entry
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasSuffix(strings.ToLower(name), ".json") {
			continue
		}
		path := filepath.Join(dir, name)
		fi, err := os.Lstat(path)
		if err != nil {
			continue
		}
		if fi.Mode()&os.ModeSymlink != 0 {
			continue
		}
		if !fi.Mode().IsRegular() {
			continue
		}
		out = append(out, Entry{Path: path, ModTime: fi.ModTime().Unix(), Size: fi.Size()})
		if len(out) >= maxFolderEntries {
			break
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].ModTime != out[j].ModTime {
			return out[i].ModTime > out[j].ModTime
		}
		return out[i].Path < out[j].Path
	})
	return out, nil
}
