package emqutiti

import (
	"fmt"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/marang/emqutiti/history"
	"github.com/marang/emqutiti/topics"
)

func TestTopicPulseExpires(t *testing.T) {
	m, _ := initialModel(nil)
	m.startTopicPulse("sensors/temp")

	if _, ok := m.topicPulsePhase("sensors/temp"); !ok {
		t.Fatalf("expected active topic pulse")
	}
	for i := 0; i < topicPulseFrames; i++ {
		m.handleAnimationTick()
	}
	if _, ok := m.topicPulsePhase("sensors/temp"); ok {
		t.Fatalf("expected expired topic pulse")
	}
}

func TestToggleMsgStartsTopicPulse(t *testing.T) {
	m, _ := initialModel(nil)
	m.topics.Items = []topics.Item{{Name: "sensors/temp", Subscribed: true}}

	m.Update(topics.ToggleMsg{Topic: "sensors/temp", Subscribed: true})

	if _, ok := m.topicPulsePhase("sensors/temp"); !ok {
		t.Fatalf("expected topic toggle pulse")
	}
}

func TestPublishStartsFallbackTopicPulse(t *testing.T) {
	m, _ := initialModel(nil)
	m.topics.Items = []topics.Item{{Name: "sensors/temp", Subscribed: true}}
	m.topics.SetSelected(0)
	m.SetFocus(idMessage)

	m.handlePublishKey()

	if _, ok := m.topicPulsePhase("sensors/temp"); !ok {
		t.Fatalf("expected publish fallback topic pulse")
	}
	if got := m.historyPulseMarker(); got == " " {
		t.Fatalf("expected history pulse")
	}
}

func TestHistoryScrollAnimationStepsToTarget(t *testing.T) {
	m, _ := initialModel(nil)
	m.startHistoryScrollAnimation(0, 1)

	if !m.ui.animation.historyScroll.active {
		t.Fatalf("expected active history scroll animation")
	}
	for i := 0; i < historyScrollFrames; i++ {
		m.handleAnimationTick()
	}
	if m.ui.animation.historyScroll.active {
		t.Fatalf("expected history scroll animation to finish")
	}
	if got := m.ui.animation.historyScroll.current; got != 1 {
		t.Fatalf("expected target scroll percent, got %f", got)
	}
}

func TestHistoryMouseWheelDoesNotStartSmoothScroll(t *testing.T) {
	m, _ := initialModel(nil)
	m.Update(tea.WindowSizeMsg{Width: 60, Height: 20})
	for i := 0; i < 30; i++ {
		m.history.Append("t", fmt.Sprintf("msg %d", i), "pub", false, "")
	}
	m.viewClient()
	m.SetFocus(idHistory)

	m.Update(tea.MouseMsg{Action: tea.MouseActionPress, Button: tea.MouseButtonWheelDown})

	if m.ui.animation.historyScroll.active {
		t.Fatalf("mouse wheel should not start smooth scroll")
	}
}

func TestIncomingMQTTMessageStartsHistoryPulse(t *testing.T) {
	m, _ := initialModel(nil)
	m.mqttClient = &MQTTClient{MessageChan: make(chan MQTTMessage)}
	m.history.SetItems([]history.Item{})

	m.handleMQTTMessage(MQTTMessage{Topic: "sensors/temp", Payload: "42"})

	if got := m.historyPulseMarker(); got == " " {
		t.Fatalf("expected incoming message history pulse")
	}
}
