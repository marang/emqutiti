//go:build linux || darwin

package emqutiti

import (
	"bytes"
	"errors"
	"io"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"
)

func TestModifiedKeyboardOutputRestoresEachAlternateScreen(t *testing.T) {
	file, err := os.CreateTemp(t.TempDir(), "terminal-output")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	output := &ctrlEnterTerminalOutput{File: file}
	if output.Fd() != file.Fd() || output.Name() != file.Name() {
		t.Fatal("output lost terminal file methods")
	}
	for range 2 {
		for _, seq := range []string{ansi.SetAltScreenSaveCursorMode, "view", ansi.ResetAltScreenSaveCursorMode} {
			if n, err := io.WriteString(output, seq); n != len(seq) || err != nil {
				t.Fatalf("WriteString(%q)=%d, %v", seq, n, err)
			}
		}
	}
	for range 2 {
		if err := output.cleanup(); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		t.Fatal(err)
	}
	data, err := io.ReadAll(file)
	if err != nil {
		t.Fatal(err)
	}
	cycle := ansi.SetAltScreenSaveCursorMode + modifiedKeysPush + "view" + modifiedKeysPop + ansi.ResetAltScreenSaveCursorMode
	if string(data) != cycle+cycle {
		t.Fatalf("protocol push/pop escaped its alternate-screen stack: %q", data)
	}
}

func TestModifiedKeyboardOutputCleanup(t *testing.T) {
	enter, exit := ansi.SetAltScreenSaveCursorMode, ansi.ResetAltScreenSaveCursorMode
	for _, test := range []struct {
		name  string
		input []string
		want  string
	}{
		{name: "no push"},
		{name: "exit without push", input: []string{exit}, want: exit},
		{name: "startup failure", input: []string{enter, "view"}, want: enter + modifiedKeysPush + "view" + modifiedKeysPop + exit},
		{name: "multiple outstanding pushes", input: []string{enter, enter}, want: enter + modifiedKeysPush + enter + modifiedKeysPush + modifiedKeysPop + modifiedKeysPop + exit},
		{name: "normal shutdown", input: []string{enter, exit}, want: enter + modifiedKeysPush + modifiedKeysPop + exit},
		{name: "released then restored", input: []string{enter, exit, enter}, want: enter + modifiedKeysPush + modifiedKeysPop + exit + enter + modifiedKeysPush + modifiedKeysPop + exit},
	} {
		t.Run(test.name, func(t *testing.T) {
			file, err := os.CreateTemp(t.TempDir(), "terminal-output")
			if err != nil {
				t.Fatal(err)
			}
			defer file.Close()
			output := &ctrlEnterTerminalOutput{File: file}
			for _, seq := range test.input {
				if n, err := io.WriteString(output, seq); n != len(seq) || err != nil {
					t.Fatalf("WriteString(%q)=%d, %v", seq, n, err)
				}
			}
			for range 2 {
				if err := output.cleanup(); err != nil {
					t.Fatal(err)
				}
			}
			// An abandoned renderer must not re-enter or repaint the terminal.
			for _, seq := range []string{enter, "late frame", exit} {
				if n, err := io.WriteString(output, seq); n != len(seq) || err != nil {
					t.Fatalf("late WriteString(%q)=%d, %v", seq, n, err)
				}
			}
			data, err := os.ReadFile(file.Name())
			if err != nil {
				t.Fatal(err)
			}
			if string(data) != test.want {
				t.Fatalf("cleanup output=%q, want %q", data, test.want)
			}
		})
	}
}

func TestModifiedKeyboardOutputConcurrentCleanup(t *testing.T) {
	file, err := os.CreateTemp(t.TempDir(), "terminal-output")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	output := &ctrlEnterTerminalOutput{File: file}
	var workers sync.WaitGroup
	for range 4 {
		workers.Go(func() {
			for range 20 {
				for _, seq := range []string{
					ansi.SetAltScreenSaveCursorMode, ansi.HideCursor,
					ansi.SetBracketedPasteMode, ansi.SetAnyEventMouseMode,
					ansi.SetSgrExtMouseMode, ansi.SetFocusEventMode, "view",
					ansi.ResetBracketedPasteMode, ansi.ShowCursor,
					ansi.ResetAnyEventMouseMode, ansi.ResetSgrExtMouseMode,
					ansi.ResetFocusEventMode, ansi.ResetAltScreenSaveCursorMode,
				} {
					if _, err := io.WriteString(output, seq); err != nil {
						t.Error(err)
					}
				}
			}
		})
	}
	for range 2 {
		workers.Go(func() {
			if err := output.cleanup(); err != nil {
				t.Error(err)
			}
		})
	}
	workers.Wait()
	data, err := os.ReadFile(file.Name())
	if err != nil {
		t.Fatal(err)
	}
	pushes, pops := bytes.Count(data, []byte(modifiedKeysPush)), bytes.Count(data, []byte(modifiedKeysPop))
	if pushes != pops {
		t.Fatalf("concurrent cleanup left pushes=%d pops=%d: %q", pushes, pops, data)
	}
}

func TestModifiedKeyboardOutputCleanupRendererModes(t *testing.T) {
	for _, mode := range []struct {
		name, enable, reset string
	}{
		{"cursor", ansi.HideCursor, ansi.ShowCursor},
		{"paste", ansi.SetBracketedPasteMode, ansi.ResetBracketedPasteMode},
		{"mouse cell", ansi.SetButtonEventMouseMode, ansi.ResetButtonEventMouseMode},
		{"mouse all", ansi.SetAnyEventMouseMode, ansi.ResetAnyEventMouseMode},
		{"mouse sgr", ansi.SetSgrExtMouseMode, ansi.ResetSgrExtMouseMode},
		{"focus", ansi.SetFocusEventMode, ansi.ResetFocusEventMode},
	} {
		for _, normalReset := range []bool{false, true} {
			name := mode.name + "/fallback"
			if normalReset {
				name = mode.name + "/normal reset"
			}
			t.Run(name, func(t *testing.T) {
				file, err := os.CreateTemp(t.TempDir(), "terminal-output")
				if err != nil {
					t.Fatal(err)
				}
				defer file.Close()
				output := &ctrlEnterTerminalOutput{File: file}
				// Repeated enables (Tea hides the cursor twice on startup) need
				// only one reset, even when no alternate screen was entered.
				for range 2 {
					if _, err := io.WriteString(output, mode.enable); err != nil {
						t.Fatal(err)
					}
				}
				if normalReset {
					if _, err := io.WriteString(output, mode.reset); err != nil {
						t.Fatal(err)
					}
				}
				for range 2 {
					if err := output.cleanup(); err != nil {
						t.Fatal(err)
					}
				}
				if _, err := io.WriteString(output, mode.enable); err != nil {
					t.Fatal(err)
				}
				data, err := os.ReadFile(file.Name())
				if err != nil {
					t.Fatal(err)
				}
				if want := mode.enable + mode.enable + mode.reset; string(data) != want {
					t.Fatalf("mode cleanup output=%q, want %q", data, want)
				}
			})
		}
	}
}

func ctrlEnterPipeInput(t *testing.T) (*ctrlEnterTerminalInput, *os.File) {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { r.Close() })
	t.Cleanup(func() { w.Close() })
	return &ctrlEnterTerminalInput{File: r}, w
}

func TestCtrlEnterUnixInputDoesNotPrefetch(t *testing.T) {
	f, w := ctrlEnterPipeInput(t)
	if err := f.File.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write([]byte("ab")); err != nil {
		t.Fatal(err)
	}
	buf := make([]byte, 256)
	if n, err := f.readNext(buf, false); n != 1 || err != nil || buf[0] != 'a' {
		t.Fatalf("first read = %d, %v, %q", n, err, buf[:n])
	}
	// The second byte must remain readable on the underlying descriptor so
	// Tea's cancelreader can poll it without waiting for another physical key.
	if n, err := f.File.Read(buf); n != 1 || err != nil || buf[0] != 'b' {
		t.Fatalf("underlying read = %d, %v, %q", n, err, buf[:n])
	}
}

func TestCtrlEnterUnixInputContinuationDeadline(t *testing.T) {
	f, w := ctrlEnterPipeInput(t)
	buf := make([]byte, 256)
	start := time.Now()
	if n, err := f.readNext(buf, true); n != 0 || !errors.Is(err, errCtrlEnterInputTimeout) {
		t.Fatalf("idle continuation = %d, %v", n, err)
	}
	if f.deadline.IsZero() || time.Since(start) > time.Second {
		t.Fatalf("continuation deadline = %v after %v", f.deadline, time.Since(start))
	}
	deadline := f.deadline
	if _, err := w.Write([]byte("ab")); err != nil {
		t.Fatal(err)
	}
	if n, err := f.readNext(buf, true); n != 0 || !errors.Is(err, errCtrlEnterInputTimeout) || !f.deadline.Equal(deadline) {
		t.Fatalf("expired continuation = %d, %v, deadline %v", n, err, f.deadline)
	}
	if n, err := f.readNext(buf, false); n != 1 || err != nil || buf[0] != 'a' || !f.deadline.IsZero() {
		t.Fatalf("fresh read = %d, %v, deadline %v", n, err, f.deadline)
	}
	if n, err := f.readNext(buf, true); n != 1 || err != nil || buf[0] != 'b' || f.deadline.IsZero() {
		t.Fatalf("ready continuation = %d, %v, deadline %v", n, err, f.deadline)
	}
}
