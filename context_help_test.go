package emqutiti

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/marang/emqutiti/topics"
)

func TestFormatTopicNames(t *testing.T) {
	got := formatTopicNames([]string{"a", "b", "c", "d"}, 3)
	if got != "a, b, c +1 more" {
		t.Fatalf("unexpected topic list %q", got)
	}
	if got := formatTopicNames(nil, 3); got != "none" {
		t.Fatalf("expected none, got %q", got)
	}
}

func TestTopicStateLegendAvoidsModePrefixes(t *testing.T) {
	legend := topicStateLegend()
	for _, unwanted := range []string{"rw", "r ", "w ", "x "} {
		if strings.Contains(legend, unwanted) {
			t.Fatalf("legend should not include compact mode prefix %q: %q", unwanted, legend)
		}
	}
}

func TestFocusContextUsesKeyboardTarget(t *testing.T) {
	m, _ := initialModel(nil)
	m.topics.Items = []topics.Item{{Name: "a", Subscribed: true}}
	m.topics.SetSelected(0)
	m.SetFocus(idTopics)

	help := m.contextHelpText()
	if !strings.HasPrefix(help, "> ") || !strings.Contains(help, "Enter unsubscribes") {
		t.Fatalf("expected focus topic action help, got %q", help)
	}
}

func TestFocusContextForTopicInputValue(t *testing.T) {
	m, _ := initialModel(nil)
	m.topics.Input.SetValue("sensors/#")
	m.SetFocus(idTopic)

	help := m.contextHelpText()
	if !strings.Contains(help, `Enter subscribes to "sensors/#"`) {
		t.Fatalf("expected input action help, got %q", help)
	}
}

func TestHoverTopicChipSetsStateWithoutSelection(t *testing.T) {
	m, _ := initialModel(nil)
	m.Update(tea.WindowSizeMsg{Width: 80, Height: 30})
	m.topics.Items = []topics.Item{
		{Name: "a", Subscribed: true},
		{Name: "b", Subscribed: true, Publish: true},
	}
	m.topics.SetSelected(0)
	m.viewClient()
	if len(m.topics.ChipBounds) < 2 {
		t.Fatalf("expected chip bounds, got %d", len(m.topics.ChipBounds))
	}

	b := m.topics.ChipBounds[1]
	m.updateHoverState(tea.MouseMsg{X: b.XPos, Y: m.clientViewportTop() + b.YPos - m.ui.viewport.YOffset})

	if m.topics.Selected() != 0 {
		t.Fatalf("hover changed selection to %d", m.topics.Selected())
	}
	if m.ui.hoveredID != idTopics || m.ui.hoveredTopic != 1 {
		t.Fatalf("unexpected hover state id=%q topic=%d", m.ui.hoveredID, m.ui.hoveredTopic)
	}
	if strings.Contains(m.contextHelpText(), `"b"`) || !strings.Contains(m.contextHelpText(), "Read subscription and publish target") {
		t.Fatalf("expected topic hover help, got %q", m.contextHelpText())
	}
}

func TestContextHelpRendersTwoLines(t *testing.T) {
	m, _ := initialModel(nil)
	m.Update(tea.WindowSizeMsg{Width: 80, Height: 30})
	m.SetFocus(idTopics)

	help := m.renderContextHelp()
	if strings.Count(help, "\n") != 1 || strings.Contains(help, topicStateLegend()) || !strings.Contains(help, "Enter toggles subscribe") {
		t.Fatalf("expected two-line context help without topic legend, got %q", help)
	}
}

func TestMessageHoverUsesPublishTargets(t *testing.T) {
	m, _ := initialModel(nil)
	m.Update(tea.WindowSizeMsg{Width: 80, Height: 30})
	m.topics.Items = []topics.Item{
		{Name: "a", Publish: true},
		{Name: "b", Publish: true},
	}
	m.viewClient()

	m.updateHoverState(tea.MouseMsg{X: 1, Y: m.clientViewportTop() + m.ui.elemPos[idMessage] + 1 - m.ui.viewport.YOffset})

	if !strings.Contains(m.contextHelpText(), "a, b") {
		t.Fatalf("expected publish targets in message help, got %q", m.contextHelpText())
	}
}

func TestMessageNoTargetHelp(t *testing.T) {
	m, _ := initialModel(nil)
	m.SetFocus(idMessage)

	if got := m.messageTargetPreview(); got != "Message -> no target" {
		t.Fatalf("unexpected target preview %q", got)
	}
	if !strings.Contains(m.contextHelpText(), "no publish target") {
		t.Fatalf("expected no-target help, got %q", m.contextHelpText())
	}
}

func TestTabClearsHoverContext(t *testing.T) {
	m, _ := initialModel(nil)
	m.Update(tea.WindowSizeMsg{Width: 80, Height: 30})
	m.topics.Items = []topics.Item{{Name: "a", Subscribed: true}}
	m.SetFocus(idTopic)
	m.viewClient()
	m.updateHoverState(tea.MouseMsg{X: 1, Y: m.clientViewportTop() + m.ui.elemPos[idMessage] + 1})
	if !strings.Contains(m.contextHelpText(), "~ Message") {
		t.Fatalf("expected hover message context, got %q", m.contextHelpText())
	}

	m.Update(tea.KeyMsg{Type: tea.KeyTab})

	if m.ui.hoveredID != "" || !strings.Contains(m.contextHelpText(), "> Topic") {
		t.Fatalf("expected focus context after tab, id=%q help=%q", m.ui.hoveredID, m.contextHelpText())
	}
}

func TestTopicInputTypingDoesNotScrollViewport(t *testing.T) {
	m, _ := initialModel(nil)
	m.Update(tea.WindowSizeMsg{Width: 80, Height: 30})
	m.viewClient()
	m.SetFocus(idTopic)
	m.ui.viewport.SetYOffset(4)

	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})

	if got := m.ui.viewport.YOffset; got != 4 {
		t.Fatalf("typing in topic input changed viewport offset to %d", got)
	}
}
