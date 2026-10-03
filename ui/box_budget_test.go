package ui

import (
	"fmt"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

func TestLegendBoxStyledContentBudget(t *testing.T) {
	label := "\x1b[35m" + strings.Repeat("Broker \u754c e\u0301 ", 40) + "LABEL_END\x1b[0m"
	longLine := "\x1b[32mBEGIN " + strings.Repeat("payload \u754c e\u0301 ", 40) + "CONTENT_END\x1b[0m"
	contents := []struct {
		name, text string
	}{
		{"empty", ""},
		{"short", "\x1b[32mBEGIN short\x1b[0m"},
		{"long", longLine},
		{"multiline", strings.Join([]string{longLine, "second", longLine, "fourth", longLine}, "\n")},
		{"trailing-newlines", longLine + "\n\n"},
	}
	states := []struct {
		name  string
		state BoxState
	}{
		{"normal", BoxState{}},
		{"focused", BoxState{Focused: true}},
		{"hovered", BoxState{Hovered: true}},
		{"focused-and-hovered", BoxState{Focused: true, Hovered: true}},
	}
	for _, width := range []int{40, 60, 80, 120} {
		for _, height := range []int{0, 1, 3, 8} {
			for _, content := range contents {
				for _, state := range states {
					name := fmt.Sprintf("%dx%d/%s/%s", width, height, content.name, state.name)
					t.Run(name, func(t *testing.T) {
						for _, scroll := range []float64{-1, 0, 0.5, 1} {
							view := LegendBoxWithState(content.text, label, width, height, ColBlue, state.state, scroll)
							bodyHeight := height
							if bodyHeight == 0 {
								bodyHeight = len(strings.Split(strings.TrimRight(content.text, "\n"), "\n"))
							}
							lines := strings.Split(view, "\n")
							if got := len(lines); got != bodyHeight+2 {
								t.Fatalf("scroll=%v height=%d want=%d\n%s", scroll, got, bodyHeight+2, ansi.Strip(view))
							}
							for y, line := range lines {
								if got := lipgloss.Width(line); got != width {
									t.Fatalf("scroll=%v row=%d width=%d want=%d: %q", scroll, y, got, width, ansi.Strip(line))
								}
							}
							plain := ansi.Strip(view)
							if strings.Contains(plain, "LABEL_END") || strings.Contains(plain, "CONTENT_END") {
								t.Fatalf("long label/content was not clipped\n%s", plain)
							}
							if content.text != "" && !strings.Contains(ansi.Strip(lines[1]), "BEGIN") {
								t.Fatalf("visible content prefix lost: %q", ansi.Strip(lines[1]))
							}
							if !state.state.Hovered {
								wrapper := LegendBox(content.text, label, width, height, ColBlue, state.state.Focused, scroll)
								if wrapper != view {
									t.Fatal("LegendBox wrapper differs from state-aware renderer")
								}
							}
							baseline := LegendBoxWithState(content.text, label, width, height, ColBlue, BoxState{}, scroll)
							if ansi.Strip(baseline) != plain {
								t.Fatal("focus/hover changed box geometry or visible content")
							}
						}
					})
				}
			}
		}
	}
}
