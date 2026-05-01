package emqutiti

import (
	"math"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/marang/emqutiti/constants"
)

const (
	animationFrameDelay = 80 * time.Millisecond
	topicPulseFrames    = 6
	historyScrollFrames = 7
)

var (
	historyPulseFrames = []string{".", "o", "O", "o", ".", " "}
	connectFrames      = []string{".", "o", "O", "o"}
	workingFrames      = []string{"Working   ", "Working.  ", "Working.. ", "Working..."}
)

type animationTickMsg time.Time

type animationState struct {
	frame         int
	tickScheduled bool
	topicPulses   map[string]int
	historyPulse  int
	historyScroll historyScrollAnimation
}

type historyScrollAnimation struct {
	active  bool
	current float64
	target  float64
	ticks   int
}

func (m *model) startAnimationTick() tea.Cmd {
	if m.ui.animation.tickScheduled || !m.animationsActive() {
		return nil
	}
	m.ui.animation.tickScheduled = true
	return tea.Tick(animationFrameDelay, func(t time.Time) tea.Msg {
		return animationTickMsg(t)
	})
}

func (m *model) handleAnimationTick() tea.Cmd {
	m.ui.animation.tickScheduled = false
	m.ui.animation.frame++
	m.stepTopicPulses()
	m.stepHistoryPulse()
	m.stepHistoryScroll()
	if m.animationsActive() {
		return m.startAnimationTick()
	}
	return nil
}

func (m *model) animationsActive() bool {
	return len(m.ui.animation.topicPulses) > 0 ||
		m.ui.animation.historyPulse > 0 ||
		m.ui.animation.historyScroll.active ||
		m.isConnecting()
}

func (m *model) ensureTopicPulses() {
	if m.ui.animation.topicPulses == nil {
		m.ui.animation.topicPulses = map[string]int{}
	}
}

func (m *model) startTopicPulse(topic string) tea.Cmd {
	if strings.TrimSpace(topic) == "" {
		return nil
	}
	m.ensureTopicPulses()
	m.ui.animation.topicPulses[topic] = topicPulseFrames
	return m.startAnimationTick()
}

func (m *model) startTopicPulses(topics []string) tea.Cmd {
	for _, topic := range topics {
		if strings.TrimSpace(topic) != "" {
			m.ensureTopicPulses()
			m.ui.animation.topicPulses[topic] = topicPulseFrames
		}
	}
	return m.startAnimationTick()
}

func (m *model) stepTopicPulses() {
	for topic, remaining := range m.ui.animation.topicPulses {
		if remaining <= 1 {
			delete(m.ui.animation.topicPulses, topic)
			continue
		}
		m.ui.animation.topicPulses[topic] = remaining - 1
	}
}

func (m *model) topicPulsePhase(topic string) (int, bool) {
	remaining := m.ui.animation.topicPulses[topic]
	if remaining <= 0 {
		return 0, false
	}
	phase := topicPulseFrames - remaining
	if phase < 0 {
		phase = 0
	}
	if phase >= topicPulseFrames {
		phase = topicPulseFrames - 1
	}
	return phase, true
}

func (m *model) topicPulsePhases() map[string]int {
	phases := map[string]int{}
	for topic := range m.ui.animation.topicPulses {
		if phase, ok := m.topicPulsePhase(topic); ok {
			phases[topic] = phase
		}
	}
	return phases
}

func (m *model) topicsWorkingText() string {
	if len(m.ui.animation.topicPulses) == 0 {
		return ""
	}
	return workingFrames[m.ui.animation.frame%len(workingFrames)]
}

func (m *model) startHistoryPulse() tea.Cmd {
	m.ui.animation.historyPulse = len(historyPulseFrames)
	return m.startAnimationTick()
}

func (m *model) stepHistoryPulse() {
	if m.ui.animation.historyPulse > 0 {
		m.ui.animation.historyPulse--
	}
}

func (m *model) historyPulseMarker() string {
	remaining := m.ui.animation.historyPulse
	if remaining <= 0 {
		return " "
	}
	idx := len(historyPulseFrames) - remaining
	if idx < 0 {
		idx = 0
	}
	if idx >= len(historyPulseFrames) {
		idx = len(historyPulseFrames) - 1
	}
	return historyPulseFrames[idx]
}

func (m *model) startHistoryScrollAnimation(from, target float64) tea.Cmd {
	if from < 0 || target < 0 || math.Abs(from-target) < 0.001 {
		return nil
	}
	m.ui.animation.historyScroll = historyScrollAnimation{
		active:  true,
		current: from,
		target:  target,
	}
	return m.startAnimationTick()
}

func (m *model) stepHistoryScroll() {
	s := &m.ui.animation.historyScroll
	if !s.active {
		return
	}
	s.ticks++
	if s.ticks >= historyScrollFrames || math.Abs(s.current-s.target) < 0.001 {
		s.current = s.target
		s.active = false
		return
	}
	s.current += (s.target - s.current) * 0.45
}

func (m *model) animatedHistoryScrollPercent(raw float64) float64 {
	if m.ui.animation.historyScroll.active {
		return m.ui.animation.historyScroll.current
	}
	return raw
}

func (m *model) connectingFrame() string {
	if !m.isConnecting() {
		return ""
	}
	return connectFrames[m.ui.animation.frame%len(connectFrames)]
}

func (m *model) isConnecting() bool {
	if strings.HasPrefix(m.connections.Connection, "Connecting") {
		return true
	}
	for _, status := range m.connections.Manager.Statuses {
		if status == "connecting" {
			return true
		}
	}
	return false
}

func isHistoryKeyboardScrollMsg(msg tea.Msg) bool {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return false
	}
	switch key.String() {
	case constants.KeyUp, constants.KeyDown, constants.KeyJ, constants.KeyK, constants.KeyPgUp, constants.KeyPgDown:
		return true
	default:
		return false
	}
}
