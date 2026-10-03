package emqutiti

import (
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/marang/emqutiti/constants"
)

func panelBorderMouse(t *testing.T, m *model, id string) tea.MouseMsg {
	t.Helper()
	m.View()
	y := m.ui.elemPos[id] + m.ui.elemHeight[id] - 2
	m.ui.viewport.SetYOffset(max(0, y-m.ui.viewport.Height/2))
	m.View()
	msg := tea.MouseMsg{X: 5, Y: m.clientViewportTop() + y - m.ui.viewport.YOffset, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft}
	if got := m.panelResizeBorderAt(msg); got != id {
		t.Fatalf("border at (%d,%d) identifies %q, want %q", msg.X, msg.Y, got, id)
	}
	return msg
}

func TestPanelResizeDragPreservesStateAndReleaseOutside(t *testing.T) {
	for _, id := range []string{idTopics, idMessage, idHistory} {
		t.Run(id, func(t *testing.T) {
			m := reviewFixture(t, 120, 40)
			m.SetMode(constants.ModeClient)
			m.SetFocus(idTopic)
			m.topics.Input.SetValue("unfinished/topic")
			m.topics.SetSelected(2)
			m.history.List().Select(3)
			m.history.SetFilterQuery("topic=site/#")
			draft, focused, selected := m.message.Input().Value(), m.FocusedID(), m.topics.Selected()
			press := panelBorderMouse(t, m, id)
			spec, _ := m.panelResizeSpec(id)
			original := spec.config.height
			m.Update(press)
			if m.ui.panelResize.id != id || !strings.Contains(m.contextHelpText(), "Resizing "+spec.name) {
				t.Fatal("bottom-border press did not capture resize or update context")
			}
			motion := press
			motion.Action, motion.Button = tea.MouseActionMotion, tea.MouseButtonLeft
			motion.X, motion.Y = -5, press.Y+2*spec.linesPerUnit
			m.Update(motion)
			if spec.config.height != original+2 {
				t.Fatalf("motion resized to %d, want %d", spec.config.height, original+2)
			}
			for _, key := range []tea.KeyMsg{{Type: tea.KeyCtrlS}, {Type: tea.KeyCtrlE}, {Type: tea.KeyEnter}, {Type: tea.KeyTab}, {Type: tea.KeyDelete}, {Type: tea.KeyRunes, Runes: []rune{'p'}}, {Type: tea.KeyRunes, Runes: []rune{'a'}}} {
				m.Update(key)
			}
			if m.message.Input().Value() != draft || m.topics.Input.Value() != "unfinished/topic" || m.FocusedID() != focused || m.topics.Selected() != selected || m.history.FilterQuery() != "topic=site/#" || m.history.List().Index() != 3 || m.pendingPublishes() != 0 {
				t.Fatal("resize changed user state or dispatched a publish")
			}
			motion.Action, motion.Button = tea.MouseActionRelease, tea.MouseButtonNone
			m.Update(motion)
			if m.ui.panelResize.id != "" || spec.config.height != original+2 {
				t.Fatal("release outside the box did not accept the resize")
			}
			m.SetMode(constants.ModeTopics)
			m.SetMode(constants.ModeClient)
			if spec.config.height != original+2 {
				t.Fatal("view switch lost an accepted height")
			}
		})
	}
}

func TestPanelResizeEscapeAndModeSwitchRollback(t *testing.T) {
	for _, id := range []string{idTopics, idMessage, idHistory} {
		for _, modeSwitch := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/mode-switch=%t", id, modeSwitch), func(t *testing.T) {
				m := reviewFixture(t, 80, 40)
				m.SetMode(constants.ModeClient)
				press := panelBorderMouse(t, m, id)
				spec, _ := m.panelResizeSpec(id)
				original, offset := spec.config.height, m.ui.viewport.YOffset
				m.Update(press)
				motion := press
				motion.Action, motion.Y = tea.MouseActionMotion, press.Y+3*spec.linesPerUnit
				m.Update(motion)
				if modeSwitch {
					m.SetMode(constants.ModeConnections)
				} else {
					m.Update(tea.KeyMsg{Type: tea.KeyEsc})
				}
				if m.ui.panelResize.id != "" || spec.config.height != original || m.ui.viewport.YOffset != offset {
					t.Fatal("cancel did not restore height and scroll anchor")
				}
			})
		}
	}
}

func TestPanelResizeRenderedShrinkRestoresScrollAnchor(t *testing.T) {
	m := reviewFixture(t, 80, 24)
	m.SetMode(constants.ModeClient)
	m.setPanelHeight(idHistory, 16)
	press := panelBorderMouse(t, m, idHistory)
	offset := m.ui.viewport.YOffset
	if offset == 0 {
		t.Fatal("fixture must exercise a scrolled viewport")
	}
	m.Update(press)
	motion := press
	motion.Action, motion.Y = tea.MouseActionMotion, press.Y-7
	m.Update(motion)
	m.View()
	m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m.View()
	if m.layout.history.height != 16 || m.ui.viewport.YOffset != offset {
		t.Fatalf("rollback height=%d offset=%d, want 16/%d", m.layout.history.height, m.ui.viewport.YOffset, offset)
	}
}

func TestPanelResizeHistoryMinimumFollowsExpandedHelp(t *testing.T) {
	for _, size := range reviewSizes {
		m := reviewFixture(t, size.width, size.height)
		m.SetMode(constants.ModeClient)
		m.SetFocus(idHistory)
		m.setPanelHeight(idHistory, 0)
		short := m.layout.history.height
		m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
		m.View()
		spec, _ := m.panelResizeSpec(idHistory)
		if !m.history.List().Help.ShowAll || spec.min <= short || m.layout.history.height < spec.min {
			t.Fatalf("expanded help height=%d min=%d short=%d at %dx%d", m.layout.history.height, spec.min, short, size.width, size.height)
		}
		m.history.SetFilterQuery("topic=site/#")
		m.View()
		spec, _ = m.panelResizeSpec(idHistory)
		if m.layout.history.height < spec.min {
			t.Fatal("filter summary clipped expanded help")
		}
		captureReviewView(t, "panel-full-help-dark", m.View(), size.width, size.height)
	}
}

func TestPanelResizeLimitsKeyboardAndReset(t *testing.T) {
	for _, size := range reviewSizes {
		for _, id := range []string{idTopics, idMessage, idHistory} {
			t.Run(fmt.Sprintf("%s/%dx%d", id, size.width, size.height), func(t *testing.T) {
				m := reviewFixture(t, size.width, size.height)
				m.SetMode(constants.ModeClient)
				m.SetFocus(id)
				spec, _ := m.panelResizeSpec(id)
				m.setPanelHeight(id, -100)
				if spec.config.height != spec.min {
					t.Fatal("resize went below the minimum height")
				}
				m.Update(tea.KeyMsg{Type: tea.KeyCtrlShiftUp})
				if spec.config.height != spec.min {
					t.Fatal("keyboard resize went below the minimum height")
				}
				m.Update(tea.KeyMsg{Type: tea.KeyCtrlShiftDown})
				if spec.config.height != min(spec.min+1, spec.max) {
					t.Fatal("keyboard resize did not use the shared limits")
				}
				m.setPanelHeight(id, 1000)
				if spec.config.height != spec.max {
					t.Fatal("resize went above the maximum height")
				}
				m.Update(tea.KeyMsg{Type: tea.KeyCtrlR})
				defaults := initLayout()
				want := map[string]int{idTopics: defaults.topics.height, idMessage: defaults.message.height, idHistory: defaults.history.height}[id]
				if spec.config.height != min(max(spec.min, want), spec.max) {
					t.Fatal("reset did not restore the bounded default height")
				}
				view := m.View()
				captureReviewView(t, "panel-reset-"+id+"-dark", view, size.width, size.height)
			})
		}
	}
}

func TestPanelResizeUsesScrolledRenderedBorders(t *testing.T) {
	for _, size := range reviewSizes {
		for _, id := range []string{idTopics, idMessage, idHistory} {
			t.Run(fmt.Sprintf("%s/%dx%d", id, size.width, size.height), func(t *testing.T) {
				m := reviewFixture(t, size.width, size.height)
				m.SetMode(constants.ModeClient)
				press := panelBorderMouse(t, m, id)
				m.Update(tea.MouseMsg{X: press.X, Y: press.Y, Action: tea.MouseActionMotion, Button: tea.MouseButtonNone})
				if !strings.Contains(m.contextHelpText(), "drag bottom border") {
					t.Fatal("bottom-border hover did not explain the resize action")
				}
				m.Update(press)
				if m.ui.panelResize.id != id {
					t.Fatal("scrolled border did not capture the correct panel")
				}
				motion := press
				motion.Action, motion.Y = tea.MouseActionMotion, press.Y+1000
				m.Update(motion)
				m.View()
				spec, _ := m.panelResizeSpec(id)
				if spec.config.height != spec.max {
					t.Fatal("drag did not clamp at the upper limit")
				}
				if !strings.Contains(ansi.Strip(m.renderContextHelp()), "[Esc] cancel") || lipgloss.Height(m.renderContextHelp()) != 2 {
					t.Fatal("resize changed the two-row fixed context help")
				}
				captureReviewView(t, "panel-drag-"+id+"-dark", m.View(), size.width, size.height)
				m.Update(tea.KeyMsg{Type: tea.KeyEsc})
			})
		}
	}
}

func TestPanelResizeIgnoresNonBordersAndModals(t *testing.T) {
	m := reviewFixture(t, 80, 40)
	m.SetMode(constants.ModeClient)
	press := panelBorderMouse(t, m, idMessage)
	for _, msg := range []tea.MouseMsg{
		{X: press.X, Y: press.Y - 1, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft},
		{X: 0, Y: press.Y, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft},
		{X: press.X, Y: press.Y, Action: tea.MouseActionPress, Button: tea.MouseButtonRight},
		{X: press.X, Y: 0, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft},
	} {
		if _, handled := m.handlePanelResize(msg); handled || m.ui.panelResize.id != "" {
			t.Fatal("non-resizable surface started a drag")
		}
	}
	m.StartConfirm("Delete?", "", nil, nil, nil)
	m.Update(press)
	if m.ui.panelResize.id != "" || m.CurrentMode() != constants.ModeConfirmDelete {
		t.Fatal("client resize took input away from a modal")
	}
}

func TestPanelResizeTerminalChangeCancelsAndClamps(t *testing.T) {
	m := reviewFixture(t, 120, 40)
	m.SetMode(constants.ModeClient)
	m.setPanelHeight(idMessage, 30)
	m.setPanelHeight(idHistory, 30)
	press := panelBorderMouse(t, m, idTopics)
	m.Update(press)
	m.Update(tea.WindowSizeMsg{Width: 40, Height: 16})
	if m.ui.panelResize.id != "" {
		t.Fatal("terminal resize left an old-coordinate drag active")
	}
	for _, id := range []string{idTopics, idMessage, idHistory} {
		spec, _ := m.panelResizeSpec(id)
		if spec.config.height < spec.min || spec.config.height > spec.max {
			t.Fatalf("terminal resize did not constrain %s", id)
		}
	}
	captureReviewView(t, "panel-terminal-resize-dark", m.View(), 40, 16)
}
