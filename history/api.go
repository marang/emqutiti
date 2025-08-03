package history

import (
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

// Mode represents an application mode from the parent model.
// It is defined as an empty interface so any concrete mode type
// can satisfy the history component's requirements.
type Mode interface{}

// Model describes the behaviour the history component expects
// from the root application model.
type Model interface {
	SetMode(Mode) tea.Cmd
	PreviousMode() Mode
	CurrentMode() Mode
	SetFocus(id string) tea.Cmd
	Width() int
	Height() int
	OverlayHelp(string) string
}

// Store defines operations for storing and querying history messages.
type Store interface {
	Append(Message)
	Search(archived bool, topics []string, start, end time.Time, payload string) []Message
	Delete(key string) error
	Archive(key string) error
	Count(archived bool) int
	Close() error
}

// Focusable represents a focusable element in the parent model.
type Focusable interface{}

// ID identifies the history list for focus management.
const ID = "history"

// NewComponent constructs a history Component bound to the provided Model.
// The supplied Store may be nil when persistence is not required.
func NewComponent(m Model, st Store) *Component {
	del := historyDelegate{}
	lst := list.New([]list.Item{}, del, 0, 0)
	lst.SetShowTitle(false)
	lst.SetShowStatusBar(false)
	lst.SetShowPagination(false)
	lst.DisableQuitKeybindings()
	hs := historyState{
		list:            lst,
		items:           []Item{},
		store:           st,
		selectionAnchor: -1,
		detail:          viewport.New(0, 0),
	}
	if st != nil {
		msgs := st.Search(false, nil, time.Time{}, time.Time{}, "")
		var items []list.Item
		hs.items, items = MessagesToItems(msgs)
		hs.list.SetItems(items)
	}
	return &Component{historyState: &hs, m: m}
}

// OpenStore opens or creates a persistent history store for the given profile.
// If profile is empty, "default" is used.
func OpenStore(profile string) (Store, error) {
	return openStore(profile)
}
