package emqutiti

import (
	"fmt"

	"github.com/charmbracelet/x/ansi"
	"github.com/marang/emqutiti/ui"
)

// renderHistorySection renders the history list box.
func (m *model) renderHistorySection() string {
	histSP := m.animatedHistoryScrollPercent(m.rawHistoryScrollPercent())

	total := len(m.history.Items())
	if st := m.history.Store(); st != nil {
		total = st.Count(m.history.ShowArchived())
	}
	shown := len(m.history.Items())
	histLabel := fmt.Sprintf("History (%d messages \u2013 Ctrl+C copy)", total)
	if m.history.FilterQuery() != "" && shown != total {
		histLabel = fmt.Sprintf("History (%d/%d messages \u2013 Ctrl+C copy)", shown, total)
	}
	if marker := m.historyPulseMarker(); marker != " " {
		histLabel = marker + " " + histLabel
	}
	listHeight := m.layout.history.height
	if m.history.FilterQuery() != "" && listHeight > 0 {
		listHeight--
	}
	m.history.List().SetSize(m.ui.width-4, listHeight)
	histContent := m.history.List().View()
	if m.history.FilterQuery() != "" {
		inner := m.ui.width - 4
		filterLine := fmt.Sprintf("Filters: %s", m.history.FilterQuery())
		filterLine = ansi.Truncate(filterLine, inner, "")
		histContent = fmt.Sprintf("%s\n%s", filterLine, histContent)
	}
	historyFocused := m.ui.focusOrder[m.ui.focusIndex] == idHistory
	historyHovered := m.ui.hoveredID == idHistory
	return ui.LegendBoxWithState(histContent, histLabel, m.ui.width-2, m.layout.history.height, ui.ColGreen, ui.BoxState{Focused: historyFocused, Hovered: historyHovered}, histSP)
}

func (m *model) rawHistoryScrollPercent() float64 {
	per := m.history.List().Paginator.PerPage
	totalItems := len(m.history.List().Items())
	if totalItems <= per {
		return -1
	}
	start := m.history.List().Paginator.Page * per
	denom := totalItems - per
	if denom <= 0 {
		return -1
	}
	return float64(start) / float64(denom)
}
