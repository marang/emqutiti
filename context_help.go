package emqutiti

import (
	"fmt"
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
	var targets []string
	for _, t := range m.topics.Items {
		if t.Publish {
			targets = append(targets, t.Name)
		}
	}
	if len(targets) > 0 {
		return targets
	}
	sel := m.topics.Selected()
	if sel >= 0 && sel < len(m.topics.Items) {
		return []string{m.topics.Items[sel].Name}
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

func topicStateHint(t topics.Item) string {
	switch {
	case t.Publish:
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

func topicActionHint(t topics.Item) string {
	name := truncateTopicName(t.Name, 36)
	subAction := "subscribes to"
	if t.Subscribed {
		subAction = "unsubscribes"
	}
	pubAction := "marks it as publish target"
	if t.Publish {
		pubAction = "clears publish target"
	}
	return fmt.Sprintf("Topic %q: %s. Enter %s; p %s.", name, topicStateHint(t), subAction, pubAction)
}

func (m *model) topicHoverHint(idx int) string {
	if idx < 0 || idx >= len(m.topics.Items) {
		return ""
	}
	t := m.topics.Items[idx]
	return fmt.Sprintf("%s. Left click selects; Enter toggles read subscription; p toggles publish target; right click/Delete removes.", topicStateHint(t))
}

func (m *model) selectedTopicHint() string {
	sel := m.topics.Selected()
	if sel < 0 || sel >= len(m.topics.Items) {
		return "Topics: no selected topic. Add a topic first."
	}
	return topicActionHint(m.topics.Items[sel])
}

func (m *model) messageTargetText(limit int) string {
	targets := m.publishTargets()
	if len(targets) == 0 {
		return "no target"
	}
	return formatTopicNames(targets, limit)
}

func (m *model) messageTargetPreview() string {
	explicit := m.explicitPublishTargets()
	switch {
	case len(explicit) > 0:
		return "Message -> publish to: " + formatTopicNames(explicit, 2)
	case len(m.publishTargets()) > 0:
		return "Message -> publish to: " + formatTopicNames(m.publishTargets(), 1)
	default:
		return "Message -> no target"
	}
}

func (m *model) messageHint() string {
	targets := m.messageTargetText(topicSummaryLimit)
	if targets == "no target" {
		return "Message: no publish target. Add or select a topic first."
	}
	return fmt.Sprintf("Message: Ctrl+S publishes to %s; Ctrl+E publishes retained.", targets)
}

func (m *model) focusHint(id string) string {
	switch id {
	case idTopic:
		topic := strings.TrimSpace(m.topics.Input.Value())
		if topic == "" {
			return "Topic input: type a topic and press Enter to subscribe."
		}
		return fmt.Sprintf("Topic input: Enter subscribes to %q.", truncateTopicName(topic, 40))
	case idTopics:
		return m.selectedTopicHint()
	case idMessage:
		return m.messageHint()
	case idHistory:
		return "History: Enter opens details; / filters; a archives; Delete removes."
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
	if hint, ok := m.hoverHint(); ok {
		return "~ " + hint
	}
	return "> " + m.focusHint(m.FocusedID())
}

func (m *model) contextHelpDetailText() string {
	id := m.FocusedID()
	if m.ui.hoveredID != "" {
		id = m.ui.hoveredID
	}
	switch id {
	case idTopic, idTopics:
		return "Enter toggles subscribe  p toggles publish  Delete removes"
	case idMessage:
		return "Publish targets are marked on topic chips | Ctrl+S publish  Ctrl+E retain"
	case idHistory:
		return "History stores received and published messages | Enter details  / filter"
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
	helpY := 0
	if m.ui.width < helpReflowWidth {
		helpY = 1
	}
	return msg.Y == helpY && msg.X >= m.ui.width-helpWidth
}

func (m *model) updateHoverState(msg tea.MouseMsg) {
	m.ui.hoveredID = ""
	m.ui.hoveredTopic = -1
	if m.pointOverHelp(msg) {
		m.ui.hoveredID = idHelp
		return
	}
	y := m.clientContentY(msg.Y)
	switch {
	case pointInElement(y, m.ui.elemPos[idTopic], 3):
		m.ui.hoveredID = idTopic
	case pointInElement(y, m.ui.elemPos[idTopics], m.layout.topics.height+4):
		m.ui.hoveredID = idTopics
		m.ui.hoveredTopic = m.topics.TopicAtPosition(msg.X, y)
	case pointInElement(y, m.ui.elemPos[idMessage], m.layout.message.height+2):
		m.ui.hoveredID = idMessage
	case pointInElement(y, m.ui.elemPos[idHistory], m.layout.history.height+2):
		m.ui.hoveredID = idHistory
	}
}

func (m *model) clearHoverState() {
	m.ui.hoveredID = ""
	m.ui.hoveredTopic = -1
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
