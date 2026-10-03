package payloads

import (
	"fmt"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/marang/emqutiti/constants"
	"github.com/marang/emqutiti/focus"
	"github.com/marang/emqutiti/ui"
)

type KeyAction func(tea.KeyMsg) tea.Cmd

// LoadMsg requests that the model load a payload for editing.
type LoadMsg struct{ Topic, Payload string }

type Component struct {
	m       Model
	status  StatusListener
	items   []Item
	list    list.Model
	actions map[string]KeyAction
	ids     []uint64
	nextID  uint64
}

// New creates a payload management component.
func New(m Model, s StatusListener) *Component {
	l := list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)
	l.DisableQuitKeybindings()
	l.SetShowTitle(false)
	p := &Component{m: m, status: s, list: l}
	p.actions = map[string]KeyAction{
		constants.KeyCtrlD: func(tea.KeyMsg) tea.Cmd { return tea.Quit },
		constants.KeyEsc:   func(tea.KeyMsg) tea.Cmd { return p.m.SetClientMode() },
		constants.KeyDelete: func(tea.KeyMsg) tea.Cmd {
			p.confirmDelete()
			return p.status.ListenStatus()
		},
		constants.KeyEnter: func(tea.KeyMsg) tea.Cmd {
			if pi, ok := p.list.SelectedItem().(Item); ok {
				return tea.Batch(
					func() tea.Msg { return LoadMsg{Topic: pi.Topic, Payload: pi.Payload} },
					p.m.SetClientMode(),
					p.status.ListenStatus(),
				)
			}
			return nil
		},
	}
	return p
}

func (p *Component) Init() tea.Cmd { return nil }

func (p *Component) Update(msg tea.Msg) tea.Cmd {
	p.SetSize(p.m.Width(), p.m.Height())
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		filterOwnsKey := p.list.FilterState() == list.Filtering || (msg.String() == constants.KeyEsc && p.list.FilterState() == list.FilterApplied)
		if act, ok := p.actions[msg.String()]; ok && (!filterOwnsKey || msg.String() == constants.KeyCtrlD) {
			return act(msg)
		}
		p.list, cmd = p.list.Update(msg)
		return tea.Batch(cmd, p.status.ListenStatus())
	case tea.MouseMsg:
		if msg.Action == tea.MouseActionPress {
			switch msg.Button {
			case tea.MouseButtonLeft, tea.MouseButtonRight:
				idx := ui.ListRowAt(p.list, msg.X-1, msg.Y-2, 2, 1)
				if idx < 0 || p.list.FilterState() == list.Filtering {
					return nil
				}
				p.list.Select(idx)
				if msg.Button == tea.MouseButtonRight {
					return p.actions[constants.KeyDelete](tea.KeyMsg{})
				}
				return p.actions[constants.KeyEnter](tea.KeyMsg{})
			}
		}
	}
	p.list, cmd = p.list.Update(msg)
	return tea.Batch(cmd, p.status.ListenStatus())
}

func (p *Component) View() string {
	m := p.m
	p.SetSize(m.Width(), m.Height())
	m.ResetElemPos()
	m.SetElemPos(IDList, 1)
	listView := p.list.View()
	help := p.footer(m.Width() - 4)
	content := lipgloss.JoinVertical(lipgloss.Left, listView, help)
	focused := m.FocusedID() == IDList
	view := ui.LegendBox(content, "Payloads", m.Width()-2, 0, ui.ColBlue, focused, -1)
	return m.OverlayHelp(view)
}

// SetSize lays out the payload manager within the full terminal, including its header.
func (p *Component) SetSize(width, height int) {
	innerWidth := max(1, width-4)
	ui.SizeManagerList(&p.list, innerWidth, height-3-lipgloss.Height(p.footer(innerWidth)))
}

func (p *Component) footer(width int) string {
	return ui.ListFooter(width, "[enter] load", "[del] delete", "[/] filter", "[esc] back")
}

func (p *Component) confirmDelete() {
	if p.list.SelectedItem() == nil {
		return
	}
	i := p.list.GlobalIndex()
	if i < 0 || i >= len(p.ids) {
		return
	}
	id, topic, focused := p.ids[i], p.items[i].Topic, p.m.FocusedID()
	p.m.StartConfirm(fmt.Sprintf("Delete payload for '%s'? [y/n]", topic), "", func() tea.Cmd {
		return p.m.SetFocus(focused)
	}, func() tea.Cmd {
		for i, candidate := range p.ids {
			if candidate == id {
				p.items = append(p.items[:i], p.items[i+1:]...)
				p.ids = append(p.ids[:i], p.ids[i+1:]...)
				p.rebuildList()
				break
			}
		}
		return p.status.ListenStatus()
	}, nil)
}

func (p *Component) rebuildList() {
	filter, state := p.list.FilterValue(), p.list.FilterState()
	items := make([]list.Item, len(p.items))
	for i, item := range p.items {
		items[i] = item
	}
	p.list.SetItems(items)
	if state != list.Unfiltered {
		p.list.SetFilterText(filter)
		if state == list.Filtering {
			p.list.SetFilterState(state)
		}
	}
}

func (p *Component) Focus() tea.Cmd { return nil }

func (p *Component) Blur() {}

// Focusables exposes focusable elements for the payloads component.
func (p *Component) Focusables() map[string]focus.Focusable {
	return map[string]focus.Focusable{IDList: &nullFocusable{}}
}

func (p *Component) Add(topic, payload string) {
	pi := Item{Topic: topic, Payload: payload}
	p.items = append(p.items, pi)
	p.nextID++
	p.ids = append(p.ids, p.nextID)
	p.rebuildList()
}

func (p *Component) Items() []Item { return p.items }

func (p *Component) SetItems(plds []Item) {
	p.items = plds
	p.ids = make([]uint64, len(plds))
	for i := range plds {
		p.nextID++
		p.ids[i] = p.nextID
	}
	p.rebuildList()
}

func (p *Component) Clear() { p.SetItems([]Item{}) }

// List exposes the underlying list model.
func (p *Component) List() *list.Model { return &p.list }

type nullFocusable struct{ focused bool }

func (n *nullFocusable) Focus()          { n.focused = true }
func (n *nullFocusable) Blur()           { n.focused = false }
func (n *nullFocusable) IsFocused() bool { return n.focused }
func (n *nullFocusable) View() string    { return "" }
