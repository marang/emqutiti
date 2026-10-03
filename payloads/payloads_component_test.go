package payloads

import (
	"fmt"
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type testModel struct {
	clientMode    bool
	width, height int
	focused       string
	prompt        string
	confirm       func() tea.Cmd
	cancel        func()
}

func (t *testModel) SetClientMode() tea.Cmd        { t.clientMode = true; return nil }
func (t *testModel) FocusedID() string             { return t.focused }
func (t *testModel) SetFocus(id string) tea.Cmd    { t.focused = id; return nil }
func (t *testModel) ResetElemPos()                 {}
func (t *testModel) SetElemPos(id string, pos int) {}
func (t *testModel) OverlayHelp(s string) string   { return "header\n" + s }
func (t *testModel) Width() int                    { return t.width }
func (t *testModel) Height() int                   { return t.height }
func (t *testModel) StartConfirm(prompt, _ string, _ func() tea.Cmd, action func() tea.Cmd, cancel func()) {
	t.prompt, t.confirm, t.cancel = prompt, action, cancel
}

type testStatus struct{}

func (testStatus) ListenStatus() tea.Cmd { return nil }

func TestEscReturnsClientMode(t *testing.T) {
	m := &testModel{}
	p := New(m, testStatus{})
	p.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if !m.clientMode {
		t.Fatalf("expected client mode")
	}
}

func payloadTestComponent() (*Component, *testModel) {
	m := &testModel{width: 80, height: 24, focused: IDList}
	p := New(m, testStatus{})
	p.SetItems([]Item{{Topic: "first", Payload: "one"}, {Topic: "target", Payload: "two"}, {Topic: "last", Payload: "three"}})
	p.SetSize(m.width, m.height)
	return p, m
}

func TestPayloadClicksOnlyHitRows(t *testing.T) {
	for _, button := range []tea.MouseButton{tea.MouseButtonLeft, tea.MouseButtonRight} {
		for _, point := range [][2]int{{0, 4}, {1, 0}, {1, 1}, {1, 2}, {1, 3}, {1, 6}, {1, 15}, {78, 4}, {1, 22}} {
			t.Run(fmt.Sprintf("%d/%v", button, point), func(t *testing.T) {
				p, m := payloadTestComponent()
				p.Update(tea.MouseMsg{Action: tea.MouseActionPress, Button: button, X: point[0], Y: point[1]})
				if m.clientMode || m.confirm != nil || len(p.Items()) != 3 {
					t.Fatalf("non-row click triggered an action at %v", point)
				}
			})
		}
	}
}

func TestPayloadClickLoadsClickedRow(t *testing.T) {
	p, m := payloadTestComponent()
	p.Update(tea.MouseMsg{Action: tea.MouseActionPress, Button: tea.MouseButtonLeft, X: 3, Y: 7})
	if !m.clientMode || p.list.SelectedItem().(Item).Topic != "target" {
		t.Fatalf("click did not load second row: selected %v", p.list.SelectedItem())
	}
}

func TestFilteredPayloadDeleteConfirmAndStableIdentity(t *testing.T) {
	p, m := payloadTestComponent()
	p.list.SetFilterText("target")
	p.Update(tea.KeyMsg{Type: tea.KeyDelete})
	if m.confirm == nil || !strings.Contains(m.prompt, "target") || len(p.Items()) != 3 {
		t.Fatal("filtered delete did not request confirmation for target")
	}
	confirm := m.confirm
	p.list.ResetFilter()
	p.list.Select(0)
	p.Update(tea.KeyMsg{Type: tea.KeyDelete})
	m.confirm()
	confirm()
	if len(p.Items()) != 1 || p.Items()[0].Topic != "last" {
		t.Fatalf("pending confirmation removed wrong payload: %v", p.Items())
	}
}

func TestPayloadDeleteCancelAndReplacement(t *testing.T) {
	p, m := payloadTestComponent()
	p.Update(tea.MouseMsg{Action: tea.MouseActionPress, Button: tea.MouseButtonRight, X: 3, Y: 7})
	if m.confirm == nil || !strings.Contains(m.prompt, "target") {
		t.Fatal("right click did not confirm clicked row")
	}
	if m.cancel != nil {
		m.cancel()
	}
	if len(p.Items()) != 3 {
		t.Fatal("cancel removed a payload")
	}
	pending := m.confirm
	p.SetItems([]Item{{Topic: "replacement", Payload: "new"}})
	pending()
	if len(p.Items()) != 1 || p.Items()[0].Topic != "replacement" {
		t.Fatal("stale confirmation removed a replacement payload")
	}
}

func TestPayloadFilterOwnsActionKeys(t *testing.T) {
	p, m := payloadTestComponent()
	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	p.list.SetFilterText("target")
	p.list.SetFilterState(list.Filtering)
	p.Update(tea.KeyMsg{Type: tea.KeyDelete})
	p.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if m.clientMode || m.confirm != nil || p.list.FilterState() != list.FilterApplied {
		t.Fatal("filter Enter/Delete triggered a payload action")
	}
	p.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if m.clientMode || p.list.FilterState() != list.Unfiltered {
		t.Fatal("Esc left manager instead of clearing its filter")
	}
}

func TestPayloadFilteredLoadUsesVisibleItem(t *testing.T) {
	p, _ := payloadTestComponent()
	p.list.SetFilterText("target")
	cmd := p.Update(tea.KeyMsg{Type: tea.KeyEnter})
	var messages []tea.Msg
	var run func(tea.Cmd)
	run = func(cmd tea.Cmd) {
		if cmd == nil {
			return
		}
		msg := cmd()
		if batch, ok := msg.(tea.BatchMsg); ok {
			for _, child := range batch {
				run(child)
			}
		} else {
			messages = append(messages, msg)
		}
	}
	run(cmd)
	if len(messages) != 1 || messages[0] != (LoadMsg{Topic: "target", Payload: "two"}) {
		t.Fatalf("loaded wrong filtered payload: %v", messages)
	}
}

func TestPayloadManagerFitsTerminal(t *testing.T) {
	for _, size := range [][2]int{{120, 40}, {80, 24}, {60, 20}, {40, 16}} {
		t.Run(fmt.Sprint(size), func(t *testing.T) {
			p, m := payloadTestComponent()
			m.width, m.height = size[0], size[1]
			for i := 0; i < 20; i++ {
				p.Add(strings.Repeat("long", 30), strings.Repeat("payload", 30))
			}
			for _, filtered := range []bool{false, true} {
				if filtered {
					p.list.SetFilterText("long")
					p.list.SetFilterState(list.Filtering)
				}
				view := p.View()
				if w, h := lipgloss.Size(view); w > size[0] || h > size[1] {
					t.Fatalf("view %dx%d exceeds %v:\n%s", w, h, size, view)
				}
				for _, action := range []string{"[enter] load", "[del] delete", "[/] filter", "[esc] back"} {
					if !strings.Contains(view, action) {
						t.Fatalf("missing action %q", action)
					}
				}
			}
		})
	}
}
