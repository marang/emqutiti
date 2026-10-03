package topics

import (
	"fmt"
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/marang/emqutiti/ui"
)

type managerModel struct {
	mockModel
	width, height int
	focused       string
	prompt        string
	confirm       func() tea.Cmd
	clientMode    bool
}

func (m *managerModel) Width() int                  { return m.width }
func (m *managerModel) Height() int                 { return m.height }
func (m *managerModel) FocusedID() string           { return m.focused }
func (m *managerModel) SetFocus(id string) tea.Cmd  { m.focused = id; return nil }
func (m *managerModel) ShowClient() tea.Cmd         { m.clientMode = true; return nil }
func (m *managerModel) OverlayHelp(v string) string { return "header\n" + v }
func (m *managerModel) StartConfirm(prompt, _ string, _ func() tea.Cmd, action func() tea.Cmd, _ func()) {
	m.prompt, m.confirm = prompt, action
}

func managerComponent() (*Component, *managerModel) {
	m := &managerModel{width: 80, height: 24, focused: idTopicsSubscribed}
	c := New(m)
	c.Items = []Item{{Name: "a", Subscribed: true}, {Name: "target", Subscribed: true}, {Name: "u1"}, {Name: "u2"}}
	c.RebuildActiveTopicList()
	c.SetSize(m.width, m.height)
	return c, m
}

func TestManagerPaneClicksIgnoreClientChipBounds(t *testing.T) {
	c, m := managerComponent()
	c.ChipBounds = []ChipBound{{XPos: 0, YPos: 0, Width: 80, Height: 24, Index: 0}}
	rightX := c.list.Width() + 3
	c.Update(tea.MouseMsg{Action: tea.MouseActionPress, Button: tea.MouseButtonLeft, X: rightX, Y: 7})
	if c.panes.active != 1 || c.Selected() != 3 || m.focused != idTopicsUnsubscribed || c.list.SelectedItem().(Item).Name != "u2" {
		t.Fatalf("wrong pane click: pane=%d selected=%d focus=%s", c.panes.active, c.Selected(), m.focused)
	}
	c.Update(tea.MouseMsg{Action: tea.MouseActionPress, Button: tea.MouseButtonRight, X: 3, Y: 7})
	if c.panes.active != 0 || c.Selected() != 1 || !strings.Contains(m.prompt, "target") || m.confirm == nil {
		t.Fatalf("wrong right-click target: pane=%d selected=%d prompt=%s", c.panes.active, c.Selected(), m.prompt)
	}
	if len(c.Items) != 4 {
		t.Fatal("right click deleted without confirmation")
	}
	c.RemoveTopic(0)
	m.confirm()
	if c.HasTopic("target") || !c.HasTopic("u1") || !c.HasTopic("u2") {
		t.Fatalf("confirmation deleted the wrong topic: %v", c.Items)
	}
}

func TestManagerPaneEmptyClicksNoOp(t *testing.T) {
	c, m := managerComponent()
	for _, point := range [][2]int{{0, 4}, {3, 0}, {3, 1}, {3, 2}, {3, 3}, {3, 6}, {3, 15}, {c.list.Width() + 1, 4}} {
		c.Update(tea.MouseMsg{Action: tea.MouseActionPress, Button: tea.MouseButtonRight, X: point[0], Y: point[1]})
		if m.confirm != nil || c.panes.active != 0 {
			t.Fatalf("non-row %v triggered action", point)
		}
	}
}

func TestManagerFilteredActionsUseVisibleTopic(t *testing.T) {
	c, m := managerComponent()
	c.list.SetFilterText("target")
	c.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})
	if c.Items[0].Publish || !c.Items[1].Publish || c.Selected() != 1 || c.list.SelectedItem().(Item).Name != "target" {
		t.Fatalf("publish used wrong filtered index: %v", c.Items)
	}
	c.Update(tea.KeyMsg{Type: tea.KeyDelete})
	if !strings.Contains(m.prompt, "target") {
		t.Fatalf("delete used wrong filtered index: %q", m.prompt)
	}
	m.confirm()
	if c.HasTopic("target") || !c.HasTopic("a") {
		t.Fatalf("deleted wrong filtered topic: %v", c.Items)
	}
}

func TestManagerFilterOwnsLettersEnterDeleteAndNavigation(t *testing.T) {
	c, m := managerComponent()
	c.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	if c.list.FilterState() != list.Filtering {
		t.Fatal("slash did not enter filter")
	}
	c.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})
	if c.list.FilterValue() != "p" || c.Items[0].Publish || c.Items[1].Publish {
		t.Fatal("typing p toggled publish instead of editing search")
	}
	c.list.SetFilterText("target")
	c.list.SetFilterState(list.Filtering)
	c.Update(tea.KeyMsg{Type: tea.KeyRight})
	c.Update(tea.KeyMsg{Type: tea.KeyDelete})
	c.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if c.panes.active != 0 || !c.Items[1].Subscribed || m.confirm != nil || c.list.FilterState() != list.FilterApplied {
		t.Fatal("filter keys triggered topic actions")
	}
	c.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if m.clientMode || c.list.FilterState() != list.Unfiltered {
		t.Fatal("Esc left manager instead of clearing its filter")
	}
}

func TestManagerPaginatedAndFilteredPaneClicks(t *testing.T) {
	c, _ := managerComponent()
	c.Items = nil
	for i := 0; i < 30; i++ {
		c.Items = append(c.Items, Item{Name: fmt.Sprintf("topic-%02d", i), Subscribed: i%2 == 0})
	}
	c.RebuildActiveTopicList()
	c.SetSize(80, 24)
	c.list.Select(c.list.Paginator.PerPage)
	want := c.list.SelectedItem().(Item).Name
	c.Update(tea.MouseMsg{Action: tea.MouseActionPress, Button: tea.MouseButtonLeft, X: 3, Y: 4})
	if c.Items[c.Selected()].Name != want {
		t.Fatalf("page click selected %q, want %q", c.Items[c.Selected()].Name, want)
	}
	c.list.SetFilterText("topic-28")
	c.Update(tea.MouseMsg{Action: tea.MouseActionPress, Button: tea.MouseButtonLeft, X: 3, Y: 4})
	if c.Items[c.Selected()].Name != "topic-28" {
		t.Fatalf("filtered click selected %q", c.Items[c.Selected()].Name)
	}
	c.SetActivePane(1)
	c.list.Select(c.list.Paginator.PerPage)
	c.syncSelection()
	c.SetActivePane(0)
	other := c.paneList(1)
	want = other.VisibleItems()[other.Paginator.Page*other.Paginator.PerPage].(Item).Name
	c.Update(tea.MouseMsg{Action: tea.MouseActionPress, Button: tea.MouseButtonLeft, X: c.list.Width() + 3, Y: 4})
	if c.Items[c.Selected()].Name != want {
		t.Fatalf("inactive page click selected %q, want %q", c.Items[c.Selected()].Name, want)
	}
}

func TestTopicManagerFitsTerminal(t *testing.T) {
	for _, size := range [][2]int{{120, 40}, {80, 24}, {60, 20}, {40, 16}} {
		t.Run(fmt.Sprint(size), func(t *testing.T) {
			c, m := managerComponent()
			m.width, m.height = size[0], size[1]
			for i := 0; i < 30; i++ {
				c.Items = append(c.Items, Item{Name: strings.Repeat("long", 20) + fmt.Sprint(i), Subscribed: i%2 == 0})
			}
			c.RebuildActiveTopicList()
			for _, filtered := range []bool{false, true} {
				if filtered {
					c.list.SetFilterText("long")
					c.list.SetFilterState(list.Filtering)
				}
				view := c.View()
				if w, h := lipgloss.Size(view); w > size[0] || h > size[1] {
					t.Fatalf("view %dx%d exceeds %v:\n%s", w, h, size, view)
				}
				for _, action := range []string{"[space] toggle", "[p] publish", "[del] delete", "[/] filter", "[esc] back"} {
					if !strings.Contains(view, action) {
						t.Fatalf("missing action %q", action)
					}
				}
				if y, _, visible := ui.ListItemBounds(c.list, c.list.Index(), 2, 1); !visible || y < 0 {
					t.Fatal("selected topic not visible")
				}
			}
		})
	}
}
