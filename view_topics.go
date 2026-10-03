package emqutiti

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/marang/emqutiti/topics"
	"github.com/marang/emqutiti/ui"
	"github.com/muesli/termenv"
)

// maxTopicChipWidth caps the width of a rendered topic name before truncation.
const maxTopicChipWidth = 40

// renderTopicChips builds styled topic chips, expanding the selected chip to
// show the full topic name by wrapping it within the viewport width. Other
// chips are truncated with an ellipsis.
func renderTopicChips(items []topics.Item, selected, hovered, width int) []string {
	return renderTopicChipsWithPulses(items, selected, hovered, width, nil)
}

func renderTopicChipsWithPulses(items []topics.Item, selected, hovered, width int, pulses map[string]int) []string {
	targets := make(map[string]bool)
	for _, name := range effectivePublishTargets(items, selected) {
		targets[name] = true
	}
	chips := make([]string, 0, len(items))
	for i, t := range items {
		st := ui.ChipInactive
		focused := i == selected
		switch {
		case targets[t.Name]:
			st = ui.ChipPublish
			if focused {
				st = ui.ChipPublishFocused
			} else if i == hovered {
				st = ui.ChipPublishHovered
			}
		case t.Subscribed:
			st = ui.Chip
			if focused {
				st = ui.ChipFocused
			} else if i == hovered {
				st = ui.ChipHovered
			}
		default:
			if focused {
				st = ui.ChipInactiveFocused
			} else if i == hovered {
				st = ui.ChipInactiveHovered
			}
		}
		if pulses != nil {
			if phase, ok := pulses[t.Name]; ok {
				st = topicPulseStyle(st, phase, focused)
			}
		}
		base := lipgloss.Width(st.Render(""))
		contentWidth := width - base
		if contentWidth < 1 {
			contentWidth = 1
		}
		if contentWidth > maxTopicChipWidth {
			contentWidth = maxTopicChipWidth
		}
		label := t.Name
		if lipgloss.ColorProfile() == termenv.Ascii {
			suffix := " [off]"
			switch {
			case t.Subscribed && targets[t.Name]:
				suffix = " [sub,pub]"
			case t.Subscribed:
				suffix = " [sub]"
			case targets[t.Name]:
				suffix = " [pub]"
			}
			if !focused {
				label = ansi.Truncate(label, max(1, contentWidth-lipgloss.Width(suffix)), "…")
			}
			label += suffix
		}
		if i == selected && lipgloss.Width(label) > contentWidth {
			wrapped := ansi.Hardwrap(label, contentWidth, false)
			lines := strings.Split(wrapped, "\n")
			for j, line := range lines {
				lw := lipgloss.Width(line)
				if lw < contentWidth {
					lines[j] = line + strings.Repeat(" ", contentWidth-lw)
				}
			}
			chips = append(chips, st.Render(topicSubscriptionLabel(strings.Join(lines, "\n"), t.Subscribed)))
			continue
		}
		if lipgloss.Width(label) > contentWidth {
			label = ansi.Truncate(label, contentWidth, "…")
		}
		chips = append(chips, st.Render(topicSubscriptionLabel(label, t.Subscribed)))
	}
	return chips
}

func topicSubscriptionLabel(label string, subscribed bool) string {
	if !subscribed || lipgloss.ColorProfile() == termenv.Ascii {
		return label
	}
	// Reset only underline attributes so the enclosing publish fill stays intact.
	start := ansi.NewStyle().Underline(true).UnderlineColor(ui.ColCyan).String()
	end := ansi.NewStyle().Underline(false).UnderlineColor(nil).String()
	lines := strings.Split(label, "\n")
	for i, line := range lines {
		words := strings.Split(line, " ")
		for j, word := range words {
			if word != "" {
				words[j] = start + word + end
			}
		}
		lines[i] = strings.Join(words, " ")
	}
	return strings.Join(lines, "\n")
}

func topicPulseStyle(st lipgloss.Style, phase int, focused bool) lipgloss.Style {
	colors := []lipgloss.TerminalColor{ui.TextMuted, ui.TextMain, ui.TextMuted, ui.TextMain, ui.TextMuted, ui.TextMain}
	if focused || st.GetBackground() == ui.ColPink {
		bright := lipgloss.Color("218")
		colors = []lipgloss.TerminalColor{ui.ColPink, bright, ui.ColPink, bright, ui.ColPink, bright}
	}
	if phase < 0 {
		phase = 0
	}
	if phase >= len(colors) {
		phase = len(colors) - 1
	}
	color := colors[phase]
	if focused {
		return st.BorderForeground(color, color, color, color)
	}
	return st.BorderForeground(color)
}

func topicChipLegend() string {
	if lipgloss.ColorProfile() == termenv.Ascii {
		return "[sub]=read  [pub]=write  [off]=inactive"
	}
	text := lipgloss.NewStyle().Foreground(ui.TextMain)
	sub := text.Render(topicSubscriptionLabel("sub", true))
	pub := lipgloss.NewStyle().
		Foreground(ui.ChipPublish.GetForeground()).
		Background(ui.ChipPublish.GetBackground()).Render("pub")
	off := text.Render("off")
	return sub + text.Render("=read  ") + pub + text.Render("=write  ") + off + text.Render("=inactive")
}

// layoutTopicViewport sets up the topic viewport and returns visible chip bounds.
func (m *model) layoutTopicViewport(chips []string) (string, []topics.ChipBound, int, int, float64) {
	chipRows, bounds := topics.LayoutChips(chips, m.ui.width-4)
	rowH := lipgloss.Height(ui.Chip.Render("test"))
	maxRows := m.layout.topics.height
	if maxRows <= 0 {
		maxRows = 1
	}
	topicsBoxHeight := maxRows * rowH
	m.topics.VP.Width = m.ui.width - 4
	m.topics.VP.Height = topicsBoxHeight
	m.topics.VP.SetContent(strings.Join(chipRows, "\n"))
	selName := ""
	if sel := m.topics.Selected(); sel >= 0 && sel < len(m.topics.Items) {
		selName = m.topics.Items[sel].Name
	}
	if selName != m.ui.topicLayoutSelection || m.ui.topicLayoutWidth != m.topics.VP.Width || m.ui.topicLayoutHeight != topicsBoxHeight {
		for _, b := range bounds {
			if b.Index != m.topics.Selected() {
				continue
			}
			if b.YPos < m.topics.VP.YOffset {
				m.topics.VP.SetYOffset(b.YPos)
			} else if b.YPos+b.Height > m.topics.VP.YOffset+topicsBoxHeight {
				m.topics.VP.SetYOffset(b.YPos + min(b.Height, topicsBoxHeight) - topicsBoxHeight)
			}
			break
		}
	}
	m.ui.topicLayoutSelection, m.ui.topicLayoutWidth, m.ui.topicLayoutHeight = selName, m.topics.VP.Width, topicsBoxHeight
	startLine := m.topics.VP.YOffset
	endLine := startLine + topicsBoxHeight
	topicsSP := -1.0
	if len(chipRows)*rowH > topicsBoxHeight {
		topicsSP = m.topics.VP.ScrollPercent()
	}
	chipContent := m.topics.VP.View()
	info := m.topicLegendInfo()
	// Working belongs to the box label, not the action budget.
	chipContent = lipgloss.JoinVertical(lipgloss.Left, chipContent, info)
	infoHeight := lipgloss.Height(info)
	visible := []topics.ChipBound{}
	for _, b := range bounds {
		if b.YPos+b.Height > startLine && b.YPos < endLine {
			top := max(b.YPos, startLine)
			b.Height = min(b.YPos+b.Height, endLine) - top
			b.YPos = top - startLine
			visible = append(visible, b)
		}
	}
	bounds = visible
	return chipContent, bounds, topicsBoxHeight, infoHeight, topicsSP
}

func (m *model) topicLegendInfo() string {
	legend := topicChipLegend()
	width := max(1, m.ui.width-6)
	actions := topicShortcutHint(width)
	text := lipgloss.NewStyle().Foreground(ui.TextMain)
	info := legend + text.Render(" | "+actions)
	if lipgloss.Width(info) > width {
		info = legend + "\n" + text.Render(actions)
	}
	return ui.InfoSubtleStyle.UnsetForeground().Render(ansi.Wrap(info, width, ""))
}

// buildTopicBoxes assembles the legend boxes for topics and the input field.
func (m *model) buildTopicBoxes(content string, boxHeight, infoHeight int, scrollPercent float64) (string, string) {
	subscribed := 0
	for _, t := range m.topics.Items {
		if t.Subscribed {
			subscribed++
		}
	}
	label := fmt.Sprintf("Topics: %d | subscribed: %d", len(m.topics.Items), subscribed)
	if working := m.topicsWorkingText(); working != "" {
		label += " | " + strings.TrimSpace(working)
	}
	topicsFocused := m.ui.focusOrder[m.ui.focusIndex] == idTopics
	topicsHovered := m.ui.hoveredID == idTopics
	scroll := scrollPercent
	if scroll >= 0 {
		scroll = scroll * float64(boxHeight-1) / float64(boxHeight+infoHeight-1)
	}
	topicsBox := ui.LegendBoxWithState(content, label, m.ui.width-2, boxHeight+infoHeight, ui.ColBlue, ui.BoxState{Focused: topicsFocused, Hovered: topicsHovered}, scroll)

	topicFocused := m.ui.focusOrder[m.ui.focusIndex] == idTopic
	topicHovered := m.ui.hoveredID == idTopic
	topicBox := ui.LegendBoxWithState(m.topics.Input.View(), "Topic", m.ui.width-2, 1, ui.ColBlue, ui.BoxState{Focused: topicFocused, Hovered: topicHovered}, -1)
	return topicsBox, topicBox
}

// renderTopicsSection renders topics and topic input boxes.
func (m *model) renderTopicsSection() (string, string, []topics.ChipBound) {
	width := m.ui.width - 4
	hovered := -1
	if m.ui.hoveredID == idTopics {
		hovered = m.ui.hoveredTopic
	}
	chips := renderTopicChipsWithPulses(m.topics.Items, m.topics.Selected(), hovered, width, m.topicPulsePhases())
	content, bounds, boxHeight, infoHeight, scroll := m.layoutTopicViewport(chips)
	topicsBox, topicBox := m.buildTopicBoxes(content, boxHeight, infoHeight, scroll)
	return topicsBox, topicBox, bounds
}
