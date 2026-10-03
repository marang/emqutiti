package emqutiti

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/marang/emqutiti/constants"
	"github.com/marang/emqutiti/ui"
)

// SetFocus moves focus to the given element id.
func (m *model) SetFocus(id string) tea.Cmd {
	if i, ok := m.ui.focusMap[id]; ok {
		m.focus.Set(i)
		m.ui.focusIndex = m.focus.Index()
	}
	m.ScrollToFocused()
	return nil
}

// focusFromMouse determines which element was clicked and focuses it.
func (m *model) focusFromMouse(y int) tea.Cmd {
	cy := m.clientContentY(y)
	chosen := ""
	maxPos := -1
	for _, id := range m.ui.focusOrder {
		if pos, ok := m.ui.elemPos[id]; ok && cy >= pos && pos > maxPos {
			chosen = id
			maxPos = pos
		}
	}
	if chosen != "" {
		if chosen != m.ui.focusOrder[m.ui.focusIndex] {
			return m.SetFocus(chosen)
		}
		return nil
	}
	if len(m.ui.focusOrder) > 0 && m.ui.focusOrder[m.ui.focusIndex] != m.ui.focusOrder[0] {
		return m.SetFocus(m.ui.focusOrder[0])
	}
	return nil
}

// ScrollToFocused ensures the focused element is visible in the viewport.
func (m *model) ScrollToFocused() {
	if len(m.ui.focusOrder) == 0 || m.CurrentMode() != constants.ModeClient {
		return
	}
	id := m.ui.focusOrder[m.ui.focusIndex]
	pos, ok := m.ui.elemPos[id]
	if !ok {
		return
	}
	if id == idHelp {
		return
	}
	top := max(0, pos-1)
	height := max(1, m.ui.elemHeight[id])
	available := max(1, m.ui.viewport.Height)
	end := top + min(height, available)
	if height > available {
		switch id {
		case idHistory:
			l := m.history.List()
			if y, rowHeight, visible := ui.ListItemBounds(*l, l.Index(), 2, 0); visible {
				row := top + 1 + y
				if m.history.FilterQuery() != "" {
					row++
				}
				end = row + rowHeight
				if end-top > available {
					top = row
				}
			}
		case idTopics:
			for _, b := range m.topics.ChipBounds {
				if b.Index == m.topics.Selected() {
					end = b.YPos + min(b.Height, available)
					if end-top > available {
						top = b.YPos
					}
					break
				}
			}
		}
	}
	if top < m.ui.viewport.YOffset {
		m.ui.viewport.SetYOffset(top)
	} else if end > m.ui.viewport.YOffset+available {
		m.ui.viewport.SetYOffset(end - available)
	}
}
