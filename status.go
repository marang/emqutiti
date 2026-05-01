package emqutiti

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	connections "github.com/marang/emqutiti/connections"
)

// handleStatusMessage processes broker status updates.
func (m *model) handleStatusMessage(msg connections.StatusMessage) tea.Cmd {
	m.ui.listeners.status = false
	m.history.Append("", string(msg), "log", false, string(msg))
	if strings.HasPrefix(string(msg), "Connected") && m.connections.Active != "" {
		m.connections.SetConnected(m.connections.Active)
		m.connections.Connection = string(msg)
		m.connections.RefreshConnectionItems()
		m.SubscribeActiveTopics()
	} else if strings.HasPrefix(string(msg), "Connection lost") && m.connections.Active != "" {
		m.connections.SetDisconnected(m.connections.Active, "")
		m.connections.Connection = string(msg)
		m.connections.RefreshConnectionItems()
	}
	return tea.Batch(m.updateClientStatus()...)
}

// handleMQTTMessage appends received MQTT messages to history.
func (m *model) handleMQTTMessage(msg MQTTMessage) tea.Cmd {
	m.ui.listeners.mqtt = false
	oldScroll := m.rawHistoryScrollPercent()
	m.history.Append(msg.Topic, msg.Payload, "sub", msg.Retained, fmt.Sprintf("Received on %s: %s", msg.Topic, msg.Payload))
	cmds := append(m.updateClientStatus(),
		m.startHistoryPulse(),
		m.startHistoryScrollAnimation(oldScroll, m.rawHistoryScrollPercent()),
	)
	return tea.Batch(cmds...)
}

// updateClientStatus returns commands to listen for connection and message updates.
func (m *model) updateClientStatus() []tea.Cmd {
	var cmds []tea.Cmd
	if cmd := m.listenStatusOnce(); cmd != nil {
		cmds = append(cmds, cmd)
	}
	if cmd := m.listenMQTTOnce(); cmd != nil {
		cmds = append(cmds, cmd)
	}
	return cmds
}

func (m *model) listenStatusOnce() tea.Cmd {
	if m.ui.listeners.status {
		return nil
	}
	if m.connections.StatusChan == nil {
		return func() tea.Msg { return nil }
	}
	m.ui.listeners.status = true
	return m.connections.ListenStatus()
}

func (m *model) listenMQTTOnce() tea.Cmd {
	if m.ui.listeners.mqtt || m.mqttClient == nil || m.mqttClient.safeMessageChan() == nil {
		return nil
	}
	m.ui.listeners.mqtt = true
	return listenMessages(m.mqttClient.safeMessageChan())
}
