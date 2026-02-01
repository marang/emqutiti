package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func TestNewSelectFieldNoOptions(t *testing.T) {
	sf, err := NewSelectField("", nil)
	if err == nil || sf != nil {
		t.Fatalf("expected error for empty options")
	}
}

func TestSelectFieldEmptyOptions(t *testing.T) {
	sf := &SelectField{}
	if v := sf.Value(); v != "" {
		t.Fatalf("expected empty value, got %q", v)
	}
	view := sf.View()
	if !strings.Contains(view, "-") {
		t.Fatalf("expected placeholder in view, got %q", view)
	}
}

func TestSelectFieldNavigatesOptions(t *testing.T) {
	sf, err := NewSelectField("one", []string{"one", "two", "three"})
	if err != nil {
		t.Fatalf("err when creating a new selectfield %v", err)
	}
	sf.Focus()

	// up from first stays at first (boundary)
	sf.Update(tea.KeyMsg{Type: tea.KeyUp})
	if got, want := sf.Index, 0; got != want {
		t.Fatalf("index=%d want=%d", got, want)
	}

	// down moves to next
	sf.Update(tea.KeyMsg{Type: tea.KeyDown})
	if got, want := sf.Index, 1; got != want {
		t.Fatalf("index=%d want=%d", got, want)
	}

	// space advances
	sf.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	if got, want := sf.Index, 2; got != want {
		t.Fatalf("index=%d want=%d", got, want)
	}

	// down from last stays at last (boundary)
	sf.Update(tea.KeyMsg{Type: tea.KeyDown})
	if got, want := sf.Index, 2; got != want {
		t.Fatalf("index=%d want=%d", got, want)
	}

	// up moves back
	sf.Update(tea.KeyMsg{Type: tea.KeyUp})
	if got, want := sf.Index, 1; got != want {
		t.Fatalf("index=%d want=%d", got, want)
	}
}

func TestSelectFieldWantsKey(t *testing.T) {
	sf, err := NewSelectField("two", []string{"one", "two", "three"})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	sf.Focus()
	sf.Index = 1 // middle item

	// In middle, field wants up/down keys.
	if !sf.WantsKey(tea.KeyMsg{Type: tea.KeyUp}) {
		t.Fatalf("expected WantsKey=true for up in middle")
	}
	if !sf.WantsKey(tea.KeyMsg{Type: tea.KeyDown}) {
		t.Fatalf("expected WantsKey=true for down in middle")
	}

	// At first item, up should pass to form.
	sf.Index = 0
	if sf.WantsKey(tea.KeyMsg{Type: tea.KeyUp}) {
		t.Fatalf("expected WantsKey=false for up at first item")
	}
	if !sf.WantsKey(tea.KeyMsg{Type: tea.KeyDown}) {
		t.Fatalf("expected WantsKey=true for down at first item")
	}

	// At last item, down should pass to form.
	sf.Index = 2
	if !sf.WantsKey(tea.KeyMsg{Type: tea.KeyUp}) {
		t.Fatalf("expected WantsKey=true for up at last item")
	}
	if sf.WantsKey(tea.KeyMsg{Type: tea.KeyDown}) {
		t.Fatalf("expected WantsKey=false for down at last item")
	}
}

func TestSelectFieldOptionsView(t *testing.T) {
	sf, err := NewSelectField("two", []string{"one", "two", "three"})
	if err != nil {
		t.Fatalf("err when creating a new selectfield %v", err)
	}
	if opts := sf.OptionsView(); opts != "" {
		t.Fatalf("expected empty options when unfocused, got %q", opts)
	}

	sf.Focus()
	expected := strings.Join([]string{
		lipgloss.NewStyle().Foreground(ColBlue).Render("one"),
		lipgloss.NewStyle().Foreground(ColPink).Render("two"),
		lipgloss.NewStyle().Foreground(ColBlue).Render("three"),
	}, "\n")
	if opts := sf.OptionsView(); opts != expected {
		t.Fatalf("options view=%q want=%q", opts, expected)
	}
}

func TestSelectFieldReadOnly(t *testing.T) {
	sf, err := NewSelectField("one", []string{"one", "two"})
	if err != nil {
		t.Fatalf("err when creating a new selectfield %v", err)
	}
	sf.SetReadOnly(true)
	sf.Focus()

	if opts := sf.OptionsView(); opts != "" {
		t.Fatalf("read-only field should not focus")
	}

	sf.Update(tea.KeyMsg{Type: tea.KeyDown})
	if got, want := sf.Value(), "one"; got != want {
		t.Fatalf("value=%s want=%s", got, want)
	}
}
