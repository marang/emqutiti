package history

import (
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/marang/emqutiti/constants"
	"github.com/marang/emqutiti/ui"
)

type historyState struct {
	list            list.Model
	items           []Item
	store           Store
	selectionAnchor int
	showArchived    bool
	filterForm      *historyFilterForm
	filterQuery     string
	detail          viewport.Model
	detailItem      Item
}

// Component provides history browsing and filtering functionality. It holds its
// own state while delegating cross component interactions back to the root
// model.
type Component struct {
	*historyState
	m Model
}

// Init performs no initialization and returns nil.
func (h *Component) Init() tea.Cmd { return nil }

// Update updates the history list. The caller must ensure the list has focus.
func (h *Component) Update(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	switch m := msg.(type) {
	case tea.MouseMsg:
		h.list, cmd = h.list.Update(m)
		if m.Action == tea.MouseActionPress && m.Button == tea.MouseButtonLeft {
			h.HandleSelection(h.list.Index(), m.Shift)
		}
		return cmd
	}
	h.list, cmd = h.list.Update(msg)
	return cmd
}

// View renders no standalone view for the history component.
func (h *Component) View() string { return "" }

// Focus gives focus to the history component. Currently a no-op.
func (h *Component) Focus() tea.Cmd { return nil }

// Blur removes focus from the history component. Currently a no-op.
func (h *Component) Blur() {}

// UpdateDetail handles input when viewing a long history payload.
func (h *Component) UpdateDetail(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case constants.KeyEsc:
			return h.m.SetMode(h.m.PreviousMode())
		case constants.KeyCtrlD:
			return tea.Quit
		}
	}
	h.detail, cmd = h.detail.Update(msg)
	return cmd
}

// UpdateFilter handles the history filter form interaction.
func (h *Component) UpdateFilter(msg tea.Msg) tea.Cmd {
	if h.filterForm == nil {
		return nil
	}
	switch t := msg.(type) {
	case tea.KeyMsg:
		switch t.String() {
		case constants.KeyEsc:
			h.filterForm = nil
			cmd := tea.Batch(h.m.SetMode(h.m.PreviousMode()), h.m.SetFocus(ID))
			return cmd
		case constants.KeyEnter:
			h.showArchived = h.filterForm.archived.Bool()
			q := h.filterForm.query()
			topics, start, end, payload := ParseQuery(q)
			msgs := h.store.Search(h.showArchived, topics, start, end, payload)
			var items []list.Item
			h.items, items = MessagesToItems(msgs)
			h.list.SetItems(items)
			h.list.FilterInput.SetValue("")
			h.list.SetFilterState(list.Unfiltered)
			h.filterQuery = q
			h.filterForm = nil
			cmd := tea.Batch(h.m.SetMode(h.m.PreviousMode()), h.m.SetFocus(ID))
			return cmd
		}
	}
	f, cmd := h.filterForm.Update(msg)
	h.filterForm = &f
	return cmd
}

// ViewDetail renders the full payload of a history message.
func (h *Component) ViewDetail() string {
	lines := strings.Split(h.detail.View(), "\n")
	help := ui.InfoStyle.Render("[esc] back")
	lines = append(lines, help)
	content := strings.Join(lines, "\n")
	sp := -1.0
	if h.detail.Height < lipgloss.Height(content) {
		sp = h.detail.ScrollPercent()
	}
	view := ui.LegendBox(content, "Message", h.m.Width()-2, h.m.Height()-2, ui.ColGreen, true, sp)
	return h.m.OverlayHelp(view)
}

// ViewFilter displays the history filter form.
func (h *Component) ViewFilter() string {
	if h.filterForm == nil {
		return ""
	}
	content := lipgloss.NewStyle().Padding(1, 2).Render(h.filterForm.View())
	box := ui.LegendBox(content, "Filter", h.m.Width()/2, 0, ui.ColBlue, true, -1)
	return lipgloss.Place(h.m.Width(), h.m.Height(), lipgloss.Center, lipgloss.Center, box)
}

// Focusables exposes focusable elements for the history component. The list is
// managed externally, so this returns an empty map.
func (h *Component) Focusables() map[string]Focusable { return map[string]Focusable{} }

// Append stores a message in the history list and optional store.
func (h *Component) Append(topic, payload, kind string, retained bool, logText string) {
	ts := time.Now()
	text := payload
	if kind == "log" {
		text = logText
	}
	hi := Item{Timestamp: ts, Topic: topic, Payload: text, Kind: kind, Archived: false, Retained: retained}
	if h.store != nil {
		h.store.Append(Message{Timestamp: ts, Topic: topic, Payload: payload, Kind: kind, Archived: false, Retained: retained})
	}
	if !h.showArchived {
		if h.filterQuery != "" {
			var items []list.Item
			h.items, items = ApplyFilter(h.filterQuery, h.store, h.showArchived)
			h.list.SetItems(items)
			h.list.Select(len(items) - 1)
		} else {
			h.items = append(h.items, hi)
			items := make([]list.Item, len(h.items))
			for i, it := range h.items {
				items[i] = it
			}
			h.list.SetItems(items)
			h.list.Select(len(items) - 1)
		}
	}
}
