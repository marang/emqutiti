package emqutiti

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/marang/emqutiti/connections"
	"github.com/marang/emqutiti/topics"
	"github.com/marang/emqutiti/ui"

	"github.com/marang/emqutiti/history"
)

type dummyToken struct{ err error }

func (t *dummyToken) Wait() bool                     { return true }
func (t *dummyToken) WaitTimeout(time.Duration) bool { return true }
func (t *dummyToken) Done() <-chan struct{} {
	ch := make(chan struct{})
	close(ch)
	return ch
}
func (t *dummyToken) Error() error { return t.err }

type fakeClient struct {
	subErr   error
	unsubErr error
}

func (c *fakeClient) IsConnected() bool                                  { return true }
func (c *fakeClient) IsConnectionOpen() bool                             { return true }
func (c *fakeClient) Connect() mqtt.Token                                { return &dummyToken{} }
func (c *fakeClient) Disconnect(uint)                                    {}
func (c *fakeClient) Publish(string, byte, bool, interface{}) mqtt.Token { return &dummyToken{} }
func (c *fakeClient) Subscribe(string, byte, mqtt.MessageHandler) mqtt.Token {
	return &dummyToken{err: c.subErr}
}
func (c *fakeClient) SubscribeMultiple(map[string]byte, mqtt.MessageHandler) mqtt.Token {
	return &dummyToken{}
}
func (c *fakeClient) Unsubscribe(...string) mqtt.Token     { return &dummyToken{err: c.unsubErr} }
func (c *fakeClient) AddRoute(string, mqtt.MessageHandler) {}
func (c *fakeClient) OptionsReader() mqtt.ClientOptionsReader {
	return mqtt.NewOptionsReader(mqtt.NewClientOptions())
}

func TestHandleMouseScrollTopics(t *testing.T) {
	m, _ := initialModel(nil)
	m.Update(tea.WindowSizeMsg{Width: 40, Height: 20})
	setupManyTopics(m, 10)
	rowH := lipgloss.Height(ui.Chip.Render("t"))
	m.layout.topics.height = rowH
	m.viewClient()
	m.SetFocus(idTopics)
	_, handled := m.handleMouseScroll(tea.MouseMsg{Action: tea.MouseActionPress, Button: tea.MouseButtonWheelDown})
	if !handled {
		t.Fatalf("expected scroll event handled")
	}
	if m.topics.VP.YOffset != rowH {
		t.Fatalf("expected scroll %d got %d", rowH, m.topics.VP.YOffset)
	}
}

func setupManyTopics(m *model, n int) {
	for i := 0; i < n; i++ {
		title := fmt.Sprintf("topic-%d", i)
		m.topics.Items = append(m.topics.Items, topics.Item{Name: title, Subscribed: true})
	}
	m.layout.topics.height = n
}

func TestHandleHistorySelectionShift(t *testing.T) {
	m, _ := initialModel(nil)
	m.history.SetItems([]history.Item{
		{Timestamp: time.Now(), Topic: "t1", Payload: "p1", Kind: "pub", Retained: false},
		{Timestamp: time.Now(), Topic: "t2", Payload: "p2", Kind: "pub", Retained: false},
		{Timestamp: time.Now(), Topic: "t3", Payload: "p3", Kind: "pub", Retained: false},
	})
	items := make([]list.Item, len(m.history.Items()))
	for i, it := range m.history.Items() {
		items[i] = it
	}
	m.history.List().SetItems(items)
	m.SetFocus(idHistory)

	m.history.HandleSelection(0, true)
	if m.history.SelectionAnchor() != 0 {
		t.Fatalf("anchor = %d, want 0", m.history.SelectionAnchor())
	}
	m.history.HandleSelection(2, true)
	for i := 0; i <= 2; i++ {
		if m.history.Items()[i].IsSelected == nil || !*m.history.Items()[i].IsSelected {
			t.Fatalf("item %d not selected", i)
		}
	}
	if m.history.SelectionAnchor() != 0 {
		t.Fatalf("anchor = %d, want 0", m.history.SelectionAnchor())
	}
}

func TestFilterHistoryList(t *testing.T) {
	m, _ := initialModel(nil)
	hs := &historyStore{}
	m.history.SetStore(hs)
	ts := time.Now()
	if err := hs.Append(history.Message{Timestamp: ts, Topic: "foo", Payload: "hello", Kind: "pub", Retained: false}); err != nil {
		t.Fatalf("Append failed: %v", err)
	}
	if err := hs.Append(history.Message{Timestamp: ts, Topic: "bar", Payload: "bye", Kind: "pub", Retained: false}); err != nil {
		t.Fatalf("Append failed: %v", err)
	}

	m.history.List().SetFilteringEnabled(true)
	m.history.List().SetFilterText("topic=foo")
	m.history.List().SetFilterState(list.Filtering)
	m.filterHistoryList()

	items := m.history.List().Items()
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	hi := items[0].(history.Item)
	if hi.Topic != "foo" {
		t.Fatalf("unexpected topic %q", hi.Topic)
	}
}

func TestHandleHistoryClick(t *testing.T) {
	m, _ := initialModel(nil)
	m.Update(tea.WindowSizeMsg{Width: 40, Height: 20})
	m.history.SetItems([]history.Item{{Timestamp: time.Now(), Topic: "t1", Payload: "p1", Kind: "pub", Retained: false}})
	items := []list.Item{m.history.Items()[0]}
	m.history.List().SetItems(items)
	m.viewClient()
	m.SetFocus(idHistory)
	y := m.ui.elemPos[idHistory] + 1
	m.history.HandleClick(tea.MouseMsg{Y: y}, m.ui.elemPos[idHistory], m.ui.viewport.YOffset)
	if m.history.List().Index() != 0 {
		t.Fatalf("expected index 0 got %d", m.history.List().Index())
	}
}

func TestHistoryScroll(t *testing.T) {
	m, _ := initialModel(nil)
	for i := 0; i < 30; i++ {
		hi := history.Item{Timestamp: time.Now(), Topic: fmt.Sprintf("t%d", i), Payload: "p", Kind: "pub", Retained: false}
		m.history.SetItems(append(m.history.Items(), hi))
	}
	items := make([]list.Item, len(m.history.Items()))
	for i, it := range m.history.Items() {
		items[i] = it
	}
	m.history.List().SetItems(items)
	m.SetFocus(idHistory)
	_, handled := m.handleMouseScroll(tea.MouseMsg{Action: tea.MouseActionPress, Button: tea.MouseButtonWheelDown})
	if !handled {
		t.Fatalf("expected scroll event handled")
	}
}

func TestUpdateClientStatus(t *testing.T) {
	m, _ := initialModel(nil)
	cmds := m.updateClientStatus()
	if len(cmds) != 1 {
		t.Fatalf("expected 1 cmd got %d", len(cmds))
	}
	m.mqttClient = &MQTTClient{MessageChan: make(chan MQTTMessage)}
	cmds = m.updateClientStatus()
	if len(cmds) != 1 {
		t.Fatalf("expected only mqtt listener after status listener is active, got %d", len(cmds))
	}
	cmds = m.updateClientStatus()
	if len(cmds) != 0 {
		t.Fatalf("expected no duplicate listeners, got %d", len(cmds))
	}
}

func TestStatusMessageRearmsStatusListener(t *testing.T) {
	m, _ := initialModel(nil)
	if len(m.updateClientStatus()) != 1 {
		t.Fatalf("expected initial status listener")
	}

	cmd := m.handleStatusMessage(connections.StatusMessage("Connected to MQTT broker"))

	if cmd == nil {
		t.Fatalf("expected rearmed status listener command")
	}
	if !m.ui.listeners.status {
		t.Fatalf("expected status listener marked active")
	}
}

func TestMQTTClosedMessageClearsListener(t *testing.T) {
	m, _ := initialModel(nil)
	m.mqttClient = &MQTTClient{MessageChan: make(chan MQTTMessage)}
	if len(m.updateClientStatus()) != 2 {
		t.Fatalf("expected status and mqtt listeners")
	}

	_, done := m.handleClientMsg(mqttListenClosedMsg{})

	if !done {
		t.Fatalf("expected closed mqtt listener message handled")
	}
	if m.ui.listeners.mqtt {
		t.Fatalf("expected mqtt listener inactive after close")
	}
}

func TestHandleTopicToggleActions(t *testing.T) {
	t.Run("subscribe", func(t *testing.T) {
		t.Setenv("HOME", t.TempDir())
		m, _ := initialModel(nil)
		m.mqttClient = &MQTTClient{Client: &fakeClient{}}
		m.handleTopicToggle(topics.ToggleMsg{Topic: "t1", Subscribed: true})
		items := m.history.Items()
		if len(items) != 1 {
			t.Fatalf("expected 1 history item, got %d", len(items))
		}
		if items[0].Payload != "Subscribed to topic: t1" {
			t.Fatalf("unexpected payload %q", items[0].Payload)
		}
	})

	t.Run("unsubscribe", func(t *testing.T) {
		t.Setenv("HOME", t.TempDir())
		m, _ := initialModel(nil)
		m.mqttClient = &MQTTClient{Client: &fakeClient{}}
		m.handleTopicToggle(topics.ToggleMsg{Topic: "t1", Subscribed: false})
		items := m.history.Items()
		if len(items) != 1 {
			t.Fatalf("expected 1 history item, got %d", len(items))
		}
		if items[0].Payload != "Unsubscribed from topic: t1" {
			t.Fatalf("unexpected payload %q", items[0].Payload)
		}
	})

	t.Run("subscribe error", func(t *testing.T) {
		t.Setenv("HOME", t.TempDir())
		m, _ := initialModel(nil)
		m.mqttClient = &MQTTClient{Client: &fakeClient{subErr: errors.New("boom")}}
		m.handleTopicToggle(topics.ToggleMsg{Topic: "t1", Subscribed: true})
		items := m.history.Items()
		if len(items) != 1 {
			t.Fatalf("expected 1 history item, got %d", len(items))
		}
		if !strings.Contains(items[0].Payload, "boom") {
			t.Fatalf("expected error payload, got %q", items[0].Payload)
		}
	})

	t.Run("unsubscribe error", func(t *testing.T) {
		t.Setenv("HOME", t.TempDir())
		m, _ := initialModel(nil)
		m.mqttClient = &MQTTClient{Client: &fakeClient{unsubErr: errors.New("boom")}}
		m.handleTopicToggle(topics.ToggleMsg{Topic: "t1", Subscribed: false})
		items := m.history.Items()
		if len(items) != 1 {
			t.Fatalf("expected 1 history item, got %d", len(items))
		}
		if !strings.Contains(items[0].Payload, "boom") {
			t.Fatalf("expected error payload, got %q", items[0].Payload)
		}
	})

	t.Run("no client", func(t *testing.T) {
		t.Setenv("HOME", t.TempDir())
		m, _ := initialModel(nil)
		m.handleTopicToggle(topics.ToggleMsg{Topic: "t1", Subscribed: true})
		items := m.history.Items()
		if len(items) != 1 {
			t.Fatalf("expected 1 history item, got %d", len(items))
		}
		if !strings.Contains(items[0].Payload, "no mqtt client") {
			t.Fatalf("unexpected payload %q", items[0].Payload)
		}
	})
}

func TestLogTopicActionEmpty(t *testing.T) {
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
}
