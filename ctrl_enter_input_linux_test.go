//go:build linux

package emqutiti

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/term"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/marang/emqutiti/connections"
	"github.com/marang/emqutiti/constants"
	"github.com/marang/emqutiti/topics"
	"golang.org/x/sys/unix"
)

func TestModifiedKeyboardTeaRestoresProtocol(t *testing.T) {
	for _, action := range []string{"quit", "cancel", "release"} {
		t.Run(action, func(t *testing.T) {
			master, slave := ctrlEnterPTY(t)
			before, err := term.GetState(slave.Fd())
			if err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			var p *tea.Program
			var releaseErr error
			resumed := false
			harness := &ctrlEnterCaptureModel{apply: func(msg tea.Msg) tea.Cmd {
				if _, ok := msg.(tea.WindowSizeMsg); !ok {
					return nil
				}
				// Wait for resize initialization (and the resumed resize) before
				// shutting down; Tea's resize goroutine must finish using Fd.
				if action == "cancel" {
					return func() tea.Msg { cancel(); return nil }
				}
				if action == "release" && !resumed {
					resumed = true
					if releaseErr = p.ReleaseTerminal(); releaseErr == nil {
						releaseErr = p.RestoreTerminal()
					}
					if releaseErr == nil {
						return nil
					}
				}
				return tea.Quit
			}}
			p = tea.NewProgram(harness,
				tea.WithInput(strings.NewReader("")),
				tea.WithOutput(&ctrlEnterTerminalOutput{File: slave}),
				tea.WithAltScreen(), tea.WithContext(ctx), tea.WithoutSignalHandler())
			_, runErr := p.Run()
			if releaseErr != nil {
				t.Fatal(releaseErr)
			}
			if action == "cancel" {
				if runErr == nil {
					t.Fatal("cancellation unexpectedly succeeded")
				}
			} else if runErr != nil {
				t.Fatal(runErr)
			}
			var output bytes.Buffer
			buf := make([]byte, 4096)
			for {
				fds := []unix.PollFd{{Fd: int32(master.Fd()), Events: unix.POLLIN}}
				if n, err := unix.Poll(fds, 10); err != nil {
					t.Fatal(err)
				} else if n == 0 {
					break
				}
				n, err := master.Read(buf)
				if err != nil {
					t.Fatal(err)
				}
				output.Write(buf[:n])
			}
			want := 1
			if action == "release" {
				want = 2
			}
			got := output.String()
			if strings.Count(got, ansi.SetAltScreenSaveCursorMode+modifiedKeysPush) != want || strings.Count(got, modifiedKeysPop+ansi.ResetAltScreenSaveCursorMode) != want {
				t.Fatalf("unbalanced protocol across %s: %q", action, got)
			}
			after, err := term.GetState(slave.Fd())
			if err != nil || !reflect.DeepEqual(before, after) {
				t.Fatal("raw terminal state not restored")
			}
		})
	}
}

func TestModifiedEnterTerminalNegotiationPublishesWithoutNewlines(t *testing.T) {
	master, slave := ctrlEnterPTY(t)
	if err := unix.IoctlSetWinsize(int(slave.Fd()), unix.TIOCSWINSZ, &unix.Winsize{Row: 30, Col: 100}); err != nil {
		t.Fatal(err)
	}
	before, err := term.GetState(slave.Fd())
	if err != nil {
		t.Fatal(err)
	}
	stdin, stdout := os.Stdin, os.Stdout
	os.Stdin, os.Stdout = slave, slave
	defer func() { os.Stdin, os.Stdout = stdin, stdout }()
	logOutput, logFlags := log.Writer(), log.Flags()
	defer func() { log.SetOutput(logOutput); log.SetFlags(logFlags) }()

	m := reviewFixture(t, 100, 30)
	m.topics.Items = []topics.Item{{Name: "output"}}
	m.topics.SetSelected(0)
	client := &publishTestClient{}
	m.mqttClient = &MQTTClient{Client: ctrlEnterOptionsClient{publishTestClient: client}}
	m.message.Input().Reset()
	var publishes []bool
	var navigation []int
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// A terminal in legacy mode encodes both modified Enter keys as CR.
	// Only an application request enables distinct encodings. Observe the
	// actual runUI output before simulating the user's physical keypresses.
	terminalDone := make(chan error, 1)
	go func() {
		var output bytes.Buffer
		buf := make([]byte, 4096)
		for ctx.Err() == nil {
			fds := []unix.PollFd{{Fd: int32(master.Fd()), Events: unix.POLLIN}}
			if n, err := unix.Poll(fds, 50); err != nil {
				terminalDone <- err
				return
			} else if n == 0 {
				continue
			}
			n, err := master.Read(buf)
			if err != nil {
				terminalDone <- err
				return
			}
			output.Write(buf[:n])
			if !strings.Contains(ansi.Strip(output.String()), "[Enter] newline") {
				continue
			}
			normal, retained, quit := "\r", "\r", "\x04"
			move := "\x1b<\x1b>"
			if strings.Contains(output.String(), "\x1b[>5u") {
				normal, retained, quit = "\x1b[13;5u", "\x1b[13;6u", "\x1b[100;5u"
				move = "\x1b[44:60;4u\x1b[46:62;4u"
			}
			_, err = master.Write([]byte("first\rsecond" + move + normal + retained + quit))
			terminalDone <- err
			return
		}
		terminalDone <- ctx.Err()
	}()

	d := &appDeps{
		initialModel: func(*connections.Connections) (*model, error) { return m, nil },
		newProgram: func(_ tea.Model, opts ...tea.ProgramOption) program {
			m.SetMode(constants.ModeClient)
			m.SetFocus(idMessage)
			harness := &ctrlEnterCaptureModel{apply: func(msg tea.Msg) tea.Cmd {
				_, cmd := m.Update(msg)
				if key, ok := msg.(tea.KeyMsg); ok && (key.String() == "alt+<" || key.String() == "alt+>") {
					navigation = append(navigation, m.message.Input().Line())
				}
				if event, ok := msg.(ctrlEnterMsg); ok && event.press {
					publishes = append(publishes, event.retained)
					applyMQTTCommand(m, cmd)
				}
				return nil
			}}
			// Keep the real renderer: protocol requests must surround the
			// alternate-screen lifecycle, not merely exist in a unit decoder.
			opts = append(opts, tea.WithContext(ctx), tea.WithoutSignalHandler())
			return tea.NewProgram(ctrlEnterViewModel{ctrlEnterCaptureModel: harness, view: m.View}, opts...)
		},
	}
	if err := runUI(d); err != nil {
		t.Errorf("runUI failed: %v", err)
	}
	cancel()
	if err := <-terminalDone; err != nil {
		t.Errorf("terminal: %v", err)
	}
	if got := m.message.Input().Value(); got != "first\nsecond" || !reflect.DeepEqual(publishes, []bool{false, true}) || client.seen["output"] != "first\nsecond" {
		t.Fatalf("modified Enter inserted newlines instead of publishing: draft=%q publishes=%v payload=%q", got, publishes, client.seen["output"])
	}
	if !reflect.DeepEqual(navigation, []int{0, 1}) {
		t.Fatalf("negotiated Alt+Shift navigation lines=%v, want [0 1]", navigation)
	}
	after, err := term.GetState(slave.Fd())
	if err != nil || !reflect.DeepEqual(before, after) {
		t.Fatal("terminal raw mode was not restored")
	}
}

type ctrlEnterViewModel struct {
	*ctrlEnterCaptureModel
	view func() string
}

type ctrlEnterOptionsClient struct{ *publishTestClient }

func (ctrlEnterOptionsClient) OptionsReader() mqtt.ClientOptionsReader {
	return mqtt.NewClient(mqtt.NewClientOptions()).OptionsReader()
}

func (m ctrlEnterViewModel) View() string { return m.view() }

func (m ctrlEnterViewModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	_, cmd := m.ctrlEnterCaptureModel.Update(msg)
	return m, cmd
}

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
