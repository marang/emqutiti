package emqutiti

import (
	"fmt"
	"runtime"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/muesli/termenv"
)

func TestMessageFooterKeepsActionsVisibleAndEditorHeight(t *testing.T) {
	profile, dark := lipgloss.ColorProfile(), lipgloss.HasDarkBackground()
	t.Cleanup(func() { lipgloss.SetColorProfile(profile); lipgloss.SetHasDarkBackground(dark) })
	for _, theme := range []string{"dark", "light", "mono"} {
		lipgloss.SetColorProfile(termenv.ANSI256)
		lipgloss.SetHasDarkBackground(theme != "light")
		if theme == "mono" {
			lipgloss.SetColorProfile(termenv.Ascii)
		}
		for _, size := range reviewSizes {
			t.Run(fmt.Sprintf("%s/%dx%d", theme, size.width, size.height), func(t *testing.T) {
				m := reviewFixture(t, size.width, size.height)
				m.ui.modifiedKeyInput = true
				m.clampPanelHeights()
				normal, retained := publishShortcutNames(runtime.GOOS)
				draft := m.message.Input().Value()
				height := m.layout.message.height + m.message.FooterHeight() + 2
				for _, focus := range []string{idTopic, idTopics, idMessage, idHistory} {
					m.SetFocus(focus)
					view := m.message.View()
					plain := ansi.Strip(view)
					for _, hint := range []string{"[" + normal + "] publish", "[" + retained + "] retained", "[Enter] newline"} {
						if !strings.Contains(plain, hint) {
							t.Fatalf("message footer lost %q with focus %s:\n%s", hint, focus, plain)
						}
					}
					if lipgloss.Width(view) != size.width-2 || lipgloss.Height(view) != height || m.message.Input().Height() != m.layout.message.height || m.message.Input().Value() != draft {
						t.Fatal("footer changed box geometry, editor height or draft")
					}
					lines := strings.Split(plain, "\n")
					if !strings.Contains(lines[len(lines)-2], "newline") {
						t.Fatal("shortcut footer is not at the bottom of the message box")
					}
				}
				captureReviewView(t, "message-footer-"+theme, m.message.View(), size.width, height)
				m.mqttClient = &MQTTClient{Client: &fakeClient{}}
				m.View()
				m.SetFocus(idMessage)
				view := m.View()
				if !strings.Contains(ansi.Strip(view), "["+retained+"] retained") || !strings.Contains(ansi.Strip(view), "[Enter] newline") {
					t.Fatalf("focusing Message did not reveal its footer:\n%s", ansi.Strip(view))
				}
				captureReviewView(t, "client-message-footer-"+theme, view, size.width, size.height)
				m.setPanelHeight(idMessage, 1000)
				if lipgloss.Height(m.message.View()) > size.height-4 {
					t.Fatal("resize limit failed to reserve shortcut footer rows")
				}
			})
		}
	}
}

func TestMessageFooterWithoutModifiedKeyAdapter(t *testing.T) {
	m := reviewFixture(t, 40, 16)
	plain := ansi.Strip(m.message.View())
	if !strings.Contains(plain, "terminal support") || !strings.Contains(plain, "[Enter] newline") || strings.Contains(plain, "[Ctrl+Enter]") || strings.Contains(plain, "[Cmd+Enter]") {
		t.Fatalf("unsupported input promises a publish action:\n%s", plain)
	}
}

func TestPublishShortcutLabelsMatchPlatforms(t *testing.T) {
	for _, goos := range []string{"linux", "darwin", "windows"} {
		modifier := "Ctrl"
		if goos == "darwin" {
			modifier = "Cmd"
		}
		normal, retained := publishShortcutNames(goos)
		if normal != modifier+"+Enter" || retained != modifier+"+Shift+Enter" {
			t.Fatalf("wrong %s publish shortcut names: %s/%s", goos, normal, retained)
		}
		if got := publishShortcutHint(goos); got != "["+normal+"] publish  ["+retained+"] retained" {
			t.Fatalf("wrong %s publish hint: %s", goos, got)
		}
	}
}
