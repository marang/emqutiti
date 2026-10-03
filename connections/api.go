package connections

import (
	tea "github.com/charmbracelet/bubbletea"
	mqtt "github.com/eclipse/paho.mqtt.golang"

	"github.com/marang/emqutiti/constants"
)

// Client defines the MQTT functions used by the connections package.
type Client interface {
	Subscribe(topic string, qos byte, callback mqtt.MessageHandler) error
	Disconnect()
}

// ConnectResult represents the outcome of a connection attempt.
type ConnectResult struct {
	Client  Client
	Profile Profile
	Err     error
}

// API defines the methods used by the connections component to interact
// with the application without depending on concrete model types.
type API interface {
	Manager() *Connections
	ListenStatus() tea.Cmd
	SendStatus(string)
	FlushStatus()
	RefreshConnectionItems()
	SubscribeActiveTopics() tea.Cmd
	ConnectionMessage() string
	SetConnectionMessage(string)
	Active() string
	BeginAdd()
	BeginEdit(index int)
	BeginDelete(index int)
	Connect(p Profile) tea.Cmd
	HandleConnectResult(msg ConnectResult) tea.Cmd
	DisconnectActive()
	ResizeTraces(width, height int)
	ResetElemPos()
	SetElemPos(id string, pos int)
	OverlayHelp(view string) string
	SetConnecting(name string)
	SetConnected(name string)
	SetDisconnected(name, detail string)
}

// Navigator exposes navigation helpers required by the component.
type Navigator interface {
	SetMode(mode constants.AppMode) tea.Cmd
	Width() int
	Height() int
}

// Focusable represents a UI element that can gain or lose focus.
type Focusable interface {
	Focus()
	Blur()
	IsFocused() bool
	View() string
}
