package payloads

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/marang/emqutiti/confirm"
	connections "github.com/marang/emqutiti/connections"
)

// IDList identifies the payload list element.
const IDList = "payload-list"

// Item represents a topic/payload pair.
type Item struct {
	Topic   string
	Payload string
}

func (p Item) FilterValue() string { return p.Topic }
func (p Item) Title() string       { return p.Topic }
func (p Item) Description() string { return p.Payload }

// Snapshot represents a stored payload for persistence.
type Snapshot = connections.PayloadSnapshot

// API exposes payload management behavior to the rest of the application.
type API interface {
	Add(topic, payload string)
	Items() []Item
	SetItems([]Item)
	Snapshot() []Snapshot
	SetSnapshot([]Snapshot)
	Clear()
}

// Model defines the dependencies the component requires from the host model.
type Model interface {
	confirm.API
	SetClientMode() tea.Cmd
	SetFocus(string) tea.Cmd
	FocusedID() string
	ResetElemPos()
	SetElemPos(id string, pos int)
	OverlayHelp(string) string
	Width() int
	Height() int
}

// StatusListener provides status updates for components.
type StatusListener interface {
	ListenStatus() tea.Cmd
}

// Ensure Component satisfies API.
var _ API = (*Component)(nil)
