//go:build linux

package emqutiti

import (
	"errors"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/term"
	"golang.org/x/sys/unix"
)

// ctrlEnterInputOptions is appended to runUI's options only. Keeping os.File's
// Fd/Name/Write/Close preserves Tea's raw mode, restoration and cancelreader.
// Non-TTY stdin keeps Tea's default /dev/tty fallback instead of redirecting it.
func ctrlEnterInputOptions() ([]tea.ProgramOption, func()) {
	if !term.IsTerminal(os.Stdin.Fd()) {
		return nil, func() {}
	}
	f := &ctrlEnterTerminalInput{File: os.Stdin}
	f.reader = newCtrlEnterInputReader(f.readNext)
	return []tea.ProgramOption{tea.WithInput(f), tea.WithFilter(f.reader.filter)}, f.reader.close
}

type ctrlEnterTerminalInput struct {
	*os.File
	reader   *ctrlEnterInputReader
	deadline time.Time
}

func (f *ctrlEnterTerminalInput) Read(dst []byte) (int, error) {
	// A rune pushed back by suffix lookahead starts a new bounded frame.
	f.deadline = time.Time{}
	return f.reader.Read(dst)
}

func (f *ctrlEnterTerminalInput) readNext(dst []byte, continuation bool) (int, error) {
	if !continuation {
		f.deadline = time.Time{}
	}
	if continuation {
		// Bound ESC latency and time spent inside Read after cancellation.
		if f.deadline.IsZero() {
			f.deadline = time.Now().Add(50 * time.Millisecond)
		}
		for {
			remaining := time.Until(f.deadline)
			if remaining <= 0 {
				return 0, errCtrlEnterInputTimeout
			}
			fds := []unix.PollFd{{Fd: int32(f.Fd()), Events: unix.POLLIN}}
			n, err := unix.Poll(fds, int((remaining+time.Millisecond-1)/time.Millisecond))
			if errors.Is(err, unix.EINTR) {
				continue
			}
			if err != nil {
				return 0, err
			}
			if n == 0 {
				return 0, errCtrlEnterInputTimeout
			}
			break
		}
	}
	// Tea's cancelreader polls this file descriptor before each Read. Do not
	// prefetch bytes into a private buffer: buffered keys would be stranded
	// until another physical key makes the descriptor readable again.
	return f.File.Read(dst[:1])
}
