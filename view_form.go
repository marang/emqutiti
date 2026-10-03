package emqutiti

import (
	"github.com/charmbracelet/lipgloss"

	"github.com/marang/emqutiti/constants"
	"github.com/marang/emqutiti/ui"
)

type connectionFormLayout struct {
	listWidth, formX, formWidth, contentHeight int
}

func brokerFormLayout(width, height int) connectionFormLayout {
	l := connectionFormLayout{formWidth: max(4, width), contentHeight: max(2, height-3)}
	if width >= 100 {
		l.listWidth = width / 2
		l.formX = l.listWidth
		l.formWidth = width - l.listWidth
	}
	return l
}

// viewForm renders the add/edit broker form alongside the list.
func (m *model) viewForm() string {
	m.ui.elemPos = map[string]int{}
	if m.connections.Form == nil {
		return ""
	}
	l := brokerFormLayout(m.ui.width, m.ui.height)
	m.connections.Form.SetSize(l.formWidth-2, l.contentHeight)
	formLabel := "Add Broker"
	if m.connections.Form.Index >= 0 {
		formLabel = "Edit Broker"
	}
	formView := ui.LegendBox(m.connections.Form.View(), formLabel, l.formWidth, l.contentHeight, ui.ColBlue, true, -1)
	if l.listWidth == 0 {
		return m.overlayHelp(formView)
	}
	m.ui.elemPos[constants.IDConnList] = 1
	m.connections.Manager.ConnectionsList.SetSize(l.listWidth-2, l.contentHeight)
	listView := ui.LegendBox(m.connections.Manager.ConnectionsList.View(), "Brokers", l.listWidth, l.contentHeight, ui.ColBlue, false, -1)
	return m.overlayHelp(lipgloss.JoinHorizontal(lipgloss.Top, listView, formView))
}
