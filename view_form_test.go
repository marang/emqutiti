package emqutiti

import (
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/marang/emqutiti/connections"
	"github.com/marang/emqutiti/constants"
	"github.com/marang/emqutiti/ui"
)

func brokerEditorModel(t *testing.T, width, height int) *model {
	t.Helper()
	t.Setenv("EMQUTITI_HOME", t.TempDir())
	mgr := connections.NewConnectionsModel()
	m, err := initialModel(&mgr)
	if err != nil {
		t.Fatal(err)
	}
	m.Update(tea.WindowSizeMsg{Width: width, Height: height})
	f := connections.NewForm(connections.Profile{
		Name: "draft", Host: strings.Repeat("host", 30), LastWillPayload: "keep this",
	}, -1)
	m.connections.Form = &f
	m.SetMode(constants.ModeEditConnection)
	return m
}

func assertBrokerEditorBounds(t *testing.T, m *model, width, height int) string {
	t.Helper()
	view := m.viewForm()
	if got := lipgloss.Height(view); got != height {
		t.Fatalf("height=%d want=%d\n%s", got, height, ansi.Strip(view))
	}
	for y, line := range strings.Split(view, "\n") {
		if got := lipgloss.Width(line); got > width {
			t.Fatalf("row %d width=%d exceeds %d: %q", y, got, width, ansi.Strip(line))
		}
	}
	return ansi.Strip(view)
}

func TestBrokerEditorFitsTerminalSizesAndTabsToLastField(t *testing.T) {
	for _, size := range []struct{ width, height int }{{120, 40}, {80, 24}, {60, 20}, {40, 16}} {
		t.Run(fmt.Sprintf("%dx%d", size.width, size.height), func(t *testing.T) {
			m := brokerEditorModel(t, size.width, size.height)
			view := assertBrokerEditorBounds(t, m, size.width, size.height)
			if got, want := strings.Contains(view, "Brokers"), size.width >= 100; got != want {
				t.Fatalf("broker sidebar visible=%v want=%v", got, want)
			}
			for i := 1; i < len(m.connections.Form.Fields); i++ {
				m.Update(tea.KeyMsg{Type: tea.KeyTab})
				if m.connections.Form.Focus != i {
					t.Fatalf("Tab focus=%d want=%d", m.connections.Form.Focus, i)
				}
				view = assertBrokerEditorBounds(t, m, size.width, size.height)
			}
			if !strings.Contains(view, "Last Will Payload:") {
				t.Fatalf("last field hidden after Tab\n%s", view)
			}
			p, err := m.connections.Form.Profile()
			if err != nil || p.Name != "draft" || p.Host != strings.Repeat("host", 30) || p.LastWillPayload != "keep this" {
				t.Fatalf("Tab lost values: profile=%#v err=%v", p, err)
			}
		})
	}
}

func TestBrokerEditorMouseFocusAfterScrollAndOptionExpansion(t *testing.T) {
	for _, size := range []struct{ width, height int }{{120, 40}, {80, 24}, {60, 20}, {40, 16}} {
		t.Run(fmt.Sprintf("%dx%d", size.width, size.height), func(t *testing.T) {
			m := brokerEditorModel(t, size.width, size.height)
			m.connections.Form.Focus = 2 // Schema expands six option rows.
			m.connections.Form.ApplyFocus()
			assertBrokerEditorBounds(t, m, size.width, size.height)
			l := brokerFormLayout(size.width, size.height)
			for range 80 {
				m.Update(tea.MouseMsg{X: l.formX + 2, Y: 4, Action: tea.MouseActionPress, Button: tea.MouseButtonWheelDown})
			}
			view := assertBrokerEditorBounds(t, m, size.width, size.height)
			clicked := false
			for y, line := range strings.Split(view, "\n") {
				if strings.Contains(line, "Last Will Payload:") {
					m.Update(tea.MouseMsg{X: l.formX + 3, Y: y, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft})
					clicked = true
					break
				}
			}
			if !clicked || m.connections.Form.Focus != len(m.connections.Form.Fields)-1 {
				t.Fatalf("last field click missed: clicked=%v focus=%d\n%s", clicked, m.connections.Form.Focus, view)
			}
			assertBrokerEditorBounds(t, m, size.width, size.height)
		})
	}
}

func TestBrokerEditorOutsideClicksPreserveForm(t *testing.T) {
	m := brokerEditorModel(t, 120, 40)
	m.viewForm()
	for _, point := range []struct{ x, y int }{{0, 2}, {59, 2}, {60, 2}, {119, 2}, {65, 1}, {65, 39}, {65, 40}} {
		m.Update(tea.MouseMsg{X: point.x, Y: point.y, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft})
		if m.connections.Form.Focus != 0 || m.connections.Form.Fields[0].(*ui.CheckField).Bool() {
			t.Fatalf("outside click at %v changed the form", point)
		}
	}
}

func TestBrokerEditorCtrlScrollRoutesToForm(t *testing.T) {
	for _, down := range []tea.KeyType{tea.KeyCtrlDown, tea.KeyCtrlJ} {
		m := brokerEditorModel(t, 80, 24)
		before := ansi.Strip(m.viewForm())
		m.Update(tea.KeyMsg{Type: down})
		after := ansi.Strip(m.viewForm())
		if before == after || m.connections.Form.Focus != 0 || m.ui.viewport.YOffset != 0 {
			t.Fatalf("%v did not scroll just the form", down)
		}
		up := tea.KeyCtrlUp
		if down == tea.KeyCtrlJ {
			up = tea.KeyCtrlK
		}
		m.Update(tea.KeyMsg{Type: up})
		if got := ansi.Strip(m.viewForm()); got != before {
			t.Fatalf("%v did not restore form scroll", up)
		}
	}
}

func TestBrokerEditorSaveKeyboardAndMouse(t *testing.T) {
	for _, mouse := range []bool{false, true} {
		t.Run(fmt.Sprintf("mouse=%v", mouse), func(t *testing.T) {
			m := brokerEditorModel(t, 40, 16)
			m.viewForm()
			if mouse {
				m.Update(tea.MouseMsg{X: 2, Y: 14, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft})
			} else {
				m.Update(tea.KeyMsg{Type: tea.KeyEnter})
			}
			if m.connections.Form != nil || m.CurrentMode() != constants.ModeConnections {
				t.Fatal("save did not leave broker editor")
			}
			profiles := m.connections.Manager.Profiles
			if len(profiles) != 1 || profiles[0].Name != "draft" || profiles[0].LastWillPayload != "keep this" {
				t.Fatalf("saved profile=%#v", profiles)
			}
		})
	}
}

func TestBrokerEditorValidationAndCancelKeyboardAndMouse(t *testing.T) {
	for _, mouse := range []bool{false, true} {
		t.Run(fmt.Sprintf("mouse=%v", mouse), func(t *testing.T) {
			m := brokerEditorModel(t, 40, 16)
			m.connections.Form.Fields[4].(*ui.TextField).SetValue("invalid") // Port
			m.connections.Form.Focus = len(m.connections.Form.Fields) - 1
			m.connections.Form.ApplyFocus()
			m.viewForm()
			if mouse {
				m.Update(tea.MouseMsg{X: 2, Y: 14, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft})
			} else {
				m.Update(tea.KeyMsg{Type: tea.KeyEnter})
			}
			if m.connections.Form == nil || len(m.connections.Manager.Profiles) != 0 {
				t.Fatal("invalid form saved or lost")
			}
			view := assertBrokerEditorBounds(t, m, 40, 16)
			if m.connections.Form.Focus != 4 || !strings.Contains(view, "Port:") || !strings.Contains(view, "Enter a whole number.") {
				t.Fatalf("validation not reachable\n%s", view)
			}
			if m.connections.Form.Fields[len(m.connections.Form.Fields)-1].Value() != "keep this" {
				t.Fatal("validation lost draft")
			}
			if mouse {
				m.Update(tea.MouseMsg{X: 16, Y: 14, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft})
			} else {
				m.Update(tea.KeyMsg{Type: tea.KeyEsc})
			}
			if m.connections.Form != nil || m.CurrentMode() != constants.ModeConnections {
				t.Fatal("cancel did not leave editor")
			}
		})
	}
}
