package emqutiti

import (
	"context"
	"io"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/marang/emqutiti/constants"
	"github.com/marang/emqutiti/history"
	"github.com/marang/emqutiti/topics"
)

func TestModifiedKeysActualParserRoutesThroughRoot(t *testing.T) {
	for _, focus := range []string{idMessage, idHistory} {
		t.Run(focus, func(t *testing.T) {
			m := reviewFixture(t, 80, 24)
			m.SetMode(constants.ModeClient)
			m.topics.Items = []topics.Item{{Name: "output"}}
			m.topics.SetSelected(0)
			client := &publishTestClient{}
			m.mqttClient = &MQTTClient{Client: client}
			m.message.Input().Reset()
			m.SetFocus(focus)
			m.history.List().Select(1)
			input := "\x1b[32;2u\x04"
			if focus == idMessage {
				input = "first\x1b[32;2u\rsecond\x1b[13;5u\x04"
			}
			r := framingCtrlEnterReader(strings.NewReader(input))
			defer r.close()
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			harness := &ctrlEnterCaptureModel{apply: func(msg tea.Msg) tea.Cmd {
				_, cmd := m.Update(msg)
				if _, ok := msg.(ctrlEnterMsg); ok {
					applyMQTTCommand(m, cmd)
				}
				return nil
			}}
			p := tea.NewProgram(harness, tea.WithInput(r), tea.WithFilter(r.filter), tea.WithOutput(io.Discard), tea.WithContext(ctx), tea.WithoutRenderer(), tea.WithoutSignalHandler())
			if _, err := p.Run(); err != nil {
				t.Fatal(err)
			}
			if focus == idMessage {
				if len(client.seen) != 1 || client.seen["output"] != "first \nsecond" || m.message.Input().Value() != "first \nsecond" || m.pendingPublishes() != 0 {
					t.Fatalf("root did not publish the unchanged draft: %+v", client.seen)
				}
			} else {
				item := m.history.List().Items()[1].(history.Item)
				if len(client.seen) != 0 || item.IsSelected == nil || !*item.IsSelected {
					t.Fatal("root did not toggle the rendered History row")
				}
			}
		})
	}
}

func TestShiftedSpaceUsesNormalTextRouting(t *testing.T) {
	for _, focus := range []string{idTopic, idMessage} {
		m := reviewFixture(t, 80, 24)
		m.SetMode(constants.ModeClient)
		m.SetFocus(focus)
		m.message.Input().Reset()
		m.topics.Input.Reset()
		m.Update(shiftedSpaceMsg{})
		value := m.topics.Input.Value()
		if focus == idMessage {
			value = m.message.Input().Value()
		}
		if value != " " || m.pendingPublishes() != 0 || len(selectedHistoryRows(m)) != 0 {
			t.Fatalf("shifted space was not editor text for %s: %q", focus, value)
		}
	}
	for _, extended := range []bool{false, true} {
		m := reviewFixture(t, 80, 24)
		m.SetMode(constants.ModeConnections)
		m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
		m.Update(tea.KeyMsg{Type: tea.KeyTab})
		if extended {
			m.Update(shiftedSpaceMsg{})
		} else {
			m.Update(tea.KeyMsg{Type: tea.KeySpace, Runes: []rune{' '}})
		}
		if !strings.Contains(m.View(), "Add Broker") {
			t.Fatal("shifted space left the broker form")
		}
	}
}

func TestModifiedKeysBlockedDuringResize(t *testing.T) {
	m := reviewFixture(t, 80, 24)
	m.SetMode(constants.ModeClient)
	m.SetFocus(idMessage)
	press := panelBorderMouse(t, m, idMessage)
	m.Update(press)
	m.Update(ctrlEnterMsg{press: true})
	m.Update(shiftedSpaceMsg{})
	if m.pendingPublishes() != 0 || len(selectedHistoryRows(m)) != 0 {
		t.Fatal("modified key bypassed resize capture")
	}
}
