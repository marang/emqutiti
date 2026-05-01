package emqutiti

import (
	list "github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/marang/emqutiti/constants"
	"github.com/marang/emqutiti/history"
)

// updateClientInputs updates form inputs, viewport and history list.
func (m *model) updateClientInputs(msg tea.Msg) []tea.Cmd {
	var cmds []tea.Cmd
	if cmd := m.topics.UpdateInput(msg); cmd != nil {
		cmds = append(cmds, cmd)
	}
	if mCmd := m.message.Update(msg); mCmd != nil {
		cmds = append(cmds, mCmd)
	}
	if vpCmd := m.updateViewport(msg); vpCmd != nil {
		cmds = append(cmds, vpCmd)
	}
	if m.FocusedID() == idHistory {
		oldScroll := -1.0
		animateScroll := isHistoryKeyboardScrollMsg(msg)
		if animateScroll {
			oldScroll = m.rawHistoryScrollPercent()
		}
		if histCmd := m.history.Update(msg); histCmd != nil {
			cmds = append(cmds, histCmd)
		}
		if animateScroll {
			if scrollCmd := m.startHistoryScrollAnimation(oldScroll, m.rawHistoryScrollPercent()); scrollCmd != nil {
				cmds = append(cmds, scrollCmd)
			}
		}
	}
	return cmds
}

// updateViewport updates the main viewport unless history handles the scroll.
func (m *model) updateViewport(msg tea.Msg) tea.Cmd {
	skipVP := false
	if m.FocusedID() == idTopic || m.FocusedID() == idMessage {
		if _, ok := msg.(tea.KeyMsg); ok {
			skipVP = true
		}
	}
	if m.FocusedID() == idHistory {
		switch mt := msg.(type) {
		case tea.KeyMsg:
			s := mt.String()
			if s == constants.KeyUp || s == constants.KeyDown || s == constants.KeyPgUp || s == constants.KeyPgDown || s == constants.KeyK || s == constants.KeyJ {
				skipVP = true
			}
		case tea.MouseMsg:
			if mt.Action == tea.MouseActionPress && (mt.Button == tea.MouseButtonWheelUp || mt.Button == tea.MouseButtonWheelDown) && m.history.CanScroll() {
				skipVP = true
			}
		}
	}
	if skipVP {
		return nil
	}
	var cmd tea.Cmd
	m.ui.viewport, cmd = m.ui.viewport.Update(msg)
	return cmd
}

// filterHistoryList refreshes history items based on the current filter state.
func (m *model) filterHistoryList() {
	if st := m.history.List().FilterState(); st == list.Filtering || st == list.FilterApplied {
		q := m.history.List().FilterInput.Value()
		hitems, litems := history.ApplyFilter(q, m.history.Store(), m.history.ShowArchived())
		m.history.SetItems(hitems)
		m.history.SetFilterQuery(q)
		m.history.List().SetItems(litems)
	} else if m.history.FilterQuery() != "" {
		hitems, litems := history.ApplyFilter(m.history.FilterQuery(), m.history.Store(), m.history.ShowArchived())
		m.history.SetItems(hitems)
		m.history.List().SetItems(litems)
	} else {
		items := make([]list.Item, len(m.history.Items()))
		for i, it := range m.history.Items() {
			items[i] = it
		}
		m.history.List().SetItems(items)
	}
}
