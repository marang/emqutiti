package topics

import (
	"strings"
	"unicode"

	"github.com/charmbracelet/lipgloss"
)

// LayoutChips lays out chips horizontally wrapping within width.
func LayoutChips(chips []string, width int) ([]string, []ChipBound) {
	var lines []string
	var row []string
	var bounds []ChipBound
	curX := 0
	rowTop := 0
	rowHeight := 0
	for i, c := range chips {
		cw := lipgloss.Width(c)
		ch := lipgloss.Height(c)
		if curX+cw > width && len(row) > 0 {
			line := lipgloss.JoinHorizontal(lipgloss.Top, row...)
			line = strings.TrimRightFunc(line, unicode.IsSpace)
			lines = append(lines, line)
			row = []string{}
			curX = 0
			rowTop += rowHeight
			rowHeight = 0
		}
		row = append(row, c)
		bounds = append(bounds, ChipBound{XPos: curX, YPos: rowTop, Width: cw, Height: ch, Index: i})
		curX += cw
		if ch > rowHeight {
			rowHeight = ch
		}
	}
	if len(row) > 0 {
		line := lipgloss.JoinHorizontal(lipgloss.Top, row...)
		line = strings.TrimRightFunc(line, unicode.IsSpace)
		lines = append(lines, line)
	}
	if len(lines) == 0 {
		lines = []string{""}
	}
	// ensure at least one bound for layout calculations
	return lines, bounds
}
