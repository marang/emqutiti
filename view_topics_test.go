package emqutiti

import (
	"fmt"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/marang/emqutiti/topics"
	"github.com/marang/emqutiti/ui"
	"github.com/muesli/termenv"
)

func TestRenderTopicChipsEmpty(t *testing.T) {
	chips := renderTopicChips(nil, 0, -1, 80)
	if len(chips) != 0 {
		t.Fatalf("expected 0 chips, got %d", len(chips))
	}
}

func TestRenderTopicChipsLarge(t *testing.T) {
	items := make([]topics.Item, 100)
	for i := range items {
		items[i] = topics.Item{Name: fmt.Sprintf("t%d", i)}
	}
	chips := renderTopicChips(items, 50, -1, 80)
	if len(chips) != len(items) {
		t.Fatalf("expected %d chips, got %d", len(items), len(chips))
	}
}

func TestRenderTopicChipsDoNotPrefixModes(t *testing.T) {
	items := []topics.Item{
		{Name: "read", Subscribed: true},
		{Name: "write", Publish: true},
		{Name: "both", Subscribed: true, Publish: true},
		{Name: "off"},
	}
	chips := renderTopicChips(items, 0, -1, 80)
	for i, prefix := range []string{"r read", "w write", "rw both", "x off"} {
		if strings.Contains(chips[i], prefix) {
			t.Fatalf("chip %d should not include mode prefix %q: %q", i, prefix, chips[i])
		}
	}
}

func TestPublishChipUsesPinkWithoutChangingGeometry(t *testing.T) {
	profile := lipgloss.ColorProfile()
	t.Cleanup(func() { lipgloss.SetColorProfile(profile) })
	lipgloss.SetColorProfile(termenv.ANSI256)
	item := topics.Item{Name: "site/topic", Publish: true}
	expected := ui.Chip.Background(ui.ColPink).Foreground(ui.ColBlack).
		BorderStyle(lipgloss.InnerHalfBlockBorder()).BorderForeground(ui.ColPink)
	for _, selected := range []int{-1, 0} {
		st := expected
		if selected == 0 {
			st = st.BorderForeground(ui.ColPink)
		}
		got := renderTopicChips([]topics.Item{item}, selected, -1, 80)[0]
		if want := st.Render(item.Name); got != want {
			t.Fatalf("publish chip does not use continuous inner-half pink fill: selected=%d got=%q want=%q", selected, got, want)
		}
		base := ui.Chip.Render(item.Name)
		if lipgloss.Width(got) != lipgloss.Width(base) || lipgloss.Height(got) != lipgloss.Height(base) {
			t.Fatal("publish color change changed chip geometry")
		}
	}
}

func TestLayoutTopicViewportEmpty(t *testing.T) {
	m, _ := initialModel(nil)
	m.ui.width = 80
	content, bounds, boxH, infoH, scroll := m.layoutTopicViewport(nil)
	if content == "" {
		t.Fatalf("expected content with info lines")
	}
	if len(bounds) != 0 {
		t.Fatalf("expected no bounds, got %d", len(bounds))
	}
	if boxH <= 0 || infoH < 1 || lipgloss.Height(content) != boxH+infoH {
		t.Fatalf("unexpected box or info height")
	}
	if scroll >= 0 {
		t.Fatalf("expected negative scroll for empty content")
	}
}

func TestTopicChipLegendMatchesStateStyles(t *testing.T) {
	profile := lipgloss.ColorProfile()
	t.Cleanup(func() { lipgloss.SetColorProfile(profile) })
	lipgloss.SetColorProfile(termenv.ANSI256)
	legend := topicChipLegend()
	for _, sample := range []string{
		topicSubscriptionLabel("sub", true),
		lipgloss.NewStyle().Foreground(ui.ChipPublish.GetForeground()).Background(ui.ChipPublish.GetBackground()).Render("pub"),
		lipgloss.NewStyle().Foreground(ui.ChipInactive.GetForeground()).Render("off"),
	} {
		if !strings.Contains(legend, sample) {
			t.Fatalf("legend does not match chip cue %q: %q", sample, legend)
		}
	}
	if got := ansi.Strip(legend); got != "sub=read  pub=write  off=inactive" {
		t.Fatalf("unexpected legend: %q", got)
	}
	lipgloss.SetColorProfile(termenv.Ascii)
	legend = topicChipLegend()
	if legend != "[sub]=read  [pub]=write  [off]=inactive" {
		t.Fatalf("no-color legend does not explain state suffixes: %q", legend)
	}
}

func TestTopicChipLegendFitsTerminalWidths(t *testing.T) {
	profile, dark := lipgloss.ColorProfile(), lipgloss.HasDarkBackground()
	t.Cleanup(func() { lipgloss.SetColorProfile(profile); lipgloss.SetHasDarkBackground(dark) })
	for _, theme := range []string{"dark", "light", "mono"} {
		lipgloss.SetColorProfile(termenv.ANSI256)
		lipgloss.SetHasDarkBackground(theme != "light")
		if theme == "mono" {
			lipgloss.SetColorProfile(termenv.Ascii)
		}
		for _, width := range []int{40, 60, 80, 120} {
			t.Run(fmt.Sprintf("%s/%d", theme, width), func(t *testing.T) {
				m := reviewFixture(t, width, 40)
				m.SetFocus(idTopics)
				chips := renderTopicChips(m.topics.Items, 0, -1, width-4)
				content, bounds, boxH, infoH, scroll := m.layoutTopicViewport(chips)
				if lipgloss.Width(content) > width-4 || lipgloss.Height(content) != boxH+infoH {
					t.Fatalf("legend exceeds content budget at width %d", width)
				}
				if len(bounds) == 0 {
					t.Fatal("legend hid chip bounds")
				}
				box, _ := m.buildTopicBoxes(content, boxH, infoH, scroll)
				plain := ansi.Strip(box)
				for _, meaning := range []string{"=read", "=write", "=inactive", "[Enter]", "[p]", "[Del] remove"} {
					if !strings.Contains(plain, meaning) {
						t.Fatalf("legend or action %q clipped at width %d\n%s", meaning, width, plain)
					}
				}
				if strings.Contains(plain, "pink=focus") {
					t.Fatal("legend still assigns pink to focus")
				}
				captureReviewView(t, "topic-legend-"+theme, box, width, lipgloss.Height(box))
			})
		}
	}
}

func TestLayoutTopicViewportLarge(t *testing.T) {
	m, _ := initialModel(nil)
	m.ui.width = 80
	items := make([]topics.Item, 200)
	for i := range items {
		items[i] = topics.Item{Name: fmt.Sprintf("t%d", i), Subscribed: true}
	}
	chips := renderTopicChips(items, 0, -1, m.ui.width-4)
	content, bounds, _, _, scroll := m.layoutTopicViewport(chips)
	if content == "" {
		t.Fatalf("expected content for large list")
	}
	if len(bounds) == 0 {
		t.Fatalf("expected bounds for large list")
	}
	if scroll < 0 {
		t.Fatalf("expected non-negative scroll for large list")
	}
}

func TestBuildTopicBoxesEmpty(t *testing.T) {
	m, _ := initialModel(nil)
	m.ui.width = 80
	topicsBox, _ := m.buildTopicBoxes("content", 1, 2, -1)
	if !strings.Contains(topicsBox, "Topics: 0 | subscribed: 0") {
		t.Fatalf("expected label 'Topics: 0 | subscribed: 0', got %q", topicsBox)
	}
}

func TestBuildTopicBoxesLarge(t *testing.T) {
	m, _ := initialModel(nil)
	m.ui.width = 80
	items := make([]topics.Item, 50)
	for i := range items {
		items[i] = topics.Item{Name: fmt.Sprintf("t%d", i), Subscribed: i%2 == 0}
	}
	m.topics.Items = items
	topicsBox, _ := m.buildTopicBoxes("content", 1, 2, 0)
	if !strings.Contains(topicsBox, "Topics: 50 | subscribed: 25") {
		t.Fatalf("expected label 'Topics: 50 | subscribed: 25', got %q", topicsBox)
	}
}

func TestBuildTopicBoxesCountsSubscriptionsNotPublishTargets(t *testing.T) {
	m := reviewFixture(t, 80, 24)
	m.topics.Items = []topics.Item{
		{Name: "test", Subscribed: true, Publish: true},
		{Name: "df", Publish: true},
		{Name: "inactive"},
		{Name: "testx"},
	}
	for selected := range m.topics.Items {
		m.topics.SetSelected(selected)
		box, _ := m.buildTopicBoxes("content", 1, 2, -1)
		if !strings.Contains(ansi.Strip(box), "Topics: 4 | subscribed: 1") {
			t.Fatalf("selection %d changed subscription count: %q", selected, box)
		}
	}
}
