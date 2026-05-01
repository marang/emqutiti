package emqutiti

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/marang/emqutiti/topics"
	"github.com/marang/emqutiti/ui"
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
	chips := make([]string, 0, len(items))
	for i, t := range items {
		st := ui.ChipInactive
		focused := i == selected
		switch {
		case t.Publish:
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
		if contentWidth < 0 {
			contentWidth = 0
		}
		if contentWidth > maxTopicChipWidth {
			contentWidth = maxTopicChipWidth
		}
		label := t.Name
		if i == selected && lipgloss.Width(label) > contentWidth {
			wrapped := ansi.Hardwrap(label, contentWidth, false)
			lines := strings.Split(wrapped, "\n")
			for j, line := range lines {
				lw := lipgloss.Width(line)
				if lw < contentWidth {
					lines[j] = line + strings.Repeat(" ", contentWidth-lw)
				}
			}
			chips = append(chips, st.Render(strings.Join(lines, "\n")))
			continue
		}
		if lipgloss.Width(label) > contentWidth {
			label = ansi.Truncate(label, contentWidth, "…")
		}
		chips = append(chips, st.Render(label))
	}
	return chips
}

func topicPulseStyle(st lipgloss.Style, phase int, focused bool) lipgloss.Style {
	colors := []lipgloss.Color{ui.ColCyan, ui.ColWhite, ui.ColPink, ui.ColWhite, ui.ColCyan, ui.ColBlue}
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
	m.topics.EnsureVisible(m.ui.width - 4)
	startLine := m.topics.VP.YOffset
	endLine := startLine + topicsBoxHeight
	topicsSP := -1.0
	if len(chipRows)*rowH > topicsBoxHeight {
		topicsSP = m.topics.VP.ScrollPercent()
	}
	chipContent := m.topics.VP.View()
	info := topicStateLegend() + " | [←/→] move  [enter] sub  [p] pub  [del] del"
	if working := m.topicsWorkingText(); working != "" {
		info = working + " | " + info
	}
	chipContent = lipgloss.JoinVertical(lipgloss.Left, chipContent, ui.InfoSubtleStyle.Render(ansi.Truncate(info, m.ui.width-6, "…")))
	infoHeight := 1
	visible := []topics.ChipBound{}
	for _, b := range bounds {
		if b.YPos >= startLine && b.YPos < endLine {
			b.YPos -= startLine
			visible = append(visible, b)
		}
	}
	bounds = visible
	return chipContent, bounds, topicsBoxHeight, infoHeight, topicsSP
}

// buildTopicBoxes assembles the legend boxes for topics and the input field.
func (m *model) buildTopicBoxes(content string, boxHeight, infoHeight int, scrollPercent float64) (string, string) {
	active := 0
	for _, t := range m.topics.Items {
		if t.Subscribed {
			active++
		}
	}
	label := fmt.Sprintf("Topics %d/%d", active, len(m.topics.Items))
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
