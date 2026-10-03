//go:build linux

package emqutiti

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"reflect"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/term"
	"golang.org/x/sys/unix"
)

// Exhaust descriptors after Tea enters raw mode and the alternate screen, but
// before it creates cancelreader's pipe. Never lower limits in the parent test.
type ctrlEnterStartupFailureModel struct {
	files []*os.File
	err   error
}

func (m *ctrlEnterStartupFailureModel) Init() tea.Cmd {
	for {
		file, err := os.Open("/dev/null")
		if err != nil {
			m.err = err
			return nil
		}
		m.files = append(m.files, file)
	}
}

func (m *ctrlEnterStartupFailureModel) View() string { return "startup probe" }
func (m *ctrlEnterStartupFailureModel) Update(tea.Msg) (tea.Model, tea.Cmd) {
	return m, nil
}

func TestModifiedKeyboardTeaStartupFailureCleanup(t *testing.T) {
	const childEnv = "EMQUTITI_KEYBOARD_STARTUP_FAILURE_CHILD"
	if os.Getenv(childEnv) != "1" {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestModifiedKeyboardTeaStartupFailureCleanup$", "-test.v")
		cmd.Env = append(os.Environ(), childEnv+"=1")
		cmd.WaitDelay = time.Second
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("startup failure subprocess: %v (timeout: %v)\n%s", err, ctx.Err(), output)
		}
		return
	}

	master, slave := ctrlEnterPTY(t)
	before, err := term.GetState(slave.Fd())
	if err != nil {
		t.Fatal(err)
	}
	stdin, stdout := os.Stdin, os.Stdout
	os.Stdin, os.Stdout = slave, slave
	defer func() { os.Stdin, os.Stdout = stdin, stdout }()
	options, cleanup := ctrlEnterInputOptions()
	defer cleanup()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	model := &ctrlEnterStartupFailureModel{}
	options = append(options, tea.WithAltScreen(), tea.WithMouseAllMotion(), tea.WithReportFocus(), tea.WithContext(ctx), tea.WithoutSignalHandler())
	p := tea.NewProgram(model, options...)
	defer p.Kill() // The subprocess isolates Tea's leaked renderer goroutine.

	var original unix.Rlimit
	if err := unix.Getrlimit(unix.RLIMIT_NOFILE, &original); err != nil {
		t.Fatal(err)
	}
	limited := original
	if limited.Cur > 128 {
		limited.Cur = 128
	}
	if err := unix.Setrlimit(unix.RLIMIT_NOFILE, &limited); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := unix.Setrlimit(unix.RLIMIT_NOFILE, &original); err != nil {
			t.Error(err)
		}
		for _, file := range model.files {
			file.Close()
		}
	}()

	_, runErr := p.Run()
	cleanup()
	cleanup()
	if !errors.Is(model.err, unix.EMFILE) || runErr == nil || !strings.Contains(runErr.Error(), "error creating cancelreader") || !strings.Contains(runErr.Error(), "too many open files") {
		t.Fatalf("did not exercise EMFILE cancelreader startup failure: Init=%v Run=%v", model.err, runErr)
	}

	var output bytes.Buffer
	buf := make([]byte, 4096)
	for {
		fds := []unix.PollFd{{Fd: int32(master.Fd()), Events: unix.POLLIN}}
		n, err := unix.Poll(fds, 10)
		if errors.Is(err, unix.EINTR) {
			continue
		}
		if err != nil {
			t.Fatal(err)
		}
		if n == 0 {
			break
		}
		n, err = master.Read(buf)
		if err != nil {
			t.Fatal(err)
		}
		output.Write(buf[:n])
	}
	got := output.String()
	for _, mode := range []struct{ enable, reset string }{
		{ansi.HideCursor, ansi.ShowCursor},
		{ansi.SetBracketedPasteMode, ansi.ResetBracketedPasteMode},
		{ansi.SetAnyEventMouseMode, ansi.ResetAnyEventMouseMode},
		{ansi.SetSgrExtMouseMode, ansi.ResetSgrExtMouseMode},
		{ansi.SetFocusEventMode, ansi.ResetFocusEventMode},
	} {
		if !strings.Contains(got, mode.enable) || strings.Count(got, mode.reset) != 1 || strings.LastIndex(got, mode.enable) > strings.Index(got, mode.reset) || strings.Index(got, mode.reset) > strings.Index(got, modifiedKeysPop) {
			t.Fatalf("startup mode %q must be reset once before keyboard pop: %q", mode.enable, got)
		}
	}
	if strings.Count(got, ansi.SetAltScreenSaveCursorMode+modifiedKeysPush) != 1 || strings.Count(got, modifiedKeysPop+ansi.ResetAltScreenSaveCursorMode) != 1 {
		t.Fatalf("startup cleanup must pop once before exiting its alternate screen: push=%d pop=%d output=%q", strings.Count(got, modifiedKeysPush), strings.Count(got, modifiedKeysPop), got)
	}
	after, err := term.GetState(slave.Fd())
	if err != nil || !reflect.DeepEqual(before, after) {
		t.Fatalf("startup cleanup did not restore raw terminal state: %v", err)
	}
}
