package emqutiti

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	connections "github.com/marang/emqutiti/connections"
)

// statusFunc reports connection status messages.
type statusFunc func(string)

// connectBroker attempts to connect to the given profile and reports status via callback.
func connectBroker(p connections.Profile, fn statusFunc) tea.Cmd {
	return func() tea.Msg {
		if fn != nil {
			brokerURL := p.BrokerURL()
			fn(fmt.Sprintf("Connecting to %s", brokerURL))
		}
		client, err := NewMQTTClient(p, fn)
		return connections.ConnectResult{Client: client, Profile: p, Err: err}
	}
}

type mqttListenClosedMsg struct{}

// listenMessages waits for incoming MQTT messages on the provided channel.
func listenMessages(ch chan MQTTMessage) tea.Cmd {
	return func() tea.Msg {
		if ch == nil {
			return mqttListenClosedMsg{}
		}
		msg, ok := <-ch
		if !ok {
			return mqttListenClosedMsg{}
		}
		return msg
	}
}
