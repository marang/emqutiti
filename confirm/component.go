package confirm

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

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
}

func NewDialog(nav Navigator, status StatusListener, returnFocus func() tea.Cmd, action func() tea.Cmd, cancel func()) *Dialog {
	return &Dialog{nav: nav, status: status, returnFocus: returnFocus, action: action, cancel: cancel}
}

func (d *Dialog) Init() tea.Cmd { return nil }

func (d *Dialog) Start(prompt, info string) {
	d.prompt = prompt
	d.info = info
	_ = d.nav.SetConfirmMode()
}

func (d *Dialog) Update(msg tea.Msg) tea.Cmd {
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
	}
	return d.status.ListenStatus()
}

func (d *Dialog) View() string {
	content := d.prompt
	if d.info != "" {
		content = lipgloss.JoinVertical(lipgloss.Left, d.prompt, d.info)
	}
	content = lipgloss.NewStyle().Padding(1, 2).Render(content)
	label := "Confirm"
	width := lipgloss.Width(content) + 2
	if lw := lipgloss.Width(label) + 4; width < lw {
		width = lw
	}
	maxWidth := d.nav.Width() - 2
	if maxWidth > 0 && width > maxWidth {
		width = maxWidth
	}
	box := ui.LegendBox(content, label, width, 0, ui.ColBlue, true, -1)
	return lipgloss.Place(d.nav.Width(), d.nav.Height(), lipgloss.Center, lipgloss.Center, box)
}

func (d *Dialog) Focus() tea.Cmd {
	d.focused = true
	return nil
}

func (d *Dialog) Blur() { d.focused = false }

func (d *Dialog) Focused() bool { return d.focused }
