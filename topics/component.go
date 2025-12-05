package topics

import (
	"fmt"
	"sort"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/marang/emqutiti/constants"
	"github.com/marang/emqutiti/focus"
	"github.com/marang/emqutiti/ui"
)

// state holds topic list state shared across components.
type state struct {
	Input      textinput.Model
	Items      []Item
	list       list.Model
	panes      topicsPanes
	selected   int
	ChipBounds []ChipBound
	VP         viewport.Model
}

func (t *state) setTopic(topic string) { t.Input.SetValue(topic) }

// SetTopic sets the topic input value.
func (c *Component) SetTopic(topic string) { c.state.setTopic(topic) }

// ToggleMsg notifies the model of topic subscription changes.
type ToggleMsg struct {
	Topic      string
	Subscribed bool
}

// Component implements topic management UI.
type Component struct {
	*state
	api Model
}

// New constructs a new Component.
func New(api Model) *Component {
	ts := initTopics()
	tc := &Component{state: &ts, api: api}
	tc.panes.subscribed.m = tc
	tc.panes.unsubscribed.m = tc
	return tc
}

func (c *Component) Init() tea.Cmd { return nil }

// Update manages the topics list UI.
func (c *Component) Update(msg tea.Msg) tea.Cmd {
	var cmd, fcmd, tcmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case constants.KeyCtrlD:
			return tea.Quit
		case constants.KeyEsc:
			return c.api.ShowClient()
		case constants.KeyLeft:
			if c.panes.active == 1 {
				fcmd = c.api.SetFocus(idTopicsSubscribed)
			}
		case constants.KeyRight:
			if c.panes.active == 0 {
				fcmd = c.api.SetFocus(idTopicsUnsubscribed)
			}
		case constants.KeyDelete:
			i := c.selected
			if i >= 0 && i < len(c.Items) {
				name := c.Items[i].Name
				rf := func() tea.Cmd { return c.api.SetFocus(c.api.FocusedID()) }
				c.api.StartConfirm(fmt.Sprintf("Delete topic '%s'? [y/n]", name), "", rf, func() tea.Cmd {
					cmd := c.RemoveTopic(i)
					c.RebuildActiveTopicList()
					return cmd
				}, nil)
				return c.api.ListenStatus()
			}
		case constants.KeyEnter, constants.KeySpaceBar:
			i := c.selected
			if i >= 0 && i < len(c.Items) {
				tcmd = c.ToggleTopic(i)
			}
		case constants.KeyP:
			i := c.selected
			if i >= 0 && i < len(c.Items) {
				c.TogglePublish(i)
			}
		}
	case tea.MouseMsg:
		if msg.Action == tea.MouseActionPress {
			switch msg.Button {
			case tea.MouseButtonLeft, tea.MouseButtonRight:
				tcmd = c.HandleClick(msg, c.VP.YOffset)
			}
		}
	}
	c.list, cmd = c.list.Update(msg)
	if c.panes.active == 0 {
		c.panes.subscribed.sel = c.list.Index()
		c.panes.subscribed.page = c.list.Paginator.Page
	} else {
		c.panes.unsubscribed.sel = c.list.Index()
		c.panes.unsubscribed.page = c.list.Paginator.Page
	}
	c.selected = c.IndexForPane(c.panes.active, c.list.Index())
	return tea.Batch(fcmd, tcmd, cmd, c.api.ListenStatus())
}

// View displays the topic manager list.
func (c *Component) View() string {
	c.api.ResetElemPos()
	c.api.SetElemPos(idTopicsSubscribed, 1)
	c.api.SetElemPos(idTopicsUnsubscribed, 1)
	help := ui.InfoStyle.Render("[space] toggle  [p] publish  [del] delete  [esc] back")
	activeView := c.list.View()
	var left, right string
	if c.panes.active == 0 {
		other := list.New(c.UnsubscribedItems(), list.NewDefaultDelegate(), c.list.Width(), c.list.Height())
		other.DisableQuitKeybindings()
		other.SetShowTitle(false)
		other.Paginator.Page = c.panes.unsubscribed.page
		other.Select(c.panes.unsubscribed.sel)
		left = ui.LegendBox(activeView, "Subscribed", c.api.Width()/2-2, 0, ui.ColBlue, c.api.FocusedID() == idTopicsSubscribed, -1)
		right = ui.LegendBox(other.View(), "Unsubscribed", c.api.Width()/2-2, 0, ui.ColBlue, false, -1)
	} else {
		other := list.New(c.SubscribedItems(), list.NewDefaultDelegate(), c.list.Width(), c.list.Height())
		other.DisableQuitKeybindings()
		other.SetShowTitle(false)
		other.Paginator.Page = c.panes.subscribed.page
		other.Select(c.panes.subscribed.sel)
		left = ui.LegendBox(other.View(), "Subscribed", c.api.Width()/2-2, 0, ui.ColBlue, false, -1)
		right = ui.LegendBox(activeView, "Unsubscribed", c.api.Width()/2-2, 0, ui.ColBlue, c.api.FocusedID() == idTopicsUnsubscribed, -1)
	}
	panes := lipgloss.JoinHorizontal(lipgloss.Top, left, right)
	content := lipgloss.JoinVertical(lipgloss.Left, panes, help)
	return c.api.OverlayHelp(content)
}

func (c *Component) Focus() tea.Cmd { return nil }

func (c *Component) Blur() {}

// UpdateInput routes messages to the topic text input.
func (c *Component) UpdateInput(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	c.Input, cmd = c.Input.Update(msg)
	return cmd
}

// Focusables exposes focusable elements for the topics component.
func (c *Component) Focusables() map[string]focus.Focusable {
	return map[string]focus.Focusable{
		idTopicsSubscribed:   &c.panes.subscribed,
		idTopicsUnsubscribed: &c.panes.unsubscribed,
		idTopic:              focus.Adapt(&c.Input),
	}
}

// HasTopic reports whether the given topic already exists in the list.
func (c *Component) HasTopic(topic string) bool {
	for _, t := range c.Items {
		if t.Name == topic {
			return true
		}
	}
	return false
}

// SortTopics orders the topic list with active topics first and keeps selection.
func (c *Component) SortTopics() {
	if len(c.Items) == 0 {
		return
	}
	sel := ""
	if c.selected >= 0 && c.selected < len(c.Items) {
		sel = c.Items[c.selected].Name
	}
	sort.SliceStable(c.Items, func(i, j int) bool {
		if c.Items[i].Subscribed != c.Items[j].Subscribed {
			return c.Items[i].Subscribed && !c.Items[j].Subscribed
		}
		return c.Items[i].Name < c.Items[j].Name
	})
	if sel != "" {
		for i, t := range c.Items {
			if t.Name == sel {
				c.selected = i
				break
			}
		}
	}
}

// SubscribedItems returns topics currently subscribed.
func (c *Component) SubscribedItems() []list.Item {
	var out []list.Item
	for _, t := range c.Items {
		if t.Subscribed {
			out = append(out, t)
		}
	}
	return out
}

// UnsubscribedItems returns topics that are not subscribed.
func (c *Component) UnsubscribedItems() []list.Item {
	var out []list.Item
	for _, t := range c.Items {
		if !t.Subscribed {
			out = append(out, t)
		}
	}
	return out
}

// IndexForPane converts a pane list index to a global topics index.
func (c *Component) IndexForPane(pane, idx int) int {
	count := -1
	for i, t := range c.Items {
		if (pane == 0 && t.Subscribed) || (pane == 1 && !t.Subscribed) {
			count++
			if count == idx {
				return i
			}
		}
	}
	return -1
}

// RebuildActiveTopicList updates the active list to show the current pane.
func (c *Component) RebuildActiveTopicList() {
	if c.panes.active == 0 {
		items := c.SubscribedItems()
		if c.panes.subscribed.sel >= len(items) {
			c.panes.subscribed.sel = len(items) - 1
		}
		if c.panes.subscribed.sel < 0 && len(items) > 0 {
			c.panes.subscribed.sel = 0
		}
		c.list.SetItems(items)
		if len(items) > 0 {
			c.list.Select(c.panes.subscribed.sel)
		}
		c.list.Paginator.Page = c.panes.subscribed.page
		c.selected = c.IndexForPane(0, c.panes.subscribed.sel)
	} else {
		items := c.UnsubscribedItems()
		if c.panes.unsubscribed.sel >= len(items) {
			c.panes.unsubscribed.sel = len(items) - 1
		}
		if c.panes.unsubscribed.sel < 0 && len(items) > 0 {
			c.panes.unsubscribed.sel = 0
		}
		c.list.SetItems(items)
		if len(items) > 0 {
			c.list.Select(c.panes.unsubscribed.sel)
		}
		c.list.Paginator.Page = c.panes.unsubscribed.page
		c.selected = c.IndexForPane(1, c.panes.unsubscribed.sel)
	}
}

// SetActivePane switches focus to the given pane index and rebuilds the list.
func (c *Component) SetActivePane(idx int) {
	if idx == c.panes.active {
		return
	}
	if c.panes.active == 0 {
		c.panes.subscribed.sel = c.list.Index()
		c.panes.subscribed.page = c.list.Paginator.Page
	} else {
		c.panes.unsubscribed.sel = c.list.Index()
		c.panes.unsubscribed.page = c.list.Paginator.Page
	}
	c.panes.active = idx
	c.RebuildActiveTopicList()
}

// ToggleTopic toggles the subscription state of the topic at index and emits an event.
func (c *Component) ToggleTopic(index int) tea.Cmd {
	if index < 0 || index >= len(c.Items) {
		return nil
	}
	name := c.Items[index].Name
	subscribed := !c.Items[index].Subscribed
	c.Items[index].Subscribed = subscribed
	c.SortTopics()
	c.RebuildActiveTopicList()
	for i, it := range c.Items {
		if it.Name == name {
			c.SetSelected(i)
			break
		}
	}
	return func() tea.Msg { return ToggleMsg{Topic: name, Subscribed: subscribed} }
}

// TogglePublish toggles the publish flag of the topic at index.
func (c *Component) TogglePublish(index int) {
	if index < 0 || index >= len(c.Items) {
		return
	}
	t := &c.Items[index]
	t.Publish = !t.Publish
	c.SetSelected(index)
	c.RebuildActiveTopicList()
}

// RemoveTopic deletes the topic at index and emits an unsubscribe event.
func (c *Component) RemoveTopic(index int) tea.Cmd {
	if index < 0 || index >= len(c.Items) {
		return nil
	}
	topic := c.Items[index].Name
	c.Items = append(c.Items[:index], c.Items[index+1:]...)
	if len(c.Items) == 0 {
		c.selected = -1
	} else if c.selected >= len(c.Items) {
		c.selected = len(c.Items) - 1
	}
	c.SortTopics()
	c.RebuildActiveTopicList()
	return func() tea.Msg { return ToggleMsg{Topic: topic, Subscribed: false} }
}

// TopicAtPosition returns the index of the topic chip at the provided coordinates or -1.
func (c *Component) TopicAtPosition(x, y int) int {
	for _, b := range c.ChipBounds {
		if x >= b.XPos && x < b.XPos+b.Width && y >= b.YPos && y < b.YPos+b.Height {
			return b.Index
		}
	}
	return -1
}

// Scroll moves the topics viewport by delta rows.
func (c *Component) Scroll(delta int) {
	rowH := lipgloss.Height(ui.Chip.Render("test"))
	if delta > 0 {
		c.VP.ScrollDown(delta * rowH)
	} else if delta < 0 {
		c.VP.ScrollUp(-delta * rowH)
	}
}

// EnsureVisible keeps the selected topic within view given the available width.
func (c *Component) EnsureVisible(width int) {
	sel := c.Selected()
	if sel < 0 || sel >= len(c.Items) {
		return
	}
	var chips []string
	for _, t := range c.Items {
		st := ui.Chip
		switch {
		case t.Publish:
			st = ui.ChipPublish
		case !t.Subscribed:
			st = ui.ChipInactive
		}
		chips = append(chips, st.Render(t.Name))
	}
	_, bounds := LayoutChips(chips, width)
	if sel >= len(bounds) {
		return
	}
	b := bounds[sel]
	if b.YPos < c.VP.YOffset {
		c.VP.SetYOffset(b.YPos)
	} else if b.YPos+b.Height > c.VP.YOffset+c.VP.Height {
		c.VP.SetYOffset(b.YPos + b.Height - c.VP.Height)
	}
}

// HandleClick processes mouse clicks on topics and triggers actions.
func (c *Component) HandleClick(msg tea.MouseMsg, vpOffset int) tea.Cmd {
	y := msg.Y + vpOffset
	idx := c.TopicAtPosition(msg.X, y)
	if idx < 0 {
		return nil
	}
	c.SetSelected(idx)
	if msg.Type == tea.MouseLeft {
		return c.ToggleTopic(idx)
	} else if msg.Type == tea.MouseRight {
		name := c.Items[idx].Name
		focused := c.api.FocusedID()
		rf := func() tea.Cmd { return c.api.SetFocus(focused) }
		c.api.StartConfirm(fmt.Sprintf("Delete topic '%s'? [y/n]", name), "", rf, func() tea.Cmd {
			return c.RemoveTopic(idx)
		}, nil)
	}
	return nil
}

// List exposes the topics list model.
func (c *Component) List() *list.Model { return &c.list }

func (c *Component) SetSelected(i int) {
	c.selected = i
	if i < 0 || i >= len(c.Items) {
		return
	}
	if c.Items[i].Subscribed {
		idx := 0
		for j := 0; j < len(c.Items); j++ {
			if c.Items[j].Subscribed {
				if j == i {
					c.panes.subscribed.sel = idx
					break
				}
				idx++
			}
		}
	} else {
		idx := 0
		for j := 0; j < len(c.Items); j++ {
			if !c.Items[j].Subscribed {
				if j == i {
					c.panes.unsubscribed.sel = idx
					break
				}
				idx++
			}
		}
	}
}

func (c *Component) Selected() int { return c.selected }

var _ API = (*Component)(nil)
