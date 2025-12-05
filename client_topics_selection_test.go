package emqutiti

import (
	"testing"

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
