package emqutiti

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"

	"github.com/marang/emqutiti/ui"
)

func (m *model) overlayHelp(view string) string {
	help := ui.HelpStyle.Render("?")
	if m.help.Focused() {
		help = ui.HelpFocused.Render("?")
	} else if m.ui.hoveredID == idHelp {
		help = ui.HelpHovered.Render("?")
	}
	m.ui.elemPos[idHelp] = 0

	actions := []string{"[Ctrl+B] brokers", "[Ctrl+D] quit", "[Ctrl+T] topics", "[Ctrl+P] payloads", "[Alt+R] traces", "[Ctrl+L] logs"}
	pad := lipgloss.Width(ui.InfoStyle.Render(""))

	lines := []string{}
	if view != "" {
		lines = strings.Split(view, "\n")
	}
	lines = append([]string{""}, lines...)

	available := max(0, m.ui.width-pad-lipgloss.Width(help))
	info := ""
	if available >= 105 {
		info = "Switch views:"
	}
	for _, action := range actions {
		candidate := action
		if info != "" {
			candidate = info + "  " + action
		}
		if runewidth.StringWidth(candidate) > available {
			break
		}
		info = candidate
	}
	if runewidth.StringWidth(info) > available {
		info = runewidth.Truncate(info, available, "")
	}
	lines[0] = lipgloss.JoinHorizontal(lipgloss.Top,
		lipgloss.NewStyle().Width(available+pad).Render(ui.InfoStyle.Render(info)), help)

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
