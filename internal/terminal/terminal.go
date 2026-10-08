package terminal

import (
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

	"golang.org/x/sys/unix"
	"golang.org/x/term"
)

// Term manages raw mode and restoration for the controlling terminal.
type Term struct {
	in         *os.File
	out        *os.File
	old        *term.State
	w          int
	h          int
	wakeR      *os.File
	wakeW      *os.File
	stopResize func()
}

// IsTTY reports whether fd refers to a terminal.
func IsTTY(fd int) bool {
	return term.IsTerminal(fd)
}

// Open switches stdin/stdout into raw mode when both are terminals.
// The UI uses the alternate screen so quitting restores the previous view.
func Open() (*Term, error) {
	if !IsTTY(int(os.Stdin.Fd())) || !IsTTY(int(os.Stdout.Fd())) {
		return nil, fmt.Errorf("interactive mode requires a terminal (stdin and stdout must be TTYs)")
	}
	old, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return nil, fmt.Errorf("enable raw mode: %w", err)
	}
	wakeR, wakeW, err := os.Pipe()
	if err != nil {
		_ = term.Restore(int(os.Stdin.Fd()), old)
		return nil, fmt.Errorf("resize pipe: %w", err)
	}
	_ = syscall.SetNonblock(int(wakeR.Fd()), true)
	_ = syscall.SetNonblock(int(wakeW.Fd()), true)
	t := &Term{in: os.Stdin, out: os.Stdout, old: old, wakeR: wakeR, wakeW: wakeW}
	t.updateSize()
	t.watchResize()
	_, _ = fmt.Fprint(t.out, "\x1b[?1049h\x1b[?7l\x1b[2J\x1b[H\x1b[?25l")
	return t, nil
}

// Close restores the terminal state.
func (t *Term) Close() {
	if t == nil || t.old == nil {
		return
	}
	if t.stopResize != nil {
		t.stopResize()
		t.stopResize = nil
	}
	if t.out != nil {
		_, _ = fmt.Fprint(t.out, "\x1b[0m\x1b[?7h\x1b[?25h\x1b[?1049l")
	}
	_ = term.Restore(int(t.in.Fd()), t.old)
	t.old = nil
	if t.wakeR != nil {
		_ = t.wakeR.Close()
		t.wakeR = nil
	}
	if t.wakeW != nil {
		_ = t.wakeW.Close()
		t.wakeW = nil
	}
}

func (t *Term) updateSize() {
	w, h, err := term.GetSize(int(t.out.Fd()))
	if err != nil || w < 1 || h < 1 {
		w, h = 80, 24
	}
	t.w, t.h = w, h
}

// Size returns the current terminal dimensions.
func (t *Term) Size() (width, height int) {
	t.updateSize()
	return t.w, t.h
}

func (t *Term) watchResize() {
	sigs := make(chan os.Signal, 4)
	done := make(chan struct{})
	signal.Notify(sigs, syscall.SIGWINCH)
	t.stopResize = func() {
		signal.Stop(sigs)
		select {
		case <-done:
		default:
			close(done)
		}
	}
	go func() {
		for {
			select {
			case <-done:
				return
			case <-sigs:
				var b [1]byte
				_, _ = t.wakeW.Write(b[:])
			}
		}
	}()
}

func (t *Term) drainWake() {
	if t.wakeR == nil {
		return
	}
	var buf [256]byte
	for {
		n, err := t.wakeR.Read(buf[:])
		if n == 0 || err != nil {
			return
		}
	}
}

// ReadEvent waits for one key, or for a terminal resize.
func (t *Term) ReadEvent() (KeyEvent, error) {
	if t == nil || t.in == nil || t.wakeR == nil {
		return KeyEvent{}, fmt.Errorf("terminal is closed")
	}
	for {
		fds := []unix.PollFd{
			{Fd: int32(t.in.Fd()), Events: unix.POLLIN},
			{Fd: int32(t.wakeR.Fd()), Events: unix.POLLIN},
		}
		_, err := unix.Poll(fds, -1)
		if err == unix.EINTR {
			continue
		}
		if err != nil {
			return KeyEvent{}, fmt.Errorf("read terminal: %w", err)
		}
		if fds[1].Revents&(unix.POLLIN|unix.POLLERR|unix.POLLHUP) != 0 {
			t.drainWake()
			t.updateSize()
			return KeyEvent{Key: KeyResize}, nil
		}
		if fds[0].Revents&unix.POLLIN != 0 {
			return ReadKey(t.in)
		}
		if fds[0].Revents&(unix.POLLHUP|unix.POLLERR|unix.POLLNVAL) != 0 {
			return KeyEvent{}, io.EOF
		}
	}
}

// Out returns stdout for drawing.
func (t *Term) Out() io.Writer { return t.out }
