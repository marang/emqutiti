package history

import (
	"fmt"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

type sizedModel struct {
	stubModel
	width, height int
}

func (m *sizedModel) Width() int                  { return m.width }
func (m *sizedModel) Height() int                 { return m.height }
func (m *sizedModel) OverlayHelp(s string) string { return "header\n" + s }

var dialogSizes = []struct{ width, height int }{{120, 40}, {80, 24}, {40, 16}, {32, 12}}

func renderedPosition(view, text string) (int, int, bool) {
	for y, line := range strings.Split(ansi.Strip(view), "\n") {
		if x := strings.Index(line, text); x >= 0 {
			return lipgloss.Width(line[:x]), y, true
		}
	}
	return 0, 0, false
}

func filterComponent(width, height int, topics []string) *Component {
	h := NewComponent(&sizedModel{width: width, height: height}, &store{})
	f := NewFilterForm(topics, "", "", time.Time{}, time.Time{}, false)
	h.SetFilterForm(&f)
	return h
}

func TestFilterStartMouseGeometry(t *testing.T) {
	for _, size := range dialogSizes {
		t.Run(fmt.Sprintf("%dx%d", size.width, size.height), func(t *testing.T) {
			h := filterComponent(size.width, size.height, nil)
			view := h.ViewFilter()
			x, y, ok := renderedPosition(view, "Start:")
			if !ok {
				t.Fatal("Start field is not visible")
			}
			h.UpdateFilter(tea.MouseMsg{X: x + 7, Y: y, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft})
			if got := h.FilterForm().Focus; got != idxFilterStart {
				t.Fatalf("click on Start focused field %d, want %d", got, idxFilterStart)
			}
		})
	}
}

func TestFilterFitsAndFocusedFieldsStayReachable(t *testing.T) {
	labels := []string{"Topic:", "Text:", "Start:", "End:", "Archived:"}
	for _, size := range dialogSizes {
		t.Run(fmt.Sprintf("%dx%d", size.width, size.height), func(t *testing.T) {
			h := filterComponent(size.width, size.height, []string{strings.Repeat("topic/", 20)})
			for i, label := range labels {
				view := h.ViewFilter()
				if w, ht := lipgloss.Width(view), lipgloss.Height(view); w > size.width || ht > size.height {
					t.Fatalf("dialog %dx%d exceeds terminal %dx%d", w, ht, size.width, size.height)
				}
				x, y, ok := renderedPosition(view, label)
				if !ok {
					t.Fatalf("focused field %s is not visible", label)
				}
				h.UpdateFilter(tea.MouseMsg{X: x + 1, Y: y, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft})
				if got := h.FilterForm().Focus; got != i {
					t.Fatalf("click on %s focused %d, want %d", label, got, i)
				}
				h.UpdateFilter(tea.KeyMsg{Type: tea.KeyTab})
			}
		})
	}
}

func TestFilterSuggestionAndBlankMouseRows(t *testing.T) {
	h := filterComponent(80, 24, []string{"topic/a", "topic/b"})
	h.UpdateFilter(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("t")})
	view := h.ViewFilter()
	x, y, ok := renderedPosition(view, "topic/b")
	if !ok {
		t.Fatal("suggestion is not visible")
	}
	h.UpdateFilter(tea.MouseMsg{X: x + 1, Y: y, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft})
	if got := h.FilterForm().Topic().Value(); got != "topic/b" {
		t.Fatalf("clicked suggestion value %q, want topic/b", got)
	}
	view = h.ViewFilter()
	x, y, ok = renderedPosition(view, "Start:")
	if !ok {
		t.Fatal("Start is not visible below suggestions")
	}
	h.UpdateFilter(tea.MouseMsg{X: x + 7, Y: y, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft})
	if got := h.FilterForm().Focus; got != idxFilterStart {
		t.Fatalf("Start below suggestions focused %d", got)
	}
	h.FilterForm().Focus = idxFilterArchived
	h.FilterForm().ApplyFocus()
	view = h.ViewFilter()
	x, y, _ = renderedPosition(view, "Archived:")
	for _, point := range [][2]int{{x + 1, y - 1}, {0, 0}, {x - 1, y}} {
		h.UpdateFilter(tea.MouseMsg{X: point[0], Y: point[1], Action: tea.MouseActionPress, Button: tea.MouseButtonLeft})
		if h.FilterForm().Archived().Bool() || h.FilterForm().Focus != idxFilterArchived {
			t.Fatalf("blank/padding click at %v changed the archived field", point)
		}
	}
}

func TestFilterScrollMouseGeometry(t *testing.T) {
	for _, size := range dialogSizes {
		t.Run(fmt.Sprintf("%dx%d", size.width, size.height), func(t *testing.T) {
			h := filterComponent(size.width, size.height, []string{strings.Repeat("topic/", 30)})
			h.UpdateFilter(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("t")})
			for n := 0; n < 30; n++ {
				h.UpdateFilter(tea.KeyMsg{Type: tea.KeyPgDown})
			}
			view := h.ViewFilter()
			x, y, ok := renderedPosition(view, "Archived:")
			if !ok {
				t.Fatal("scrolling did not reach Archived")
			}
			h.UpdateFilter(tea.MouseMsg{X: x + 10, Y: y, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft})
			if h.FilterForm().Focus != idxFilterArchived || !h.FilterForm().Archived().Bool() {
				t.Fatal("scrolled Archived checkbox click missed its field")
			}
		})
	}
}
