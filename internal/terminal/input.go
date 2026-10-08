package terminal

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"golang.org/x/sys/unix"
)

// Key represents one decoded key press.
type Key int

const (
	KeyNone Key = iota
	KeyUp
	KeyDown
	KeyPageUp
	KeyPageDown
	KeyEnter
	KeyEsc
	KeyTab
	KeyBackspace
	KeyQuit
	KeyDetail
	KeyPID
	KeyFilter
	KeyRune
	KeyResize
)

// KeyEvent is a decoded input event.
type KeyEvent struct {
	Key  Key
	Rune rune
}

// ReadKey reads one key from raw terminal input.
func ReadKey(r io.Reader) (KeyEvent, error) {
	first, ok, err := readByte(r, false)
	if err != nil {
		return KeyEvent{}, err
	}
	if !ok {
		return KeyEvent{Key: KeyNone}, nil
	}
	switch first {
	case 3, 4: // Ctrl+C, Ctrl+D
		return KeyEvent{Key: KeyQuit}, nil
	case '\r', '\n':
		return KeyEvent{Key: KeyEnter}, nil
	case 27:
		prefix, ok, err := readByte(r, true)
		if err != nil || !ok || (prefix != '[' && prefix != 'O') {
			return KeyEvent{Key: KeyEsc}, err
		}
		code, ok, err := readByte(r, true)
		if err != nil || !ok {
			return KeyEvent{Key: KeyEsc}, err
		}
		switch code {
		case 'A':
			return KeyEvent{Key: KeyUp}, nil
		case 'B':
			return KeyEvent{Key: KeyDown}, nil
		case '5', '6':
			if prefix == '[' {
				end, ok, err := readByte(r, true)
				if err != nil {
					return KeyEvent{}, err
				}
				if ok && end == '~' {
					if code == '5' {
						return KeyEvent{Key: KeyPageUp}, nil
					}
					return KeyEvent{Key: KeyPageDown}, nil
				}
			}
		}
		return KeyEvent{Key: KeyEsc}, nil
	case 9:
		return KeyEvent{Key: KeyTab}, nil
	case 127, 8:
		return KeyEvent{Key: KeyBackspace}, nil
	case 'q':
		return KeyEvent{Key: KeyQuit, Rune: 'q'}, nil
	case 'j':
		return KeyEvent{Key: KeyDown, Rune: 'j'}, nil
	case 'k':
		return KeyEvent{Key: KeyUp, Rune: 'k'}, nil
	case 'p':
		return KeyEvent{Key: KeyPID, Rune: 'p'}, nil
	case '/':
		return KeyEvent{Key: KeyFilter, Rune: '/'}, nil
	default:
		rn, _ := decodeRune([]byte{first})
		if rn != 0 {
			return KeyEvent{Key: KeyRune, Rune: rn}, nil
		}
	}
	return KeyEvent{Key: KeyNone}, nil
}

// readByte consumes exactly one byte so a read cannot discard later keypresses.
// After Escape, a short wait distinguishes an arrow sequence from Esc alone.
func readByte(r io.Reader, afterEscape bool) (byte, bool, error) {
	if afterEscape {
		if f, ok := r.(*os.File); ok && IsTTY(int(f.Fd())) {
			fds := []unix.PollFd{{Fd: int32(f.Fd()), Events: unix.POLLIN}}
			n, err := unix.Poll(fds, 40)
			if err == unix.EINTR || n == 0 {
				return 0, false, nil
			}
			if err != nil {
				return 0, false, err
			}
		}
	}
	var buf [1]byte
	n, err := r.Read(buf[:])
	if n == 1 {
		return buf[0], true, nil
	}
	if err == io.EOF {
		return 0, false, nil
	}
	return 0, false, err
}

func decodeRune(b []byte) (rune, int) {
	if len(b) == 0 {
		return 0, 0
	}
	return rune(b[0]), 1
}

// ReadLine reads a visible line for prompts (caller draws the prompt).
func ReadLine(r io.Reader, w io.Writer, initial string) (string, error) {
	line := initial
	for {
		_, _ = fmt.Fprintf(w, "\r%s ", line)
		ev, err := ReadKey(r)
		if err != nil {
			return "", err
		}
		switch ev.Key {
		case KeyEnter:
			_, _ = fmt.Fprint(w, "\r\n")
			return strings.TrimSpace(line), nil
		case KeyEsc:
			_, _ = fmt.Fprint(w, "\r\n")
			return "", fmt.Errorf("cancelled")
		case KeyBackspace:
			if line != "" {
				line = line[:len(line)-1]
			}
		case KeyRune:
			line += string(ev.Rune)
		case KeyQuit:
			return "", fmt.Errorf("cancelled")
		}
	}
}

// ParsePID parses a decimal PID from prompt input.
func ParsePID(s string) (int, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("empty PID")
	}
	pid, err := strconv.Atoi(s)
	if err != nil || pid <= 0 {
		return 0, fmt.Errorf("invalid PID %q", s)
	}
	return pid, nil
}

// DrainBufferedKeys discards pending input after a resize or mode change.
func DrainBufferedKeys(r io.Reader) {
	br, ok := r.(*bufio.Reader)
	if !ok {
		return
	}
	for br.Buffered() > 0 {
		if _, err := br.ReadByte(); err != nil {
			break
		}
	}
}
