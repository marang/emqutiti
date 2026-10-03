//go:build linux

package emqutiti

import (
	"context"
	"fmt"
	"io"
	"os"
	"reflect"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/term"
	"golang.org/x/sys/unix"
)

func ctrlEnterPTY(t *testing.T) (*os.File, *os.File) {
	t.Helper()
	master, err := os.OpenFile("/dev/ptmx", os.O_RDWR|unix.O_NOCTTY, 0)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { master.Close() })
	if err := unix.IoctlSetPointerInt(int(master.Fd()), unix.TIOCSPTLCK, 0); err != nil {
		t.Fatal(err)
	}
	n, err := unix.IoctlGetInt(int(master.Fd()), unix.TIOCGPTN)
	if err != nil {
		t.Fatal(err)
	}
	slave, err := os.OpenFile(fmt.Sprintf("/dev/pts/%d", n), os.O_RDWR|unix.O_NOCTTY, 0)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { slave.Close() })
	return master, slave
}

func TestCtrlEnterLinuxTTYActualTeaInput(t *testing.T) {
	master, slave := ctrlEnterPTY(t)
	before, err := term.GetState(slave.Fd())
	if err != nil {
		t.Fatal(err)
	}
	f := &ctrlEnterTerminalInput{File: slave}
	f.reader = newCtrlEnterInputReader(f.readNext)
	defer f.reader.close()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	ready := make(chan struct{})
	m := &ctrlEnterCaptureModel{init: func() tea.Msg { close(ready); return nil }}
	p := tea.NewProgram(m, tea.WithInput(f), tea.WithOutput(io.Discard), tea.WithContext(ctx), tea.WithoutSignalHandler(), tea.WithFilter(f.reader.filter))
	writerDone := make(chan error, 1)
	go func() {
		// Tea invokes Init only after installing raw mode (which flushes input).
		select {
		case <-ready:
		case <-ctx.Done():
			writerDone <- ctx.Err()
			return
		}
		input := "\x1b[13;5u\r\x1b[200~paste\x1b[13;5u\x1b[13;6u\x1b[201~\x1b[27;5;13~\x1b[13;6u\x1b[27;6;13~\x04"
		for _, b := range []byte(input) {
			if _, err := master.Write([]byte{b}); err != nil {
				writerDone <- err
				return
			}
			time.Sleep(time.Millisecond)
		}
		writerDone <- nil
	}()
	if _, err := p.Run(); err != nil {
		t.Errorf("%v; messages: %#v", err, m.messages)
	}
	if err := <-writerDone; err != nil {
		t.Error(err)
	}
	if got := ctrlEnterEvents(m.messages); !reflect.DeepEqual(got, []ctrlEnterMsg{{press: true}, {press: true}, {press: true, retained: true}, {press: true, retained: true}}) {
		t.Fatalf("TTY events = %#v", got)
	}
	after, err := term.GetState(slave.Fd())
	if err != nil || !reflect.DeepEqual(before, after) {
		t.Fatal("Tea did not restore raw terminal state")
	}
}

func TestCtrlEnterLinuxTTYIdleEscape(t *testing.T) {
	master, slave := ctrlEnterPTY(t)
	state, err := term.MakeRaw(slave.Fd())
	if err != nil {
		t.Fatal(err)
	}
	defer term.Restore(slave.Fd(), state)
	f := &ctrlEnterTerminalInput{File: slave}
	f.reader = newCtrlEnterInputReader(f.readNext)
	if _, err := master.Write([]byte("\x1b")); err != nil {
		t.Fatal(err)
	}
	start := time.Now()
	buf := make([]byte, 256)
	n, err := f.Read(buf)
	if err != nil || string(buf[:n]) != "\x1b" || time.Since(start) > time.Second {
		t.Fatalf("idle ESC = %q, %v after %v", buf[:n], err, time.Since(start))
	}
}
