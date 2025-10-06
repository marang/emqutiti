package ui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// Test that plain j/k are consumed by TextField and do not move focus.
func TestTextFieldConsumesJK(t *testing.T) {
	tf1 := NewTextField("", "")
	tf2 := NewTextField("", "")
	f := Form{Fields: []Field{tf1, tf2}, Focus: 0}
	f.ApplyFocus()

	for _, tt := range []struct{ r rune }{{'j'}, {'k'}} {
		key := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{tt.r}}
		f.CycleFocus(key)
		f.ApplyFocus()
		f.Fields[f.Focus].Update(key)
		if f.Focus != 0 {
			t.Fatalf("focus moved on %q", string(tt.r))
		}
		if tf1.Value() != string(tt.r) {
			t.Fatalf("field value=%q want=%q", tf1.Value(), string(tt.r))
		}
		tf1.SetValue("")
	}
}

// Test that SuggestField also consumes j/k without losing focus.
func TestSuggestFieldConsumesJK(t *testing.T) {
	sf := NewSuggestField([]string{"foo"}, "topic")
	tf := NewTextField("", "")
	f := Form{Fields: []Field{sf, tf}, Focus: 0}
	f.ApplyFocus()

	key := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
	f.CycleFocus(key)
	f.ApplyFocus()
	f.Fields[f.Focus].Update(key)
	if f.Focus != 0 {
		t.Fatalf("focus moved on j")
	}
	if sf.Value() != "j" {
		t.Fatalf("field value=%q want=j", sf.Value())
	}
}

func TestRFC3339FieldSanitisesInput(t *testing.T) {
	tf := NewTextField("", "", WithRFC3339())
	tf.Focus()

	inputs := []tea.KeyMsg{
		{Type: tea.KeyRunes, Runes: []rune("2025-12-")},
		{Type: tea.KeyRunes, Runes: []rune("22t14:00:00")},
		{Type: tea.KeyRunes, Runes: []rune(".12z")},
		{Type: tea.KeyRunes, Runes: []rune("oops")},
	}
	for _, msg := range inputs {
		tf.Update(msg)
	}

	if got, want := tf.Value(), "2025-12-22T14:00:00.12Z"; got != want {
		t.Fatalf("value=%q want=%q", got, want)
	}

	if err := tf.Err(); err != nil {
		t.Fatalf("unexpected validation error: %v", err)
	}
}

func TestRFC3339FieldSetValueFiltersInvalidRunes(t *testing.T) {
	tf := NewTextField("2025-06zz", "", WithRFC3339())
	if got, want := tf.Value(), "2025-06"; got != want {
		t.Fatalf("value=%q want=%q", got, want)
	}
}
