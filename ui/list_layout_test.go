package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/lipgloss"
)

type layoutItem string

func (i layoutItem) FilterValue() string { return string(i) }
func (i layoutItem) Title() string       { return string(i) }
func (i layoutItem) Description() string { return "description" }

func TestListRowGeometry(t *testing.T) {
	items := []list.Item{layoutItem("a"), layoutItem("b"), layoutItem("c"), layoutItem("d"), layoutItem("e")}
	l := list.New(items, list.NewDefaultDelegate(), 36, 12)
	SizeManagerList(&l, 36, 12)
	l.Select(l.Paginator.PerPage)
	start := l.Paginator.Page * l.Paginator.PerPage
	if got := ListRowAt(l, 0, 2, 2, 1); got != start {
		t.Fatalf("page row maps to %d, want %d", got, start)
	}
	for _, point := range [][2]int{{0, 0}, {0, 1}, {0, 4}, {-1, 2}, {36, 2}, {0, 11}} {
		if got := ListRowAt(l, point[0], point[1], 2, 1); got != -1 {
			t.Fatalf("non-row %v maps to %d", point, got)
		}
	}
	y, height, visible := ListItemBounds(l, start, 2, 1)
	if y != 2 || height != 2 || !visible {
		t.Fatalf("bounds got (%d,%d,%v)", y, height, visible)
	}
	if _, _, visible := ListItemBounds(l, 0, 2, 1); visible {
		t.Fatal("off-page item marked visible")
	}
}

func TestHistoryListItemBoundsWithoutChrome(t *testing.T) {
	d := list.NewDefaultDelegate()
	d.SetSpacing(0)
	l := list.New([]list.Item{layoutItem("one"), layoutItem("two")}, d, 40, 10)
	l.SetShowTitle(false)
	l.SetShowFilter(false)
	l.SetShowStatusBar(false)
	l.SetShowPagination(false)
	y, height, visible := ListItemBounds(l, 1, 2, 0)
	if y != 2 || height != 2 || !visible {
		t.Fatalf("history bounds got (%d,%d,%v)", y, height, visible)
	}
}

func TestListFooterWrapsWholeActions(t *testing.T) {
	footer := ListFooter(36, "[enter] start/stop", "[del] delete", "[/] filter", "[esc] back")
	if lipgloss.Width(footer) > 36 || !strings.Contains(footer, "[enter] start/stop") || !strings.Contains(footer, "[esc] back") {
		t.Fatalf("invalid footer: %s", footer)
	}
}
