package connections

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/marang/emqutiti/constants"
)

type testNav struct{ mode constants.AppMode }

func (t *testNav) SetMode(m constants.AppMode) tea.Cmd { t.mode = m; return nil }
func (t *testNav) Width() int                          { return 80 }
func (t *testNav) Height() int                         { return 24 }

type testAPI struct {
	began bool
	mgr   *Connections
}

func (t *testAPI) Manager() *Connections                     { return t.mgr }
func (t *testAPI) ListenStatus() tea.Cmd                     { return nil }
func (t *testAPI) SendStatus(string)                         {}
func (t *testAPI) FlushStatus()                              {}
func (t *testAPI) RefreshConnectionItems()                   {}
func (t *testAPI) SubscribeActiveTopics() tea.Cmd            { return nil }
func (t *testAPI) ConnectionMessage() string                 { return "" }
func (t *testAPI) SetConnectionMessage(string)               {}
func (t *testAPI) Active() string                            { return "" }
func (t *testAPI) BeginAdd()                                 { t.began = true }
func (t *testAPI) BeginEdit(int)                             {}
func (t *testAPI) BeginDelete(int)                           {}
func (t *testAPI) Connect(Profile) tea.Cmd                   { return nil }
func (t *testAPI) HandleConnectResult(ConnectResult) tea.Cmd { return nil }
func (t *testAPI) DisconnectActive()                         {}
func (t *testAPI) ResizeTraces(int, int)                     {}
func (t *testAPI) ResetElemPos()                             {}
func (t *testAPI) SetElemPos(string, int)                    {}
func (t *testAPI) OverlayHelp(view string) string            { return view }
func (t *testAPI) SetConnecting(string)                      {}
func (t *testAPI) SetConnected(string)                       {}
func (t *testAPI) SetDisconnected(string, string)            {}

func TestAddKeyTriggersBeginAdd(t *testing.T) {
	mgr := NewConnectionsModel()
	api := &testAPI{mgr: &mgr}
	nav := &testNav{}
	c := NewComponent(nav, api)
	c.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	if !api.began {
		t.Fatalf("expected BeginAdd called")
	}
	if nav.mode != constants.ModeEditConnection {
		t.Fatalf("expected mode %v, got %v", constants.ModeEditConnection, nav.mode)
	}
}
