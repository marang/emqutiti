package emqutiti

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/marang/emqutiti/connections"
	"github.com/marang/emqutiti/constants"
	"github.com/marang/emqutiti/history"
	"github.com/marang/emqutiti/topics"
)

type mockToken struct{ err error }

func (t *mockToken) Wait() bool                     { return true }
func (t *mockToken) WaitTimeout(time.Duration) bool { return true }
func (t *mockToken) Done() <-chan struct{}          { ch := make(chan struct{}); close(ch); return ch }
func (t *mockToken) Error() error                   { return t.err }

type failingClient struct {
	subErr   error
	unsubErr error
}

func (c *failingClient) IsConnected() bool                                  { return true }
func (c *failingClient) IsConnectionOpen() bool                             { return true }
func (c *failingClient) Connect() mqtt.Token                                { return &mockToken{} }
func (c *failingClient) Disconnect(uint)                                    {}
func (c *failingClient) Publish(string, byte, bool, interface{}) mqtt.Token { return &mockToken{} }
func (c *failingClient) Subscribe(string, byte, mqtt.MessageHandler) mqtt.Token {
	return &mockToken{err: c.subErr}
}
func (c *failingClient) SubscribeMultiple(map[string]byte, mqtt.MessageHandler) mqtt.Token {
	return &mockToken{}
}
func (c *failingClient) Unsubscribe(...string) mqtt.Token     { return &mockToken{err: c.unsubErr} }
func (c *failingClient) AddRoute(string, mqtt.MessageHandler) {}
func (c *failingClient) OptionsReader() mqtt.ClientOptionsReader {
	return mqtt.NewOptionsReader(mqtt.NewClientOptions())
}

// Test copy behavior when history items are selected.
func TestHandleClientKeyCopySelected(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	m, _ := initialModel(nil)
	sel := true
	hi := history.Item{Timestamp: time.Now(), Topic: "t1", Payload: "msg1", Kind: "pub", Retained: false, IsSelected: &sel}
	m.history.SetItems([]history.Item{hi})
	m.history.List().SetItems([]list.Item{hi})
	m.history.List().Select(0)
	m.SetFocus(idHistory)

	HandleClientKey(m, tea.KeyMsg{Type: tea.KeyCtrlC})

	if len(m.history.Items()) != 2 {
		t.Fatalf("expected error appended to history, got %d items", len(m.history.Items()))
	}
	if m.history.Items()[1].Kind != "log" {
		t.Fatalf("expected last item kind 'log', got %q", m.history.Items()[1].Kind)
	}
}

// Test disconnect behavior clears connection state.
func TestHandleClientKeyDisconnect(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	conn := connections.NewConnectionsModel()
	conn.Profiles = []connections.Profile{{Name: "test"}}
	conn.Statuses["test"] = "connected"
	conn.Errors["test"] = "oops"

	m, _ := initialModel(&conn)
	m.mqttClient = &MQTTClient{}
	m.connections.Connection = "test"
	m.connections.Active = "test"
	m.connections.SetConnected("test")
	m.connections.Manager.Errors["test"] = "boom"

	HandleClientKey(m, tea.KeyMsg{Type: tea.KeyCtrlX})
	if m.CurrentMode() != constants.ModeConfirmDelete {
		t.Fatalf("expected confirm mode, got %v", m.CurrentMode())
	}
	m.connections.StatusChan = nil
	cmd := m.confirm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	if cmd == nil {
		t.Fatalf("expected command after confirm")
	}
	cmd()
	m.updateClient(reconnectPromptMsg("test"))
	if m.CurrentMode() != constants.ModeConfirmDelete {
		t.Fatalf("expected reconnect confirm mode, got %v", m.CurrentMode())
	}
	cmd = m.confirm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	if cmd == nil {
		t.Fatalf("expected command after reconnect prompt")
	}
	cmd()

	if m.mqttClient != nil {
		t.Fatalf("expected mqttClient nil after disconnect")
	}
	if m.connections.Connection != "" || m.connections.Active != "" {
		t.Fatalf("expected connection cleared, got %q %q", m.connections.Connection, m.connections.Active)
	}
	if st := m.connections.Manager.Statuses["test"]; st != "disconnected" {
		t.Fatalf("expected status 'disconnected', got %q", st)
	}
	if err := m.connections.Manager.Errors["test"]; err != "" {
		t.Fatalf("expected error cleared, got %q", err)
	}
	if m.CurrentMode() != constants.ModeConnections {
		t.Fatalf("expected mode %v, got %v", constants.ModeConnections, m.CurrentMode())
	}
}

// Test that pressing '/' while history is focused starts the filter form.
func TestHandleClientKeyFilterInitiation(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	m, _ := initialModel(nil)
	idx := 3 // history index in focus order
	m.focus.Set(idx)
	m.ui.focusIndex = idx

	HandleClientKey(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})

	if m.ui.modeStack[0] != constants.ModeHistoryFilter {
		t.Fatalf("expected ModeHistoryFilter, got %v", m.ui.modeStack[0])
	}
	if m.history.FilterForm() == nil {
		t.Fatalf("expected filter form to be initialized")
	}
}

// Test error handling when topic subscription fails.
func TestHandleTopicToggleSubscribeError(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	m, _ := initialModel(nil)
	m.mqttClient = &MQTTClient{Client: &failingClient{subErr: errors.New("sub boom")}}
	applyMQTTCommand(m, m.handleTopicToggle(topics.ToggleMsg{Topic: "t1", Subscribed: true}))
	items := m.history.Items()
	if len(items) != 1 {
		t.Fatalf("expected 1 history item, got %d", len(items))
	}
	if items[0].Kind != "log" || !strings.Contains(items[0].Payload, "sub boom") {
		t.Fatalf("expected log with error, got kind %q payload %q", items[0].Kind, items[0].Payload)
	}
}

// Test error handling when topic unsubscription fails.
func TestHandleTopicToggleUnsubscribeError(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	m, _ := initialModel(nil)
	m.mqttClient = &MQTTClient{Client: &failingClient{unsubErr: errors.New("unsub boom")}}
	applyMQTTCommand(m, m.handleTopicToggle(topics.ToggleMsg{Topic: "t1", Subscribed: false}))
	items := m.history.Items()
	if len(items) != 1 {
		t.Fatalf("expected 1 history item, got %d", len(items))
	}
	if items[0].Kind != "log" || !strings.Contains(items[0].Payload, "unsub boom") {
		t.Fatalf("expected log with error, got kind %q payload %q", items[0].Kind, items[0].Payload)
	}
}

// Test logTopicAction handles empty and valid actions.
func TestLogTopicAction(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		t.Setenv("HOME", t.TempDir())
		m, _ := initialModel(nil)
		m.logTopicAction("t1", "", nil)
		items := m.history.Items()
		if len(items) != 1 {
			t.Fatalf("expected 1 history item, got %d", len(items))
		}
		if items[0].Kind != "log" || items[0].Payload != "No action specified for topic: t1" {
			t.Fatalf("unexpected log item: kind %q payload %q", items[0].Kind, items[0].Payload)
		}
	})

	t.Run("subscribe", func(t *testing.T) {
		t.Setenv("HOME", t.TempDir())
		m, _ := initialModel(nil)
		m.logTopicAction("t1", "subscribe", nil)
		items := m.history.Items()
		if len(items) != 1 {
			t.Fatalf("expected 1 history item, got %d", len(items))
		}
		if items[0].Kind != "log" || items[0].Payload != "Subscribed to topic: t1" {
			t.Fatalf("unexpected log item: kind %q payload %q", items[0].Kind, items[0].Payload)
		}
	})

	t.Run("unsubscribe", func(t *testing.T) {
		t.Setenv("HOME", t.TempDir())
		m, _ := initialModel(nil)
		m.logTopicAction("t1", "unsubscribe", nil)
		items := m.history.Items()
		if len(items) != 1 {
			t.Fatalf("expected 1 history item, got %d", len(items))
		}
		if items[0].Kind != "log" || items[0].Payload != "Unsubscribed from topic: t1" {
			t.Fatalf("unexpected log item: kind %q payload %q", items[0].Kind, items[0].Payload)
		}
	})

	t.Run("unknown", func(t *testing.T) {
		t.Setenv("HOME", t.TempDir())
		m, _ := initialModel(nil)
		m.logTopicAction("t1", "invalid", nil)
		items := m.history.Items()
		if len(items) != 1 {
			t.Fatalf("expected 1 history item, got %d", len(items))
		}
		exp := "Unknown action for topic: t1"
		if items[0].Kind != "log" || items[0].Payload != exp {
			t.Fatalf("unexpected log item: kind %q payload %q", items[0].Kind, items[0].Payload)
		}
	})
}
