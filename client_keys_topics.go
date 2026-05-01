package emqutiti

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/marang/emqutiti/constants"
	"github.com/marang/emqutiti/topics"
)

// handleLeftKey moves topic selection left.
func (m *model) handleLeftKey() tea.Cmd {
	if m.ui.focusOrder[m.ui.focusIndex] == idTopics && len(m.topics.Items) > 0 {
		sel := m.topics.Selected()
		m.topics.SetSelected((sel - 1 + len(m.topics.Items)) % len(m.topics.Items))
		m.topics.EnsureVisible(m.ui.width - 4)
	}
	return nil
}

// handleRightKey moves topic selection right.
func (m *model) handleRightKey() tea.Cmd {
	if m.ui.focusOrder[m.ui.focusIndex] == idTopics && len(m.topics.Items) > 0 {
		sel := m.topics.Selected()
		m.topics.SetSelected((sel + 1) % len(m.topics.Items))
		m.topics.EnsureVisible(m.ui.width - 4)
	}
	return nil
}

// handleTopicScroll handles scroll keys when topics are focused.
func (m *model) handleTopicScroll(key string) tea.Cmd {
	delta := -1
	if key == constants.KeyDown || key == constants.KeyJ {
		delta = 1
	}
	m.topics.Scroll(delta)
	return nil
}

// handleEnterKey handles Enter for topic input, toggling, and history detail.
func (m *model) handleEnterKey() tea.Cmd {
	switch m.ui.focusOrder[m.ui.focusIndex] {
	case idTopic:
		topic := strings.TrimSpace(m.topics.Input.Value())
		if topic != "" && !m.topics.HasTopic(topic) {
			m.topics.Items = append(m.topics.Items, topics.Item{Name: topic, Subscribed: true})
			m.topics.SortTopics()
			if m.CurrentMode() == constants.ModeTopics {
				m.topics.RebuildActiveTopicList()
			}
			m.topics.Input.SetValue("")
			return func() tea.Msg { return topics.ToggleMsg{Topic: topic, Subscribed: true} }
		}
	case idTopics:
		sel := m.topics.Selected()
		if sel >= 0 && sel < len(m.topics.Items) {
			cmd := m.topics.ToggleTopic(sel)
			m.topics.EnsureVisible(m.ui.width - 4)
			return cmd
		}
	case idHistory:
		return m.handleHistoryViewKey()
	}
	return nil
}

// handleDeleteTopicKey deletes the selected topic.
func (m *model) handleDeleteTopicKey() tea.Cmd {
	idx := m.topics.Selected()
	name := m.topics.Items[idx].Name
	rf := func() tea.Cmd { return m.SetFocus(m.ui.focusOrder[m.ui.focusIndex]) }
	m.StartConfirm(fmt.Sprintf("Delete topic '%s'? [y/n]", name), "", rf, func() tea.Cmd {
		cmd := m.topics.RemoveTopic(idx)
		if m.CurrentMode() == constants.ModeTopics {
			m.topics.RebuildActiveTopicList()
		}
		return cmd
	}, nil)
	return nil
}

// handleTogglePublishKey toggles the publish flag on the selected topic.
func (m *model) handleTogglePublishKey() tea.Cmd {
	if m.ui.focusOrder[m.ui.focusIndex] == idTopics {
		sel := m.topics.Selected()
		if sel >= 0 && sel < len(m.topics.Items) {
			name := m.topics.Items[sel].Name
			m.topics.TogglePublish(sel)
			m.topics.EnsureVisible(m.ui.width - 4)
			return m.startTopicPulse(name)
		}
	}
	return nil
}
