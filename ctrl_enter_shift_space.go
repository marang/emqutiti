package emqutiti

import (
	"bytes"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/marang/emqutiti/constants"
)

// shiftedSpaceMsg preserves the modifier without aliasing ordinary space/text.
type shiftedSpaceMsg struct{}

func decodeShiftedSpace(raw []byte) (shiftedSpaceMsg, bool) {
	// ANSI framing is supplied by the shared reader. Accept only this exact
	// CSI-u press signature, not legacy space, extra modifiers or event types.
	return shiftedSpaceMsg{}, bytes.Equal(raw, []byte("\x1b[32;2u"))
}

// handleShiftedSpaceMsg toggles the current non-archived History row, using
// the existing Space path. Consume the typed event outside that context too.
func (m *model) handleShiftedSpaceMsg(msg tea.Msg) (tea.Cmd, bool) {
	if _, ok := msg.(shiftedSpaceMsg); !ok {
		return nil, false
	}
	if m.ui.panelResize.id != "" {
		return nil, true
	}
	m.clearHoverState()
	if m.CurrentMode() == constants.ModeClient && m.FocusedID() == idHistory {
		return m.handleSpaceKey(), true
	}
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeySpace, Runes: []rune{' '}})
	return cmd, true
}
