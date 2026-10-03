package traces

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	connections "github.com/marang/emqutiti/connections"
	"github.com/marang/emqutiti/constants"
	"github.com/marang/emqutiti/ui"
)

type testAPI struct {
	mode          constants.AppMode
	width, height int
	focused       string
	prompt        string
	confirm       func() tea.Cmd
	logs          []string
}

func (t *testAPI) StartConfirm(prompt, _ string, _ func() tea.Cmd, action func() tea.Cmd, _ func()) {
	t.prompt, t.confirm = prompt, action
}
func (t *testAPI) SetModeClient() tea.Cmd                         { t.mode = constants.ModeClient; return nil }
func (t *testAPI) SetModeTracer() tea.Cmd                         { t.mode = constants.ModeTracer; return nil }
func (t *testAPI) SetModeEditTrace() tea.Cmd                      { t.mode = constants.ModeEditTrace; return nil }
func (t *testAPI) SetModeViewTrace() tea.Cmd                      { t.mode = constants.ModeViewTrace; return nil }
func (t *testAPI) SetModeTraceFilter() tea.Cmd                    { t.mode = constants.ModeTraceFilter; return nil }
func (t *testAPI) SetFocus(id string) tea.Cmd                     { t.focused = id; return nil }
func (t *testAPI) FocusedID() string                              { return t.focused }
func (t *testAPI) ResetElemPos()                                  {}
func (t *testAPI) SetElemPos(string, int)                         {}
func (t *testAPI) OverlayHelp(v string) string                    { return "header\n" + v }
func (t *testAPI) Profiles() []connections.Profile                { return nil }
func (t *testAPI) ActiveConnection() string                       { return "" }
func (t *testAPI) SubscribedTopics() []string                     { return nil }
func (t *testAPI) LogHistory(_, _, _ string, _ bool, text string) { t.logs = append(t.logs, text) }
func (t *testAPI) TraceHeight() int                               { return 0 }
func (t *testAPI) SetTraceHeight(int)                             {}
func (t *testAPI) Width() int {
	if t.width == 0 {
		return 80
	}
	return t.width
}
func (t *testAPI) Height() int {
	if t.height == 0 {
		return 24
	}
	return t.height
}
func (t *testAPI) NewClient(connections.Profile) (Client, error) { return nil, nil }

type noopStore struct{}

func (noopStore) LoadTraces() map[string]TracerConfig                         { return nil }
func (noopStore) SaveTraces(map[string]TracerConfig) error                    { return nil }
func (noopStore) AddTrace(TracerConfig) error                                 { return nil }
func (noopStore) RemoveTrace(string) error                                    { return nil }
func (noopStore) Messages(string, string) ([]TracerMessage, error)            { return nil, nil }
func (noopStore) HasData(string, string) (bool, error)                        { return false, nil }
func (noopStore) ClearData(string, string) error                              { return nil }
func (noopStore) LoadCounts(string, string, []string) (map[string]int, error) { return nil, nil }

func TestEscSetsClientMode(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	api := &testAPI{}
	c := NewComponent(api, State{}, &noopStore{})
	c.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if api.mode != constants.ModeClient {
		t.Fatalf("expected mode %v, got %v", constants.ModeClient, api.mode)
	}
}

func traceTestComponent() (*Component, *testAPI) {
	api := &testAPI{focused: IDList, mode: constants.ModeTracer}
	c := NewComponent(api, State{}, &noopStore{})
	c.items = []*traceItem{
		{key: "first", cfg: TracerConfig{Key: "first", End: time.Unix(1, 0)}, loaded: true},
		{key: "target", cfg: TracerConfig{Key: "target", End: time.Unix(1, 0)}, loaded: true},
		{key: "last", cfg: TracerConfig{Key: "last", End: time.Unix(1, 0)}, loaded: true},
	}
	c.list.SetItems([]list.Item{c.items[0], c.items[1], c.items[2]})
	c.SetSize(api.Width(), api.Height())
	return c, api
}

func TestTraceFilterOwnsActionKeysWithNonemptyList(t *testing.T) {
	c, api := traceTestComponent()
	c.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	if c.list.FilterState() != list.Filtering {
		t.Fatal("slash did not enter filter on a nonempty list")
	}
	c.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	c.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'v'}})
	if c.form != nil || c.list.FilterValue() != "av" || api.mode != constants.ModeTracer {
		t.Fatal("typing action letters opened a trace action instead of search")
	}
	c.list.SetFilterText("target")
	c.list.SetFilterState(list.Filtering)
	c.Update(tea.KeyMsg{Type: tea.KeyDelete})
	c.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if api.confirm != nil || len(api.logs) > 0 || c.list.FilterState() != list.FilterApplied {
		t.Fatal("filter Enter/Delete triggered trace actions")
	}
	c.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if api.mode != constants.ModeTracer || c.list.FilterState() != list.Unfiltered {
		t.Fatal("Esc left traces instead of clearing filter")
	}
}

func TestTraceFilteredActionsUseVisibleTrace(t *testing.T) {
	c, api := traceTestComponent()
	c.list.SetFilterText("target")
	c.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if len(api.logs) != 1 || !strings.Contains(api.logs[0], "'target'") {
		t.Fatalf("started wrong filtered trace: %v", api.logs)
	}
	c.Update(tea.KeyMsg{Type: tea.KeyDelete})
	if api.confirm == nil || !strings.Contains(api.prompt, "target") {
		t.Fatalf("delete targeted wrong filtered trace: %q", api.prompt)
	}
	pending := api.confirm
	c.list.ResetFilter()
	c.list.Select(0)
	c.Update(tea.KeyMsg{Type: tea.KeyDelete})
	api.confirm()
	pending()
	if len(c.items) != 1 || c.items[0].key != "last" {
		t.Fatalf("pending deletion targeted wrong trace: %v", c.items)
	}
}

func TestTraceRowClicks(t *testing.T) {
	c, api := traceTestComponent()
	c.Update(tea.MouseMsg{Action: tea.MouseActionPress, Button: tea.MouseButtonRight, X: 3, Y: 15})
	if api.confirm != nil {
		t.Fatal("empty trace area opened confirmation")
	}
	c.Update(tea.MouseMsg{Action: tea.MouseActionPress, Button: tea.MouseButtonRight, X: 3, Y: 7})
	if api.confirm == nil || !strings.Contains(api.prompt, "target") || len(c.items) != 3 {
		t.Fatal("right click did not confirm the clicked trace")
	}
}

func TestTraceManagerFitsTerminal(t *testing.T) {
	for _, size := range [][2]int{{120, 40}, {80, 24}, {60, 20}, {40, 16}} {
		t.Run(fmt.Sprint(size), func(t *testing.T) {
			c, api := traceTestComponent()
			api.width, api.height = size[0], size[1]
			for i := 0; i < 20; i++ {
				item := &traceItem{key: strings.Repeat("long", 30), cfg: TracerConfig{}, loaded: true}
				c.items = append(c.items, item)
				c.list.InsertItem(len(c.list.Items()), item)
			}
			for _, filtered := range []bool{false, true} {
				if filtered {
					c.list.SetFilterText("long")
					c.list.SetFilterState(list.Filtering)
				}
				view := c.View()
				if w, h := lipgloss.Size(view); w > size[0] || h > size[1] {
					t.Fatalf("view %dx%d exceeds %v:\n%s", w, h, size, view)
				}
				for _, action := range []string{"[a] add", "[enter] start/stop", "[v] view", "[del] delete", "[/] filter", "[esc] back"} {
					if !strings.Contains(view, action) {
						t.Fatalf("missing action %q", action)
					}
				}
			}
		})
	}
}

func TestTraceFormFitsTerminalAndPreservesFields(t *testing.T) {
	profiles := make([]string, 30)
	for i := range profiles {
		profiles[i] = strings.Repeat("profile", 20) + fmt.Sprint(i)
	}
	for _, size := range [][2]int{{120, 40}, {80, 24}, {60, 20}, {40, 16}} {
		t.Run(fmt.Sprint(size), func(t *testing.T) {
			c, api := traceTestComponent()
			api.width, api.height = size[0], size[1]
			api.focused = IDForm
			form := newTraceForm(profiles, profiles[0], []string{strings.Repeat("topic", 30)})
			form.Fields[idxTraceKey].(*ui.TextField).SetValue(strings.Repeat("key", 30))
			c.form = &form
			for focus := range form.Fields {
				c.form.Focus = focus
				for _, errMsg := range []string{"", strings.Repeat("validation", 40)} {
					c.form.errMsg = errMsg
					view := c.ViewForm()
					if w, h := lipgloss.Size(view); w > size[0] || h > size[1] {
						t.Fatalf("form %dx%d exceeds %v:\n%s", w, h, size, view)
					}
					if !strings.Contains(view, "[enter] save") || !strings.Contains(view, "[esc] cancel") {
						t.Fatal("form hid save/cancel actions")
					}
					if form.Fields[idxTraceKey].Value() != strings.Repeat("key", 30) {
						t.Fatal("resizing changed the field value")
					}
				}
			}
		})
	}
}

func TestTraceFormClickUsesVisibleFieldRows(t *testing.T) {
	c, api := traceTestComponent()
	api.focused, api.width, api.height = IDForm, 40, 16
	form := newTraceForm([]string{"one", "two", "three", "four"}, "one", nil)
	form.Focus = idxTraceProfile
	c.form = &form
	c.ViewForm()
	topicsY := c.form.rows[idxTraceTopics] + 2
	c.UpdateForm(tea.MouseMsg{Action: tea.MouseActionPress, Button: tea.MouseButtonLeft, X: 3, Y: topicsY})
	if c.form.Focus != idxTraceTopics {
		t.Fatalf("clicked topics selected field %d", c.form.Focus)
	}
	c.UpdateForm(tea.MouseMsg{Action: tea.MouseActionPress, Button: tea.MouseButtonLeft, X: 0, Y: 2})
	if c.form.Focus != idxTraceTopics {
		t.Fatal("border click changed form focus")
	}
}
