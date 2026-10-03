package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/lipgloss"
)

// SizeManagerList sizes a default-delegate list with compact, single-row chrome.
// Managers render their action footer separately, so Bubbles' extra help is hidden.
func SizeManagerList(l *list.Model, width, height int) {
	width, height = max(1, width), max(1, height)
	l.SetShowTitle(false)
	l.SetShowHelp(false)
	l.Styles.TitleBar = l.Styles.TitleBar.PaddingBottom(0).MaxWidth(width).MaxHeight(1)
	l.Styles.StatusBar = l.Styles.StatusBar.PaddingBottom(0).MaxWidth(width).MaxHeight(1)
	l.SetSize(width, height)
	l.FilterInput.Width = max(1, width-lipgloss.Width(l.FilterInput.Prompt)-l.Styles.TitleBar.GetHorizontalFrameSize()-1)
}

// ListRowAt returns a visible-list index only for an actual default-delegate row.
// Coordinates are relative to the list, excluding its surrounding box/header.
func ListRowAt(l list.Model, x, y, itemHeight, spacing int) int {
	if x < 0 || x >= l.Width() || y < 0 || y >= l.Height() || itemHeight < 1 || spacing < 0 {
		return -1
	}
	row := y - listContentTop(l)
	stride := itemHeight + spacing
	if row < 0 || row%stride >= itemHeight || row/stride >= l.Paginator.PerPage {
		return -1
	}
	index := l.Paginator.Page*l.Paginator.PerPage + row/stride
	if index >= len(l.VisibleItems()) {
		return -1
	}
	return index
}

// ListItemBounds returns a visible item's y/height relative to the list origin.
// The index addresses VisibleItems, not the underlying unfiltered slice.
func ListItemBounds(l list.Model, index, itemHeight, spacing int) (y, height int, visible bool) {
	if itemHeight < 1 || spacing < 0 || index < 0 || index >= len(l.VisibleItems()) || l.Paginator.PerPage < 1 {
		return 0, 0, false
	}
	start := l.Paginator.Page * l.Paginator.PerPage
	if index < start || index >= start+l.Paginator.PerPage {
		return 0, 0, false
	}
	y = listContentTop(l) + (index-start)*(itemHeight+spacing)
	return y, itemHeight, y+itemHeight <= l.Height()
}

func listContentTop(l list.Model) int {
	top := 0
	if l.ShowTitle() || (l.ShowFilter() && l.FilteringEnabled()) {
		top++
		if l.ShowTitle() || l.FilterState() == list.Filtering {
			top += l.Styles.TitleBar.GetVerticalFrameSize()
		}
	}
	if l.ShowStatusBar() {
		top += 1 + l.Styles.StatusBar.GetVerticalFrameSize()
	}
	return top
}

// ListFooter wraps whole action labels without separating a key from its action.
func ListFooter(width int, actions ...string) string {
	width = max(1, width-InfoStyle.GetHorizontalFrameSize())
	var rows []string
	line := ""
	for _, action := range actions {
		if line != "" && lipgloss.Width(line)+2+lipgloss.Width(action) > width {
			rows = append(rows, line)
			line = ""
		}
		if line != "" {
			line += "  "
		}
		line += action
	}
	rows = append(rows, line)
	return InfoStyle.Render(strings.Join(rows, "\n"))
}
