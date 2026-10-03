package emqutiti

import (
	"testing"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/marang/emqutiti/topics"
)

func TestHandlePublishKeyFlags(t *testing.T) {
	m, _ := initialModel(nil)
	m.mqttClient = &MQTTClient{Client: &mockClient{}}
	m.topics.Items = []topics.Item{
		{Name: "a", Publish: true},
		{Name: "b"},
		{Name: "c", Publish: true},
	}
	m.message.SetPayload("hello")
	m.SetFocus(idMessage)
	applyMQTTCommand(m, m.handlePublishKey())
	items := m.payloads.Items()
	if len(items) != 2 {
		t.Fatalf("expected 2 payloads, got %d", len(items))
	}
	if items[0].Topic != "a" || items[1].Topic != "c" {
		t.Fatalf("unexpected topics: %+v", items)
	}
}

func TestHandlePublishKeyFallback(t *testing.T) {
	m, _ := initialModel(nil)
	m.mqttClient = &MQTTClient{Client: &mockClient{}}
	m.topics.Items = []topics.Item{
		{Name: "a"},
		{Name: "b"},
	}
	m.topics.SetSelected(1)
	m.message.SetPayload("hi")
	m.SetFocus(idMessage)
	applyMQTTCommand(m, m.handlePublishKey())
	items := m.payloads.Items()
	if len(items) != 1 {
		t.Fatalf("expected 1 payload, got %d", len(items))
	}
	if items[0].Topic != "b" {
		t.Fatalf("expected topic 'b', got %q", items[0].Topic)
	}
}

type stubToken struct{}

func (stubToken) Wait() bool                     { return true }
func (stubToken) WaitTimeout(time.Duration) bool { return true }
func (stubToken) Done() <-chan struct{}          { ch := make(chan struct{}); close(ch); return ch }
func (stubToken) Error() error                   { return nil }

type mockClient struct{ retained bool }

func (m *mockClient) IsConnected() bool      { return true }
func (m *mockClient) IsConnectionOpen() bool { return true }
func (m *mockClient) Connect() mqtt.Token    { return stubToken{} }
func (m *mockClient) Disconnect(uint)        {}
func (m *mockClient) Publish(topic string, qos byte, retained bool, payload interface{}) mqtt.Token {
	m.retained = retained
	return stubToken{}
}
func (m *mockClient) Subscribe(string, byte, mqtt.MessageHandler) mqtt.Token { return stubToken{} }
func (m *mockClient) SubscribeMultiple(map[string]byte, mqtt.MessageHandler) mqtt.Token {
	return stubToken{}
}
func (m *mockClient) Unsubscribe(...string) mqtt.Token        { return stubToken{} }
func (m *mockClient) AddRoute(string, mqtt.MessageHandler)    {}
func (m *mockClient) OptionsReader() mqtt.ClientOptionsReader { return mqtt.ClientOptionsReader{} }

func TestHandlePublishRetainKey(t *testing.T) {
	m, _ := initialModel(nil)
	fc := &mockClient{}
	m.mqttClient = &MQTTClient{Client: fc}
	m.topics.Items = []topics.Item{{Name: "a"}}
	m.topics.SetSelected(0)
	m.message.SetPayload("hi")
	m.SetFocus(idMessage)
	applyMQTTCommand(m, m.handlePublishRetainKey())
	if !fc.retained {
		t.Fatalf("expected retained publish")
	}
	items := m.history.Items()
	if len(items) == 0 || !items[len(items)-1].Retained {
		t.Fatalf("expected history item marked retained")
	}
}
