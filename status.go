package emqutiti

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	connections "github.com/marang/emqutiti/connections"
)

// handleStatusMessage processes broker status updates.
func (m *model) handleStatusMessage(msg connections.StatusMessage) tea.Cmd {
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
	return m.connections.ListenStatus()
}

// handleMQTTMessage appends received MQTT messages to history.
func (m *model) handleMQTTMessage(msg MQTTMessage) tea.Cmd {
	m.history.Append(msg.Topic, msg.Payload, "sub", msg.Retained, fmt.Sprintf("Received on %s: %s", msg.Topic, msg.Payload))
	return listenMessages(m.mqttClient.MessageChan)
}

// updateClientStatus returns commands to listen for connection and message updates.
func (m *model) updateClientStatus() []tea.Cmd {
	cmds := []tea.Cmd{m.connections.ListenStatus()}
	if m.mqttClient != nil {
		cmds = append(cmds, listenMessages(m.mqttClient.MessageChan))
	}
	return cmds
}
