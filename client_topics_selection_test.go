package emqutiti

import (
	"errors"
	"fmt"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/marang/emqutiti/constants"
	"github.com/marang/emqutiti/topics"
)

func TestTopicSelectionPersistsAcrossFocus(t *testing.T) {
	m, _ := initialModel(nil)
	m.topics.Items = []topics.Item{{Name: "a"}, {Name: "b"}, {Name: "c"}}
	m.topics.SetSelected(1)

	// Simulate focus cycling forward to topics
	m.focus.Set(4) // idHelp index so tab wraps to idTopic then idTopics
	m.ui.focusIndex = 4
	m.handleTabKey()
	m.handleTabKey()
	if m.topics.Selected() != 1 {
		t.Fatalf("expected selected index 1 after Tab, got %d", m.topics.Selected())
	}

	// Simulate focus cycling backward to topics
	m.focus.Set(2) // idMessage index so shift+tab goes to idTopics
	m.ui.focusIndex = 2
	m.handleShiftTabKey()
	if m.topics.Selected() != 1 {
		t.Fatalf("expected selected index 1 after Shift+Tab, got %d", m.topics.Selected())
	}
}

func TestTopicInputInitiallyBlurred(t *testing.T) {
	m, _ := initialModel(nil)
	if m.topics.Input.Focused() {
		t.Fatalf("topic input should not be focused on init")
	}
}

func TestToggleTopicKeepsSelection(t *testing.T) {
	m, _ := initialModel(nil)
	m.topics.Items = []topics.Item{
		{Name: "a", Subscribed: true},
		{Name: "b", Subscribed: true},
		{Name: "c", Subscribed: true},
	}
	m.topics.SetSelected(2)
	m.focus.Set(1)
	m.ui.focusIndex = 1
	m.handleEnterKey()
	if m.topics.Items[m.topics.Selected()].Name != "c" {
		t.Fatalf("expected to stay on topic 'c', got %q", m.topics.Items[m.topics.Selected()].Name)
	}
}

func TestSubscribeCompletionKeepsSelectedTopic(t *testing.T) {
	m := reviewFixture(t, 80, 24)
	m.mqttClient = &MQTTClient{Client: &failingClient{}}
	m.topics.Items = []topics.Item{
		{Name: "df", Subscribed: true},
		{Name: "long/topic", Subscribed: true},
		{Name: "test", Subscribed: true},
		{Name: "testx"},
	}
	m.topics.RebuildActiveTopicList()
	m.SetMode(constants.ModeClient)
	m.SetFocus(idTopics)
	m.topics.SetSelected(3)
	toggle := HandleClientKey(m, tea.KeyMsg{Type: tea.KeyEnter})
	if toggle == nil {
		t.Fatal("Enter did not queue a subscription")
	}
	_, subscribe := m.Update(toggle())
	if subscribe == nil {
		t.Fatal("topic toggle did not dispatch the MQTT request")
	}
	if name := m.topics.Items[m.topics.Selected()].Name; name != "testx" {
		t.Fatalf("Enter changed selection before MQTT result to %q", name)
	}
	applyMQTTCommand(m, subscribe)
	selected := m.topics.Selected()
	if selected < 0 || selected >= len(m.topics.Items) {
		t.Fatalf("subscription completion lost selection: %d", selected)
	}
	if name := m.topics.Items[selected].Name; name != "testx" {
		t.Fatalf("subscription completion moved selection to %q, want testx", name)
	}
	if m.FocusedID() != idTopics || !m.topics.Items[selected].Subscribed {
		t.Fatalf("subscription completion changed focus or failed to subscribe: focus=%s item=%+v", m.FocusedID(), m.topics.Items[selected])
	}
	if targets := m.publishTargets(); len(targets) != 1 || targets[0] != "testx" {
		t.Fatalf("subscription completion changed publish fallback: %v", targets)
	}
}

func TestSubscriptionResultPreservesLiveSelection(t *testing.T) {
	for _, subscribed := range []bool{false, true} {
		for _, fail := range []bool{false, true} {
			for _, changeSelection := range []bool{false, true} {
				t.Run(fmt.Sprintf("subscribed=%t/failure=%t/new-selection=%t", subscribed, fail, changeSelection), func(t *testing.T) {
					m := reviewFixture(t, 80, 24)
					client := &failingClient{}
					if fail {
						client.subErr = errors.New("subscribe denied")
						client.unsubErr = errors.New("unsubscribe denied")
					}
					m.mqttClient = &MQTTClient{Client: client}
					m.topics.Items = []topics.Item{
						{Name: "df", Subscribed: true},
						{Name: "test", Subscribed: true},
						{Name: "testx", Subscribed: subscribed},
					}
					m.topics.SortTopics()
					m.topics.RebuildActiveTopicList()
					m.SetMode(constants.ModeClient)
					m.SetFocus(idTopics)
					m.topics.SetSelected(m.topicIndexByName("testx"))
					toggle := HandleClientKey(m, tea.KeyMsg{Type: tea.KeyEnter})
					if toggle == nil {
						t.Fatal("Enter did not queue a subscription toggle")
					}
					_, request := m.Update(toggle())
					if request == nil {
						t.Fatal("subscription toggle did not dispatch an MQTT request")
					}
					wantSelection, wantFocus := "testx", idTopics
					if changeSelection {
						wantSelection, wantFocus = "test", idMessage
						m.topics.SetSelected(m.topicIndexByName(wantSelection))
						m.SetFocus(wantFocus)
					}
					applyMQTTCommand(m, request)
					selected := m.topics.Selected()
					if selected < 0 || selected >= len(m.topics.Items) || m.topics.Items[selected].Name != wantSelection {
						t.Fatalf("MQTT result lost live selection %q: index=%d items=%+v", wantSelection, selected, m.topics.Items)
					}
					wantSubscribed := !subscribed
					if fail {
						wantSubscribed = subscribed
					}
					if item := m.topics.Items[m.topicIndexByName("testx")]; item.Subscribed != wantSubscribed {
						t.Fatalf("MQTT result reconciled subscription incorrectly: %+v", item)
					}
					if m.FocusedID() != wantFocus {
						t.Fatalf("MQTT result changed focus: got %q, want %q", m.FocusedID(), wantFocus)
					}
					if targets := m.publishTargets(); len(targets) != 1 || targets[0] != wantSelection {
						t.Fatalf("MQTT result changed publish fallback: %v", targets)
					}
				})
			}
		}
	}
}

func TestTogglePublishKeepsSelection(t *testing.T) {
	m, _ := initialModel(nil)
	m.topics.Items = []topics.Item{
		{Name: "a", Subscribed: true},
		{Name: "b", Subscribed: true},
		{Name: "c", Subscribed: true},
	}
	m.topics.SetSelected(2)
	m.focus.Set(1)
	m.ui.focusIndex = 1
	m.handleTogglePublishKey()
	if m.topics.Items[m.topics.Selected()].Name != "c" {
		t.Fatalf("expected to stay on topic 'c', got %q", m.topics.Items[m.topics.Selected()].Name)
	}
}

func TestEnterTogglesSelectedTopicName(t *testing.T) {
	m, _ := initialModel(nil)
	m.topics.Items = []topics.Item{{Name: "a", Subscribed: true}, {Name: "b", Subscribed: true}}
	m.topics.SetSelected(1)
	m.focus.Set(1)
	m.ui.focusIndex = 1

	cmd := m.handleEnterKey()
	if cmd == nil {
		t.Fatalf("expected command when toggling topic")
	}
	msg := cmd()
	toggle, ok := msg.(topics.ToggleMsg)
	if !ok {
		t.Fatalf("expected ToggleMsg, got %T", msg)
	}
	if toggle.Topic != "b" {
		t.Fatalf("expected toggle for topic 'b', got %q", toggle.Topic)
	}
}

func TestMultipleTogglesFollowSelection(t *testing.T) {
	m, _ := initialModel(nil)
	m.topics.Items = []topics.Item{{Name: "a", Subscribed: true}, {Name: "b", Subscribed: true}, {Name: "c", Subscribed: true}}

	m.topics.SetSelected(0)
	m.focus.Set(1)
	m.ui.focusIndex = 1

	firstSelection := m.topics.Items[m.topics.Selected()].Name
	first := m.handleEnterKey()
	if msg := first(); msg != nil {
		toggle := msg.(topics.ToggleMsg)
		if toggle.Topic != firstSelection {
			t.Fatalf("expected first toggle for %q, got %q", firstSelection, toggle.Topic)
		}
	}
	firstIdx := -1
	for i, it := range m.topics.Items {
		if it.Name == firstSelection {
			firstIdx = i
			break
		}
	}
	if firstIdx < 0 {
		t.Fatalf("first selection %q not found", firstSelection)
	}
	if m.topics.Items[firstIdx].Subscribed {
		t.Fatalf("expected first topic to be unsubscribed")
	}

	targetName := "c"
	for i, it := range m.topics.Items {
		if it.Name == targetName {
			m.topics.SetSelected(i)
			break
		}
	}
	secondSelection := m.topics.Items[m.topics.Selected()].Name
	second := m.handleEnterKey()
	if msg := second(); msg != nil {
		toggle := msg.(topics.ToggleMsg)
		if toggle.Topic != secondSelection {
			t.Fatalf("expected second toggle for %q, got %q", secondSelection, toggle.Topic)
		}
	}
	secondIdx := -1
	for i, it := range m.topics.Items {
		if it.Name == secondSelection {
			secondIdx = i
			break
		}
	}
	if secondIdx < 0 {
		t.Fatalf("second selection %q not found", secondSelection)
	}
	if m.topics.Items[secondIdx].Subscribed {
		t.Fatalf("expected third topic to be unsubscribed")
	}
	middleIdx := -1
	for i, it := range m.topics.Items {
		if it.Name == "b" {
			middleIdx = i
			break
		}
	}
	if middleIdx < 0 {
		t.Fatalf("middle topic not found")
	}
	if !m.topics.Items[middleIdx].Subscribed {
		t.Fatalf("expected middle topic to remain subscribed")
	}
}
