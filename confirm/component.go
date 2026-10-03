package confirm

import (
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/marang/emqutiti/constants"
	"github.com/marang/emqutiti/ui"
)

// Dialog manages confirmation dialogs.
type Dialog struct {
	nav    Navigator
	status StatusListener

	prompt      string
	info        string
	action      func() tea.Cmd
	cancel      func()
	returnFocus func() tea.Cmd
	focused     bool
	body        viewport.Model
	bodyText    string
}

type dialogLayout struct {
	width, padding int
	left, top      int
	bodyX, bodyY   int
	decisionY      int
}

func NewDialog(nav Navigator, status StatusListener, returnFocus func() tea.Cmd, action func() tea.Cmd, cancel func()) *Dialog {
	return &Dialog{nav: nav, status: status, returnFocus: returnFocus, action: action, cancel: cancel, body: viewport.New(0, 0)}
}

func (d *Dialog) Init() tea.Cmd { return nil }

func (d *Dialog) Start(prompt, info string) {
	d.prompt = prompt
	d.info = info
	d.body.GotoTop()
	_ = d.nav.SetConfirmMode()
}

func (d *Dialog) Update(msg tea.Msg) tea.Cmd {
	layout := d.layout()
	switch t := msg.(type) {
	case tea.KeyMsg:
		switch t.String() {
		case constants.KeyCtrlD:
			return tea.Quit
		case constants.KeyY:
			var acmd tea.Cmd
			if d.action != nil {
				acmd = d.action()
				d.action = nil
			}
			if d.cancel != nil {
				d.cancel = nil
			}
			cmd := d.nav.SetPreviousMode()
			cmds := []tea.Cmd{cmd, d.status.ListenStatus()}
			if acmd != nil {
				cmds = append(cmds, acmd)
			}
			if d.returnFocus != nil {
				cmds = append(cmds, d.returnFocus())
				d.returnFocus = nil
			} else {
				d.nav.ScrollToFocused()
			}
			return tea.Batch(cmds...)
		case constants.KeyN, constants.KeyEsc:
			if d.cancel != nil {
				d.cancel()
				d.cancel = nil
			}
			cmd := d.nav.SetPreviousMode()
			cmds := []tea.Cmd{cmd, d.status.ListenStatus()}
			if d.returnFocus != nil {
				cmds = append(cmds, d.returnFocus())
				d.returnFocus = nil
			} else {
				d.nav.ScrollToFocused()
			}
			return tea.Batch(cmds...)
		}
		var cmd tea.Cmd
		switch t.String() {
		case constants.KeyCtrlUp, constants.KeyCtrlK:
			d.body.ScrollUp(1)
		case constants.KeyCtrlDown, constants.KeyCtrlJ:
			d.body.ScrollDown(1)
		default:
			d.body, cmd = d.body.Update(msg)
		}
		return tea.Batch(cmd, d.status.ListenStatus())
	case tea.MouseMsg:
		if t.Action == tea.MouseActionPress && t.Button == tea.MouseButtonLeft && t.Y == layout.decisionY {
			switch t.X - layout.bodyX {
			case 1:
				return d.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(constants.KeyY)})
			case 3:
				return d.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(constants.KeyN)})
			}
		}
		if t.X >= layout.bodyX && t.X < layout.bodyX+d.body.Width && t.Y >= layout.bodyY && t.Y < layout.bodyY+d.body.Height {
			var cmd tea.Cmd
			d.body, cmd = d.body.Update(msg)
			return tea.Batch(cmd, d.status.ListenStatus())
		}
	}
	return d.status.ListenStatus()
}

func (d *Dialog) layout() dialogLayout {
	// Keep the decision hint outside the viewport so long targets cannot hide it.
	content := strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(d.prompt), "[y/n]"))
	if d.info != "" {
		content += "\n" + d.info
	}
	width := max(lipgloss.Width(content)+6, lipgloss.Width("Confirm")+4)
	width = max(10, min(width, d.nav.Width()-2))
	innerWidth := max(1, width-6)
	if d.body.Width != innerWidth || d.bodyText != content {
		d.body.Width = innerWidth
		d.body.SetContent(ansi.WrapWc(content, innerWidth, ""))
		d.bodyText = content
	}
	padding := 1
	if d.nav.Height() < 7 {
		padding = 0
	}
	d.body.Height = min(d.body.TotalLineCount(), max(1, d.nav.Height()-4-2*padding))
	d.body.SetYOffset(d.body.YOffset)
	left := max(0, (d.nav.Width()-width)/2)
	top := max(0, (d.nav.Height()-d.body.Height-4-2*padding)/2)
	bodyX, bodyY := left+3, top+1+padding
	return dialogLayout{width: width, padding: padding, left: left, top: top, bodyX: bodyX, bodyY: bodyY, decisionY: bodyY + d.body.Height + 1}
}

func (d *Dialog) View() string {
	layout := d.layout()
	content := lipgloss.NewStyle().Padding(layout.padding, 2).Render(d.body.View() + "\n\n[y/n]")
	sp := -1.0
	if d.body.TotalLineCount() > d.body.Height {
		sp = d.body.ScrollPercent()
	}
	box := ui.LegendBox(content, "Confirm", layout.width, lipgloss.Height(content), ui.ColBlue, true, sp)
	return lipgloss.Place(d.nav.Width(), d.nav.Height(), lipgloss.Center, lipgloss.Center, box)
}

func (d *Dialog) Focus() tea.Cmd {
	d.focused = true
	return nil
}

func (d *Dialog) Blur() { d.focused = false }

func (d *Dialog) Focused() bool { return d.focused }
