package emqutiti

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/marang/emqutiti/constants"
	"github.com/marang/emqutiti/history"
	"github.com/marang/emqutiti/internal/clipboardutil"
	"github.com/marang/emqutiti/topics"
	"github.com/marang/emqutiti/ui"
)

func selectedHistoryRows(m *model) []int {
	var selected []int
	for i, item := range m.history.Items() {
		if item.IsSelected != nil && *item.IsSelected {
			selected = append(selected, i)
		}
	}
	return selected
}

func TestHistoryAdditiveMouseAndSpaceSelection(t *testing.T) {
	for _, filtered := range []bool{false, true} {
		t.Run(fmt.Sprintf("filtered=%t", filtered), func(t *testing.T) {
			m := reviewFixture(t, 80, 24)
			if filtered {
				m.history.SetStore(&historyStore{})
				for _, item := range m.history.Items() {
					if err := m.history.Store().Append(history.Message{Timestamp: item.Timestamp, Topic: item.Topic, Payload: item.Payload, Kind: item.Kind}); err != nil {
						t.Fatal(err)
					}
				}
				m.history.SetFilterQuery("payload=payload-")
			}
			m.SetMode(constants.ModeClient)
			m.SetFocus(idHistory)
			m.View()
			m.Update(tea.KeyMsg{Type: tea.KeyShiftDown})
			m.Update(tea.KeyMsg{Type: tea.KeyShiftDown})
			m.history.List().Select(5)
			offset := m.ui.viewport.YOffset
			m.Update(tea.KeyMsg{Type: tea.KeySpace}) // Legacy terminals encode Shift+Space identically.
			if m.ui.viewport.YOffset != offset {
				t.Fatal("selection Space paged the outer viewport")
			}
			if got := selectedHistoryRows(m); !slices.Equal(got, []int{0, 1, 2, 5}) {
				t.Fatalf("Space replaced existing marks: %v", got)
			}
			m.history.List().Select(7)
			m.View()
			row, _, ok := ui.ListItemBounds(*m.history.List(), 7, 2, 0)
			if !ok {
				t.Fatal("clicked row is not visible")
			}
			y := m.ui.elemPos[idHistory] + row + m.clientViewportTop() - m.ui.viewport.YOffset
			if filtered {
				y++
			}
			m.Update(tea.MouseMsg{X: 3, Y: y, Type: tea.MouseLeft, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft, Shift: true})
			if got := selectedHistoryRows(m); !slices.Equal(got, []int{0, 1, 2, 5, 6, 7}) {
				t.Fatalf("Shift-click did not add its range: %v", got)
			}
			m.Update(tea.KeyMsg{Type: tea.KeySpace})
			want := []int{0, 1, 2, 5, 6}
			if got := selectedHistoryRows(m); !slices.Equal(got, want) {
				t.Fatalf("Space did not toggle only the current row: %v", got)
			}
			if filtered {
				m.history.Append("outside/filter", "unrelated incoming message", "sub", false, "")
				if got := selectedHistoryRows(m); !slices.Equal(got, want) {
					t.Fatalf("filtered append discarded marks: %v", got)
				}
			}
			var expected []string
			for _, i := range want {
				item := m.history.Items()[i]
				expected = append(expected, item.Topic+": "+item.Payload)
			}
			oldCopy := clipboardutil.Copy
			t.Cleanup(func() { clipboardutil.Copy = oldCopy })
			var copied string
			clipboardutil.Copy = func(text string) error { copied = text; return nil }
			m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
			if copied != strings.Join(expected, "\n") {
				t.Fatalf("copy did not match additive marks: %q", copied)
			}
			if got := selectedHistoryRows(m); !slices.Equal(got, want) {
				t.Fatalf("copy log discarded marks: %v", got)
			}
			m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
			if copied != strings.Join(expected, "\n") {
				t.Fatal("repeated copy lost the additive selection")
			}
		})
	}
}

func TestHistorySpaceSelectionRespectsEditorAndArchive(t *testing.T) {
	m := reviewFixture(t, 80, 24)
	m.SetMode(constants.ModeClient)
	m.SetFocus(idMessage)
	m.message.Input().SetValue("draft")
	m.message.Input().CursorEnd()
	m.Update(tea.KeyMsg{Type: tea.KeySpace, Runes: []rune{' '}})
	if !strings.Contains(m.message.Input().Value(), "draft ") || len(selectedHistoryRows(m)) != 0 {
		t.Fatal("selection key stole editor text")
	}
	m.SetFocus(idHistory)
	m.history.SetShowArchived(true)
	m.Update(tea.KeyMsg{Type: tea.KeySpace})
	if len(selectedHistoryRows(m)) != 0 {
		t.Fatal("archive allowed selection toggles")
	}
}

func TestPublishFailureSurvivesHistoryFilteringAndReload(t *testing.T) {
	m := reviewFixture(t, 80, 24)
	m.history.SetStore(&historyStore{})
	m.SetMode(constants.ModeClient)
	m.topics.Items = []topics.Item{{Name: "failed/outgoing"}}
	m.topics.SetSelected(0)
	m.message.Input().SetValue("draft")
	m.SetFocus(idMessage)
	applyMQTTCommand(m, m.handlePublishKey())
	want := m.mqttOps.publishError
	if want == "" {
		t.Fatal("fixture did not produce a publish failure")
	}
	m.history.SetFilterQuery("payload=Publish failed")
	m.filterHistoryList()
	if items := m.history.Items(); len(items) != 1 || items[0].Payload != want {
		t.Fatalf("failure lost after filter: %+v", items)
	}
	m.SetFocus(idHistory)
	m.handleClearFilterKey()
	if items := m.history.Items(); len(items) != 1 || items[0].Payload != want {
		t.Fatalf("failure lost after clearing filter: %+v", items)
	}
	loaded := history.NewComponent(historyModelAdapter{m}, m.history.Store())
	if items := loaded.Items(); len(items) != 1 || items[0].Payload != want {
		t.Fatalf("failure lost after reload: %+v", items)
	}
}
