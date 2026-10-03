package emqutiti

import (
	"context"
	"errors"
	"io"
	"reflect"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/marang/emqutiti/constants"
	"github.com/marang/emqutiti/topics"
)

type ctrlEnterCaptureModel struct {
	messages []tea.Msg
	init     tea.Cmd
	apply    func(tea.Msg) tea.Cmd
}

func (m *ctrlEnterCaptureModel) Init() tea.Cmd { return m.init }
func (m *ctrlEnterCaptureModel) View() string  { return "" }
func (m *ctrlEnterCaptureModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok && key.Type == tea.KeyCtrlD {
		return m, tea.Quit
	}
	m.messages = append(m.messages, msg)
	if m.apply != nil {
		return m, m.apply(msg)
	}
	return m, nil
}

func captureCtrlEnterInput(t *testing.T, input io.Reader) []tea.Msg {
	t.Helper()
	return captureCtrlEnterInputWithUpdate(t, input, nil)
}

func captureCtrlEnterInputWithUpdate(t *testing.T, input io.Reader, apply func(tea.Msg) tea.Cmd) []tea.Msg {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	m := &ctrlEnterCaptureModel{apply: apply}
	r := framingCtrlEnterReader(input)
	defer r.close()
	p := tea.NewProgram(m, tea.WithInput(r), tea.WithFilter(r.filter), tea.WithOutput(io.Discard), tea.WithContext(ctx), tea.WithoutRenderer(), tea.WithoutSignalHandler())
	if _, err := p.Run(); err != nil {
		t.Fatal(err)
	}
	return m.messages
}

func ctrlEnterEvents(messages []tea.Msg) []ctrlEnterMsg {
	var events []ctrlEnterMsg
	for _, msg := range messages {
		if event, ok := msg.(ctrlEnterMsg); ok {
			events = append(events, event)
		}
	}
	return events
}

func TestCtrlEnterActualTeaParser(t *testing.T) {
	for _, raw := range []string{"\x1b[13;5u", "\x1b[13;5:1u", "\x1b[27;5;13~"} {
		t.Run(raw, func(t *testing.T) {
			msgs := captureCtrlEnterInput(t, strings.NewReader(raw+"\x04"))
			if got := ctrlEnterEvents(msgs); !reflect.DeepEqual(got, []ctrlEnterMsg{{press: true}}) {
				t.Fatalf("events = %#v, messages = %#v", got, msgs)
			}
		})
	}
}

func TestCtrlEnterTeaParserPreservesEditorNewlineAndPaste(t *testing.T) {
	m := reviewFixture(t, 80, 24)
	m.ui.modeStack = []constants.AppMode{constants.ModeClient}
	m.message.SetPayload("")
	m.SetFocus(idMessage)
	paste := "pasted ctrl+enter\x1b[13;5u\x1b[27;5;13~"
	r := framingCtrlEnterReader(strings.NewReader("first\rsecond\x1b[200~" + paste + "\x1b[201~\x04"))
	defer r.close()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	harness := &ctrlEnterCaptureModel{apply: func(msg tea.Msg) tea.Cmd {
		if cmd, handled := m.handleCtrlEnterMsg(msg); handled {
			return cmd
		}
		_, cmd := m.Update(msg)
		return cmd
	}}
	p := tea.NewProgram(harness, tea.WithInput(r), tea.WithFilter(r.filter), tea.WithOutput(io.Discard), tea.WithContext(ctx), tea.WithoutRenderer(), tea.WithoutSignalHandler())
	if _, err := p.Run(); err != nil {
		t.Fatal(err)
	}
	// Textarea removes pasted control characters, but they remain paste data
	// throughout routing and can never become publish events.
	if got := m.message.Input().Value(); !strings.HasPrefix(got, "first\nsecondpasted ctrl+enter") || m.pendingPublishes() != 0 || len(ctrlEnterEvents(harness.messages)) != 0 {
		t.Fatalf("editor draft = %q; pending = %d", got, m.pendingPublishes())
	}
}

func TestCtrlEnterTeaParserRejectsAliasesAndPaste(t *testing.T) {
	pasted := "text\r\n\x1b[13;5u\x1b[27;5;13~\x13"
	msgs := captureCtrlEnterInput(t, strings.NewReader("\r\nctrl+enter\x1b[1;5P\x1b[13;6u\x1b[200~"+pasted+"\x1b[201~\x04"))
	if got := ctrlEnterEvents(msgs); len(got) != 0 {
		t.Fatalf("unsafe aliases: %#v", got)
	}
	var enters int
	var paste string
	for _, msg := range msgs {
		if key, ok := msg.(tea.KeyMsg); ok {
			if key.Type == tea.KeyEnter || key.Type == tea.KeyCtrlJ {
				enters++
			}
			if key.Paste {
				paste = string(key.Runes)
			}
		}
	}
	if enters != 2 || paste != pasted {
		t.Fatalf("newlines = %d, paste = %q", enters, paste)
	}
}

func TestCtrlEnterRepeatReleaseIgnored(t *testing.T) {
	msgs := captureCtrlEnterInput(t, strings.NewReader("\x1b[13;5:1u\x1b[13;5:2u\x1b[13;5:3u\x04"))
	if got := ctrlEnterEvents(msgs); !reflect.DeepEqual(got, []ctrlEnterMsg{{true}, {false}, {false}}) {
		t.Fatalf("events = %#v", got)
	}
}

func TestCtrlEnterDecoderFailsClosed(t *testing.T) {
	for _, raw := range []string{
		"\r", "\n", "ctrl+enter", "\x1b[13u", "\x1b[13;1u", "\x1b[13;4u", "\x1b[13;6u", "\x1b[13;7u",
		"\x1b[13;5:0u", "\x1b[13;5:4u", "\x1b[13;5:1;13u", "\x1b[13:13;5u", "\x1b[13;5;1u",
		"\x1b[13;;5u", "\x1b[13;5:u", "\x1b[13;5 u", "\x1b[>13;5u", "\x1b[13;\x005u",
		"\x1b[13;5uX", "\x1b[13;5", "\x1b[27;5;10~", "\x1b[1;5P", "\x1b[99999999999999999999999999999999999999;5u",
	} {
		if event, ok := decodeCtrlEnter([]byte(raw)); ok {
			t.Errorf("accepted %q as %#v", raw, event)
		}
	}
	for _, msg := range []tea.Msg{nil, []byte("\x1b[13;5u"), "\x1b[13;5u", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("\x1b[13;5u"), Paste: true}} {
		if !reflect.DeepEqual(normalizeCtrlEnterMsg(msg), msg) {
			t.Errorf("normalized non-diagnostic type %T", msg)
		}
	}
}

func TestCtrlEnterDispatchScope(t *testing.T) {
	m := reviewFixture(t, 80, 24)
	m.topics.Items = []topics.Item{{Name: "target", Publish: true}}
	for _, mode := range []constants.AppMode{constants.ModeConnections, constants.ModePayloads, constants.ModeConfirmDelete, constants.ModeLogs} {
		m.ui.modeStack = []constants.AppMode{mode}
		m.SetFocus(idMessage)
		if cmd, handled := m.handleCtrlEnterMsg(ctrlEnterMsg{true}); cmd != nil || !handled {
			t.Fatalf("published in mode %v", mode)
		}
	}
	m.ui.modeStack = []constants.AppMode{constants.ModeClient}
	for _, focus := range []string{idTopics, idHistory} {
		m.SetFocus(focus)
		if cmd, handled := m.handleCtrlEnterMsg(ctrlEnterMsg{true}); cmd != nil || !handled {
			t.Fatalf("published with focus %s", focus)
		}
	}
	m.SetFocus(idMessage)
	m.ui.panelResize = panelResizeState{id: idMessage}
	if cmd, handled := m.handleCtrlEnterMsg(ctrlEnterMsg{true}); cmd != nil || !handled || m.pendingPublishes() != 0 {
		t.Fatal("active resize did not suppress publishing")
	}
	m.ui.panelResize = panelResizeState{}
	if cmd, handled := m.handleCtrlEnterMsg(ctrlEnterMsg{false}); cmd != nil || !handled {
		t.Fatal("repeat/release published")
	}
	if _, handled := m.handleCtrlEnterMsg(tea.KeyMsg{Type: tea.KeyEnter}); handled {
		t.Fatal("plain Enter consumed")
	}
}

func TestCtrlEnterReusesPublishSnapshotAndPending(t *testing.T) {
	m := reviewFixture(t, 80, 24)
	m.ui.modeStack = []constants.AppMode{constants.ModeClient}
	m.history.SetItems(nil)
	m.payloads.Clear()
	c := &publishTestClient{errTopic: "bad"}
	m.mqttClient = &MQTTClient{Client: c}
	m.topics.Items = []topics.Item{{Name: "good", Publish: true}, {Name: "bad", Publish: true}}
	m.message.SetPayload("original")
	m.SetFocus(idMessage)
	cmd, handled := m.handleCtrlEnterMsg(ctrlEnterMsg{true})
	if !handled || cmd == nil || m.pendingPublishes() != 2 || len(m.history.Items()) != 0 {
		t.Fatal("publish was not dispatched asynchronously")
	}
	if repeated, _ := m.handleCtrlEnterMsg(ctrlEnterMsg{true}); repeated != nil {
		t.Fatal("duplicated pending publish")
	}
	m.message.SetPayload("edited draft")
	m.topics.Items[0].Name = "new target"
	applyMQTTCommand(m, cmd)
	if c.seen["good"] != "original" || c.seen["bad"] != "original" || c.retained || m.message.Input().Value() != "edited draft" {
		t.Fatal("snapshot, non-retained flag or draft was not preserved")
	}
	items := m.history.Items()
	if len(items) != 2 || items[0].Kind != "pub" || items[1].Kind != "log" || m.pendingPublishes() != 0 {
		t.Fatalf("partial results = %#v", items)
	}
}

func TestCtrlEnterOfflinePreservesDraft(t *testing.T) {
	m := reviewFixture(t, 80, 24)
	m.ui.modeStack = []constants.AppMode{constants.ModeClient}
	m.history.SetItems(nil)
	m.payloads.Clear()
	m.topics.Items = []topics.Item{{Name: "target", Publish: true}}
	m.message.SetPayload("offline draft")
	m.SetFocus(idMessage)
	cmd, _ := m.handleCtrlEnterMsg(ctrlEnterMsg{true})
	applyMQTTCommand(m, cmd)
	if m.message.Input().Value() != "offline draft" || m.pendingPublishes() != 0 || len(m.payloads.Items()) != 0 || len(m.history.Items()) != 1 || m.history.Items()[0].Kind != "log" {
		t.Fatal("offline publish did not reuse error path")
	}
}

type ctrlEnterChunkReader struct {
	chunks [][]byte
	err    error
}

func (r *ctrlEnterChunkReader) Read(dst []byte) (int, error) {
	if len(r.chunks) == 0 {
		return 0, io.EOF
	}
	n := copy(dst, r.chunks[0])
	r.chunks[0] = r.chunks[0][n:]
	if len(r.chunks[0]) == 0 {
		r.chunks = r.chunks[1:]
	}
	if len(r.chunks) == 0 {
		return n, r.err
	}
	return n, nil
}

func framingCtrlEnterReader(r io.Reader) *ctrlEnterInputReader {
	return newCtrlEnterInputReader(func(dst []byte, _ bool) (int, error) { return r.Read(dst) })
}

func TestCtrlEnterFramingEveryReadSplitAndOrder(t *testing.T) {
	input := "before\r\x1b[13;5uafter\x1b[27;5;13~\x1b[200~pasted\x1b[13;5u\x1b[201~end\x04"
	for split := 1; split < len(input); split++ {
		r := &ctrlEnterChunkReader{chunks: [][]byte{[]byte(input[:split]), []byte(input[split:])}}
		msgs := captureCtrlEnterInput(t, r)
		var ordered []string
		for _, msg := range msgs {
			switch msg := msg.(type) {
			case ctrlEnterMsg:
				ordered = append(ordered, "publish")
			case tea.KeyMsg:
				if msg.Paste {
					ordered = append(ordered, "paste:"+string(msg.Runes))
				} else {
					ordered = append(ordered, msg.String())
				}
			}
		}
		want := "b|e|f|o|r|e|enter|publish|a|f|t|e|r|publish|paste:pasted\x1b[13;5u|e|n|d"
		if got := strings.Join(ordered, "|"); got != want {
			t.Fatalf("split %d: %q", split, got)
		}
	}
}

func TestCtrlEnterFramingByteSplitsPasteAndUTF8(t *testing.T) {
	input := "\xc3\xa4\x1b[200~" + strings.Repeat("\xc3\xa4\x1b[13;5u\r\n", 100) + "\x1b[201~\x1b[13;5u\x04"
	var chunks [][]byte
	for i := range []byte(input) {
		chunks = append(chunks, []byte{input[i]})
	}
	msgs := captureCtrlEnterInput(t, &ctrlEnterChunkReader{chunks: chunks})
	if got := ctrlEnterEvents(msgs); !reflect.DeepEqual(got, []ctrlEnterMsg{{true}}) {
		t.Fatalf("paste triggered publishing: %#v", got)
	}
	var paste string
	for _, msg := range msgs {
		if key, ok := msg.(tea.KeyMsg); ok && key.Paste {
			paste = string(key.Runes)
		}
	}
	if paste != strings.Repeat("\xc3\xa4\x1b[13;5u\r\n", 100) {
		t.Fatal("paste corrupted across byte splits")
	}
}

func TestCtrlEnterFramingErrorsAndSmallBuffers(t *testing.T) {
	sourceErr := errors.New("source failed")
	r := framingCtrlEnterReader(&ctrlEnterChunkReader{chunks: [][]byte{[]byte("\x1b[13;5u")}, err: sourceErr})
	if n, err := r.Read(make([]byte, 2)); n != 0 || !errors.Is(err, io.ErrShortBuffer) {
		t.Fatalf("small read = %d, %v", n, err)
	}
	buf := make([]byte, 256)
	if n, err := r.Read(buf); err != nil || !strings.HasPrefix(string(buf[:n]), "\x1b[13;5u") {
		t.Fatalf("data before error = %q, %v", buf[:n], err)
	}
	r.filter(nil, tea.KeyMsg{Type: tea.KeyEnter}) // non-CSI messages cannot acknowledge
	r.ack <- struct{}{}
	if n, err := r.Read(buf); n != 0 || !errors.Is(err, sourceErr) {
		t.Fatalf("deferred error = %d, %v", n, err)
	}
	if n, err := r.Read(nil); n != 0 || err != nil {
		t.Fatal("zero-length read failed")
	}
}

func TestCtrlEnterFramingUnknownAndMalformedNeverPublish(t *testing.T) {
	for _, input := range []string{
		"\x1b[13;5;99u", "\x1b[13;5:4u", "\x1b[13;18446744073709551621u",
		"\x1b]0;unknown\x1b[13;5u", "\x1bPunknown\x1b[13;5u",
		"\x1b]" + strings.Repeat("x", 126) + "\x1b[13;5u",
		"\x1b[" + strings.Repeat("0", 126) + "13;5u",
	} {
		msgs := captureCtrlEnterInput(t, strings.NewReader(input+"\x04"))
		if got := ctrlEnterEvents(msgs); len(got) != 0 {
			t.Fatalf("published from unknown/malformed %q: %#v", input, got)
		}
	}
}

func TestCtrlEnterFramingTimeoutAndClose(t *testing.T) {
	step := 0
	r := newCtrlEnterInputReader(func(dst []byte, continuation bool) (int, error) {
		step++
		switch step {
		case 1:
			return copy(dst, "\x1b"), nil
		case 2:
			if !continuation {
				t.Error("ESC continuation was not bounded")
			}
			return 0, errCtrlEnterInputTimeout
		case 3:
			return copy(dst, "\x1b[13;5u"), nil
		default:
			return 0, io.EOF
		}
	})
	buf := make([]byte, 256)
	if n, err := r.Read(buf); err != nil || string(buf[:n]) != "\x1b" {
		t.Fatalf("idle ESC = %q, %v", buf[:n], err)
	}
	if n, err := r.Read(buf); err != nil || !strings.HasPrefix(string(buf[:n]), "\x1b[13;5u") {
		t.Fatalf("next complete frame = %q, %v", buf[:n], err)
	}
	finished := make(chan error, 1)
	go func() { _, err := r.Read(buf); finished <- err }()
	r.close()
	r.close()
	select {
	case err := <-finished:
		if !errors.Is(err, io.EOF) {
			t.Fatalf("closed reader = %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("close failed to release frame handshake")
	}
}

func TestCtrlEnterFramingNoProgressAndInvalidCount(t *testing.T) {
	for _, count := range []int{0, -1, 257} {
		r := newCtrlEnterInputReader(func([]byte, bool) (int, error) { return count, nil })
		if n, err := r.Read(make([]byte, 256)); n != 0 || err == nil {
			t.Fatalf("invalid count %d: %d, %v", count, n, err)
		}
	}
}
