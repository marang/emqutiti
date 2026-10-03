package emqutiti

import (
	"fmt"
	"slices"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/marang/emqutiti/topics"
	"github.com/marang/emqutiti/ui"
)

const topicSummaryLimit = 3

func formatTopicNames(names []string, limit int) string {
	if len(names) == 0 {
		return "none"
	}
	if limit <= 0 || len(names) <= limit {
		return strings.Join(names, ", ")
	}
	return fmt.Sprintf("%s +%d more", strings.Join(names[:limit], ", "), len(names)-limit)
}

func truncateTopicName(name string, width int) string {
	if width < 1 {
		return ""
	}
	return ansi.Truncate(name, width, "…")
}

func (m *model) publishTargets() []string {
	return effectivePublishTargets(m.topics.Items, m.topics.Selected())
}

func effectivePublishTargets(items []topics.Item, selected int) []string {
	var targets []string
	for _, t := range items {
		if t.Publish {
			targets = append(targets, t.Name)
		}
	}
	if len(targets) > 0 {
		return targets
	}
	if selected >= 0 && selected < len(items) {
		return []string{items[selected].Name}
	}
	return nil
}

func (m *model) explicitPublishTargets() []string {
	var targets []string
	for _, t := range m.topics.Items {
		if t.Publish {
			targets = append(targets, t.Name)
		}
	}
	return targets
}

func topicStateLegend() string {
	return "subscribed = read  publish target = write  inactive = off"
}

func topicStateHint(t topics.Item, publishing bool) string {
	switch {
	case publishing:
		if t.Subscribed {
			return "Read subscription and publish target"
		}
		return "Publish target"
	case t.Subscribed:
		return "Read subscription"
	default:
		return "Off"
	}
}

func topicShortcutHint(width int) string {
	full := "[Enter] toggle sub  [p] toggle pub  [Del] remove"
	if lipgloss.Width(full) <= width {
		return full
	}
	return "[Enter] sub  [p] pub  [Del] remove"
}

func (m *model) topicHoverHint(idx int) string {
	if idx < 0 || idx >= len(m.topics.Items) {
		return ""
	}
	t := m.topics.Items[idx]
	return topicStateHint(t, slices.Contains(m.publishTargets(), t.Name)) + "."
}

func (m *model) selectedTopicHint() string {
	sel := m.topics.Selected()
	if sel < 0 || sel >= len(m.topics.Items) {
		return "Topics: no selected topic. Add a topic first."
	}
	return "Topics: " + m.topicHoverHint(sel)
}

func (m *model) messageTargetText(limit int) string {
	targets := m.publishTargets()
	if len(targets) == 0 {
		return "no target"
	}
	return formatTopicNames(targets, limit)
}

func (m *model) messageTargetPreview() string {
	mode := "selected"
	targets := m.publishTargets()
	if targets := m.pendingPublishTargets(); len(targets) > 0 {
		return "Message -> publishing to: " + formatTopicNames(targets, 2)
	}
	if len(targets) == 0 {
		return "Message -> no target"
	}
	if len(m.explicitPublishTargets()) > 0 {
		mode = "marked"
	}
	prefix := "Message -> publish to (" + mode + "): "
	if m.ui.width > 0 && m.ui.width-6-lipgloss.Width(prefix) < 8 {
		prefix = "Publish to (" + mode + "): "
	}
	return prefix + formatTopicNames(targets, 2)
}

func (m *model) messageHint() string {
	if targets := m.pendingPublishTargets(); len(targets) > 0 {
		return "Message: publishing to " + formatTopicNames(targets, topicSummaryLimit) + "."
	}
	if m.mqttOps.publishError != "" {
		return "Message: " + m.mqttOps.publishError
	}
	targets := m.messageTargetText(topicSummaryLimit)
	if targets == "no target" {
		return "Message: no publish target. Add or select a topic first."
	}
	if !m.isConnected() {
		return fmt.Sprintf("Message: broker disconnected. Publish to: %s.", targets)
	}
	return "Message: publish to " + targets + "."
}

func (m *model) historyHint() string {
	if m.history.ShowArchived() {
		return "History: archived messages."
	}
	full := "History: [Shift+Up/Down]/[Shift+Click] range; [Shift+Space] toggle"
	if lipgloss.Width(full)+2 <= m.ui.width-4 {
		return full
	}
	return "History: [Shift+Up/Down] select"
}

func (m *model) focusHint(id string) string {
	switch id {
	case idTopic:
		topic := strings.TrimSpace(m.topics.Input.Value())
		if topic == "" {
			return "Topic input: type a topic and press [Enter] to subscribe."
		}
		if m.topics.HasTopic(topic) {
			return fmt.Sprintf("Topic input: %q already exists.", truncateTopicName(topic, 40))
		}
		return fmt.Sprintf("Topic input: [Enter] subscribes to %q.", truncateTopicName(topic, 40))
	case idTopics:
		return m.selectedTopicHint()
	case idMessage:
		return m.messageHint()
	case idHistory:
		return m.historyHint()
	case idHelp:
		return "Help: click ? or focus it to open the full shortcut and workflow guide."
	default:
		return "Hover or focus an area to see what it does."
	}
}

func (m *model) hoverHint() (string, bool) {
	switch m.ui.hoveredID {
	case idTopic, idMessage, idHistory, idHelp:
		return m.focusHint(m.ui.hoveredID), true
	case idTopics:
		if m.ui.hoveredTopic >= 0 {
			return m.topicHoverHint(m.ui.hoveredTopic), true
		}
		return "Topics: publish targets receive outgoing messages; read subscriptions receive broker messages.", true
	default:
		return "", false
	}
}

func (m *model) contextHelpText() string {
	if hint := m.panelResizeHint(); hint != "" {
		return hint
	}
	if hint, ok := m.hoverHint(); ok {
		return "~ " + hint
	}
	return "> " + m.focusHint(m.FocusedID())
}

func (m *model) contextHelpDetailText() string {
	if m.ui.panelResize.id != "" {
		return "[Esc] cancel | Release to apply"
	}
	if m.ui.hoveredResizeID != "" {
		return "Hold left mouse button and drag"
	}
	id := m.FocusedID()
	if m.ui.hoveredID != "" {
		id = m.ui.hoveredID
	}
	switch id {
	case idTopic:
		if topic := strings.TrimSpace(m.topics.Input.Value()); topic != "" && !m.topics.HasTopic(topic) {
			full := "[Enter] adds and subscribes | [Tab] to topics"
			if lipgloss.Width(full) <= m.ui.width-4 {
				return full
			}
			return "[Enter] subscribe  [Tab] topics"
		}
		return "[Tab] to topics | Type a new topic"
	case idTopics:
		return topicShortcutHint(m.ui.width - 4)
	case idMessage:
		if m.pendingPublishes() > 0 {
			return "Waiting for MQTT result | Draft remains editable"
		}
		if !m.isConnected() {
			full := "[Ctrl+B] brokers | [Ctrl+S] reports disconnected"
			if lipgloss.Width(full) <= m.ui.width-4 {
				return full
			}
			return "[Ctrl+B] brokers  [Ctrl+S] offline"
		}
		if len(m.publishTargets()) == 0 {
			return "Select or add a topic before publishing"
		}
		full := "[Ctrl+S]/[Ctrl+Enter*] publish  [Ctrl+E] retained"
		if m.ui.modifiedKeyInput && lipgloss.Width(full) <= m.ui.width-4 {
			return full
		}
		return "[Ctrl+S] publish  [Ctrl+E] retained"
	case idHistory:
		full := "[Enter] details  [/] filter  [Ctrl+C] copy"
		if lipgloss.Width(full) <= m.ui.width-4 {
			return full
		}
		return "[Enter] view [/] find [Ctrl+C] copy"
	case idHelp:
		return "Open help for full shortcuts and MQTT workflow notes"
	default:
		return "Hover or focus an area to see available actions"
	}
}

func (m *model) renderContextHelp() string {
	width := m.ui.width - 2
	if width < 1 {
		width = 1
	}
	innerWidth := width - 2
	if innerWidth < 1 {
		innerWidth = width
	}
	lines := []string{
		ansi.Truncate(m.contextHelpText(), innerWidth, "…"),
		ansi.Truncate(m.contextHelpDetailText(), innerWidth, "…"),
	}
	return ui.ContextHelpStyle.Width(width).Height(2).Render(strings.Join(lines, "\n"))
}

func (m *model) pointOverHelp(msg tea.MouseMsg) bool {
	helpWidth := lipgloss.Width(ui.HelpStyle.Render("?"))
	return msg.Y == 0 && msg.X >= m.ui.width-helpWidth && msg.X < m.ui.width
}

func (m *model) updateHoverState(msg tea.MouseMsg) {
	m.ui.hoveredID = ""
	m.ui.hoveredTopic = -1
	m.ui.hoveredResizeID = m.panelResizeBorderAt(msg)
	if m.pointOverHelp(msg) {
		m.ui.hoveredID = idHelp
		return
	}
	y := m.clientContentY(msg.Y)
	switch {
	case pointInElement(y, m.ui.elemPos[idTopic]-1, m.ui.elemHeight[idTopic]):
		m.ui.hoveredID = idTopic
	case pointInElement(y, m.ui.elemPos[idTopics]-1, m.ui.elemHeight[idTopics]):
		m.ui.hoveredID = idTopics
		m.ui.hoveredTopic = m.topics.TopicAtPosition(msg.X, y)
	case pointInElement(y, m.ui.elemPos[idMessage]-1, m.ui.elemHeight[idMessage]):
		m.ui.hoveredID = idMessage
	case pointInElement(y, m.ui.elemPos[idHistory]-1, m.ui.elemHeight[idHistory]):
		m.ui.hoveredID = idHistory
	}
}

func (m *model) clearHoverState() {
	m.ui.hoveredID = ""
	m.ui.hoveredTopic = -1
	m.ui.hoveredResizeID = ""
}

func (m *model) clientViewportTop() int {
	return 2 + contextHelpHeight(m.renderContextHelp())
}

func (m *model) clientContentY(screenY int) int {
	return screenY - m.clientViewportTop() + m.ui.viewport.YOffset
}

func (m *model) clientContentMouseMsg(msg tea.MouseMsg) tea.MouseMsg {
	msg.Y = m.clientContentY(msg.Y)
	return msg
}

func pointInElement(y, top, height int) bool {
	if height < 1 {
		height = 1
	}
	return y >= top && y < top+height
}

func contextHelpHeight(help string) int {
	if help == "" {
		return 0
	}
	return lipgloss.Height(help)
}
