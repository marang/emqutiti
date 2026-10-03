//go:build linux || darwin

package emqutiti

import (
	"bytes"
	"errors"
	"io"
	"os"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
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
	initialState, _ := term.GetState(f.Fd())
	options := []tea.ProgramOption{tea.WithInput(f), tea.WithFilter(f.reader.filter)}
	var output *ctrlEnterTerminalOutput
	if term.IsTerminal(os.Stdout.Fd()) {
		output = &ctrlEnterTerminalOutput{File: os.Stdout}
		options = append(options, tea.WithOutput(output))
	}
	var once sync.Once
	return options, func() {
		once.Do(func() {
			f.reader.close()
			if output != nil {
				_ = output.cleanup()
			}
			// Tea's early startup errors can skip both renderer and raw-mode
			// restoration. Keep this snapshot independent of Tea's Run path.
			if initialState != nil {
				_ = term.Restore(f.Fd(), initialState)
			}
		})
	}
}

// Kitty flags 1 (disambiguate) and 4 (report alternate/shifted key codes).
const modifiedKeysPush = "\x1b[>5u"
const modifiedKeysPop = "\x1b[<u"

// These are the standalone mode sequences emitted by Tea's renderer. Track
// resets too, so normal shutdown/release leaves no work for the fallback.
var ctrlEnterRendererModes = [...]struct{ enable, reset string }{
	{ansi.HideCursor, ansi.ShowCursor},
	{ansi.SetBracketedPasteMode, ansi.ResetBracketedPasteMode},
	{ansi.SetButtonEventMouseMode, ansi.ResetButtonEventMouseMode},
	{ansi.SetAnyEventMouseMode, ansi.ResetAnyEventMouseMode},
	{ansi.SetSgrExtMouseMode, ansi.ResetSgrExtMouseMode},
	{ansi.SetFocusEventMode, ansi.ResetFocusEventMode},
}

// Tea 1.x has no keyboard-protocol option. Attach the request to its renderer's
// screen transitions so the push/pop use the same stack, including shutdown,
// panic recovery, and temporary terminal release. File methods preserve sizing.
type ctrlEnterTerminalOutput struct {
	*os.File
	mu        sync.Mutex
	pushes    int
	altScreen bool
	closed    bool
	modes     [len(ctrlEnterRendererModes)]bool
}

func (f *ctrlEnterTerminalOutput) WriteString(data string) (int, error) {
	return f.Write([]byte(data))
}

func (f *ctrlEnterTerminalOutput) Write(data []byte) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	// An early Run error can leave Tea's renderer alive after cleanup.
	if f.closed {
		return len(data), nil
	}
	if bytes.Equal(data, []byte(ansi.ResetAltScreenSaveCursorMode)) {
		if err := f.popKeyboardMode(); err != nil {
			return 0, err
		}
	}
	n, err := f.File.Write(data)
	if err == nil && n == len(data) {
		for i, mode := range ctrlEnterRendererModes {
			switch {
			case bytes.Equal(data, []byte(mode.enable)):
				f.modes[i] = true
			case bytes.Equal(data, []byte(mode.reset)):
				f.modes[i] = false
			}
		}
		switch {
		case bytes.Equal(data, []byte(ansi.SetAltScreenSaveCursorMode)):
			f.altScreen = true
			if _, err = io.WriteString(f.File, modifiedKeysPush); err == nil {
				f.pushes++
			}
		case bytes.Equal(data, []byte(ansi.ResetAltScreenSaveCursorMode)):
			f.altScreen = false
		}
	}
	return n, err
}

// popKeyboardMode requires mu and drains only this adapter's outstanding pushes.
func (f *ctrlEnterTerminalOutput) popKeyboardMode() error {
	for f.pushes > 0 {
		if _, err := io.WriteString(f.File, modifiedKeysPop); err != nil {
			return err
		}
		f.pushes--
	}
	return nil
}

// cleanup is a fallback for Run paths that never shut down the renderer. Pop
// before leaving the alternate screen: Kitty keeps a separate stack per screen.
func (f *ctrlEnterTerminalOutput) cleanup() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.closed = true
	for i, mode := range ctrlEnterRendererModes {
		if f.modes[i] {
			if _, err := io.WriteString(f.File, mode.reset); err != nil {
				return err
			}
			f.modes[i] = false
		}
	}
	if err := f.popKeyboardMode(); err != nil {
		return err
	}
	if f.altScreen {
		if _, err := io.WriteString(f.File, ansi.ResetAltScreenSaveCursorMode); err != nil {
			return err
		}
		f.altScreen = false
	}
	return nil
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
