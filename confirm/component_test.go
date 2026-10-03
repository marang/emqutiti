package confirm

import (
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

type testNavigator struct {
	width, height int
	returned      bool
}

func (*testNavigator) SetConfirmMode() tea.Cmd    { return nil }
func (n *testNavigator) SetPreviousMode() tea.Cmd { n.returned = true; return nil }
func (n *testNavigator) Width() int               { return n.width }
func (n *testNavigator) Height() int              { return n.height }
func (*testNavigator) ScrollToFocused()           {}
func (*testNavigator) ListenStatus() tea.Cmd      { return nil }

func TestLongConfirmationFitsAndScrolls(t *testing.T) {
	for _, size := range []struct{ width, height int }{{120, 40}, {80, 24}, {40, 16}, {32, 12}} {
		t.Run(fmt.Sprintf("%dx%d", size.width, size.height), func(t *testing.T) {
			nav := &testNavigator{width: size.width, height: size.height}
			d := NewDialog(nav, nav, nil, nil, nil)
			d.Start("Delete "+strings.Repeat("target", 60)+"? [y/n]", strings.Repeat("details\n", 60)+"TAIL")
			for i := 0; i < 30; i++ {
				view := d.View()
				if w, h := lipgloss.Width(view), lipgloss.Height(view); w > size.width || h > size.height {
					t.Fatalf("confirmation %dx%d exceeds terminal %dx%d", w, h, size.width, size.height)
				}
				if !strings.Contains(view, "[y/n]") {
					t.Fatal("confirmation decisions are not visible")
				}
				d.Update(tea.KeyMsg{Type: tea.KeyPgDown})
			}
			if !strings.Contains(d.View(), "TAIL") {
				t.Fatal("long confirmation tail is not reachable by scrolling")
			}
		})
	}
}

func TestConfirmationDecisionsRemainReachable(t *testing.T) {
	for _, accept := range []bool{true, false} {
		t.Run(fmt.Sprintf("accept=%t", accept), func(t *testing.T) {
			nav := &testNavigator{width: 32, height: 12}
			acted, cancelled, refocused := 0, 0, 0
			d := NewDialog(nav, nav, func() tea.Cmd { refocused++; return nil }, func() tea.Cmd { acted++; return nil }, func() { cancelled++ })
			d.Start(strings.Repeat("long target ", 40)+"[y/n]", "")
			d.Update(tea.KeyMsg{Type: tea.KeyPgDown})
			letter := "n"
			if accept {
				letter = "y"
			}
			d.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(letter)})
			if !nav.returned || refocused != 1 || (accept && (acted != 1 || cancelled != 0)) || (!accept && (acted != 0 || cancelled != 1)) {
				t.Fatalf("bad decision result: returned=%t acted=%d cancelled=%d refocused=%d", nav.returned, acted, cancelled, refocused)
			}
		})
	}
}

func TestConfirmationMouseDecisions(t *testing.T) {
	for _, letter := range []string{"y", "n"} {
		t.Run(letter, func(t *testing.T) {
			nav := &testNavigator{width: 80, height: 24}
			acted, cancelled := false, false
			d := NewDialog(nav, nav, nil, func() tea.Cmd { acted = true; return nil }, func() { cancelled = true })
			d.Start("Delete target? [y/n]", strings.Repeat("details\n", 60))
			for y, line := range strings.Split(ansi.Strip(d.View()), "\n") {
				if start := strings.Index(line, "[y/n]"); start >= 0 {
					x := lipgloss.Width(line[:start]) + 1
					if letter == "n" {
						x += 2
					}
					d.Update(tea.MouseMsg{X: x, Y: y, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft})
					break
				}
			}
			if !nav.returned || (letter == "y" && !acted) || (letter == "n" && !cancelled) {
				t.Fatal("mouse click did not reach its confirmation decision")
			}
		})
	}
}
