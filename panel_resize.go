package emqutiti

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/marang/emqutiti/constants"
	"github.com/marang/emqutiti/ui"
)

type panelResizeState struct {
	id             string
	startY         int
	originalHeight int
	viewportOffset int
}

type panelResizeSpec struct {
	config       *boxConfig
	name         string
	min, max     int
	linesPerUnit int
}

func (m *model) panelResizeSpec(id string) (panelResizeSpec, bool) {
	available := max(1, m.ui.height-4)
	spec := panelResizeSpec{min: 1, max: max(1, available-2), linesPerUnit: 1}
	switch id {
	case idTopics:
		spec.config, spec.name = &m.layout.topics, "Topics"
		spec.linesPerUnit = lipgloss.Height(ui.Chip.Render(""))
		spec.max = max(1, (available-2-lipgloss.Height(m.topicLegendInfo()))/spec.linesPerUnit)
	case idMessage:
		spec.config, spec.name = &m.layout.message, "Message"
		spec.max = max(1, spec.max-m.message.FooterHeight())
		if limit := m.message.Input().MaxHeight; limit > 0 {
			spec.max = min(spec.max, limit)
		}
	case idHistory:
		spec.config, spec.name = &m.layout.history, "History"
		l := m.history.List()
		spec.min = 3
		if l.ShowHelp() {
			spec.min += lipgloss.Height(l.Help.View(*l)) + l.Styles.HelpStyle.GetVerticalFrameSize()
		}
		if m.history.FilterQuery() != "" {
			spec.min++
		}
		spec.max = max(spec.min, spec.max)
	default:
		return panelResizeSpec{}, false
	}
	return spec, true
}

func (m *model) setPanelHeight(id string, height int) {
	spec, ok := m.panelResizeSpec(id)
	if !ok {
		return
	}
	height = min(max(spec.min, height), spec.max)
	if spec.config.height == height {
		return
	}
	spec.config.height = height
	switch id {
	case idTopics:
		// A resize preserves the topic viewport anchor instead of revealing selection.
		m.ui.topicLayoutHeight = height * spec.linesPerUnit
	case idMessage:
		m.message.Input().SetHeight(height)
	case idHistory:
		m.history.List().SetSize(max(1, m.ui.width-4), height)
	}
}

func (m *model) clampPanelHeights() {
	if m.ui.height <= 0 {
		return
	}
	for _, id := range []string{idTopics, idMessage, idHistory} {
		spec, _ := m.panelResizeSpec(id)
		m.setPanelHeight(id, spec.config.height)
	}
}

func (m *model) resizeFocusedPanel(delta int) tea.Cmd {
	if m.CurrentMode() == constants.ModeClient {
		if spec, ok := m.panelResizeSpec(m.FocusedID()); ok {
			m.setPanelHeight(m.FocusedID(), spec.config.height+delta)
		}
	}
	return nil
}

func (m *model) resetFocusedPanel() tea.Cmd {
	if m.CurrentMode() != constants.ModeClient {
		return nil
	}
	defaults := initLayout()
	heights := map[string]int{idTopics: defaults.topics.height, idMessage: defaults.message.height, idHistory: defaults.history.height}
	if height, ok := heights[m.FocusedID()]; ok {
		m.setPanelHeight(m.FocusedID(), height)
	}
	return nil
}

func (m *model) panelResizeBorderAt(msg tea.MouseMsg) string {
	if m.CurrentMode() != constants.ModeClient || msg.X < 1 || msg.X >= m.ui.width-1 ||
		msg.Y < m.clientViewportTop() || msg.Y >= m.clientViewportTop()+m.ui.viewport.Height {
		return ""
	}
	y := m.clientContentY(msg.Y)
	for _, id := range []string{idTopics, idMessage, idHistory} {
		pos, ok := m.ui.elemPos[id]
		if ok && m.ui.elemHeight[id] >= 2 && y == pos+m.ui.elemHeight[id]-2 {
			return id
		}
	}
	return ""
}

func (m *model) cancelPanelResize() {
	drag := m.ui.panelResize
	if drag.id == "" {
		return
	}
	m.setPanelHeight(drag.id, drag.originalHeight)
	// Rebuild the original content before SetYOffset applies its content bounds.
	m.viewClient()
	m.ui.viewport.SetYOffset(drag.viewportOffset)
	m.ui.panelResize = panelResizeState{}
	m.clearHoverState()
}

// handlePanelResize captures the whole drag before focus, editor and chip routing.
func (m *model) handlePanelResize(msg tea.Msg) (tea.Cmd, bool) {
	drag := m.ui.panelResize
	if drag.id != "" {
		switch msg := msg.(type) {
		case ctrlEnterMsg, shiftedSpaceMsg:
			return nil, true
		case tea.WindowSizeMsg:
			m.cancelPanelResize()
		case tea.KeyMsg:
			if msg.Type == tea.KeyCtrlD {
				m.cancelPanelResize()
				return nil, false
			}
			if msg.Type == tea.KeyEsc {
				m.cancelPanelResize()
			}
			return nil, true
		case tea.MouseMsg:
			if msg.Action == tea.MouseActionMotion || msg.Action == tea.MouseActionRelease {
				spec, _ := m.panelResizeSpec(drag.id)
				m.setPanelHeight(drag.id, drag.originalHeight+(msg.Y-drag.startY)/spec.linesPerUnit)
				if msg.Action == tea.MouseActionRelease || msg.Button != tea.MouseButtonLeft {
					m.ui.panelResize = panelResizeState{}
					m.updateHoverState(msg)
				}
			}
			return nil, true
		}
	}
	if msg, ok := msg.(tea.MouseMsg); ok && msg.Action == tea.MouseActionPress && msg.Button == tea.MouseButtonLeft {
		if id := m.panelResizeBorderAt(msg); id != "" {
			spec, _ := m.panelResizeSpec(id)
			m.ui.panelResize = panelResizeState{id: id, startY: msg.Y, originalHeight: spec.config.height, viewportOffset: m.ui.viewport.YOffset}
			return nil, true
		}
	}
	return nil, false
}

func (m *model) panelResizeHint() string {
	if id := m.ui.panelResize.id; id != "" {
		spec, _ := m.panelResizeSpec(id)
		return fmt.Sprintf("> Resizing %s: %d lines.", spec.name, spec.config.height*spec.linesPerUnit)
	}
	if spec, ok := m.panelResizeSpec(m.ui.hoveredResizeID); ok {
		return "~ " + spec.name + ": drag bottom border to resize."
	}
	return ""
}
