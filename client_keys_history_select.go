package emqutiti

import tea "github.com/charmbracelet/bubbletea"

// handleSpaceKey toggles selection in history.
func (m *model) handleSpaceKey() tea.Cmd {
	if m.ui.focusOrder[m.ui.focusIndex] == idHistory && !m.history.ShowArchived() {
		idx := m.history.List().Index()
		hitems := m.history.Items()
		if idx >= 0 && idx < len(hitems) {
			if hitems[idx].IsSelected != nil && *hitems[idx].IsSelected {
				hitems[idx].IsSelected = nil
			} else {
				v := true
				hitems[idx].IsSelected = &v
			}
			m.history.SetItems(hitems)
			m.history.SetSelectionAnchor(idx)
		}
	}
	return nil
}

// handleShiftUpKey extends selection upward in history.
func (m *model) handleShiftUpKey() tea.Cmd {
	if m.ui.focusOrder[m.ui.focusIndex] == idHistory && !m.history.ShowArchived() {
		oldScroll := m.rawHistoryScrollPercent()
		if m.history.SelectionAnchor() == -1 {
			m.history.SetSelectionAnchor(m.history.List().Index())
			if a := m.history.SelectionAnchor(); a >= 0 && a < len(m.history.Items()) {
				hitems := m.history.Items()
				v := true
				hitems[a].IsSelected = &v
				m.history.SetItems(hitems)
			}
		}
		if m.history.List().Index() > 0 {
			m.history.List().CursorUp()
			idx := m.history.List().Index()
			m.history.UpdateSelectionRange(idx)
		}
		return m.startHistoryScrollAnimation(oldScroll, m.rawHistoryScrollPercent())
	}
	return nil
}

// handleShiftDownKey extends selection downward in history.
func (m *model) handleShiftDownKey() tea.Cmd {
	if m.ui.focusOrder[m.ui.focusIndex] == idHistory && !m.history.ShowArchived() {
		oldScroll := m.rawHistoryScrollPercent()
		if m.history.SelectionAnchor() == -1 {
			m.history.SetSelectionAnchor(m.history.List().Index())
			if a := m.history.SelectionAnchor(); a >= 0 && a < len(m.history.Items()) {
				hitems := m.history.Items()
				v := true
				hitems[a].IsSelected = &v
				m.history.SetItems(hitems)
			}
		}
		if m.history.List().Index() < len(m.history.List().Items())-1 {
			m.history.List().CursorDown()
			idx := m.history.List().Index()
			m.history.UpdateSelectionRange(idx)
		}
		return m.startHistoryScrollAnimation(oldScroll, m.rawHistoryScrollPercent())
	}
	return nil
}

// handleSelectAllKey selects or clears all history items.
func (m *model) handleSelectAllKey() tea.Cmd {
	if m.ui.focusOrder[m.ui.focusIndex] == idHistory && !m.history.ShowArchived() {
		hitems := m.history.Items()
		allSelected := true
		for i := range hitems {
			if hitems[i].IsSelected == nil || !*hitems[i].IsSelected {
				allSelected = false
				break
			}
		}
		if allSelected {
			for i := range hitems {
				hitems[i].IsSelected = nil
			}
			m.history.SetSelectionAnchor(-1)
		} else {
			for i := range hitems {
				v := true
				hitems[i].IsSelected = &v
			}
			if len(hitems) > 0 {
				m.history.SetSelectionAnchor(0)
			}
		}
		m.history.SetItems(hitems)
	}
	return nil
}

// handleHistoryScroll handles history scroll keys.
func (m *model) handleHistoryScroll(_ string) tea.Cmd {
	// keep current selection and anchor
	return nil
}
