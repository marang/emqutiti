package emqutiti

import (
	"fmt"
	"slices"

	tea "github.com/charmbracelet/bubbletea"
)

type mqttOperationState struct {
	nextID        uint64
	publishes     map[uint64]publishRequest
	subscriptions map[string]subscriptionRequest
	publishError  string
}

type mqttRequest struct {
	id     uint64
	client *MQTTClient
	broker string
}

type publishRequest struct {
	mqttRequest
	topic, payload string
	retained       bool
}

type publishResultMsg struct {
	id  uint64
	err error
}

type subscriptionRequest struct {
	mqttRequest
	topic                     string
	target, desired, previous bool
}

type subscriptionResultMsg struct {
	topic string
	id    uint64
	err   error
}

func (m *model) newMQTTRequest() mqttRequest {
	m.mqttOps.nextID++
	return mqttRequest{id: m.mqttOps.nextID, client: m.mqttClient, broker: m.connections.Active}
}

func (m *model) currentMQTTRequest(r mqttRequest) bool {
	return r.client == m.mqttClient && r.broker == m.connections.Active
}

func mqttRequestError(r mqttRequest) error {
	if r.client == nil || r.client.Client == nil {
		return fmt.Errorf("no mqtt client")
	}
	if !r.client.Client.IsConnected() {
		return fmt.Errorf("broker disconnected")
	}
	return nil
}

func (m *model) pendingPublishes() int {
	count := 0
	for _, r := range m.mqttOps.publishes {
		if m.currentMQTTRequest(r.mqttRequest) {
			count++
		}
	}
	return count
}

func (m *model) pendingPublishTargets() []string {
	var ids []uint64
	for id, r := range m.mqttOps.publishes {
		if m.currentMQTTRequest(r.mqttRequest) {
			ids = append(ids, id)
		}
	}
	slices.Sort(ids)
	var targets []string
	for _, id := range ids {
		targets = append(targets, m.mqttOps.publishes[id].topic)
	}
	return targets
}

func (m *model) queuePublish(topic, payload string, retained bool) tea.Cmd {
	r := publishRequest{mqttRequest: m.newMQTTRequest(), topic: topic, payload: payload, retained: retained}
	if m.mqttOps.publishes == nil {
		m.mqttOps.publishes = map[uint64]publishRequest{}
	}
	m.mqttOps.publishes[r.id] = r
	return func() tea.Msg {
		err := mqttRequestError(r.mqttRequest)
		if err == nil {
			err = r.client.Publish(r.topic, 0, r.retained, r.payload)
		}
		return publishResultMsg{id: r.id, err: err}
	}
}

func (m *model) handlePublishResult(msg publishResultMsg) tea.Cmd {
	r, ok := m.mqttOps.publishes[msg.id]
	if !ok {
		return nil
	}
	delete(m.mqttOps.publishes, msg.id)
	if !m.currentMQTTRequest(r.mqttRequest) {
		return nil
	}
	if msg.err != nil {
		text := fmt.Sprintf("Publish failed for %s: %v", r.topic, msg.err)
		m.mqttOps.publishError = text
		m.history.Append(r.topic, "", "log", false, text)
		return nil
	}
	m.payloads.Add(r.topic, r.payload)
	m.history.Append(r.topic, r.payload, "pub", r.retained, "")
	return tea.Batch(m.startTopicPulse(r.topic), m.startHistoryPulse())
}

// Only one subscription request per topic runs at once; rapid toggles queue
// the latest desired state rather than racing broker acknowledgements.
func (m *model) queueSubscription(topic string, desired, previous bool) tea.Cmd {
	if r, ok := m.mqttOps.subscriptions[topic]; ok && m.currentMQTTRequest(r.mqttRequest) {
		r.desired = desired
		m.mqttOps.subscriptions[topic] = r
		return nil
	}
	r := subscriptionRequest{mqttRequest: m.newMQTTRequest(), topic: topic, target: desired, desired: desired, previous: previous}
	if m.mqttOps.subscriptions == nil {
		m.mqttOps.subscriptions = map[string]subscriptionRequest{}
	}
	m.mqttOps.subscriptions[topic] = r
	return func() tea.Msg {
		err := mqttRequestError(r.mqttRequest)
		if err == nil {
			if r.target {
				err = r.client.Subscribe(r.topic, 0, nil)
			} else {
				err = r.client.Unsubscribe(r.topic)
			}
		}
		return subscriptionResultMsg{topic: r.topic, id: r.id, err: err}
	}
}

func (m *model) handleSubscriptionResult(msg subscriptionResultMsg) tea.Cmd {
	r, ok := m.mqttOps.subscriptions[msg.topic]
	if !ok || r.id != msg.id {
		return nil
	}
	delete(m.mqttOps.subscriptions, msg.topic)
	if !m.currentMQTTRequest(r.mqttRequest) {
		return nil
	}
	action := "unsubscribe"
	if r.target {
		action = "subscribe"
	}
	m.logTopicAction(r.topic, action, msg.err)
	actual := r.previous
	if msg.err == nil {
		actual = r.target
	}
	if r.desired != actual && r.desired != r.target {
		return m.queueSubscription(r.topic, r.desired, actual)
	}
	if idx := m.topicIndexByName(r.topic); idx >= 0 {
		m.topics.Items[idx].Subscribed = actual
		m.topics.SortTopics()
		m.topics.RebuildActiveTopicList()
		if msg.err == nil {
			return m.startTopicPulse(r.topic)
		}
	}
	return nil
}
