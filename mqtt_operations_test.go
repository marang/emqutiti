package emqutiti

import (
	"errors"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/marang/emqutiti/topics"
)

func applyMQTTCommand(m *model, cmd tea.Cmd) {
	if cmd == nil {
		return
	}
	msg := cmd()
	if batch, ok := msg.(tea.BatchMsg); ok {
		for _, child := range batch {
			applyMQTTCommand(m, child)
		}
		return
	}
	m.Update(msg)
}

type publishTestClient struct {
	mockClient
	errTopic string
	seen     map[string]string
}

func (c *publishTestClient) Publish(topic string, qos byte, retained bool, payload interface{}) mqtt.Token {
	if c.seen == nil {
		c.seen = map[string]string{}
	}
	c.seen[topic] = payload.(string)
	if topic == c.errTopic {
		return &mockToken{err: errors.New("connection lost")}
	}
	return c.mockClient.Publish(topic, qos, retained, payload)
}

func TestPublishOfflineReportsFailureWithoutSuccess(t *testing.T) {
	m, _ := initialModel(nil)
	m.topics.Items = []topics.Item{{Name: "probe/outgoing", Publish: true}}
	m.message.SetPayload("offline payload")
	m.SetFocus(idMessage)
	cmd := m.handlePublishKey()
	if len(m.history.Items()) != 0 || m.pendingPublishes() != 1 {
		t.Fatal("dispatch must be pending, not successful")
	}
	applyMQTTCommand(m, cmd)
	items := m.history.Items()
	if len(items) != 1 || items[0].Kind != "log" || !strings.Contains(items[0].Payload, "Publish failed") {
		t.Fatalf("expected failure, got %+v", items)
	}
	if len(m.ui.animation.topicPulses) != 0 || m.ui.animation.historyPulse != 0 || len(m.payloads.Items()) != 0 {
		t.Fatal("offline request created success feedback")
	}
	if m.message.Input().Value() != "offline payload" || m.pendingPublishes() != 0 {
		t.Fatal("draft lost or pending request leaked")
	}
}

func TestPublishPartialResultsKeepSnapshotAndDraft(t *testing.T) {
	m, _ := initialModel(nil)
	c := &publishTestClient{errTopic: "bad"}
	m.mqttClient = &MQTTClient{Client: c}
	m.topics.Items = []topics.Item{{Name: "good", Publish: true}, {Name: "bad", Publish: true}}
	m.message.SetPayload("original")
	m.SetFocus(idMessage)
	cmd := m.handlePublishKey()
	if m.handlePublishKey() != nil {
		t.Fatal("pending request should not be duplicated")
	}
	m.message.SetPayload("new draft")
	m.topics.Items[0].Name = "changed"
	applyMQTTCommand(m, cmd)
	if c.seen["good"] != "original" || c.seen["bad"] != "original" || len(c.seen) != 2 {
		t.Fatalf("request did not keep its snapshot: %+v", c.seen)
	}
	items := m.history.Items()
	if len(items) != 2 || items[0].Kind != "pub" || items[1].Kind != "log" {
		t.Fatalf("expected independent results, got %+v", items)
	}
	if _, ok := m.topicPulsePhase("bad"); ok {
		t.Fatal("failed target must not pulse")
	}
	if m.message.Input().Value() != "new draft" {
		t.Fatal("completion replaced edited draft")
	}
}

func TestPublishStaleBrokerAndDuplicateResultsIgnored(t *testing.T) {
	m, _ := initialModel(nil)
	m.mqttClient = &MQTTClient{Client: &mockClient{}}
	m.connections.Active = "first"
	m.topics.Items = []topics.Item{{Name: "a", Publish: true}}
	m.SetFocus(idMessage)
	cmd := m.handlePublishKey()
	result := cmd()
	m.connections.Active = "second"
	m.Update(result)
	if len(m.history.Items()) != 0 || len(m.payloads.Items()) != 0 || len(m.mqttOps.publishes) != 0 {
		t.Fatal("stale result affected another broker")
	}
	m.connections.Active = "first"
	m.Update(result)
	if len(m.history.Items()) != 0 {
		t.Fatal("duplicate stale result resurrected")
	}
}

type blockingPublishToken struct {
	entered chan struct{}
	release chan struct{}
}

func (t *blockingPublishToken) Wait() bool { <-t.release; return true }
func (t *blockingPublishToken) WaitTimeout(timeout time.Duration) bool {
	close(t.entered)
	select {
	case <-t.release:
		return true
	case <-time.After(timeout):
		return false
	}
}
func (t *blockingPublishToken) Done() <-chan struct{} { return t.release }
func (t *blockingPublishToken) Error() error          { return nil }

type blockingPublishClient struct {
	mockClient
	token mqtt.Token
}

func (c *blockingPublishClient) Publish(string, byte, bool, interface{}) mqtt.Token { return c.token }

func TestPublishCommandLeavesUpdateResponsiveWhileTokenWaits(t *testing.T) {
	m, _ := initialModel(nil)
	token := &blockingPublishToken{entered: make(chan struct{}), release: make(chan struct{})}
	m.mqttClient = &MQTTClient{Client: &blockingPublishClient{token: token}, publishTimeout: 5 * time.Second}
	m.topics.Items = []topics.Item{{Name: "a", Publish: true}}
	m.SetFocus(idMessage)
	cmd := m.handlePublishKey()
	result := make(chan tea.Msg, 1)
	go func() { result <- cmd() }()
	t.Cleanup(func() {
		close(token.release)
		select {
		case <-result:
		case <-time.After(time.Second):
			t.Error("publish worker did not exit after release")
		}
	})
	select {
	case <-token.entered:
	case <-time.After(time.Second):
		t.Fatal("publish command did not begin waiting")
	}
	m.Update(tea.KeyMsg{Type: tea.KeyTab})
	if m.FocusedID() != idHistory || m.pendingPublishes() != 1 || len(m.history.Items()) != 0 {
		t.Fatal("update did not remain responsive while publish pending")
	}
}

type disconnectedPublishClient struct{ mockClient }

func (*disconnectedPublishClient) IsConnected() bool { return false }

func TestPublishDisconnectedAndTimeoutNeverReportSuccess(t *testing.T) {
	for _, timeout := range []bool{false, true} {
		t.Run(map[bool]string{false: "disconnected", true: "timeout"}[timeout], func(t *testing.T) {
			m := reviewFixture(t, 80, 24)
			m.history.SetItems(nil)
			m.history.List().SetItems(nil)
			m.payloads.Clear()
			m.mqttClient = &MQTTClient{Client: &disconnectedPublishClient{}}
			if timeout {
				token := &blockingPublishToken{entered: make(chan struct{}), release: make(chan struct{})}
				m.mqttClient = &MQTTClient{Client: &blockingPublishClient{token: token}, publishTimeout: time.Millisecond}
			}
			m.topics.Items = []topics.Item{{Name: "target", Publish: true}}
			m.SetFocus(idMessage)
			applyMQTTCommand(m, m.handlePublishKey())
			items := m.history.Items()
			if len(items) != 1 || items[0].Kind != "log" || len(m.payloads.Items()) != 0 || m.pendingPublishes() != 0 || len(m.ui.animation.topicPulses) != 0 {
				t.Fatalf("failure created success feedback or leaked pending state: %+v", items)
			}
		})
	}
}

func TestSubscriptionStaleBrokerResultsIgnored(t *testing.T) {
	m := reviewFixture(t, 80, 24)
	m.mqttClient = &MQTTClient{Client: &failingClient{subErr: errors.New("old broker denied")}}
	m.connections.Active = "old"
	m.topics.Items = []topics.Item{{Name: "t", Subscribed: true, Publish: true}}
	cmd := m.handleTopicToggle(topics.ToggleMsg{Topic: "t", Subscribed: true})
	result := cmd()
	m.mqttClient = &MQTTClient{Client: &mockClient{}}
	m.connections.Active = "new"
	m.Update(result)
	if !m.topics.Items[0].Subscribed || !m.topics.Items[0].Publish || len(m.mqttOps.subscriptions) != 0 || len(m.ui.animation.topicPulses) != 0 {
		t.Fatal("old broker result affected new subscription state")
	}
}

func TestRestoredSubscriptionsArePendingAndReconcileOnFailure(t *testing.T) {
	m := reviewFixture(t, 80, 24)
	m.mqttClient = &MQTTClient{Client: &failingClient{subErr: errors.New("denied")}}
	m.topics.Items = []topics.Item{{Name: "t", Subscribed: true, Publish: true}, {Name: "inactive"}}
	cmd := m.SubscribeActiveTopics()
	if len(m.mqttOps.subscriptions) != 1 {
		t.Fatal("restored subscriptions were not dispatched asynchronously")
	}
	applyMQTTCommand(m, cmd)
	idx := m.topicIndexByName("t")
	if m.topics.Items[idx].Subscribed || !m.topics.Items[idx].Publish || len(m.mqttOps.subscriptions) != 0 {
		t.Fatal("restored subscription failure kept an incorrect active state")
	}
}

func TestSubscriptionFailureReconcilesStateWithoutPulse(t *testing.T) {
	m, _ := initialModel(nil)
	m.mqttClient = &MQTTClient{Client: &failingClient{subErr: errors.New("denied")}}
	m.topics.Items = []topics.Item{{Name: "t", Subscribed: true, Publish: true}}
	applyMQTTCommand(m, m.handleTopicToggle(topics.ToggleMsg{Topic: "t", Subscribed: true}))
	if m.topics.Items[0].Subscribed || !m.topics.Items[0].Publish || len(m.ui.animation.topicPulses) != 0 {
		t.Fatal("failed subscription did not roll back independently of publish state")
	}
}

func TestSubscriptionRapidTogglesSerializeLatestDesiredState(t *testing.T) {
	m, _ := initialModel(nil)
	m.mqttClient = &MQTTClient{Client: &mockClient{}}
	m.topics.Items = []topics.Item{{Name: "t", Subscribed: true}}
	first := m.handleTopicToggle(topics.ToggleMsg{Topic: "t", Subscribed: true})
	m.topics.Items[0].Subscribed = false
	if cmd := m.handleTopicToggle(topics.ToggleMsg{Topic: "t", Subscribed: false}); cmd != nil {
		t.Fatal("second toggle must wait for first result")
	}
	_, next := m.Update(first())
	if next == nil || len(m.mqttOps.subscriptions) != 1 {
		t.Fatal("latest desired state was not queued")
	}
	applyMQTTCommand(m, next)
	if m.topics.Items[0].Subscribed || len(m.mqttOps.subscriptions) != 0 {
		t.Fatal("queued unsubscribe did not settle")
	}
}
