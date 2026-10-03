package emqutiti

import (
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/marang/emqutiti/constants"
)

const (
	focusNext = 1
	focusPrev = -1
)

// calcHistorySize returns the width and height for the history list.
// It defaults the height when the current value is zero.
func calcHistorySize(width, height, currentHeight int) (int, int) {
	if currentHeight == 0 {
		currentHeight = (height-1)/3 + 10
	}
	return calcMessageWidth(width), currentHeight
}

// calcMessageWidth returns the width for message inputs and lists.
func calcMessageWidth(width int) int {
	return width - 4
}

// calcConnectionsSize returns the width and height for the connections list.
func calcConnectionsSize(width, height int) (int, int) {
	return calcMessageWidth(width), height - 6
}

// calcTopicsInputWidth returns the width for the topics input.
// It subtracts space for the prompt and cursor so the surrounding box
// stays on a single line.
func calcTopicsInputWidth(width int) int {
	w := calcMessageWidth(width) - 3
	if w < 0 {
		return 0
	}
	return w
}

// calcTraceHeight returns the height for trace views, defaulting if zero.
func calcTraceHeight(height, currentHeight int) int {
	if currentHeight == 0 {
		return height - 6
	}
	return currentHeight
}

// calcTraceListSize returns size for trace lists.
func calcTraceListSize(width, height int) (int, int) {
	return calcMessageWidth(width), height - 4
}

// calcViewportHeight returns the viewport height, reserving two lines for headers.
func calcViewportHeight(height int) int {
	return height - 2
}

// handleWindowSize adjusts the layout when the terminal is resized.
func (m *model) handleWindowSize(msg tea.WindowSizeMsg) tea.Cmd {
	m.ui.width = msg.Width
	m.ui.height = msg.Height
	m.clampPanelHeights()
	cw, ch := calcConnectionsSize(msg.Width, msg.Height)
	m.connections.Manager.ConnectionsList.SetSize(cw, ch)
	// textinput.View() renders the prompt and cursor in addition
	// to the configured width. Reduce the width slightly so the
	// surrounding box stays within the terminal boundaries.
	m.topics.Input.Width = calcTopicsInputWidth(msg.Width)
	m.message.Input().SetWidth(calcMessageWidth(msg.Width))
	m.message.Input().SetHeight(m.layout.message.height)
	hw, hh := calcHistorySize(msg.Width, msg.Height, m.layout.history.height)
	m.layout.history.height = hh
	m.history.List().SetSize(hw, hh)
	m.layout.trace.height = calcTraceHeight(msg.Height, m.layout.trace.height)
	m.traces.ViewList().SetSize(calcMessageWidth(msg.Width), m.layout.trace.height)
	tw, th := calcTraceListSize(msg.Width, msg.Height)
	m.traces.List().SetSize(tw, th)
	m.topics.SetSize(msg.Width, msg.Height)
	m.payloads.SetSize(msg.Width, msg.Height)
	m.traces.SetSize(msg.Width, msg.Height)
	m.help.SetSize(msg.Width, msg.Height)
	m.logs.SetSize(msg.Width, msg.Height)
	m.history.SetDetailSize(msg.Width, msg.Height)
	m.ui.viewport.Width = msg.Width
	// Reserve two lines for the info header at the top of the view.
	m.ui.viewport.Height = max(1, calcViewportHeight(msg.Height)-2)
	return nil
}

// cycleFocus moves focus forward or backward through the focus order.
// It updates the focus index and ensures the topics list selection is valid.
// A non-zero return indicates the focus changed.
func (m *model) cycleFocus(direction int) (tea.Cmd, bool) {
	if len(m.ui.focusOrder) == 0 {
		return nil, false
	}
	switch direction {
	case focusNext:
		m.focus.Next()
	case focusPrev:
		m.focus.Prev()
	default:
		return nil, false
	}
	m.ui.focusIndex = m.focus.Index()
	id := m.ui.focusOrder[m.ui.focusIndex]
	cmd := m.SetFocus(id)
	if id == idTopics {
		if len(m.topics.Items) > 0 {
			sel := m.topics.Selected()
			if sel < 0 || sel >= len(m.topics.Items) {
				m.topics.SetSelected(0)
			}
			m.topics.EnsureVisible(m.ui.width - 4)
		} else {
			m.topics.SetSelected(-1)
		}
	}
	return cmd, true
}

// handleKeyNav processes global navigation key presses.
func (m *model) handleKeyNav(msg tea.KeyMsg) (tea.Cmd, bool) {
	key := msg.String()
	if key == constants.KeyTab || key == constants.KeyShiftTab {
		var l *list.Model
		switch m.CurrentMode() {
		case constants.ModeTopics:
			l = m.topics.List()
		case constants.ModePayloads:
			l = m.payloads.List()
		case constants.ModeTracer:
			l = m.traces.List()
		case constants.ModeConnections:
			l = &m.connections.Manager.ConnectionsList
		}
		if l != nil && l.FilterState() == list.Filtering {
			return m.components[m.CurrentMode()].Update(msg), true
		}
	}
	switch key {
	case constants.KeyCtrlUp, constants.KeyCtrlK:
		if m.CurrentMode() != constants.ModeClient {
			return m.scrollCurrentMode(msg, tea.KeyUp)
		}
		m.ui.viewport.ScrollUp(1)
		return nil, true
	case constants.KeyCtrlDown, constants.KeyCtrlJ:
		if m.CurrentMode() != constants.ModeClient {
			return m.scrollCurrentMode(msg, tea.KeyDown)
		}
		m.ui.viewport.ScrollDown(1)
		return nil, true
	case constants.KeyTab:
		if m.CurrentMode() == constants.ModeTraceFilter {
			return m.traces.UpdateFilter(msg), true
		}
		if m.CurrentMode() == constants.ModeEditTrace {
			return m.traces.UpdateForm(msg), true
		}
		if m.CurrentMode() == constants.ModeHistoryFilter {
			return m.history.UpdateFilter(msg), true
		}
		if m.CurrentMode() == constants.ModeEditConnection {
			return m.updateConnectionForm(msg), true
		}
		if cmd, ok := m.cycleFocus(focusNext); ok {
			return cmd, true
		}
	case constants.KeyShiftTab:
		if m.CurrentMode() == constants.ModeTraceFilter {
			return m.traces.UpdateFilter(msg), true
		}
		if m.CurrentMode() == constants.ModeEditTrace {
			return m.traces.UpdateForm(msg), true
		}
		if m.CurrentMode() == constants.ModeHistoryFilter {
			return m.history.UpdateFilter(msg), true
		}
		if m.CurrentMode() == constants.ModeEditConnection {
			return m.updateConnectionForm(msg), true
		}
		if cmd, ok := m.cycleFocus(focusPrev); ok {
			return cmd, true
		}
	}

	if m.CurrentMode() != constants.ModeHistoryFilter &&
		(key == constants.KeyEnter || key == constants.KeySpaceBar || key == constants.KeySpace) &&
		m.help.Focused() {
		return m.SetMode(constants.ModeHelp), true
	}
	return nil, false
}

func (m *model) scrollCurrentMode(msg tea.KeyMsg, arrow tea.KeyType) (tea.Cmd, bool) {
	switch m.CurrentMode() {
	case constants.ModeEditConnection:
		return m.updateConnectionForm(msg), true
	case constants.ModeHistoryFilter:
		return m.history.UpdateFilter(msg), true
	case constants.ModeTraceFilter:
		return m.traces.UpdateFilter(msg), true
	case constants.ModeConfirmDelete:
		return m.confirm.Update(msg), true
	case constants.ModeHistoryDetail:
		return m.history.UpdateDetail(tea.KeyMsg{Type: arrow}), true
	case constants.ModeEditTrace:
		return m.traces.UpdateForm(tea.KeyMsg{Type: arrow}), true
	default:
		if c, ok := m.components[m.CurrentMode()]; ok {
			return c.Update(tea.KeyMsg{Type: arrow}), true
		}
	}
	return nil, false
}
