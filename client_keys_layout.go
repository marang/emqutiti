package emqutiti

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/marang/emqutiti/constants"
)

// handleTabKey moves focus forward.
func (m *model) handleTabKey() tea.Cmd {
	if len(m.ui.focusOrder) > 0 {
		m.focus.Next()
		m.ui.focusIndex = m.focus.Index()
		id := m.ui.focusOrder[m.ui.focusIndex]
		m.SetFocus(id)
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
	}
	return nil
}

// handleShiftTabKey moves focus backward.
func (m *model) handleShiftTabKey() tea.Cmd {
	if len(m.ui.focusOrder) > 0 {
		m.focus.Prev()
		m.ui.focusIndex = m.focus.Index()
		id := m.ui.focusOrder[m.ui.focusIndex]
		m.SetFocus(id)
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
	}
	return nil
}

// handleResizeUpKey reduces the height of the focused pane.
func (m *model) handleResizeUpKey() tea.Cmd {
	id := m.ui.focusOrder[m.ui.focusIndex]
	switch id {
	case idMessage:
		if m.layout.message.height > 1 {
			m.layout.message.height--
			m.message.Input().SetHeight(m.layout.message.height)
		}
	case idHistory:
		if m.layout.history.height > 1 {
			m.layout.history.height--
			m.history.List().SetSize(m.ui.width-4, m.layout.history.height)
		}
	case idTopics:
		if m.layout.topics.height > 1 {
			m.layout.topics.height--
		}
	}
	return nil
}

// handleResizeDownKey increases the height of the focused pane.
func (m *model) handleResizeDownKey() tea.Cmd {
	id := m.ui.focusOrder[m.ui.focusIndex]
	switch id {
	case idMessage:
		m.layout.message.height++
		m.message.Input().SetHeight(m.layout.message.height)
	case idHistory:
		m.layout.history.height++
		m.history.List().SetSize(m.ui.width-4, m.layout.history.height)
	case idTopics:
		m.layout.topics.height++
	}
	return nil
}

// handleModeSwitchKey switches application modes for special key combos.
func (m *model) handleModeSwitchKey(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case constants.KeyCtrl1:
		m.layout.topics.collapsed = !m.layout.topics.collapsed
		return nil
	case constants.KeyCtrl2:
		m.layout.message.collapsed = !m.layout.message.collapsed
		if !m.layout.message.collapsed {
			m.message.Input().SetHeight(m.layout.message.height)
		}
		return nil
	case constants.KeyCtrl3:
		m.layout.history.collapsed = !m.layout.history.collapsed
		if !m.layout.history.collapsed {
			m.history.List().SetSize(m.ui.width-4, m.layout.history.height)
		}
		return nil
	case constants.KeyCtrlB:
		if err := m.connections.Manager.LoadProfiles(""); err != nil {
			m.history.Append("", err.Error(), "log", false, err.Error())
		}
		m.connections.RefreshConnectionItems()
		m.connections.SaveCurrent(m.topics.Snapshot(), m.payloads.Snapshot())
		m.traces.SavePlannedTraces()
		return m.SetMode(constants.ModeConnections)
	case constants.KeyCtrlT:
		m.topics.SetActivePane(0)
		m.topics.RebuildActiveTopicList()
		m.topics.SetSelected(0)
		m.topics.List().SetSize(m.ui.width/2-4, m.ui.height-4)
		return m.SetMode(constants.ModeTopics)
	case constants.KeyCtrlP:
		m.payloads.List().SetSize(m.ui.width-4, m.ui.height-4)
		return m.SetMode(constants.ModePayloads)
	case constants.KeyCtrlR:
		m.traces.List().SetSize(m.ui.width-4, m.ui.height-4)
		return m.SetMode(constants.ModeTracer)
	case constants.KeyCtrlL:
		m.logs.SetSize(m.ui.width, m.ui.height)
		m.logs.Focus()
		return m.SetMode(constants.ModeLogs)
	default:
		return nil
	}
}
