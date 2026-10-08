package terminal_test

import (
	"strings"
	"testing"

	"github.com/dahnutz/procsift-ui/internal/terminal"
)

func TestIsTTYWithoutTerminal(t *testing.T) {
	// stdin in go test is usually not a TTY
	if terminal.IsTTY(0) {
		t.Skip("stdin is a TTY in this environment")
	}
}

func TestReadKeyDecoding(t *testing.T) {
	tests := []struct {
		in  string
		key terminal.Key
	}{
		{"q", terminal.KeyQuit},
		{"j", terminal.KeyDown},
		{"k", terminal.KeyUp},
		{"\x1b[A", terminal.KeyUp},
		{"\x1b[B", terminal.KeyDown},
		{"\x1b[5~", terminal.KeyPageUp},
		{"\x1b[6~", terminal.KeyPageDown},
		{"\r", terminal.KeyEnter},
	}
	for _, tc := range tests {
		ev, err := terminal.ReadKey(strings.NewReader(tc.in))
		if err != nil {
			t.Fatal(err)
		}
		if ev.Key != tc.key {
			t.Fatalf("%q: got %v want %v", tc.in, ev.Key, tc.key)
		}
	}
}

func TestReadKeyKeepsFollowingKey(t *testing.T) {
	r := strings.NewReader("jk")
	first, err := terminal.ReadKey(r)
	if err != nil || first.Key != terminal.KeyDown {
		t.Fatalf("first key: %+v, %v", first, err)
	}
	second, err := terminal.ReadKey(r)
	if err != nil || second.Key != terminal.KeyUp {
		t.Fatalf("second key: %+v, %v", second, err)
	}
}
