//go:build linux

package emqutiti

import (
	"context"
	"io"
	"reflect"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/term"
)

func captureCtrlEnterPTYInput(t *testing.T, input string, split int) []tea.Msg {
	t.Helper()
	return captureCtrlEnterPTYInputWithDelay(t, input, split, time.Millisecond, nil)
}

func captureCtrlEnterPTYInputWithDelay(t *testing.T, input string, split int, delay time.Duration, apply func(tea.Msg) tea.Cmd) []tea.Msg {
	t.Helper()
	master, slave := ctrlEnterPTY(t)
	before, err := term.GetState(slave.Fd())
	if err != nil {
		t.Fatal(err)
	}
	f := &ctrlEnterTerminalInput{File: slave}
	f.reader = newCtrlEnterInputReader(f.readNext)
	defer f.reader.close()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	ready := make(chan struct{})
	m := &ctrlEnterCaptureModel{init: func() tea.Msg { close(ready); return nil }, apply: apply}
	p := tea.NewProgram(m, tea.WithInput(f), tea.WithOutput(io.Discard), tea.WithContext(ctx), tea.WithoutSignalHandler(), tea.WithFilter(f.reader.filter))
	writerDone := make(chan error, 1)
	go func() {
		select {
		case <-ready:
		case <-ctx.Done():
			writerDone <- ctx.Err()
			return
		}
		chunks := []string{input[:split], input[split:]}
		for _, chunk := range chunks {
			if _, err := master.Write([]byte(chunk)); err != nil {
				writerDone <- err
				return
			}
			time.Sleep(delay)
		}
		writerDone <- nil
	}()
	_, runErr := p.Run()
	if err := <-writerDone; err != nil {
		t.Error(err)
	}
	if runErr != nil {
		t.Fatalf("PTY input timed out/failed: %v; messages = %#v", runErr, m.messages)
	}
	after, err := term.GetState(slave.Fd())
	if err != nil || !reflect.DeepEqual(before, after) {
		t.Fatal("Tea did not restore raw terminal state")
	}
	return m.messages
}

func TestCtrlEnterFramingSS3PTYEverySplit(t *testing.T) {
	for _, test := range ctrlEnterSS3Cases() {
		input := test.raw + "\x04"
		for split := 1; split < len(input); split++ {
			messages := captureCtrlEnterPTYInput(t, input, split)
			if got := ctrlEnterInputKeys(messages); !reflect.DeepEqual(got, []tea.KeyMsg{test.key}) {
				t.Fatalf("%q split %d: keys = %#v, want %#v", test.raw, split, got, test.key)
			}
		}
	}
}

func TestCtrlEnterFramingNestedPastePTYEverySplit(t *testing.T) {
	for _, prefix := range []string{"\x1b]x", "\x1bPx", "\x1bO", "\x1b\x1bO"} {
		t.Run(prefix, func(t *testing.T) {
			input := prefix + "\x1b[200~abc\x1b[201~\x04"
			want := ctrlEnterInputKeys(captureCtrlEnterBaselineInput(t, input))
			for split := 1; split < len(input); split++ {
				messages := captureCtrlEnterPTYInput(t, input, split)
				if got := ctrlEnterInputKeys(messages); !reflect.DeepEqual(got, want) {
					t.Fatalf("split %d: keys = %#v, baseline = %#v", split, got, want)
				}
			}
		})
	}
}

func TestCtrlEnterFramingBareControlProtectedPastePTYEverySplit(t *testing.T) {
	payload := "abc\x1b[13;5u\x1b[32;2u"
	for _, prefix := range []string{"\x1bO", "\x1b\x1bO", "\x1bP", "\x1b]"} {
		t.Run(prefix, func(t *testing.T) {
			input := prefix + "\x1b[200~" + payload + "\x1b[201~\x04"
			for split := 1; split < len(input); split++ {
				assertCtrlEnterProtectedPaste(t, captureCtrlEnterPTYInput(t, input, split), payload)
			}
		})
	}
}

func TestCtrlEnterFramingTeaKeyCorpusPTYEverySplit(t *testing.T) {
	for _, raw := range ctrlEnterTeaKeyCorpus(t) {
		input := raw + "\x04"
		want := captureCtrlEnterBaselineInput(t, input)
		for split := 1; split < len(input); split++ {
			got := captureCtrlEnterPTYInput(t, input, split)
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("%q split %d: messages = %#v, baseline = %#v", raw, split, got, want)
			}
		}
	}
}

func TestCtrlEnterFramingIncompleteX10PTYDoesNotEatQuit(t *testing.T) {
	for _, raw := range []string{"\x1b[M", "\x1b[Ma"} {
		input := raw + "\x04"
		want := ctrlEnterInputKeys(captureCtrlEnterBaselineInput(t, input))
		for split := 1; split < len(input); split++ {
			got := ctrlEnterInputKeys(captureCtrlEnterPTYInput(t, input, split))
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("%q split %d: keys = %#v, baseline = %#v", raw, split, got, want)
			}
		}
	}
}

func TestCtrlEnterFramingSuffixUTF8PTYEverySplit(t *testing.T) {
	for _, prefix := range []string{"\x1bO", "\x1b\x1bO", "\x1b[O", "\x1b\x1b[O", "\x1b[[", "\x1b\x1b[["} {
		for _, runeBytes := range []string{"\xc3\xa4", "\xf0\x9f\x98\x80"} {
			input := prefix + runeBytes + "\x04"
			want := ctrlEnterAtomicInputKeys(captureCtrlEnterBaselineInput(t, input))
			for split := 1; split < len(input); split++ {
				got := ctrlEnterAtomicInputKeys(captureCtrlEnterPTYInput(t, input, split))
				if !reflect.DeepEqual(got, want) {
					t.Fatalf("%q split %d: keys = %#v, baseline = %#v", input, split, got, want)
				}
			}
		}
	}
}

func TestCtrlEnterFramingSuffixUTF8PTYTimeout(t *testing.T) {
	for _, prefix := range []string{"\x1bO", "\x1b\x1bO", "\x1b[O", "\x1b[["} {
		for _, runeBytes := range []string{"\xc3\xa4", "\xf0\x9f\x98\x80"} {
			input := prefix + runeBytes + "\x04"
			want := ctrlEnterAtomicInputKeys(captureCtrlEnterBaselineInput(t, input))
			for split := 0; split < len(runeBytes); split++ {
				got := ctrlEnterAtomicInputKeys(captureCtrlEnterPTYInputWithDelay(t, input, len(prefix)+split, 75*time.Millisecond, nil))
				if !reflect.DeepEqual(got, want) {
					t.Fatalf("%q rune split %d: keys = %#v, baseline = %#v", input, split, got, want)
				}
			}
		}
	}
}

func TestCtrlEnterFramingSuffixUTF8PTYRootEditor(t *testing.T) {
	input := "\x1bO\xc3\xa4\x04"
	for _, delay := range []time.Duration{time.Millisecond, 75 * time.Millisecond} {
		for split := 1; split < len(input); split++ {
			m := ctrlEnterUnicodeEditorFixture(t)
			messages := captureCtrlEnterPTYInputWithDelay(t, input, split, delay, func(msg tea.Msg) tea.Cmd {
				_, cmd := m.Update(msg)
				return cmd
			})
			if got := m.message.Input().Value(); got != "seedO\xc3\xa4" || m.pendingPublishes() != 0 || len(ctrlEnterEvents(messages)) != 0 {
				t.Fatalf("delay %v split %d: editor draft = %q; pending = %d", delay, split, got, m.pendingPublishes())
			}
		}
	}
}

func TestCtrlEnterFramingNestedPastePTYAroundFrameLimit(t *testing.T) {
	payload := "abc\x1b[13;5u\x1b[32;2u"
	for _, control := range []string{"\x1b]", "\x1bP"} {
		for length := 120; length <= 130; length++ {
			input := control + strings.Repeat("x", length) + "\x1b[200~" + payload + "\x1b[201~\x1b[13;5u\x04"
			messages := captureCtrlEnterPTYInput(t, input, length+4)
			assertCtrlEnterNestedPaste(t, messages, payload)
		}
	}
}
