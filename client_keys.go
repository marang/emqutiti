package emqutiti

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/marang/emqutiti/constants"
)

type reconnectPromptMsg string

// Handler processes keyboard input in client mode.
type Handler interface {
	// HandleClientKey reacts to a key message and optionally returns a Tea command.
	HandleClientKey(msg tea.KeyMsg) tea.Cmd
}

// HandleClientKey dispatches a key message to the provided handler.
func HandleClientKey(h Handler, msg tea.KeyMsg) tea.Cmd {
	if h == nil {
		return nil
	}
	return h.HandleClientKey(msg)
}

// HandleClientKey processes keyboard events in client mode.
func (m *model) HandleClientKey(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case constants.KeyCtrlD:
		return m.handleQuitKey()
	case constants.KeyCtrlC:
		return m.handleCopyKey()
	case constants.KeyCtrlX:
		return m.handleDisconnectKey()
	case constants.KeySlash:
		return m.handleHistoryFilterKey()
	case constants.KeyCtrlF:
		return m.handleClearFilterKey()
	case constants.KeySpace:
		return m.handleSpaceKey()
	case constants.KeyShiftUp:
		return m.handleShiftUpKey()
	case constants.KeyShiftDown:
		return m.handleShiftDownKey()
	case constants.KeyTab:
		return m.handleTabKey()
	case constants.KeyShiftTab:
		return m.handleShiftTabKey()
	case constants.KeyLeft:
		return m.handleLeftKey()
	case constants.KeyRight:
		return m.handleRightKey()
	case constants.KeyCtrlShiftUp:
		return m.handleResizeUpKey()
	case constants.KeyCtrlShiftDown:
		return m.handleResizeDownKey()
	case constants.KeyCtrlA:
		return m.handleSelectAllKey()
	case constants.KeyUp, constants.KeyDown, constants.KeyK, constants.KeyJ:
		return m.handleScrollKeys(msg.String())
	case constants.KeyCtrlE:
		return m.handlePublishRetainKey()
	case constants.KeyCtrlS:
		if m.ui.focusOrder[m.ui.focusIndex] == idMessage {
			return m.handlePublishKey()
		}
		return nil
	case constants.KeyEnter:
		return m.handleEnterKey()
	case constants.KeyP:
		return m.handleTogglePublishKey()
	case constants.KeyA:
		return m.handleArchiveKey()
	case constants.KeyDelete:
		return m.handleDeleteKey()
	default:
		return m.handleModeSwitchKey(msg)
	}
}

// handleQuitKey saves current state and quits the application.
func (m *model) handleQuitKey() tea.Cmd {
	m.connections.SaveCurrent(
		m.topics.Snapshot(),
		m.payloads.Snapshot(),
		m.layout.message.height,
		m.layout.topics.height,
		m.layout.history.height,
	)
	m.traces.SavePlannedTraces()
	return tea.Quit
}

// handleDisconnectKey disconnects from the active broker after confirmation.
func (m *model) handleDisconnectKey() tea.Cmd {
	if m.mqttClient == nil {
		return m.SetMode(constants.ModeConnections)
	}
	name := m.connections.Active
	m.StartConfirm(
		fmt.Sprintf("Disconnect from '%s'? [y/n]", name),
		"You'll return to the broker manager where you can reconnect.",
		nil,
		func() tea.Cmd {
			m.mqttClient.Disconnect()
			m.connections.SetDisconnected(name, "")
			m.connections.RefreshConnectionItems()
			m.connections.Connection = ""
			m.connections.Active = ""
			m.mqttClient = nil
			return func() tea.Msg { return reconnectPromptMsg(name) }
		},
		nil,
	)
	return nil
}

// handleScrollKeys dispatches scroll events based on focus.
func (m *model) handleScrollKeys(key string) tea.Cmd {
	switch m.ui.focusOrder[m.ui.focusIndex] {
	case idHistory:
		return m.handleHistoryScroll(key)
	case idTopics:
		return m.handleTopicScroll(key)
	default:
		return nil
	}
}

// publishMessage publishes the current message to flagged topics or the
// selected topic if none are flagged. When retained is true, the message is
// published with the retained flag and noted in history.
func (m *model) publishMessage(retained bool) {
	if m.ui.focusOrder[m.ui.focusIndex] != idMessage {
		return
	}
	payload := m.message.Input().Value()
	var targets []string
	for _, t := range m.topics.Items {
		if t.Publish {
			targets = append(targets, t.Name)
		}
	}
	if len(targets) == 0 {
		sel := m.topics.Selected()
		if sel >= 0 && sel < len(m.topics.Items) {
			targets = append(targets, m.topics.Items[sel].Name)
		}
	}
	for _, topic := range targets {
		m.payloads.Add(topic, payload)
		msg := fmt.Sprintf("Published to %s: %s", topic, payload)
		if retained {
			msg = fmt.Sprintf("Published retained to %s: %s", topic, payload)
		}
		m.history.Append(topic, payload, "pub", retained, msg)
		if m.mqttClient != nil {
			m.mqttClient.Publish(topic, 0, retained, payload)
		}
	}
}

// handlePublishKey publishes the current message without the retained flag.
func (m *model) handlePublishKey() tea.Cmd {
	m.publishMessage(false)
	return nil
}

// handlePublishRetainKey publishes the current message with the retained flag.
func (m *model) handlePublishRetainKey() tea.Cmd {
	m.publishMessage(true)
	return nil
}

// handleDeleteKey dispatches deletion based on focus.
func (m *model) handleDeleteKey() tea.Cmd {
	switch m.ui.focusOrder[m.ui.focusIndex] {
	case idHistory:
		if !m.history.ShowArchived() {
			return m.handleDeleteHistoryKey()
		}
	case idTopics:
		sel := m.topics.Selected()
		if sel >= 0 && sel < len(m.topics.Items) {
			return m.handleDeleteTopicKey()
		}
	}
	return nil
}
