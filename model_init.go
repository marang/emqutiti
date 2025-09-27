package emqutiti

import (
	"fmt"
	"os"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/marang/emqutiti/confirm"
	"github.com/marang/emqutiti/connections"
	"github.com/marang/emqutiti/constants"
	"github.com/marang/emqutiti/help"
	"github.com/marang/emqutiti/history"
	"github.com/marang/emqutiti/logs"
	"github.com/marang/emqutiti/message"
	"github.com/marang/emqutiti/payloads"
	"github.com/marang/emqutiti/topics"
	"github.com/marang/emqutiti/traces"
	"github.com/marang/emqutiti/ui"
)

type navAdapter struct{ navigator }

func (n navAdapter) SetMode(mode constants.AppMode) tea.Cmd { return n.navigator.SetMode(mode) }
func (n navAdapter) Width() int                             { return n.navigator.Width() }
func (n navAdapter) Height() int                            { return n.navigator.Height() }
func (n navAdapter) PreviousMode() constants.AppMode        { return n.navigator.PreviousMode() }

func initMessage() message.State {
	ta := textarea.New()
	ta.Placeholder = "Enter Message"
	ta.CharLimit = 10000
	ta.ShowLineNumbers = false
	ta.SetPromptFunc(0, func(i int) string {
		return fmt.Sprintf("%d> ", i+1)
	})
	promptColor := ui.ColGray
	ta.FocusedStyle.Prompt = lipgloss.NewStyle().Foreground(promptColor)
	ta.BlurredStyle.Prompt = lipgloss.NewStyle().Foreground(promptColor)
	ta.Blur()
	ta.Cursor.Style = ui.NoCursor
	ta.SetWidth(0)
	ta.SetHeight(6)
	ta.FocusedStyle.CursorLine = ui.FocusedStyle
	ta.BlurredStyle.CursorLine = ui.BlurredStyle
	ms := message.State{
		TA: ta,
	}
	return ms
}

func savedLayoutProfile(state connections.State) string {
	if profileName != "" {
		return profileName
	}
	if state.Active != "" {
		return state.Active
	}
	return state.Manager.DefaultProfileName
}

func applySnapshotHeights(layout *layoutConfig, snap connections.ConnectionSnapshot) {
	if snap.MessageHeight > 0 {
		layout.message.height = snap.MessageHeight
	}
	if snap.TopicsHeight > 0 {
		layout.topics.height = snap.TopicsHeight
	}
	if snap.HistoryHeight > 0 {
		layout.history.height = snap.HistoryHeight
	}
}

// initialModel creates the main program model with optional connection data.
func initialModel(conns *connections.Connections) (*model, error) {
	order := append([]string(nil), focusByMode[constants.ModeClient]...)
	cs, loadErr := initConnections(conns)
	st, herr := history.OpenStore("")
	if herr != nil && loadErr == nil {
		loadErr = herr
	}
	if herr != nil {
		fmt.Fprintf(os.Stderr, "history store error: %v\n", herr)
		st = nil
	}
	ms := initMessage()
	layout := initLayout()
	initialProfile := savedLayoutProfile(cs)
	if snap, ok := cs.Saved[initialProfile]; ok {
		applySnapshotHeights(&layout, snap)
	}
	ms.TA.SetHeight(layout.message.height)
	tr := traces.Init()
	m := &model{
		connections: cs,
		ui:          initUI(order),
		layout:      layout,
	}
	m.history = history.NewComponent(historyModelAdapter{m}, st)
	if herr != nil {
		m.history.Append("", "", "log", false, fmt.Sprintf("history store error: %v", herr))
	}
	m.message = message.NewComponent(m, ms)
	m.logs = logs.New(navAdapter{m}, &m.ui.width, &m.ui.height, &m.ui.elemPos)
	m.help = help.New(navAdapter{m}, &m.ui.width, &m.ui.height, &m.ui.elemPos)
	m.confirm = confirm.NewDialog(m, m, nil, nil, nil)
	connComp := connections.NewComponent(navAdapter{m}, m)
	m.topics = topics.New(m)
	m.payloads = payloads.New(m, &m.connections)
	m.traces = traces.NewComponent(m, tr, m.tracesStore())
	m.applySavedLayout(initialProfile)
	initComponents(m, order, connComp)
	m.SetFocus(idTopics)
	if err := initImporter(m); err != nil {
		return nil, err
	}
	m.topics.RebuildActiveTopicList()
	return m, loadErr
}

func (m *model) applySavedLayout(profile string) {
	if profile == "" {
		return
	}
	snap, ok := m.connections.Saved[profile]
	if !ok {
		return
	}
	applySnapshotHeights(&m.layout, snap)
	if m.message != nil {
		m.message.Input().SetHeight(m.layout.message.height)
	}
	if m.history != nil && m.ui.width > 0 && m.ui.height > 0 {
		hw, hh := calcHistorySize(m.ui.width, m.ui.height, m.layout.history.height)
		m.layout.history.height = hh
		m.history.List().SetSize(hw, hh)
	}
}

// Init enables initial Tea behavior such as mouse support.
func (m *model) Init() tea.Cmd {
	cmds := []tea.Cmd{tea.EnableMouseCellMotion}
	if profileName == "" {
		if name := m.connections.Manager.DefaultProfileName; name != "" {
			for _, p := range m.connections.Manager.Profiles {
				if p.Name == name {
					cmds = append(cmds, m.Connect(p))
					break
				}
			}
		}
	}
	return tea.Batch(cmds...)
}
