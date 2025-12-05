package topics

import (
	tea "github.com/charmbracelet/bubbletea"
	"testing"
)

func TestHandleClickToggles(t *testing.T) {
	c := newTestComponent()
	c.Items = []Item{{Name: "a", Subscribed: true}}
	c.ChipBounds = []ChipBound{{XPos: 0, YPos: 0, Width: 5, Height: 1, Index: 0}}
	msg := tea.MouseMsg{Type: tea.MouseLeft, X: 0, Y: 0}
	c.HandleClick(msg, 0)
	if c.Items[0].Subscribed {
		t.Fatalf("expected toggle on click")
	}
}

func TestHandleClickUsesBoundIndex(t *testing.T) {
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
	states := map[string]bool{}
	for _, it := range c.Items {
		states[it.Name] = it.Subscribed
	}
	if states["b"] {
		t.Fatalf("expected click to toggle item at bound index")
	}
	if !states["a"] || !states["c"] {
		t.Fatalf("unexpected toggle of other items")
	}
	if cmd == nil {
		t.Fatalf("expected toggle command")
	}
	if msg := cmd(); msg != nil {
		tm, ok := msg.(ToggleMsg)
		if !ok {
			t.Fatalf("expected ToggleMsg, got %T", msg)
		}
		if tm.Topic != "b" {
			t.Fatalf("expected toggle topic 'b', got %q", tm.Topic)
		}
	}
}
