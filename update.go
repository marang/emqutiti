package emqutiti

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/marang/emqutiti/connections"
	"github.com/marang/emqutiti/payloads"
	"github.com/marang/emqutiti/topics"
	"github.com/marang/emqutiti/traces"
)

// Update routes messages based on the current mode.
func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if cmd, handled := m.handlePanelResize(msg); handled {
		return m, cmd
	}
	if cmd, handled := m.handleCtrlEnterMsg(msg); handled {
		return m, cmd
	}
	if cmd, handled := m.handleShiftedSpaceMsg(msg); handled {
		return m, cmd
	}
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		return m, m.handleWindowSize(msg)
	case animationTickMsg:
		return m, m.handleAnimationTick()
	case publishResultMsg:
		return m, m.handlePublishResult(msg)
	case subscriptionResultMsg:
		return m, m.handleSubscriptionResult(msg)
	case traces.ReportMsg:
		return m, m.traces.HandleReport(msg)
	case connections.StatusMessage:
		return m, m.handleStatusMessage(msg)
	case MQTTMessage:
		return m, m.handleMQTTMessage(msg)
	case mqttListenClosedMsg:
		m.ui.listeners.mqtt = false
		return m, nil
	case topics.ToggleMsg:
		return m, m.handleTopicToggle(msg)
	case payloads.LoadMsg:
		m.topics.SetTopic(msg.Topic)
		m.message.SetPayload(msg.Payload)
		return m, nil
	case tea.MouseMsg:
		if cmd := m.handleMouse(msg); cmd != nil {
			return m, cmd
		}
	case tea.KeyMsg:
		m.clearHoverState()
		if cmd, handled := m.handleKeyNav(msg); handled {
			return m, cmd
		}
	}

	if c, ok := m.components[m.CurrentMode()]; ok {
		cmd := c.Update(msg)
		return m, cmd
	}
	return m, nil
}
