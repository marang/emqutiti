package emqutiti

import (
	"reflect"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/marang/emqutiti/constants"
)

// ctrlEnterMsg is distinct from KeyMsg: no legacy key or text aliases it.
type ctrlEnterMsg struct{ press bool }

// normalizeCtrlEnterMsg runs only inside the framing reader's input filter.
// Bubble Tea 1.3.10 has no exported extended-key event. Fail closed if its
// diagnostic type changes; never parse String(), KeyRunes, or pasted text.
func normalizeCtrlEnterMsg(msg tea.Msg) tea.Msg {
	if raw, ok := ctrlEnterCSIBytes(msg); ok {
		if event, ok := decodeCtrlEnter(raw); ok {
			return event
		}
	}
	return msg
}

func ctrlEnterCSIBytes(msg tea.Msg) ([]byte, bool) {
	v := reflect.ValueOf(msg)
	if !v.IsValid() {
		return nil, false
	}
	t := v.Type()
	if t.PkgPath() != "github.com/charmbracelet/bubbletea" || t.Name() != "unknownCSISequenceMsg" || t.Kind() != reflect.Slice || t.Elem().Kind() != reflect.Uint8 {
		return nil, false
	}
	return v.Bytes(), true
}

func decodeCtrlEnter(raw []byte) (ctrlEnterMsg, bool) {
	if len(raw) < 7 || len(raw) > 32 || raw[0] != '\x1b' || raw[1] != '[' {
		return ctrlEnterMsg{}, false
	}
	final := raw[len(raw)-1]
	if final != 'u' && final != '~' {
		return ctrlEnterMsg{}, false
	}
	// The ANSI parser intentionally tolerates embedded controls and truncates
	// excessive parameters. Reject those before interpreting a keyboard event.
	digits := 0
	for _, b := range raw[2 : len(raw)-1] {
		if (b < '0' || b > '9') && b != ';' && b != ':' {
			return ctrlEnterMsg{}, false
		}
		if b >= '0' && b <= '9' {
			digits++
			if digits > 6 {
				return ctrlEnterMsg{}, false
			}
		} else {
			digits = 0
		}
	}
	p := ansi.NewParser()
	var event ctrlEnterMsg
	var matched bool
	p.SetHandler(ansi.Handler{HandleCsi: func(cmd ansi.Cmd, params ansi.Params) {
		if int(cmd) != int(final) {
			return
		}
		plain := func(i, want int) bool {
			return i < len(params) && !params[i].HasMore() && params[i].Param(-1) == want
		}
		if final == '~' {
			matched = len(params) == 3 && plain(0, 27) && plain(1, 5) && plain(2, 13)
			event.press = matched
			return
		}
		if !plain(0, 13) || len(params) < 2 || params[1].Param(-1) != 5 {
			return
		}
		if len(params) == 2 && !params[1].HasMore() {
			matched, event.press = true, true
		} else if len(params) == 3 && params[1].HasMore() && !params[2].HasMore() {
			kind := params[2].Param(-1)
			matched, event.press = kind >= 1 && kind <= 3, kind == 1
		}
	}})
	for _, b := range raw {
		p.Advance(b)
	}
	return event, matched
}

// handleCtrlEnterMsg consumes this event in every mode, but publishes only
// on a press in the client message editor. It reuses the Ctrl+S command path.
func (m *model) handleCtrlEnterMsg(msg tea.Msg) (tea.Cmd, bool) {
	event, ok := msg.(ctrlEnterMsg)
	if !ok {
		return nil, false
	}
	if !event.press || m.ui.panelResize.id != "" || m.CurrentMode() != constants.ModeClient || m.FocusedID() != idMessage {
		return nil, true
	}
	m.clearHoverState()
	return m.handlePublishKey(), true
}
