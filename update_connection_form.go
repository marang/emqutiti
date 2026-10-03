package emqutiti

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/marang/emqutiti/connections"
	"github.com/marang/emqutiti/constants"
)

// updateConnectionForm handles the add/edit connection form.
func (m *model) updateConnectionForm(msg tea.Msg) tea.Cmd {
	if m.connections.Form == nil {
		return nil
	}
	l := brokerFormLayout(m.ui.width, m.ui.height)
	m.connections.Form.SetSize(l.formWidth-2, l.contentHeight)
	if mouse, ok := msg.(tea.MouseMsg); ok {
		// One global header row and the top/left box borders precede content.
		mouse.X -= l.formX + 1
		mouse.Y -= 2
		if mouse.X < 0 || mouse.X >= l.formWidth-2 || mouse.Y < 0 || mouse.Y >= l.contentHeight {
			return nil
		}
		if mouse.Action == tea.MouseActionPress && mouse.Button == tea.MouseButtonLeft {
			switch m.connections.Form.ActionAt(mouse.X, mouse.Y) {
			case connections.SaveForm:
				return m.saveConnectionForm()
			case connections.CancelForm:
				return m.cancelConnectionForm()
			}
		}
		msg = mouse
	}
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case constants.KeyCtrlD:
			return tea.Quit
		case constants.KeyEsc:
			return m.cancelConnectionForm()
		case constants.KeyEnter:
			return m.saveConnectionForm()
		}
	}
	f, cmd := m.connections.Form.Update(msg)
	m.connections.Form = &f
	return tea.Batch(cmd, m.connections.ListenStatus())
}

func (m *model) saveConnectionForm() tea.Cmd {
	p, err := m.connections.Form.Validate()
	if err != nil {
		m.connections.SendStatus(err.Error())
		return m.connections.ListenStatus()
	}
	if m.connections.Form.Index >= 0 {
		m.connections.Manager.EditConnection(m.connections.Form.Index, p)
	} else {
		m.connections.Manager.AddConnection(p)
	}
	m.connections.RefreshConnectionItems()
	return m.cancelConnectionForm()
}

func (m *model) cancelConnectionForm() tea.Cmd {
	cmd := m.SetMode(constants.ModeConnections)
	m.connections.Form = nil
	return cmd
}
