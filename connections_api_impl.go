package emqutiti

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/marang/emqutiti/connections"
	"github.com/marang/emqutiti/history"
)

// Manager returns the underlying Connections manager.
func (m *model) Manager() *connections.Connections { return &m.connections.Manager }

func (m *model) SetConnecting(name string) { m.connections.SetConnecting(name) }
func (m *model) SetConnected(name string)  { m.connections.SetConnected(name) }
func (m *model) SetDisconnected(name, detail string) {
	m.connections.SetDisconnected(name, detail)
}

func (m *model) SendStatus(msg string) { m.connections.SendStatus(msg) }
func (m *model) FlushStatus()          { m.connections.FlushStatus() }

func (m *model) RefreshConnectionItems() { m.connections.RefreshConnectionItems() }
func (m *model) SubscribeActiveTopics() {
	if m.mqttClient == nil {
		return
	}
	for _, t := range m.topics.Items {
		if t.Subscribed {
			if err := m.mqttClient.Subscribe(t.Name, 0, nil); err != nil {
				m.connections.SendStatus(fmt.Sprintf("Subscribe error for %s: %v", t.Name, err))
			}
		}
	}
}
func (m *model) ConnectionMessage() string       { return m.connections.Connection }
func (m *model) SetConnectionMessage(msg string) { m.connections.Connection = msg }
func (m *model) Active() string                  { return m.connections.Active }

func (m *model) validProfileIndex(idx int) bool {
	return idx >= 0 && idx < len(m.connections.Manager.Profiles)
}
func (m *model) BeginAdd() {
	f := connections.NewForm(connections.Profile{}, -1)
	m.connections.Form = &f
}
func (m *model) BeginEdit(index int) {
	if m.validProfileIndex(index) {
		f := connections.NewForm(m.connections.Manager.Profiles[index], index)
		m.connections.Form = &f
	}
}
func (m *model) BeginDelete(index int) {
	if !m.validProfileIndex(index) {
		return
	}
	name := m.connections.Manager.Profiles[index].Name
	info := "This also deletes history and traces"
	rf := func() tea.Cmd { return m.SetFocus(m.ui.focusOrder[m.ui.focusIndex]) }
	m.StartConfirm(
		fmt.Sprintf("Delete broker '%s'? [y/n]", name),
		info,
		rf,
		func() tea.Cmd {
			m.connections.Manager.DeleteConnection(index)
			m.RefreshConnectionItems()
			return nil
		},
		nil,
	)
}
func (m *model) Connect(p connections.Profile) tea.Cmd {
	m.connections.FlushStatus()
	if p.FromEnv {
		connections.ApplyEnvVars(&p)
	}
	connections.ApplyDefaultPassword(&p)
	m.connections.SetConnecting(p.Name)
	brokerURL := fmt.Sprintf("%s://%s:%d", p.Schema, p.Host, p.Port)
	m.connections.Connection = "Connecting to " + brokerURL
	m.RefreshConnectionItems()
	return connectBroker(p, m.connections.SendStatus)
}
func (m *model) HandleConnectResult(msg connections.ConnectResult) {
	profile := msg.Profile
	brokerURL := fmt.Sprintf("%s://%s:%d", profile.Schema, profile.Host, profile.Port)
	if err := msg.Err; err != nil {
		m.connections.SetDisconnected(profile.Name, fmt.Sprintf("Failed to connect to %s: %v", brokerURL, err))
		m.connections.Connection = fmt.Sprintf("Failed to connect to %s: %v", brokerURL, err)
		m.RefreshConnectionItems()
		return
	}
	m.mqttClient = msg.Client.(*MQTTClient)
	m.connections.Active = profile.Name
	if st := m.history.Store(); st != nil {
		st.Close()
	}
	idx, err := history.OpenStore(profile.Name)
	if err != nil {
		m.connections.SendStatus(fmt.Sprintf("History open error for %s: %v", profile.Name, err))
	} else if idx != nil {
		m.history.SetStore(idx)
		msgs := idx.Search(false, nil, time.Time{}, time.Time{}, "")
		hitems := make([]history.Item, len(msgs))
		items := make([]list.Item, len(msgs))
		for i, mmsg := range msgs {
			hi := history.Item{Timestamp: mmsg.Timestamp, Topic: mmsg.Topic, Payload: mmsg.Payload, Kind: mmsg.Kind, Archived: mmsg.Archived, Retained: mmsg.Retained}
			hitems[i] = hi
			items[i] = hi
		}
		m.history.SetItems(hitems)
		m.history.List().SetItems(items)
	}
	ts, ps := m.connections.RestoreState(profile.Name)
	m.topics.SetSnapshot(ts)
	m.payloads.SetSnapshot(ps)
	m.applySavedLayout(profile.Name)
	m.topics.SortTopics()
	m.topics.RebuildActiveTopicList()
	m.SubscribeActiveTopics()
	m.connections.Connection = "Connected to " + brokerURL
	m.connections.SetConnected(profile.Name)
	m.RefreshConnectionItems()
}
func (m *model) DisconnectActive() {
	if m.mqttClient != nil {
		m.mqttClient.Disconnect()
		m.connections.SetDisconnected(m.connections.Active, "")
		m.RefreshConnectionItems()
		m.connections.Connection = ""
		m.connections.Active = ""
		m.mqttClient = nil
	}
}
func (m *model) ResizeTraces(width, height int) { m.traces.List().SetSize(width, height) }

var _ connections.API = (*model)(nil)
