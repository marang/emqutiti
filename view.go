package emqutiti

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"

	"github.com/marang/emqutiti/ui"
)

const helpReflowWidth = 60

func (m *model) availableInfoWidth(helpWidth, pad int, stacked bool) int {
	available := m.ui.width - pad
	if !stacked {
		available -= helpWidth
	}
	if available < 0 {
		available = 0
	}
	return available
}

func (m *model) renderFirstLine(info, help string, available, pad int, stacked bool) string {
	if stacked {
		return lipgloss.NewStyle().Width(available + pad).Render(ui.InfoStyle.Render(info))
	}
	return lipgloss.JoinHorizontal(lipgloss.Top,
		lipgloss.NewStyle().Width(available+pad).Render(ui.InfoStyle.Render(info)), help)
}

func (m *model) renderSecondLine(lines []string, help string, stacked bool) []string {
	if !stacked {
		return lines
	}
	if len(lines) > 1 {
		secondAvail := m.ui.width - lipgloss.Width(help)
		if secondAvail < 0 {
			secondAvail = 0
		}
		second := lipgloss.JoinHorizontal(lipgloss.Top,
			lipgloss.NewStyle().Width(secondAvail).Render(lines[1]), help)
		lines[1] = second
	} else {
		lines = append(lines, help)
	}
	return lines
}

func (m *model) overlayHelp(view string) string {
	help := ui.HelpStyle.Render("?")
	if m.help.Focused() {
		help = ui.HelpFocused.Render("?")
	}
	m.ui.elemPos[idHelp] = 0

	info := "Switch views: Ctrl+B brokers, Ctrl+T topics, Ctrl+P payloads, Alt+R traces, Ctrl+L logs, Ctrl+D quit."
	pad := lipgloss.Width(ui.InfoStyle.Render(""))

	lines := []string{}
	if view != "" {
		lines = strings.Split(view, "\n")
	}
	lines = append([]string{""}, lines...)

	stacked := m.ui.width < helpReflowWidth
	available := m.availableInfoWidth(lipgloss.Width(help), pad, stacked)
	if runewidth.StringWidth(info) > available {
		info = runewidth.Truncate(info, available, "")
	}
	lines[0] = m.renderFirstLine(info, help, available, pad, stacked)
	lines = m.renderSecondLine(lines, help, stacked)

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

// OverlayHelp wraps overlayHelp to satisfy component interfaces.
func (m *model) OverlayHelp(view string) string { return m.overlayHelp(view) }

// View renders the application UI based on the current mode.
func (m *model) View() string {
	if c, ok := m.components[m.CurrentMode()]; ok {
		return c.View()
	}
	return ""
}
