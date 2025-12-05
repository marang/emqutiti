package emqutiti

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/marang/emqutiti/constants"
	"github.com/marang/emqutiti/history"
)

// copyHistoryItems writes the provided history items to the clipboard and logs
// the result.
func (m *model) copyHistoryItems(items []history.Item) (int, error) {
	var parts []string
	for _, hi := range items {
		text := hi.Payload
		if hi.Kind != "log" {
			text = fmt.Sprintf("%s: %s", hi.Topic, hi.Payload)
		}
		parts = append(parts, text)
	}
	if len(parts) == 0 {
		return 0, nil
	}
	if err := clipboard.WriteAll(strings.Join(parts, "\n")); err != nil {
		m.history.Append("", err.Error(), "log", false, err.Error())
		return 0, err
	}
	msg := "Copied item"
	if len(parts) > 1 {
		msg = fmt.Sprintf("Copied %d item(s)", len(parts))
	}
	m.history.Append("", msg, "log", false, msg)
	return len(parts), nil
}

// handleCopyKey copies selected or current history items to the clipboard.
func (m *model) handleCopyKey() tea.Cmd {
	if m.ui.focusOrder[m.ui.focusIndex] != idHistory {
		return nil
	}
	var selected []history.Item
	hitems := m.history.Items()
	for _, it := range hitems {
		if it.IsSelected != nil && *it.IsSelected {
			selected = append(selected, it)
		}
	}
	switch {
	case len(selected) > 0:
		m.copyHistoryItems(selected)
	case len(hitems) > 0:
		idx := m.history.List().Index()
		if idx >= 0 && idx < len(hitems) {
			m.copyHistoryItems([]history.Item{hitems[idx]})
		}
	}
	return nil
}

// handleHistoryFilterKey opens the history filter when focused on history.
func (m *model) handleHistoryFilterKey() tea.Cmd {
	if m.ui.focusOrder[m.ui.focusIndex] == idHistory {
		return m.startHistoryFilter()
	}
	return nil
}

// handleClearFilterKey clears any active history filters.
func (m *model) handleClearFilterKey() tea.Cmd {
	if m.ui.focusOrder[m.ui.focusIndex] != idHistory {
		return nil
	}
	m.history.SetFilterQuery("")
	m.history.List().FilterInput.SetValue("")
	m.history.List().SetFilterState(list.Unfiltered)
	var msgs []history.Message
	if m.history.ShowArchived() {
		msgs = m.history.Store().Search(true, nil, time.Time{}, time.Time{}, "")
	} else {
		msgs = m.history.Store().Search(false, nil, time.Time{}, time.Time{}, "")
	}
	hitems, items := history.MessagesToItems(msgs)
	m.history.SetItems(hitems)
	m.history.List().SetItems(items)
	return nil
}

// handleHistoryViewKey opens a detail view for long history payloads.
func (m *model) handleHistoryViewKey() tea.Cmd {
	if m.ui.focusOrder[m.ui.focusIndex] != idHistory {
		return nil
	}
	idx := m.history.List().Index()
	if idx < 0 || idx >= len(m.history.List().Items()) {
		return nil
	}
	hi := m.history.List().Items()[idx].(history.Item)
	if utf8.RuneCountInString(hi.Payload) <= historyPreviewLimit {
		return nil
	}
	m.history.SetDetailItem(hi)
	m.history.Detail().SetContent(history.FormatDetailPayload(hi.Payload))
	m.history.Detail().SetYOffset(0)
	return m.SetMode(constants.ModeHistoryDetail)
}
