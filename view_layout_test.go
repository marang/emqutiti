package emqutiti

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestOverlayHelpStaysInlineOnNarrowWidth(t *testing.T) {
	m, _ := initialModel(nil)
	m.Update(tea.WindowSizeMsg{Width: 40, Height: 16})
	out := m.overlayHelp("status")
	lines := strings.Split(out, "\n")
	if len(lines) < 2 {
		t.Fatalf("expected at least two lines, got %d", len(lines))
	}
	if !strings.Contains(lines[0], "?") || !strings.Contains(lines[0], "[Ctrl+B] brokers") {
		t.Fatalf("expected help and complete action on first line: %q", lines[0])
	}
	if strings.Contains(lines[1], "?") || strings.TrimSpace(lines[1]) != "status" {
		t.Fatalf("expected unchanged content on second line: %q", lines[1])
	}
}

func TestOverlayHelpInlineOnWideWidth(t *testing.T) {
	m, _ := initialModel(nil)
	m.Update(tea.WindowSizeMsg{Width: 70, Height: 20})
	out := m.overlayHelp("status")
	lines := strings.Split(out, "\n")
	if !strings.Contains(lines[0], "?") || !strings.Contains(lines[0], "[Ctrl+B] brokers") || !strings.Contains(lines[0], "[Ctrl+D] quit") {
		t.Fatalf("expected help and info on first line: %q", lines[0])
	}
	if len(lines) < 2 || strings.Contains(lines[1], "?") || !strings.Contains(lines[1], "status") {
		t.Fatalf("unexpected second line: %v", lines)
	}
}
