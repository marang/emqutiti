package topics

import (
	tea "github.com/charmbracelet/bubbletea"
	"testing"
)

func TestHandleClickFocusesOnly(t *testing.T) {
	c := newTestComponent()
	c.Items = []Item{{Name: "a", Subscribed: true}}
	c.ChipBounds = []ChipBound{{XPos: 0, YPos: 0, Width: 5, Height: 1, Index: 0}}
	msg := tea.MouseMsg{Type: tea.MouseLeft, X: 0, Y: 0}
	cmd := c.HandleClick(msg, 0)
	// Left click should NOT toggle subscription, only focus.
	if !c.Items[0].Subscribed {
		t.Fatalf("left click should not toggle subscription")
	}
	if c.Selected() != 0 {
		t.Fatalf("expected selected index 0, got %d", c.Selected())
	}
	if cmd != nil {
		t.Fatalf("left click should not return a command")
	}
}

func TestHandleClickSelectsBoundIndex(t *testing.T) {
	c := newTestComponent()
	c.Items = []Item{{Name: "a", Subscribed: true}, {Name: "b", Subscribed: true}, {Name: "c", Subscribed: true}}
	c.ChipBounds = []ChipBound{
		{XPos: 0, YPos: 0, Width: 5, Height: 1, Index: 1},
		{XPos: 6, YPos: 0, Width: 5, Height: 1, Index: 2},
	}
	msg := tea.MouseMsg{Type: tea.MouseLeft, X: 0, Y: 0}
	idx := c.TopicAtPosition(msg.X, msg.Y)
	if idx != 1 {
		t.Fatalf("expected position to map to index 1, got %d", idx)
	}
	cmd := c.HandleClick(msg, 0)
	// All items should remain subscribed (no toggle on click).
	for _, it := range c.Items {
		if !it.Subscribed {
			t.Fatalf("left click should not toggle any item, %q was toggled", it.Name)
		}
	}
	// The clicked chip should be selected.
	if c.Selected() != 1 {
		t.Fatalf("expected selected index 1, got %d", c.Selected())
	}
	if cmd != nil {
		t.Fatalf("left click should not return a command")
	}
}

func TestEnterTogglesSelectedTopic(t *testing.T) {
	c := newTestComponent()
	c.Items = []Item{{Name: "a", Subscribed: true}, {Name: "b", Subscribed: false}}
	c.SetSelected(0)

	// Simulate Enter key via ToggleTopic (called from Update on Enter/Space).
	cmd := c.ToggleTopic(0)
	if c.Items[0].Subscribed {
		t.Fatalf("expected topic 'a' to be unsubscribed after toggle")
	}
	if cmd == nil {
		t.Fatalf("expected toggle command")
	}
	msg := cmd()
	tm, ok := msg.(ToggleMsg)
	if !ok {
		t.Fatalf("expected ToggleMsg, got %T", msg)
	}
	if tm.Topic != "a" || tm.Subscribed {
		t.Fatalf("unexpected ToggleMsg: %+v", tm)
	}
}
