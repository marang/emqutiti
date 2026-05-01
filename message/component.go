package message

import (
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"

	"github.com/marang/emqutiti/focus"
	"github.com/marang/emqutiti/ui"
)

// State holds the textarea model for composing messages.
type State struct {
	TA textarea.Model
}

// Component implements the message editor.
type Component struct {
	*State
	m Model
}

// NewComponent creates a message editor component.
func NewComponent(m Model, s State) *Component {
	return &Component{State: &s, m: m}
}

func (c *Component) Init() tea.Cmd { return nil }

// Update handles textarea updates when editing messages.
func (c *Component) Update(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	c.TA, cmd = c.TA.Update(msg)
	return cmd
}

// View renders the message editor box.
func (c *Component) View() string {
	msgContent := c.TA.View()
	msgLines := c.TA.LineCount()
	msgHeight := c.m.MessageHeight()
	msgSP := -1.0
	if msgLines > msgHeight {
		off := c.TA.Line() - msgHeight + 1
		if off < 0 {
			off = 0
		}
		maxOff := msgLines - msgHeight
		if off > maxOff {
			off = maxOff
		}
		if maxOff > 0 {
			msgSP = float64(off) / float64(maxOff)
		}
	}
	focused := c.m.FocusedID() == ID
	hovered := c.m.HoveredID() == ID
	label := c.m.MessageTargetPreview()
	if maxLabel := c.m.Width() - 6; maxLabel > 0 {
		label = ansi.Truncate(label, maxLabel, "…")
	}
	return ui.LegendBoxWithState(msgContent, label, c.m.Width()-2, msgHeight, ui.ColBlue, ui.BoxState{Focused: focused, Hovered: hovered}, msgSP)
}

func (c *Component) Focus() tea.Cmd { return c.TA.Focus() }

func (c *Component) Blur() { c.TA.Blur() }

// Input returns the textarea model.
func (c *Component) Input() *textarea.Model { return &c.TA }

// SetPayload updates the textarea with the provided payload.
func (c *Component) SetPayload(payload string) { c.TA.SetValue(payload) }

// Focusables exposes focusable elements for the message component.
func (c *Component) Focusables() map[string]focus.Focusable {
	return map[string]focus.Focusable{ID: focus.Adapt(&c.TA)}
}
