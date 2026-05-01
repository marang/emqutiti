package emqutiti

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/marang/emqutiti/topics"
	"github.com/marang/emqutiti/ui"
)

// isConnected reports whether the MQTT client is actually connected.
func (m *model) isConnected() bool {
	return m.mqttClient != nil && m.mqttClient.Client != nil && m.mqttClient.Client.IsConnected()
}

// clientInfoLine renders the connection status.
func (m *model) clientInfoLine() string {
	clientID := ""
	if m.mqttClient != nil && m.mqttClient.Client != nil {
		r := m.mqttClient.Client.OptionsReader()
		clientID = r.ClientID()
	}
	status := strings.TrimSpace(m.connections.Connection + " " + clientID)
	st := ui.InfoSubtleStyle
	if m.isConnected() {
		st = st.Foreground(ui.ColGreen)
	} else if strings.HasPrefix(m.connections.Connection, "Connecting") {
		// Keep subtle style for connecting state.
		st = ui.InfoSubtleStyle
		if frame := m.connectingFrame(); frame != "" {
			status = frame + " " + status
		}
	} else {
		// Disconnected, connection lost, or failed.
		st = st.Foreground(ui.ColWarn)
	}
	return st.Render(status)
}

// viewClient renders the main client view.
func (m *model) viewClient() string {
	m.ui.elemPos = map[string]int{}
	statusLine := m.clientInfoLine()
	var parts []string
	y := 1

	topicsBox, topicBox, bounds := m.renderTopicsSection()
	parts = append(parts, topicBox, topicsBox)

	m.ui.elemPos[idTopic] = y
	y += lipgloss.Height(topicBox)
	m.ui.elemPos[idTopics] = y
	y += lipgloss.Height(topicsBox)

	m.topics.ChipBounds = make([]topics.ChipBound, len(bounds))
	for i, b := range bounds {
		m.topics.ChipBounds[i] = topics.ChipBound{
			XPos:   b.XPos,
			YPos:   b.YPos,
			Width:  b.Width,
			Height: b.Height,
			Index:  b.Index,
		}
	}
	startX := 2
	startY := m.ui.elemPos[idTopics] + 1
	for i := range m.topics.ChipBounds {
		m.topics.ChipBounds[i].XPos += startX
		m.topics.ChipBounds[i].YPos += startY
	}

	messageBox := m.message.View()
	parts = append(parts, messageBox)
	m.ui.elemPos[idMessage] = y
	y += lipgloss.Height(messageBox)

	messagesBox := m.renderHistorySection()
	parts = append(parts, messagesBox)
	m.ui.elemPos[idHistory] = y

	content := lipgloss.JoinVertical(lipgloss.Left, parts...)

	box := lipgloss.NewStyle().Width(m.ui.width).Padding(0, 1, 1, 1).Render(content)
	contextHelp := m.renderContextHelp()
	m.ui.viewport.SetContent(box)
	m.ui.viewport.Width = m.ui.width
	// Deduct the global shortcut header, connection status line, and top
	// context help so the hint remains fixed while the main content scrolls.
	m.ui.viewport.Height = m.ui.height - 2 - contextHelpHeight(contextHelp)
	if m.ui.viewport.Height < 1 {
		m.ui.viewport.Height = 1
	}

	view := m.ui.viewport.View()
	contentView := lipgloss.JoinVertical(lipgloss.Left, statusLine, contextHelp, view)
	return m.overlayHelp(contentView)
}
