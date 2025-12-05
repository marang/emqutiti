package emqutiti

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/marang/emqutiti/confirm"
	"github.com/marang/emqutiti/constants"
	"github.com/marang/emqutiti/focus"
	"github.com/marang/emqutiti/history"
)

// FocusedID returns the identifier of the currently focused element.
func (m *model) FocusedID() string { return m.ui.focusOrder[m.ui.focusIndex] }

// ListenStatus proxies connection status updates for components.
func (m *model) ListenStatus() tea.Cmd { return m.connections.ListenStatus() }

// ResetElemPos clears cached element positions.
func (m *model) ResetElemPos() { m.ui.elemPos = map[string]int{} }

// SetElemPos records the position of a UI element.
func (m *model) SetElemPos(id string, pos int) { m.ui.elemPos[id] = pos }

// StartConfirm displays a confirmation dialog and runs the action on accept.
func (m *model) StartConfirm(prompt, info string, returnFocus func() tea.Cmd, action func() tea.Cmd, cancel func()) {
	m.confirm = confirm.NewDialog(m, m, returnFocus, action, cancel)
	m.confirm.Start(prompt, info)
	m.components[constants.ModeConfirmDelete] = m.confirm
}

// startHistoryFilter opens the history filter form.
func (m *model) startHistoryFilter() tea.Cmd {
	var topics []string
	for _, t := range m.topics.Items {
		topics = append(topics, t.Name)
	}
	var topic, payload string
	var start, end time.Time
	if m.history.FilterQuery() != "" {
		ts, s, e, p := history.ParseQuery(m.history.FilterQuery())
		if len(ts) > 0 {
			topic = ts[0]
		}
		start, end, payload = s, e, p
	} else {
		end = time.Now()
		start = end.Add(-time.Hour)
	}
	hf := history.NewFilterForm(topics, topic, payload, start, end, m.history.ShowArchived())
	m.history.SetFilterForm(&hf)
	return m.SetMode(constants.ModeHistoryFilter)
}

// SetMode updates the current mode and focus order.
func (m *model) SetMode(mode constants.AppMode) tea.Cmd {
	prevMode := m.CurrentMode()
	if m.focus != nil && len(m.ui.focusOrder) > m.ui.focusIndex {
		if f, ok := m.focusables[m.ui.focusOrder[m.ui.focusIndex]]; ok {
			f.Blur()
		}
	}
	if m.ui.focusMemory == nil {
		m.ui.focusMemory = map[constants.AppMode]int{}
	}
	if len(m.ui.focusOrder) > 0 {
		m.ui.focusMemory[prevMode] = m.ui.focusIndex
	}
	// push mode to stack
	if len(m.ui.modeStack) == 0 || m.ui.modeStack[0] != mode {
		m.ui.modeStack = append([]constants.AppMode{mode}, m.ui.modeStack...)
	} else {
		m.ui.modeStack[0] = mode
	}
	// remove any other occurrences of this mode to keep order meaningful
	for i := 1; i < len(m.ui.modeStack); {
		if m.ui.modeStack[i] == mode {
			m.ui.modeStack = append(m.ui.modeStack[:i], m.ui.modeStack[i+1:]...)
		} else {
			i++
		}
	}
	// prune help mode from history to avoid navigation dead-ends
	if mode != constants.ModeHelp {
		for i := 1; i < len(m.ui.modeStack); {
			if m.ui.modeStack[i] == constants.ModeHelp {
				m.ui.modeStack = append(m.ui.modeStack[:i], m.ui.modeStack[i+1:]...)
			} else {
				i++
			}
		}
	}
	if len(m.ui.modeStack) > 10 {
		m.ui.modeStack = m.ui.modeStack[:10]
	}
	order, ok := focusByMode[mode]
	if !ok || len(order) == 0 {
		order = []string{idHelp}
	}
	m.ui.focusOrder = append([]string(nil), order...)
	m.ui.focusMap = make(map[string]int, len(order))
	items := make([]focus.Focusable, len(order))
	for i, id := range order {
		m.ui.focusMap[id] = i
		if f := m.focusables[id]; f != nil {
			items[i] = f
		} else {
			items[i] = &focus.NullFocusable{}
		}
	}
	m.focus = focus.NewFocusMap(items)
	if idx, ok := m.ui.focusMemory[mode]; ok && idx >= 0 && idx < len(items) {
		m.focus.Set(idx)
	}
	m.ui.focusIndex = m.focus.Index()
	m.help.Blur()
	return nil
}

// CurrentMode returns the active application mode.
func (m *model) CurrentMode() constants.AppMode {
	if len(m.ui.modeStack) == 0 {
		return constants.ModeClient
	}
	return m.ui.modeStack[0]
}

// PreviousMode returns the last mode before the current one.
func (m *model) PreviousMode() constants.AppMode {
	if len(m.ui.modeStack) > 1 {
		return m.ui.modeStack[1]
	}
	return m.CurrentMode()
}

// SetConfirmMode switches to the confirmation screen.
func (m *model) SetConfirmMode() tea.Cmd { return m.SetMode(constants.ModeConfirmDelete) }

// SetPreviousMode returns to the prior screen.
func (m *model) SetPreviousMode() tea.Cmd { return m.SetMode(m.PreviousMode()) }

// Width returns the current UI width.
func (m *model) Width() int { return m.ui.width }

// MessageHeight returns the configured message box height.
func (m *model) MessageHeight() int { return m.layout.message.height }

// Height returns the current UI height.
func (m *model) Height() int { return m.ui.height }

// SetClientMode switches to the main client screen.
func (m *model) SetClientMode() tea.Cmd { return m.SetMode(constants.ModeClient) }
