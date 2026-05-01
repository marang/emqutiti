package emqutiti

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/marang/emqutiti/constants"
	"github.com/marang/emqutiti/ui"
)

// isHistoryFocused reports if the history list has focus.
func (m *model) isHistoryFocused() bool {
	return m.FocusedID() == idHistory
}

// isTopicsFocused reports if the topics view has focus.
func (m *model) isTopicsFocused() bool {
	return m.FocusedID() == idTopics
}

// handleMouseScroll processes scroll wheel events.
// It returns a command and a boolean indicating if the event was handled.
func (m *model) handleMouseScroll(msg tea.MouseMsg) (tea.Cmd, bool) {
	if msg.Action == tea.MouseActionPress && (msg.Button == tea.MouseButtonWheelUp || msg.Button == tea.MouseButtonWheelDown) {
		if m.isHistoryFocused() && m.history.CanScroll() {
			return m.history.Scroll(msg), true
		}
		if m.isTopicsFocused() {
			delta := -1
			if msg.Button == tea.MouseButtonWheelDown {
				delta = 1
			}
			m.topics.Scroll(delta)
			return nil, true
		}
		return nil, false
	}
	return nil, false
}

// handleHelpClick switches to help mode when the icon is clicked.
func (m *model) handleHelpClick(msg tea.MouseMsg) (tea.Cmd, bool) {
	if msg.Action != tea.MouseActionPress || msg.Button != tea.MouseButtonLeft {
		return nil, false
	}
	helpWidth := lipgloss.Width(ui.HelpStyle.Render("?"))
	helpY := 0
	if m.ui.width < helpReflowWidth {
		helpY = 1
	}
	if msg.Y == helpY && msg.X >= m.ui.width-helpWidth {
		return m.SetMode(constants.ModeHelp), true
	}
	return nil, false
}

// handleMouseLeft manages left-click focus and selection.
func (m *model) handleMouseLeft(msg tea.MouseMsg) tea.Cmd {
	if cmd, handled := m.handleHelpClick(msg); handled {
		return cmd
	}
	cmd := m.focusFromMouse(msg.Y)
	if m.isHistoryFocused() && !m.history.ShowArchived() {
		contentMsg := m.clientContentMouseMsg(msg)
		m.history.HandleClick(contentMsg, m.ui.elemPos[idHistory], 0)
	}
	helpWidth := lipgloss.Width(ui.HelpStyle.Render("?"))
	helpY := 0
	xOffset := 0
	if m.ui.width < helpReflowWidth {
		helpY = 1
		xOffset = 1
	}
	if msg.Y == helpY && msg.X >= m.ui.width-helpWidth+xOffset {
		m.SetMode(constants.ModeHelp)
	}
	return cmd
}

// handleMouse processes mouse events common to all modes.
func (m *model) handleMouse(msg tea.MouseMsg) tea.Cmd {
	if cmd, handled := m.handleHelpClick(msg); handled {
		return cmd
	}
	return nil
}

// handleClientMouse processes mouse events in client mode.
func (m *model) handleClientMouse(msg tea.MouseMsg) tea.Cmd {
	m.updateHoverState(msg)
	cmd, handled := m.handleMouseScroll(msg)
	if handled {
		return cmd
	}
	if msg.Action == tea.MouseActionPress && (msg.Button == tea.MouseButtonWheelUp || msg.Button == tea.MouseButtonWheelDown) {
		if msg.Button == tea.MouseButtonWheelUp {
			m.ui.viewport.ScrollUp(1)
		} else {
			m.ui.viewport.ScrollDown(1)
		}
		return nil
	}
	var cmds []tea.Cmd
	if msg.Type == tea.MouseLeft {
		if cmd := m.handleMouseLeft(msg); cmd != nil {
			cmds = append(cmds, cmd)
		}
	}
	if msg.Type == tea.MouseLeft || msg.Type == tea.MouseRight {
		contentMsg := m.clientContentMouseMsg(msg)
		if cmd := m.topics.HandleClick(contentMsg, 0); cmd != nil {
			cmds = append(cmds, cmd)
		}
	}
	if len(cmds) == 0 {
		return nil
	}
	return tea.Batch(cmds...)
}
