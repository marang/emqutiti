package traces

import (
	"github.com/charmbracelet/lipgloss"

	"github.com/marang/emqutiti/ui"
)

// viewTraces lists configured traces and their state.
func (t *Component) viewTraces() string {
	t.SetSize(t.api.Width(), t.api.Height())
	t.api.ResetElemPos()
	t.api.SetElemPos(IDList, 1)
	listView := t.list.View()
	help := t.footer(t.api.Width() - 4)
	content := lipgloss.JoinVertical(lipgloss.Left, listView, help)
	focused := t.api.FocusedID() == IDList
	view := ui.LegendBox(content, "Traces", t.api.Width()-2, 0, ui.ColBlue, focused, -1)
	return t.api.OverlayHelp(view)
}

// SetSize lays out the trace manager/form within the terminal, including its header.
func (t *Component) SetSize(width, height int) {
	innerWidth := max(1, width-4)
	ui.SizeManagerList(&t.list, innerWidth, height-3-lipgloss.Height(t.footer(innerWidth)))
	if t.form != nil {
		t.form.SetSize(innerWidth, max(1, height-3))
	}
}

func (t *Component) footer(width int) string {
	return ui.ListFooter(width, "[a] add", "[enter] start/stop", "[v] view", "[del] delete", "[/] filter", "[esc] back")
}
