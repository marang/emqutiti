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
	m.ui.width = helpReflowWidth - 20
	msg := tea.MouseMsg{X: m.ui.width - 1, Y: 1, Type: tea.MouseLeft, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress}
	m.Update(msg)
	if m.CurrentMode() != constants.ModeHelp {
		t.Fatalf("expected ModeHelp, got %v", m.CurrentMode())
	}
}

func TestHelpIconHoverDoesNotOpenHelp(t *testing.T) {
	m, _ := initialModel(nil)
	m.ui.width = helpReflowWidth - 20
	msg := tea.MouseMsg{X: m.ui.width - 1, Y: 1, Type: tea.MouseMotion, Action: tea.MouseActionMotion}
	m.Update(msg)
	if m.CurrentMode() == constants.ModeHelp {
		t.Fatalf("hover should not open help")
	}
}

// Test that the help icon reflows to the second line when space is limited.
func TestHelpIconReflows(t *testing.T) {
	m, _ := initialModel(nil)
	m.ui.width = helpReflowWidth - 20
	view := m.overlayHelp("")
	lines := strings.Split(view, "\n")
	if len(lines) < 2 || strings.Contains(lines[0], "?") || !strings.Contains(lines[1], "?") {
		t.Fatalf("help icon did not reflow: %q", view)
	}
}
