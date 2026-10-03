package topics

import (
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"
	"github.com/marang/emqutiti/ui"
)

func initTopics() state {
	ti := textinput.New()
	ti.Placeholder = "Enter new Topic"
	ti.CharLimit = 128
	ti.Prompt = "> "
	ti.PromptStyle = lipgloss.NewStyle().Foreground(ui.TextMuted)
	ti.PlaceholderStyle = lipgloss.NewStyle().Foreground(ui.TextMuted)
	ti.Cursor.Style = ui.CursorStyle
	ti.TextStyle = ui.FocusedStyle
	ti.Width = 40
	topicsList := list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)
	topicsList.DisableQuitKeybindings()
	topicsList.SetShowTitle(false)
	ts := state{
		Input: ti,
		Items: []Item{},
		list:  topicsList,
		panes: topicsPanes{
			subscribed:   paneState{sel: 0, page: 0, index: 0},
			unsubscribed: paneState{sel: 0, page: 0, index: 1},
			active:       0,
		},
		selected:   -1,
		ChipBounds: []ChipBound{},
		VP:         viewport.New(0, 0),
	}
	return ts
}
