package history

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/marang/emqutiti/ui"
	"github.com/muesli/termenv"
)

func TestHistoryTimestampAndLogUseReadableText(t *testing.T) {
	profile, dark := lipgloss.ColorProfile(), lipgloss.HasDarkBackground()
	t.Cleanup(func() { lipgloss.SetColorProfile(profile); lipgloss.SetHasDarkBackground(dark) })
	lipgloss.SetColorProfile(termenv.ANSI256)
	ts := time.Date(2026, time.October, 3, 12, 34, 56, 789000000, time.UTC)
	for _, dark := range []bool{false, true} {
		lipgloss.SetHasDarkBackground(dark)
		for _, kind := range []string{"sub", "pub", "log"} {
			for _, selected := range []bool{false, true} {
				item := Item{Timestamp: ts, Topic: "test", Kind: kind, Payload: "ready", IsSelected: &selected}
				m := list.New([]list.Item{item}, historyDelegate{}, 80, 6)
				var out bytes.Buffer
				historyDelegate{}.Render(&out, m, 0, item)
				want := " " + ts.Format("2006-01-02 15:04:05.000") + ":"
				if kind == "log" {
					want = strings.TrimSpace(want) + " "
				}
				if !strings.Contains(out.String(), lipgloss.NewStyle().Foreground(ui.TextMuted).Render(want)) {
					t.Fatalf("timestamp does not use adaptive medium gray: dark=%t kind=%s selected=%t view=%q", dark, kind, selected, out.String())
				}
				if kind == "log" && !strings.Contains(out.String(), lipgloss.NewStyle().Foreground(ui.TextMain).Render("ready")) {
					t.Fatal("log text lost contrast when its timestamp was subdued")
				}
			}
		}
	}
}

func TestHistoryListHelpUsesReadableText(t *testing.T) {
	profile, dark := lipgloss.ColorProfile(), lipgloss.HasDarkBackground()
	t.Cleanup(func() { lipgloss.SetColorProfile(profile); lipgloss.SetHasDarkBackground(dark) })
	lipgloss.SetColorProfile(termenv.ANSI256)
	for _, dark := range []bool{false, true} {
		lipgloss.SetHasDarkBackground(dark)
		c := NewComponent(stubModel{}, nil)
		c.list.SetSize(80, 12)
		c.list.SetItems([]list.Item{Item{Kind: "sub", Topic: "test"}, Item{Kind: "pub", Topic: "test"}})
		for _, st := range []lipgloss.Style{c.list.Help.Styles.ShortKey, c.list.Help.Styles.ShortDesc, c.list.Help.Styles.FullKey, c.list.Help.Styles.FullDesc} {
			if st.GetForeground() != ui.TextMain {
				t.Fatal("history key or description uses a subdued color")
			}
		}
		for _, showAll := range []bool{false, true} {
			c.list.Help.ShowAll = showAll
			view := c.list.Help.View(c.list)
			if !strings.Contains(view, lipgloss.NewStyle().Foreground(ui.TextMain).Render("/")) ||
				!strings.Contains(view, lipgloss.NewStyle().Foreground(ui.TextMain).Render("filter")) {
				t.Fatalf("history key or description still uses dark help defaults: dark=%t full=%t view=%q", dark, showAll, view)
			}
		}
		if c.list.Styles.NoItems.GetForeground() != ui.TextMain {
			t.Fatal("empty-history hint uses a subdued color")
		}
	}
}

func TestHistoryDelegateTwoLineBudgetPreservesPayload(t *testing.T) {
	ts := time.Date(2026, time.October, 3, 12, 34, 56, 789000000, time.UTC)
	longTopic := strings.Repeat("factory/sensor/\u754c/e\u0301/", 60)
	longPayload := strings.Repeat("long payload \u754c e\u0301 ", historyPreviewLimit) + "PAYLOAD_END"
	cases := []struct {
		name            string
		item            Item
		wantTimestamp   bool
		wantNewlineMark bool
	}{
		{"pub-long-topic", Item{Topic: longTopic, Kind: "pub", Payload: "ready"}, false, false},
		{"sub-long-retained-topic", Item{Topic: longTopic, Kind: "sub", Payload: longPayload, Retained: true}, false, false},
		{"styled-long-topic-and-payload", Item{Topic: "\x1b[35m" + longTopic + "\x1b[0m", Kind: "pub", Payload: "\x1b[32m" + longPayload + "\x1b[0m"}, false, false},
		{"timestamp", Item{Topic: "t", Kind: "sub", Payload: "ready"}, true, false},
		{"log-timestamp", Item{Kind: "log", Payload: "ready"}, true, false},
		{"long-log", Item{Kind: "log", Payload: longPayload}, true, false},
		{"multiline-payload", Item{Topic: "t", Kind: "pub", Payload: "one\r\ntwo\nthree\r\n"}, true, true},
		{"long-multiline-payload", Item{Topic: "t", Kind: "sub", Payload: "one\r\ntwo\n" + longPayload}, true, true},
		{"multiline-log", Item{Kind: "log", Payload: "one\r\ntwo\n" + longPayload}, true, false},
		{"empty-payload", Item{Topic: "t", Kind: "sub"}, true, false},
		{"empty-log", Item{Kind: "log"}, true, false},
	}
	states := []struct {
		name              string
		selected, focused bool
	}{
		{"normal", false, false},
		{"focused", false, true},
		{"selected", true, false},
		{"selected-and-focused", true, true},
	}
	d := historyDelegate{}
	if d.Height() != 2 || d.Spacing() != 0 {
		t.Fatalf("delegate height=%d spacing=%d want 2/0", d.Height(), d.Spacing())
	}
	for _, width := range []int{8, 16, 24, 40, 60, 80, 120} {
		for _, tc := range cases {
			for _, state := range states {
				t.Run(fmt.Sprintf("width=%d/%s/%s", width, tc.name, state.name), func(t *testing.T) {
					item := tc.item
					item.Timestamp = ts
					selected := state.selected
					item.IsSelected = &selected
					before := item
					m := list.New([]list.Item{item, Item{Kind: "log", Timestamp: ts}}, d, width, 20)
					if !state.focused {
						m.Select(1)
					}
					var out bytes.Buffer
					d.Render(&out, m, 0, item)
					view := out.String()
					lines := strings.Split(view, "\n")
					if len(lines) != d.Height() {
						t.Fatalf("rendered %d rows want 2\n%s", len(lines), ansi.Strip(view))
					}
					for y, line := range lines {
						if got := lipgloss.Width(line); got != width {
							t.Fatalf("row=%d width=%d want=%d: %q", y, got, width, ansi.Strip(line))
						}
						if !strings.HasPrefix(ansi.Strip(line), "\u2503 ") {
							t.Fatalf("row=%d missing history prefix: %q", y, ansi.Strip(line))
						}
					}
					plain := ansi.Strip(view)
					if tc.wantTimestamp && width >= 40 && !strings.Contains(plain, ts.Format("2006-01-02 15:04:05.000")) {
						t.Fatalf("timestamp lost\n%s", plain)
					}
					if tc.wantNewlineMark && width >= 24 && !strings.Contains(plain, "\u23ce") {
						t.Fatalf("multiline preview has no newline marker\n%s", plain)
					}
					if item.Kind == "log" && strings.TrimSpace(strings.TrimPrefix(ansi.Strip(lines[1]), "\u2503 ")) != "" {
						t.Fatalf("log's second row is not blank: %q", ansi.Strip(lines[1]))
					}
					if strings.Contains(plain, "PAYLOAD_END") {
						t.Fatalf("long payload did not produce a bounded preview\n%s", plain)
					}
					if item != before || m.Items()[0].(Item) != before || selected != state.selected {
						t.Fatal("rendering modified the full history item or selection")
					}
				})
			}
		}
	}
}
