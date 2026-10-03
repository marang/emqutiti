package emqutiti

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/marang/emqutiti/connections"
	"github.com/marang/emqutiti/constants"
	"github.com/marang/emqutiti/history"
	"github.com/marang/emqutiti/topics"
	"github.com/marang/emqutiti/traces"
	"github.com/marang/emqutiti/ui"
	"github.com/muesli/termenv"
)

var reviewSizes = []struct{ width, height int }{{120, 40}, {80, 24}, {60, 20}, {40, 16}}

func reviewFixture(t *testing.T, width, height int) *model {
	t.Helper()
	t.Setenv("EMQUTITI_HOME", t.TempDir())
	mgr := connections.NewConnectionsModel()
	mgr.Profiles = []connections.Profile{{Name: "demo", Schema: "tcp", Host: "localhost", Port: 1883}, {Name: strings.Repeat("long-broker-", 10)}}
	store := traces.FileStore{}
	if err := store.SaveTraces(map[string]traces.TracerConfig{
		"capture-demo": {Key: "capture-demo", Profile: "demo", Topics: []string{"site/temperature", "site/status"}},
	}); err != nil {
		t.Fatal(err)
	}
	m, err := initialModel(&mgr)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if st := m.history.Store(); st != nil {
			st.Close()
		}
	})
	m.RefreshConnectionItems()
	m.Update(tea.WindowSizeMsg{Width: width, Height: height})
	m.topics.Items = []topics.Item{
		{Name: "site/temperature", Subscribed: true},
		{Name: "site/command", Publish: true},
		{Name: "site/status", Subscribed: true, Publish: true},
		{Name: "site/inactive"},
		{Name: strings.Repeat("site/long-topic/", 6), Subscribed: true},
	}
	m.topics.RebuildActiveTopicList()
	m.topics.SetSelected(0)
	m.message.SetPayload("{\n  \"command\": \"on\"\n}")
	m.payloads.Add("site/command", "on")
	m.payloads.Add("site/status", strings.Repeat("long payload ", 30))
	var hs []history.Item
	var items []list.Item
	for i := range 20 {
		it := history.Item{Timestamp: time.Date(2026, 10, 3, 9, 0, i, 0, time.UTC), Topic: fmt.Sprintf("site/item-%02d", i), Payload: fmt.Sprintf("payload-%02d", i), Kind: "sub"}
		hs, items = append(hs, it), append(items, it)
	}
	m.history.SetItems(hs)
	m.history.List().SetItems(items)
	return m
}

func captureReviewView(t *testing.T, name, view string, width, height int) {
	t.Helper()
	if got := lipgloss.Width(view); got > width {
		t.Fatalf("%s width %d > %d\n%s", name, got, width, ansi.Strip(view))
	}
	if got := lipgloss.Height(view); got > height {
		t.Fatalf("%s height %d > %d\n%s", name, got, height, ansi.Strip(view))
	}
	if dir := os.Getenv("EMQUTITI_REVIEW_CAPTURE_DIR"); dir != "" {
		path := filepath.Join(dir, fmt.Sprintf("%s-%dx%d.ansi", name, width, height))
		if err := os.WriteFile(path, []byte(view), 0600); err != nil {
			t.Fatal(err)
		}
	}
}

func TestLAB226ViewsFitTerminalBudgets(t *testing.T) {
	profile, dark := lipgloss.ColorProfile(), lipgloss.HasDarkBackground()
	t.Cleanup(func() { lipgloss.SetColorProfile(profile); lipgloss.SetHasDarkBackground(dark) })
	for _, theme := range []string{"dark", "light", "mono"} {
		lipgloss.SetColorProfile(termenv.ANSI256)
		lipgloss.SetHasDarkBackground(theme != "light")
		if theme == "mono" {
			lipgloss.SetColorProfile(termenv.Ascii)
		}
		for _, size := range reviewSizes {
			t.Run(fmt.Sprintf("%s/%dx%d", theme, size.width, size.height), func(t *testing.T) {
				m := reviewFixture(t, size.width, size.height)
				capture := func(name string) { captureReviewView(t, name+"-"+theme, m.View(), size.width, size.height) }
				m.SetFocus(idTopic)
				capture("client")
				m.connections.Connection = "Connection lost: " + strings.Repeat("broker/", 50)
				capture("client-long-status")
				m.SetFocus(idHistory)
				capture("client-history-focus")
				for _, mode := range []constants.AppMode{constants.ModeTopics, constants.ModePayloads, constants.ModeTracer, constants.ModeConnections, constants.ModeHelp, constants.ModeLogs} {
					m.SetMode(mode)
					capture(fmt.Sprintf("mode-%d", mode))
				}
				m.SetMode(constants.ModeConnections)
				m.BeginAdd()
				m.SetMode(constants.ModeEditConnection)
				capture("broker-form")
				m.SetMode(constants.ModeTracer)
				m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
				capture("trace-form")
				m.SetMode(constants.ModeClient)
				m.history.SetDetailItem(history.Item{Payload: strings.Repeat("complete payload/", 100) + "TAIL"})
				m.SetMode(constants.ModeHistoryDetail)
				capture("history-detail")
				m.SetMode(constants.ModeClient)
				m.startHistoryFilter()
				capture("history-filter")
				m.SetMode(constants.ModeClient)
				m.StartConfirm("Delete topic '"+strings.Repeat("site/topic/", 30)+"'? [y/n]", "This action cannot be undone.", nil, nil, nil)
				capture("confirm")
			})
		}
	}
}

func TestLAB226HistoryFocusShowsActiveEntryAndPayload(t *testing.T) {
	for _, size := range reviewSizes {
		t.Run(fmt.Sprintf("%dx%d", size.width, size.height), func(t *testing.T) {
			m := reviewFixture(t, size.width, size.height)
			m.View()
			for _, index := range []int{0, 9, 19} {
				m.history.List().Select(index)
				m.View()
				m.SetFocus(idHistory)
				view := ansi.Strip(m.View())
				if !strings.Contains(view, fmt.Sprintf("site/item-%02d", index)) || !strings.Contains(view, fmt.Sprintf("payload-%02d", index)) {
					t.Fatalf("active entry %d not fully visible\n%s", index, view)
				}
				m.SetFocus(idMessage)
			}
			if m.message.Input().Value() != "{\n  \"command\": \"on\"\n}" {
				t.Fatal("navigation changed the draft")
			}
		})
	}
}

func TestLAB226ReopenedDetailKeepsFullPayload(t *testing.T) {
	m := reviewFixture(t, 80, 24)
	item := history.Item{Topic: "detail", Payload: strings.Repeat("x", 800) + "TAIL", Kind: "sub"}
	m.history.SetItems([]history.Item{item})
	m.history.List().SetItems([]list.Item{item})
	m.SetFocus(idHistory)
	for range 2 {
		m.handleHistoryViewKey()
		m.View()
		m.history.Detail().GotoBottom()
		if !strings.Contains(ansi.Strip(m.View()), "TAIL") {
			t.Fatal("payload tail became inaccessible after reopening")
		}
		m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	}
}

func TestLAB226MouseUsesCoordinatesBeforeFocusScroll(t *testing.T) {
	m := reviewFixture(t, 40, 16)
	m.View()
	m.SetFocus(idMessage)
	m.ui.viewport.SetYOffset(m.ui.elemPos[idHistory] - 1)
	m.View()
	// Click the second history entry while focusing the first one would scroll.
	row, _, visible := ui.ListItemBounds(*m.history.List(), 1, 2, 0)
	if !visible {
		t.Fatal("test fixture does not show the clicked history entry")
	}
	y := m.ui.elemPos[idHistory] + row - m.ui.viewport.YOffset + m.clientViewportTop()
	m.Update(tea.MouseMsg{X: 3, Y: y, Type: tea.MouseLeft, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress})
	if m.FocusedID() != idHistory || m.history.List().Index() != 1 {
		t.Fatalf("click moved after focus scrolling: focus=%s row=%d", m.FocusedID(), m.history.List().Index())
	}
}

func TestLAB226HistoryClickIncludesFilterLineAndIgnoresChrome(t *testing.T) {
	m := reviewFixture(t, 40, 16)
	m.history.SetFilterQuery("payload:ready")
	m.View()
	m.history.List().Select(0)
	m.SetFocus(idHistory)
	row, _, _ := ui.ListItemBounds(*m.history.List(), 1, 2, 0)
	top := m.ui.elemPos[idHistory]
	m.history.HandleClick(tea.MouseMsg{X: 3, Y: top + 1 + row}, top, 0)
	if m.history.List().Index() != 1 {
		t.Fatal("filter status line shifted the clicked history row")
	}
	for _, msg := range []tea.MouseMsg{{X: 0, Y: top + 1 + row}, {X: 3, Y: top}, {X: 3, Y: top + m.history.List().Height() + 1}} {
		m.history.HandleClick(msg, top, 0)
		if m.history.List().Index() != 1 {
			t.Fatal("chrome or blank click selected an invisible history entry")
		}
	}
}

func TestLAB226RootHistoryOutsideClicksPreserveRange(t *testing.T) {
	m := reviewFixture(t, 40, 16)
	m.View()
	m.SetFocus(idHistory)
	m.View()
	row, _, _ := ui.ListItemBounds(*m.history.List(), 1, 2, 0)
	y := m.ui.elemPos[idHistory] + row - m.ui.viewport.YOffset + m.clientViewportTop()
	click := tea.MouseMsg{X: 1, Y: y, Type: tea.MouseLeft, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress, Shift: true}
	m.Update(click)
	if m.history.SelectionAnchor() != -1 {
		t.Fatal("Shift-border click created an anchor without hitting a rendered row")
	}
	for _, item := range m.history.Items() {
		if item.IsSelected != nil {
			t.Fatal("Shift-border click selected an item")
		}
	}
	click.X = 3
	m.Update(click)
	if m.history.SelectionAnchor() != 1 {
		t.Fatal("valid Shift-click did not establish selection")
	}
	m.history.HandleSelection(2, true)
	for _, point := range []struct{ x, y int }{{1, y}, {0, y}, {3, m.clientViewportTop() + m.ui.elemPos[idHistory] - 1 - m.ui.viewport.YOffset}} {
		click.X, click.Y, click.Shift = point.x, point.y, false
		m.Update(click)
		if m.history.SelectionAnchor() != 1 {
			t.Fatal("outside click cleared the range anchor")
		}
		for i, item := range m.history.Items() {
			selected := item.IsSelected != nil && *item.IsSelected
			if selected != (i == 1 || i == 2) {
				t.Fatalf("outside click changed selection at %d", i)
			}
		}
	}
}

func TestLAB226TopicConfirmationSurvivesSubscriptionResort(t *testing.T) {
	m := reviewFixture(t, 40, 16)
	m.mqttClient = &MQTTClient{Client: &failingClient{subErr: fmt.Errorf("denied")}}
	m.topics.Items = []topics.Item{{Name: "z-confirmed", Subscribed: true}, {Name: "a-keep"}}
	m.topics.SetSelected(0)
	m.SetFocus(idTopics)
	pending := m.handleTopicToggle(topics.ToggleMsg{Topic: "z-confirmed", Subscribed: true})
	m.Update(tea.KeyMsg{Type: tea.KeyDelete})
	if m.CurrentMode() != constants.ModeConfirmDelete {
		t.Fatal("topic deletion did not ask for confirmation")
	}
	m.Update(pending())
	if m.topics.Items[0].Name != "a-keep" {
		t.Fatal("test did not exercise subscription-result resorting")
	}
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	if len(m.topics.Items) != 1 || m.topics.Items[0].Name != "a-keep" || m.FocusedID() != idTopics {
		t.Fatal("confirmation deleted a different topic or lost chip focus")
	}
}

func TestLAB226TraceFormTabRemainsInForm(t *testing.T) {
	m := reviewFixture(t, 40, 16)
	m.SetMode(constants.ModeTracer)
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("draft-key")})
	for range 2 {
		m.Update(tea.KeyMsg{Type: tea.KeyTab})
		if m.FocusedID() != traces.IDForm {
			t.Fatal("Tab moved application focus out of the trace form")
		}
	}
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/new-topic")})
	if !strings.Contains(ansi.Strip(m.View()), "draft-key") {
		t.Fatal("Tab discarded the trace key")
	}
}

func TestLAB226ConfirmCtrlScrollDoesNotScrollClient(t *testing.T) {
	m := reviewFixture(t, 40, 16)
	m.StartConfirm(strings.Repeat("long topic/", 100), "", nil, nil, nil)
	before := m.View()
	m.Update(tea.KeyMsg{Type: tea.KeyCtrlDown})
	if m.View() == before || m.ui.viewport.YOffset != 0 {
		t.Fatal("Ctrl+Down did not scroll just the dialog")
	}
}

func TestLAB226TabAppliesManagerSearchBeforeChangingFocus(t *testing.T) {
	for _, mode := range []constants.AppMode{constants.ModeTopics, constants.ModePayloads, constants.ModeTracer, constants.ModeConnections} {
		for _, key := range []tea.KeyType{tea.KeyTab, tea.KeyShiftTab} {
			t.Run(fmt.Sprintf("mode-%d/key-%d", mode, key), func(t *testing.T) {
				m := reviewFixture(t, 80, 24)
				m.SetMode(mode)
				var l *list.Model
				switch mode {
				case constants.ModeTopics:
					l = m.topics.List()
				case constants.ModePayloads:
					l = m.payloads.List()
				case constants.ModeConnections:
					l = &m.connections.Manager.ConnectionsList
				default:
					l = m.traces.List()
				}
				l.SetFilterText("e")
				l.SetFilterState(list.Filtering)
				focus := m.FocusedID()
				m.Update(tea.KeyMsg{Type: key})
				if l.FilterState() != list.FilterApplied || l.FilterValue() != "e" || m.FocusedID() != focus {
					t.Fatalf("Tab lost search or focus: filter=%q state=%v focus=%s", l.FilterValue(), l.FilterState(), m.FocusedID())
				}
			})
		}
	}
}

func TestLAB226BrokerSearchEnterAndEditUseFilteredProfile(t *testing.T) {
	m := reviewFixture(t, 40, 16)
	m.SetMode(constants.ModeConnections)
	l := &m.connections.Manager.ConnectionsList
	l.SetFilterText("long-broker")
	l.SetFilterState(list.Filtering)
	m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if l.FilterState() != list.FilterApplied || m.CurrentMode() != constants.ModeConnections || m.connections.Connection != "" {
		t.Fatal("Enter during search connected instead of applying the filter")
	}
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	if m.connections.Form == nil || m.connections.Form.Index != 1 {
		t.Fatal("edit selected a different profile than the filtered row")
	}
}

func TestLAB226ChipStatesIndependentOfFocusHoverAndPulse(t *testing.T) {
	profile := lipgloss.ColorProfile()
	t.Cleanup(func() { lipgloss.SetColorProfile(profile) })
	for _, profile := range []termenv.Profile{termenv.ANSI256, termenv.Ascii} {
		lipgloss.SetColorProfile(profile)
		var states []string
		pulsedStates := make([][]string, 6)
		for _, item := range []topics.Item{{Name: "site/topic"}, {Name: "site/topic", Subscribed: true}, {Name: "site/topic", Publish: true}, {Name: "site/topic", Subscribed: true, Publish: true}} {
			// A separate explicit target keeps selection from enabling the fallback.
			items := []topics.Item{item, {Name: "other/target", Publish: true}}
			base := renderTopicChips(items, 0, -1, 40)[0]
			states = append(states, base)
			for phase := range 6 {
				pulsed := renderTopicChipsWithPulses(items, 0, 0, 40, map[string]int{item.Name: phase})[0]
				if lipgloss.Width(pulsed) != lipgloss.Width(base) || lipgloss.Height(pulsed) != lipgloss.Height(base) || ansi.Strip(pulsed) != ansi.Strip(base) {
					t.Fatal("pulse changed chip geometry or label")
				}
				pulsedStates[phase] = append(pulsedStates[phase], pulsed)
			}
			if got := renderTopicChips(items, -1, 0, 40)[0]; got != renderTopicChips(items, -1, -1, 40)[0] {
				t.Fatal("hover changed chip appearance")
			}
		}
		for i := range states {
			for j := 0; j < i; j++ {
				if states[i] == states[j] {
					t.Fatalf("chip states %d and %d are indistinguishable", i, j)
				}
			}
		}
		for _, states := range pulsedStates {
			if states[2] == states[3] {
				t.Fatal("pulse overwrote the independent subscription cue")
			}
		}
	}
}

func TestLAB226ChipEncodingComparison(t *testing.T) {
	if os.Getenv("EMQUTITI_REVIEW_CAPTURE_DIR") == "" {
		t.Skip("opt-in design comparison captures")
	}
	profile := lipgloss.ColorProfile()
	t.Cleanup(func() { lipgloss.SetColorProfile(profile) })
	lipgloss.SetColorProfile(termenv.ANSI256)
	items := []topics.Item{{Name: "site/topic"}, {Name: "site/topic", Subscribed: true}, {Name: "site/topic", Publish: true}, {Name: "site/topic", Subscribed: true, Publish: true}}
	var rows []string
	rows = append(rows, "States: inactive | subscribe | publish | both", "Underline, resting:", lipgloss.JoinHorizontal(lipgloss.Top, renderTopicChips(items, -1, -1, 30)...))
	for _, focused := range []bool{false, true} {
		var underline, border []string
		for _, item := range items {
			selected := -1
			if focused {
				selected = 0
			}
			fixture := []topics.Item{item, {Name: "other/target", Publish: true}}
			underline = append(underline, renderTopicChipsWithPulses(fixture, selected, -1, 30, map[string]int{item.Name: 1})[0])
			st := ui.ChipInactive
			if item.Subscribed {
				st = ui.Chip
			}
			if item.Publish {
				st = ui.ChipPublish
			}
			if item.Subscribed {
				st = st.BorderTopForeground(ui.ColGreen)
			}
			if focused {
				st = st.BorderTopForeground(ui.ColPink).BorderLeftForeground(ui.ColPink)
			}
			border = append(border, topicPulseStyle(st, 1, focused).Render(item.Name))
		}
		if border[2] != border[3] {
			t.Fatal("comparison no longer reproduces pulse overwriting the border-only subscribe cue")
		}
		rows = append(rows, fmt.Sprintf("Underline, pulse focused=%v:", focused), lipgloss.JoinHorizontal(lipgloss.Top, underline...), "Border cue, same pulse:", lipgloss.JoinHorizontal(lipgloss.Top, border...))
	}
	captureReviewView(t, "chip-comparison-dark", strings.Join(rows, "\n"), 80, 40)
}
