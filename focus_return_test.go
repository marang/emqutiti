package emqutiti

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/marang/emqutiti/constants"
)

func TestHistoryDetailReturnRestoresHistoryFocus(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	m, _ := initialModel(nil)

	historyIdx := 3 // index of history in client focus order
	m.focus.Set(historyIdx)
	m.ui.focusIndex = historyIdx

	// Enter history detail mode
	m.SetMode(constants.ModeHistoryDetail)
	if m.CurrentMode() != constants.ModeHistoryDetail {
		t.Fatalf("expected history detail mode, got %v", m.CurrentMode())
	}

	// Escape back to previous mode
	if cmd := m.history.UpdateDetail(tea.KeyMsg{Type: tea.KeyEsc}); cmd != nil {
		cmd()
	}

	if m.CurrentMode() != constants.ModeClient {
		t.Fatalf("expected client mode after returning, got %v", m.CurrentMode())
	}
	if m.FocusedID() != idHistory {
		t.Fatalf("expected history to regain focus, got %s", m.FocusedID())
	}
}
