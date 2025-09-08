package emqutiti

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/marang/emqutiti/topics"
	"github.com/marang/emqutiti/ui"
)

// clientInfoLine renders the connection status.
func (m *model) clientInfoLine() string {
	clientID := ""
	if m.mqttClient != nil {
		r := m.mqttClient.Client.OptionsReader()
		clientID = r.ClientID()
	}
	status := strings.TrimSpace(m.connections.Connection + " " + clientID)
	st := ui.InfoSubtleStyle
	if strings.HasPrefix(m.connections.Connection, "Connected") {
		st = st.Foreground(ui.ColGreen)
	} else if strings.HasPrefix(m.connections.Connection, "Connection lost") || strings.HasPrefix(m.connections.Connection, "Failed") {
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

	if !m.layout.topics.collapsed {
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
			}
		}
		startX := 2
		startY := m.ui.elemPos[idTopics] + 1
		for i := range m.topics.ChipBounds {
			m.topics.ChipBounds[i].XPos += startX
			m.topics.ChipBounds[i].YPos += startY
		}
	} else {
		m.topics.ChipBounds = nil
	}

	if !m.layout.message.collapsed {
		messageBox := m.message.View()
		parts = append(parts, messageBox)
		m.ui.elemPos[idMessage] = y
		y += lipgloss.Height(messageBox)
	}

	if !m.layout.history.collapsed {
		messagesBox := m.renderHistorySection()
		parts = append(parts, messagesBox)
		m.ui.elemPos[idHistory] = y
	}

	content := lipgloss.JoinVertical(lipgloss.Left, parts...)

	box := lipgloss.NewStyle().Width(m.ui.width).Padding(0, 1, 1, 1).Render(content)
	m.ui.viewport.SetContent(box)
	m.ui.viewport.Width = m.ui.width
	// Deduct two lines for the info header.
	m.ui.viewport.Height = m.ui.height - 2

	view := m.ui.viewport.View()
	contentView := lipgloss.JoinVertical(lipgloss.Left, statusLine, view)
	return m.overlayHelp(contentView)
}
