package emqutiti

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/marang/emqutiti/constants"
)

// Test that clicking the help icon opens the help view.
func TestHelpIconClick(t *testing.T) {
	m, _ := initialModel(nil)
	m.ui.width = 40
	msg := tea.MouseMsg{X: m.ui.width - 1, Y: 0, Type: tea.MouseLeft, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress}
	m.Update(msg)
	if m.CurrentMode() != constants.ModeHelp {
		t.Fatalf("expected ModeHelp, got %v", m.CurrentMode())
	}
}

func TestHelpIconHoverDoesNotOpenHelp(t *testing.T) {
	m, _ := initialModel(nil)
	m.ui.width = 40
	msg := tea.MouseMsg{X: m.ui.width - 1, Y: 0, Type: tea.MouseMotion, Action: tea.MouseActionMotion}
	m.Update(msg)
	if m.CurrentMode() == constants.ModeHelp {
		t.Fatalf("hover should not open help")
	}
}

func TestHelpIconUsesOneHeaderRow(t *testing.T) {
	m, _ := initialModel(nil)
	m.ui.width = 40
	view := m.overlayHelp("")
	lines := strings.Split(view, "\n")
	if len(lines) != 1 || !strings.Contains(lines[0], "?") {
		t.Fatalf("help icon must stay in the header: %q", view)
	}
}
