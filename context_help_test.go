package emqutiti

import (
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"

	"github.com/marang/emqutiti/internal/clipboardutil"
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
	if !strings.HasPrefix(help, "> ") || !strings.Contains(help, "Read subscription") || strings.Contains(help, "Enter") {
		t.Fatalf("expected focus topic action help, got %q", help)
	}
}

func TestTopicInputDetailNeverPromisesChipActions(t *testing.T) {
	m, _ := initialModel(nil)
	m.SetFocus(idTopic)
	for _, value := range []string{"", "probe", "existing"} {
		m.topics.Items = []topics.Item{{Name: "existing"}}
		m.topics.Input.SetValue(value)
		help := m.contextHelpDetailText()
		for _, action := range []string{"toggle", "[Del]", "[p]"} {
			if strings.Contains(help, action) {
				t.Fatalf("input promises chip action for %q: %q", value, help)
			}
		}
	}
}

func TestHistoryContextSeparatesStateAndActions(t *testing.T) {
	m := reviewFixture(t, 40, 16)
	m.SetFocus(idHistory)
	detail := m.contextHelpDetailText()
	if strings.Contains(detail, "messages") || !strings.Contains(detail, "[Enter] view") || !strings.Contains(detail, "[/] find") || !strings.Contains(m.renderContextHelp(), "[Ctrl+C] copy") {
		t.Fatalf("history actions are duplicated or clipped: %q", detail)
	}
}

func TestPublishHintOnlyOffersExtendedKeysWithAdapter(t *testing.T) {
	m := reviewFixture(t, 80, 24)
	m.mqttClient = &MQTTClient{Client: &fakeClient{}}
	m.SetFocus(idMessage)
	if strings.Contains(m.contextHelpDetailText(), "Ctrl+Enter") {
		t.Fatal("extended shortcut offered without its input adapter")
	}
	m.ui.modifiedKeyInput = true
	if !strings.Contains(m.contextHelpDetailText(), "[Ctrl+Enter*]") {
		t.Fatal("extended shortcut missing on supported input")
	}
	m.ui.width = 40
	if strings.Contains(m.contextHelpDetailText(), "Ctrl+Enter") || !strings.Contains(m.contextHelpDetailText(), "[Ctrl+E]") {
		t.Fatal("narrow extended-key hint hid the portable commands")
	}
}

func TestHistoryContextExplainsRangeSelectionAndCopy(t *testing.T) {
	for _, width := range []int{40, 80} {
		t.Run(fmt.Sprint(width), func(t *testing.T) {
			m := reviewFixture(t, width, 24)
			m.SetFocus(idHistory)
			help := ansi.Strip(m.renderContextHelp())
			if !strings.Contains(help, "[Shift+Up/Down]") || !strings.Contains(help, "[Ctrl+C] copy") {
				t.Fatalf("range selection or copy shortcut is missing: %q", help)
			}
			if width == 80 && !strings.Contains(help, "[Shift+Click]") {
				t.Fatalf("mouse range selection is missing: %q", help)
			}
			m.ui.hoveredID = idHistory
			if !strings.Contains(m.contextHelpText(), "[Shift+Up/Down]") {
				t.Fatal("hover help omitted range selection")
			}
			m.history.SetShowArchived(true)
			if strings.Contains(m.contextHelpText(), "[Shift+") {
				t.Fatal("archived history advertises unavailable selection shortcuts")
			}
			m.history.SetShowArchived(false)
			m.ui.hoveredID = idTopics
			if strings.Contains(m.contextHelpText(), "[Shift+") || strings.Contains(m.contextHelpDetailText(), "[Ctrl+C]") {
				t.Fatal("topic hover advertises history selection or copy")
			}
		})
	}
}

func TestHistoryShiftRangeAndCopyMatchHelp(t *testing.T) {
	originalCopy := clipboardutil.Copy
	t.Cleanup(func() { clipboardutil.Copy = originalCopy })
	var copied string
	clipboardutil.Copy = func(text string) error { copied = text; return nil }
	m := reviewFixture(t, 80, 24)
	m.SetFocus(idHistory)
	m.Update(tea.KeyMsg{Type: tea.KeyShiftDown})
	m.Update(tea.KeyMsg{Type: tea.KeyShiftDown})
	for i, item := range m.history.Items() {
		selected := item.IsSelected != nil && *item.IsSelected
		if selected != (i < 3) {
			t.Fatalf("Shift+Down selection mismatch at entry %d", i)
		}
	}
	m.Update(tea.KeyMsg{Type: tea.KeyShiftUp})
	m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	if copied != "site/item-00: payload-00\nsite/item-01: payload-01" {
		t.Fatalf("Ctrl+C did not copy the selected range: %q", copied)
	}
	hitems := m.history.Items()
	for i := range hitems {
		hitems[i].IsSelected = nil
	}
	m.history.SetItems(hitems)
	m.history.List().Select(3)
	m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	if copied != "site/item-03: payload-03" {
		t.Fatalf("Ctrl+C without a selection did not copy the current entry: %q", copied)
	}
}

func TestPendingHelpUsesSnapshotTargetsAndFailureIsVisible(t *testing.T) {
	m, _ := initialModel(nil)
	m.topics.Items = []topics.Item{{Name: "first", Publish: true}}
	m.message.SetPayload("draft")
	m.SetFocus(idMessage)
	cmd := m.handlePublishKey()
	m.topics.Items[0].Name = "changed"
	if got := m.messageTargetPreview(); !strings.Contains(got, "first") || strings.Contains(got, "changed") {
		t.Fatalf("pending label does not match in-flight targets: %q", got)
	}
	applyMQTTCommand(m, cmd)
	if !strings.Contains(m.contextHelpText(), "Publish failed for first") {
		t.Fatalf("failure missing from focused help: %q", m.contextHelpText())
	}
}

func TestFocusContextForTopicInputValue(t *testing.T) {
	m, _ := initialModel(nil)
	m.topics.Input.SetValue("sensors/#")
	m.SetFocus(idTopic)

	help := m.contextHelpText()
	if !strings.Contains(help, `[Enter] subscribes to "sensors/#"`) {
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
	if strings.Count(help, "\n") != 1 || strings.Contains(help, topicStateLegend()) || !strings.Contains(help, "[Enter] toggle sub") {
		t.Fatalf("expected two-line context help without topic legend, got %q", help)
	}
}

func TestContextShortcutKeysUseBrackets(t *testing.T) {
	m := reviewFixture(t, 80, 24)
	for _, tc := range []struct {
		name, focus, input string
		connected          bool
		keys               []string
	}{
		{"empty topic input", idTopic, "", false, []string{"[Tab]"}},
		{"new topic input", idTopic, "new/topic", false, []string{"[Enter]", "[Tab]"}},
		{"existing topic input", idTopic, "site/status", false, []string{"[Tab]"}},
		{"topic chips", idTopics, "", false, []string{"[Enter]", "[p]", "[Del]"}},
		{"disconnected editor", idMessage, "", false, []string{"[Ctrl+B]", "[Ctrl+S]"}},
		{"connected editor", idMessage, "", true, []string{"[Ctrl+S]", "[Ctrl+E]"}},
		{"history", idHistory, "", false, []string{"[Enter]", "[/]", "[Ctrl+C]"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m.mqttClient = nil
			if tc.connected {
				m.mqttClient = &MQTTClient{Client: &fakeClient{}}
			}
			m.topics.Input.SetValue(tc.input)
			m.clearHoverState()
			m.SetFocus(tc.focus)
			for _, key := range tc.keys {
				if !strings.Contains(m.contextHelpDetailText(), key) {
					t.Fatalf("shortcut %s lacks brackets: %q", key, m.contextHelpDetailText())
				}
			}
		})
	}
}

func TestContextShortcutsRemainVisibleOnNarrowTerminal(t *testing.T) {
	m := reviewFixture(t, 40, 16)
	for _, tc := range []struct {
		focus, input string
		connected    bool
		keys         []string
	}{
		{idTopic, "new/topic", false, []string{"[Enter]", "[Tab]"}},
		{idTopics, "", false, []string{"[Enter]", "[p]", "[Del]"}},
		{idMessage, "", false, []string{"[Ctrl+B]", "[Ctrl+S]"}},
		{idMessage, "", true, []string{"[Ctrl+S]", "[Ctrl+E]"}},
		{idHistory, "", false, []string{"[Enter]", "[/]", "[Ctrl+C]"}},
	} {
		m.mqttClient = nil
		if tc.connected {
			m.mqttClient = &MQTTClient{Client: &fakeClient{}}
		}
		m.topics.Input.SetValue(tc.input)
		m.SetFocus(tc.focus)
		help := ansi.Strip(m.renderContextHelp())
		lines := strings.Split(help, "\n")
		if len(lines) != 2 {
			t.Fatal("shortcut hints changed the two-row help layout")
		}
		if strings.Contains(lines[1], "…") {
			t.Fatalf("shortcut row is clipped at focus %s: %q", tc.focus, lines[1])
		}
		for _, key := range tc.keys {
			if !strings.Contains(help, key) {
				t.Fatalf("shortcut %s is clipped at focus %s: %q", key, tc.focus, help)
			}
		}
		captureReviewView(t, "context-shortcuts-"+tc.focus, m.renderContextHelp(), 40, 2)
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
