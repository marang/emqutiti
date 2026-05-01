package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

// TestLegendBoxLayouts renders LegendBox at various sizes to ensure it
// doesn't panic and that borders remain aligned.
func TestLegendBoxLayouts(t *testing.T) {
	cases := []struct {
		name          string
		width, height int
		content       string
	}{
		{
			name:    "narrow",
			width:   5,
			height:  1,
			content: "hi",
		},
		{
			name:    "wide",
			width:   20,
			height:  5,
			content: "one\ntwo\nthree",
		},
		{
			name:    "tinyHeight",
			width:   10,
			height:  2,
			content: "a\nb\nc",
		},
		{
			name:    "autoHeight",
			width:   15,
			height:  0,
			content: "foo\nbar\nbaz\nqux",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			var box string
			func() {
				defer func() {
					if r := recover(); r != nil {
						t.Fatalf("LegendBox panicked: %v", r)
					}
				}()
				box = LegendBox(tc.content, "Lbl", tc.width, tc.height, ColGreen, false, -1)
			}()

			t.Logf("\n%s", box)

			lines := strings.Split(box, "\n")
			if len(lines) == 0 {
				t.Fatalf("no output")
			}
			w := lipgloss.Width(lines[0])
			for i, l := range lines {
				if got := lipgloss.Width(l); got != w {
					t.Fatalf("line %d width=%d want=%d", i, got, w)
				}
			}
		})
	}
}

func TestLegendBoxTextFieldFocused(t *testing.T) {
	tf := NewTextField("", "")
	tf.Model.Width = 5
	tf.Focus()
	view := tf.View()
	width := lipgloss.Width(view) + 2
	box := LegendBox(view, "Lbl", width, 0, ColGreen, true, -1)
	lines := strings.Split(box, "\n")
	if len(lines) != 3 {
		t.Fatalf("box height=%d want=3", len(lines))
	}
	content := lines[1]
	b := lipgloss.RoundedBorder()
	left := lipgloss.NewStyle().Foreground(ColGreen).Render(b.Left)
	if !strings.HasPrefix(content, left) {
		t.Fatalf("missing left border: %q", content)
	}
	right := lipgloss.NewStyle().Foreground(ColGreen).Render(b.Right)
	if !strings.HasSuffix(content, right) {
		t.Fatalf("missing right border: %q", content)
	}
}

func TestLegendBoxFocusWinsOverHover(t *testing.T) {
	lipgloss.SetColorProfile(termenv.TrueColor)
	normal := LegendBoxWithState("body", "Lbl", 20, 1, ColGreen, BoxState{}, -1)
	hovered := LegendBoxWithState("body", "Lbl", 20, 1, ColGreen, BoxState{Hovered: true}, -1)
	focused := LegendBoxWithState("body", "Lbl", 20, 1, ColGreen, BoxState{Focused: true, Hovered: true}, -1)
	if normal != hovered {
		t.Fatalf("hover should not visually change the box")
	}
	if focused == normal {
		t.Fatalf("focused box should differ from normal")
	}
}
