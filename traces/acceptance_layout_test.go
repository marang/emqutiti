package traces

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/muesli/termenv"

	"github.com/marang/emqutiti/connections"
	"github.com/marang/emqutiti/constants"
	"github.com/marang/emqutiti/ui"
)

type reviewTraceClient struct{ calls int }

func (c *reviewTraceClient) Subscribe(string, byte, mqtt.MessageHandler) error {
	c.calls++
	return nil
}
func (c *reviewTraceClient) Unsubscribe(string) error { c.calls++; return nil }
func (c *reviewTraceClient) Disconnect()              { c.calls++ }

type reviewTraceAPI struct {
	testAPI
	client      *reviewTraceClient
	clientCalls int
}

func (a *reviewTraceAPI) NewClient(connections.Profile) (Client, error) {
	a.clientCalls++
	return a.client, nil
}

func (a *reviewTraceAPI) OverlayHelp(view string) string {
	return ui.InfoSubtleStyle.Render("emqutiti | Traces") + "\n" + view
}

func populatedReviewTraces(width, height int) (*Component, *reviewTraceAPI) {
	client := &reviewTraceClient{}
	api := &reviewTraceAPI{
		testAPI: testAPI{width: width, height: height, focused: IDList, mode: constants.ModeTracer},
		client:  client,
	}
	topics := []string{
		"plant/zone-3/temperature",
		"factory/east/line-7/controllers/production/telemetry/" + strings.Repeat("long-segment/", 8) + "#",
	}
	configs := []TracerConfig{
		{Key: "running-live-telemetry-" + strings.Repeat("production-line-", 8), Start: time.Unix(0, 0)},
		{Key: "planned-overnight-recording-" + strings.Repeat("controllers-", 8), Start: time.Date(9999, 1, 1, 0, 0, 0, 0, time.UTC)},
		{Key: "stopped-shift-report-" + strings.Repeat("temperature-sensors-", 8), Start: time.Date(2026, 1, 1, 9, 0, 0, 0, time.UTC), End: time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)},
	}
	var items []*traceItem
	for i, cfg := range configs {
		cfg.Profile, cfg.Topics = "review-broker", append([]string(nil), topics...)
		tracer := newTracer(cfg, client)
		tracer.counts = map[string]int{topics[0]: 1204 + i, topics[1]: 58 + i}
		tracer.running, tracer.ready = i == 0, i == 0
		items = append(items, &traceItem{key: cfg.Key, cfg: cfg, tracer: tracer, loaded: true})
	}
	for i := 0; i < 18; i++ {
		cfg := TracerConfig{Key: fmt.Sprintf("stopped-archived-shift-%02d", i), Profile: "review-broker", Topics: append([]string(nil), topics...)}
		items = append(items, &traceItem{key: cfg.Key, cfg: cfg, counts: map[string]int{topics[0]: 42 + i}, loaded: true})
	}
	component := NewComponent(api, State{items: items}, noopStore{})
	component.SetSize(width, height)
	return component, api
}

func captureTraceReview(t *testing.T, name, view string) {
	t.Helper()
	dir := os.Getenv("EMQUTITI_REVIEW_CAPTURE_DIR")
	if dir == "" {
		return
	}
	if !filepath.IsAbs(dir) {
		t.Fatal("EMQUTITI_REVIEW_CAPTURE_DIR must be an absolute path")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("create capture directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, name+".ansi"), []byte(view), 0o600); err != nil {
		t.Fatalf("write ANSI capture: %v", err)
	}
}

func assertTraceReviewFits(t *testing.T, view string, width, height int, actions ...string) {
	t.Helper()
	if renderWidth, renderHeight := lipgloss.Size(view); renderWidth > width || renderHeight > height {
		t.Fatalf("render %dx%d exceeds terminal %dx%d:\n%s", renderWidth, renderHeight, width, height, view)
	}
	plain := ansi.Strip(view)
	if !strings.HasPrefix(plain, " emqutiti | Traces\n") {
		t.Fatal("capture must include exactly one root header row")
	}
	for _, action := range actions {
		if !strings.Contains(plain, action) {
			t.Fatalf("missing visible action %q", action)
		}
	}
}

func TestTraceAcceptanceLayout(t *testing.T) {
	themes := []struct {
		name    string
		profile termenv.Profile
		dark    bool
	}{
		{name: "dark", profile: termenv.TrueColor, dark: true},
		{name: "light", profile: termenv.TrueColor},
		{name: "mono", profile: termenv.Ascii, dark: true},
	}
	for _, theme := range themes {
		t.Run(theme.name, func(t *testing.T) {
			// Lip Gloss uses process-global color settings; these cases stay serial.
			profile, dark := lipgloss.ColorProfile(), lipgloss.HasDarkBackground()
			t.Cleanup(func() {
				lipgloss.SetColorProfile(profile)
				lipgloss.SetHasDarkBackground(dark)
			})
			lipgloss.SetColorProfile(theme.profile)
			lipgloss.SetHasDarkBackground(theme.dark)
			for _, size := range [][2]int{{120, 40}, {80, 24}, {60, 20}, {40, 16}} {
				t.Run(fmt.Sprintf("%dx%d", size[0], size[1]), func(t *testing.T) {
					component, api := populatedReviewTraces(size[0], size[1])
					for i, status := range []string{"running", "planned", "stopped"} {
						if !strings.HasPrefix(component.items[i].Description(), status+" ") {
							t.Fatalf("fixture %d does not represent %s", i, status)
						}
					}
					name := fmt.Sprintf("active-traces-%s-%dx%d", theme.name, size[0], size[1])
					for _, filtering := range []bool{false, true} {
						captureName := name
						if filtering {
							component.list.SetFilterText("running")
							component.list.SetFilterState(list.Filtering)
							captureName += "-filtering"
						}
						view := component.View()
						assertTraceReviewFits(t, view, size[0], size[1], "[a] add", "[enter] start/stop", "[v] view", "[del] delete", "[/] filter", "[esc] back")
						plain := ansi.Strip(view)
						if len(component.list.VisibleItems()) == 0 || !strings.Contains(plain, "running") {
							t.Fatal("acceptance view must show a populated, running trace")
						}
						if filtering && !strings.Contains(plain, "Filter: running") {
							t.Fatal("capture did not show the active filter input")
						}
						if theme.name == "mono" && plain != view {
							t.Fatal("monochrome capture contains ANSI styling")
						}
						if theme.name != "mono" && !strings.Contains(view, "\x1b[") {
							t.Fatal("color capture has no ANSI styling")
						}
						captureTraceReview(t, captureName, view)
					}
					captureTraceReviewForms(t, component, api, theme.name)
					if api.clientCalls != 0 || api.client.calls != 0 {
						t.Fatal("rendering performed client operations")
					}
				})
			}
		})
	}
}

func captureTraceReviewForms(t *testing.T, component *Component, api *reviewTraceAPI, theme string) {
	t.Helper()
	profiles := make([]string, 20)
	for i := range profiles {
		profiles[i] = fmt.Sprintf("broker-%02d-", i) + strings.Repeat("production-", 8)
	}
	api.focused = IDForm
	for _, state := range []string{"select", "error"} {
		form := newTraceForm(profiles, profiles[12], component.items[0].cfg.Topics)
		form.Fields[idxTraceKey].(*ui.TextField).SetValue("new-trace-" + strings.Repeat("plant-telemetry-", 10))
		form.Fields[idxTraceStart].(*ui.TextField).SetValue("2026-10-03T09:00:00Z")
		form.Fields[idxTraceEnd].(*ui.TextField).SetValue("2026-10-03T10:00:00Z")
		form.Focus = idxTraceProfile
		if state == "error" {
			form.Focus = idxTraceKey
			form.errMsg = "trace key exists: " + strings.Repeat("production-recording-", 20)
		}
		component.form = &form
		view := component.ViewForm()
		assertTraceReviewFits(t, view, api.Width(), api.Height(), "[enter] save", "[esc] cancel")
		plain := ansi.Strip(view)
		if state == "select" && strings.Count(plain, "broker-") < 2 {
			t.Fatal("select capture did not show profile options")
		}
		if state == "error" && !strings.Contains(plain, "trace key exists:") {
			t.Fatal("error capture hid the validation error")
		}
		captureTraceReview(t, fmt.Sprintf("trace-form-%s-%s-%dx%d", state, theme, api.Width(), api.Height()), view)
	}
}
