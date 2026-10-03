package emqutiti

import (
	"image/color"
	"slices"
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/marang/emqutiti/constants"
	"github.com/marang/emqutiti/topics"
	"github.com/marang/emqutiti/ui"
	"github.com/muesli/termenv"
)

func TestPublishChipInteriorFillAndCyanSubscription(t *testing.T) {
	profile := lipgloss.ColorProfile()
	t.Cleanup(func() { lipgloss.SetColorProfile(profile) })
	lipgloss.SetColorProfile(termenv.ANSI256)
	item := topics.Item{Name: "read/write", Subscribed: true, Publish: true}
	view := renderTopicChips([]topics.Item{item}, 0, -1, 80)[0]
	var bg, underlineColor color.Color
	underlined, x, y, painted, letters := false, 0, 0, 0, 0
	pinkR, pinkG, pinkB, _ := ui.ColPink.RGBA()
	cyanR, cyanG, cyanB, _ := ui.ColCyan.RGBA()
	width := lipgloss.Width(view) - 1 // The right margin is not part of the chip.
	p := ansi.NewParser()
	p.SetHandler(ansi.Handler{
		HandleCsi: func(cmd ansi.Cmd, params ansi.Params) {
			if int(cmd) != 'm' {
				return
			}
			for i := 0; i < len(params); i++ {
				switch params[i].Param(0) {
				case 0:
					bg, underlineColor, underlined = nil, nil, false
				case 4:
					underlined = true
				case 24:
					underlined = false
				case 48:
					i += ansi.ReadStyleColor(params[i:], &bg) - 1
				case 49:
					bg = nil
				case 58:
					i += ansi.ReadStyleColor(params[i:], &underlineColor) - 1
				case 59:
					underlineColor = nil
				}
			}
		},
		Execute: func(b byte) {
			if b == '\n' {
				x = 0
				y++
			}
		},
		Print: func(r rune) {
			if x > 0 && x < width-1 && y == 1 {
				if bg == nil {
					t.Fatalf("chip cell %d (%q) has no pink background", x, r)
				}
				red, green, blue, _ := bg.RGBA()
				if red != pinkR || green != pinkG || blue != pinkB {
					t.Fatalf("chip cell %d (%q) background is not pink", x, r)
				}
				painted++
			} else if bg != nil {
				t.Fatalf("border/margin cell (%d,%d) (%q) has a background", x, y, r)
			}
			if strings.ContainsRune(item.Name, r) {
				if !underlined || underlineColor == nil {
					t.Fatalf("subscribed letter %q has no colored underline", r)
				}
				red, green, blue, _ := underlineColor.RGBA()
				if red != cyanR || green != cyanG || blue != cyanB {
					t.Fatalf("subscribed letter %q underline is not cyan", r)
				}
				letters++
			} else if underlined {
				t.Fatalf("underline leaked into padding/border %q", r)
			}
			x++
		},
	})
	for _, b := range []byte(view) {
		p.Advance(b)
	}
	if painted != width-2 || letters != len(item.Name) {
		t.Fatalf("painted=%d letters=%d, want %d/%d", painted, letters, width-2, len(item.Name))
	}
}

func TestTopicLegendUsesConsistentForeground(t *testing.T) {
	profile, dark := lipgloss.ColorProfile(), lipgloss.HasDarkBackground()
	t.Cleanup(func() { lipgloss.SetColorProfile(profile); lipgloss.SetHasDarkBackground(dark) })
	lipgloss.SetColorProfile(termenv.ANSI256)
	for _, dark := range []bool{false, true} {
		lipgloss.SetHasDarkBackground(dark)
		for _, width := range []int{40, 120} {
			m := reviewFixture(t, width, 24)
			view := m.topicLegendInfo()
			var fg, bg color.Color
			mainR, mainG, mainB, _ := ui.TextMain.RGBA()
			p := ansi.NewParser()
			p.SetHandler(ansi.Handler{
				HandleCsi: func(cmd ansi.Cmd, params ansi.Params) {
					if int(cmd) != 'm' {
						return
					}
					for i := 0; i < len(params); i++ {
						switch params[i].Param(0) {
						case 0:
							fg, bg = nil, nil
						case 30:
							fg = ui.ColBlack
						case 38:
							i += ansi.ReadStyleColor(params[i:], &fg) - 1
						case 39:
							fg = nil
						case 48:
							i += ansi.ReadStyleColor(params[i:], &bg) - 1
						case 49:
							bg = nil
						case 58:
							var underlineColor color.Color
							i += ansi.ReadStyleColor(params[i:], &underlineColor) - 1
						}
					}
				},
				Print: func(r rune) {
					if r == ' ' || bg != nil {
						return // The pink publish sample keeps its contrasting dark text.
					}
					if fg == nil {
						t.Fatalf("legend character %q has no explicit foreground (dark=%t width=%d)", r, dark, width)
					}
					red, green, blue, _ := fg.RGBA()
					if red != mainR || green != mainG || blue != mainB {
						t.Fatalf("legend character %q does not use the shared readable foreground (dark=%t width=%d)", r, dark, width)
					}
				},
			})
			for _, b := range []byte(view) {
				p.Advance(b)
			}
		}
	}
}

func TestSubscribedTopicsAndHintKeepCyanUnderline(t *testing.T) {
	profile, dark := lipgloss.ColorProfile(), lipgloss.HasDarkBackground()
	t.Cleanup(func() { lipgloss.SetColorProfile(profile); lipgloss.SetHasDarkBackground(dark) })
	for _, profile := range []termenv.Profile{termenv.ANSI256, termenv.TrueColor} {
		lipgloss.SetColorProfile(profile)
		for _, dark := range []bool{false, true} {
			lipgloss.SetHasDarkBackground(dark)
			for _, width := range []int{40, 120} {
				m := reviewFixture(t, width, 24)
				m.SetFocus(idTopics)
				for _, selected := range []int{0, 2} {
					m.topics.SetSelected(selected)
					m.startTopicPulse(m.topics.Items[selected].Name)
					m.handleAnimationTick()
					box, _, _ := m.renderTopicsSection()
					var underlineColor color.Color
					underlined, letters := false, 0
					cyanR, cyanG, cyanB, _ := ui.ColCyan.RGBA()
					p := ansi.NewParser()
					p.SetHandler(ansi.Handler{
						HandleCsi: func(cmd ansi.Cmd, params ansi.Params) {
							if int(cmd) != 'm' {
								return
							}
							for i := 0; i < len(params); i++ {
								switch params[i].Param(0) {
								case 0:
									underlined, underlineColor = false, nil
								case 4:
									underlined = true
								case 24:
									underlined = false
								case 38, 48:
									var ignored color.Color
									i += ansi.ReadStyleColor(params[i:], &ignored) - 1
								case 58:
									i += ansi.ReadStyleColor(params[i:], &underlineColor) - 1
								case 59:
									underlineColor = nil
								}
							}
						},
						Print: func(r rune) {
							if !underlined {
								return
							}
							if underlineColor == nil {
								t.Fatalf("underlined %q lost its color (profile=%v dark=%t width=%d selected=%d)", r, profile, dark, width, selected)
							}
							red, green, blue, _ := underlineColor.RGBA()
							if red != cyanR || green != cyanG || blue != cyanB {
								t.Fatalf("underlined %q is not cyan", r)
							}
							letters++
						},
					})
					for _, b := range []byte(box) {
						p.Advance(b)
					}
					if letters < len(m.topics.Items[selected].Name)+len("sub") {
						t.Fatal("subscribed topic or legend underline is missing")
					}
				}
			}
		}
	}
}

func TestInputPromptsAndPlaceholdersUseMediumGray(t *testing.T) {
	profile, dark := lipgloss.ColorProfile(), lipgloss.HasDarkBackground()
	t.Cleanup(func() { lipgloss.SetColorProfile(profile); lipgloss.SetHasDarkBackground(dark) })
	lipgloss.SetColorProfile(termenv.ANSI256)
	for _, dark := range []bool{false, true} {
		lipgloss.SetHasDarkBackground(dark)
		m := reviewFixture(t, 80, 24)
		m.message.Input().Reset()
		for _, st := range []lipgloss.Style{m.topics.Input.PromptStyle, m.topics.Input.PlaceholderStyle,
			m.message.Input().FocusedStyle.Prompt, m.message.Input().BlurredStyle.Prompt,
			m.message.Input().FocusedStyle.Placeholder, m.message.Input().BlurredStyle.Placeholder} {
			if st.GetForeground() != ui.TextMuted {
				t.Fatal("input hint is not adaptive medium gray")
			}
		}
		view := lipgloss.JoinVertical(lipgloss.Left, m.topics.Input.View(), m.message.Input().View())
		if !strings.Contains(ansi.Strip(view), "Enter new Topic") || !strings.Contains(ansi.Strip(view), "Enter Message") {
			t.Fatal("empty input hints are missing")
		}
		theme := "dark"
		if !dark {
			theme = "light"
		}
		captureReviewView(t, "input-hints-"+theme, view, 80, lipgloss.Height(view))
	}
}

func TestTopicChipsMatchEffectivePublishTargets(t *testing.T) {
	profile, dark := lipgloss.ColorProfile(), lipgloss.HasDarkBackground()
	t.Cleanup(func() { lipgloss.SetColorProfile(profile); lipgloss.SetHasDarkBackground(dark) })
	for _, theme := range []string{"dark", "light", "mono"} {
		lipgloss.SetColorProfile(termenv.ANSI256)
		lipgloss.SetHasDarkBackground(theme != "light")
		if theme == "mono" {
			lipgloss.SetColorProfile(termenv.Ascii)
		}
		for _, tc := range []struct {
			name     string
			items    []topics.Item
			selected int
			targets  []string
		}{
			{"empty", nil, -1, nil},
			{"subscribed fallback", []topics.Item{{Name: "read", Subscribed: true}, {Name: "off"}}, 0, []string{"read"}},
			{"unsubscribed fallback", []topics.Item{{Name: "read", Subscribed: true}, {Name: "off"}}, 1, []string{"off"}},
			{"marked overrides selection", []topics.Item{{Name: "read", Subscribed: true}, {Name: "pub", Publish: true}}, 0, []string{"pub"}},
			{"multiple marked", []topics.Item{{Name: "read", Subscribed: true}, {Name: "pub", Publish: true}, {Name: "both", Subscribed: true, Publish: true}}, 0, []string{"pub", "both"}},
			{"no selection", []topics.Item{{Name: "read", Subscribed: true}, {Name: "off"}}, -1, nil},
			{"invalid selection", []topics.Item{{Name: "off"}}, 8, nil},
		} {
			t.Run(theme+"/"+tc.name, func(t *testing.T) {
				m := reviewFixture(t, 80, 24)
				m.topics.Items = tc.items
				m.topics.SetSelected(tc.selected)
				if got := m.publishTargets(); !slices.Equal(got, tc.targets) {
					t.Fatalf("publish targets = %v, want %v", got, tc.targets)
				}
				chips := renderTopicChips(tc.items, tc.selected, -1, 80)
				for i, item := range tc.items {
					publishing := slices.Contains(tc.targets, item.Name)
					st := ui.Chip
					if publishing {
						st = ui.ChipPublish
					}
					if i == tc.selected {
						st = st.BorderForeground(ui.ColPink)
					}
					label := item.Name
					if theme == "mono" {
						suffix := " [off]"
						switch {
						case item.Subscribed && publishing:
							suffix = " [sub,pub]"
						case item.Subscribed:
							suffix = " [sub]"
						case publishing:
							suffix = " [pub]"
						}
						label += suffix
					}
					if want := st.Render(topicSubscriptionLabel(label, item.Subscribed)); chips[i] != want {
						t.Fatalf("chip %q does not represent its effective state: got=%q want=%q", item.Name, chips[i], want)
					}
					if hint := strings.ToLower(m.topicHoverHint(i)); strings.Contains(hint, "publish target") != publishing {
						t.Fatalf("chip %q tooltip contradicts publish state: %q", item.Name, hint)
					}
				}
				if len(chips) > 0 {
					name := "topic-state-" + theme + "-" + strings.ReplaceAll(tc.name, " ", "-")
					captureReviewView(t, name, lipgloss.JoinHorizontal(lipgloss.Top, chips...), 80, 3)
				}
			})
		}
	}
}

func TestHistoryContrastPresentationFitsTerminalBudgets(t *testing.T) {
	profile, dark := lipgloss.ColorProfile(), lipgloss.HasDarkBackground()
	t.Cleanup(func() { lipgloss.SetColorProfile(profile); lipgloss.SetHasDarkBackground(dark) })
	for _, theme := range []string{"dark", "light", "mono"} {
		lipgloss.SetColorProfile(termenv.ANSI256)
		lipgloss.SetHasDarkBackground(theme != "light")
		if theme == "mono" {
			lipgloss.SetColorProfile(termenv.Ascii)
		}
		for _, width := range []int{40, 80, 120} {
			m := reviewFixture(t, width, 40)
			items := m.history.Items()[:4]
			items[1].Kind = "pub"
			items[2].Kind, items[2].Payload = "log", "Ready"
			selected := true
			items[3].IsSelected = &selected
			m.history.SetItems(items)
			var rows []list.Item
			for _, item := range items {
				rows = append(rows, item)
			}
			m.history.List().SetItems(rows)
			m.layout.history.height = 12
			box := m.renderHistorySection()
			if !strings.Contains(ansi.Strip(box), "Ready") {
				t.Fatalf("history text clipped at width %d: %q", width, ansi.Strip(box))
			}
			captureReviewView(t, "history-contrast-"+theme, box, width, lipgloss.Height(box))
		}
	}
}

func TestTopicSelectionAndPublishMarkingStayInSync(t *testing.T) {
	m := reviewFixture(t, 80, 24)
	m.topics.Items = []topics.Item{{Name: "read", Subscribed: true}, {Name: "off"}}
	m.topics.SetSelected(0)
	m.SetMode(constants.ModeClient)
	m.SetFocus(idTopics)
	check := func(targets []string, mode string) {
		t.Helper()
		if got := m.publishTargets(); !slices.Equal(got, targets) {
			t.Fatalf("publish targets = %v, want %v", got, targets)
		}
		if title := m.messageTargetPreview(); !strings.Contains(title, "("+mode+"): "+strings.Join(targets, ", ")) {
			t.Fatalf("message title does not explain selection mode: %q", title)
		}
	}
	mark := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}}
	check([]string{"read"}, "selected")
	m.Update(mark)
	check([]string{"read"}, "marked")
	m.Update(tea.KeyMsg{Type: tea.KeyRight})
	check([]string{"read"}, "marked")
	if got := m.selectedTopicHint(); got != "Topics: Off." {
		t.Fatalf("unmarked selected topic is incorrectly presented as publishing: %q", got)
	}
	m.Update(mark)
	check([]string{"read", "off"}, "marked")
	m.Update(tea.KeyMsg{Type: tea.KeyLeft})
	m.Update(mark)
	check([]string{"off"}, "marked")
	m.Update(tea.KeyMsg{Type: tea.KeyRight})
	m.Update(mark)
	check([]string{"off"}, "selected")
	if m.topics.Items[1].Publish || m.selectedTopicHint() != "Topics: Publish target." {
		t.Fatal("fallback was persisted as a mark or omitted from the tooltip")
	}
	client := &publishTestClient{}
	m.mqttClient = &MQTTClient{Client: client}
	m.SetFocus(idMessage)
	applyMQTTCommand(m, m.handlePublishKey())
	if len(client.seen) != 1 || client.seen["off"] != m.message.Input().Value() {
		t.Fatalf("actual publish does not match the presented destination: %v", client.seen)
	}
}

func TestMessageTitleShowsTargetModeInNarrowTerminals(t *testing.T) {
	for _, width := range []int{40, 60, 80, 120} {
		m := reviewFixture(t, width, 24)
		m.topics.Items = []topics.Item{{Name: "test", Subscribed: true}}
		m.topics.SetSelected(0)
		for _, marked := range []bool{false, true} {
			m.topics.Items[0].Publish = marked
			mode := "selected"
			if marked {
				mode = "marked"
			}
			view := ansi.Strip(m.message.View())
			if !strings.Contains(view, "("+mode+"): test") || lipgloss.Width(view) > width-2 {
				t.Fatalf("message target or mode is clipped at width %d: %q", width, view)
			}
			captureReviewView(t, "message-target-"+mode+"-dark", m.message.View(), width, lipgloss.Height(view))
		}
	}
}
