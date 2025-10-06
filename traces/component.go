package traces

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	connections "github.com/marang/emqutiti/connections"
	"github.com/marang/emqutiti/constants"
	"github.com/marang/emqutiti/focus"
	"github.com/marang/emqutiti/history"
	"github.com/marang/emqutiti/ui"
)

type KeyAction func(tea.KeyMsg) tea.Cmd

// traceItem represents a single trace configuration and its runtime tracer.
type traceItem struct {
	key    string
	cfg    TracerConfig
	tracer *Tracer
	counts map[string]int
	loaded bool
}

func (t *traceItem) FilterValue() string { return t.key }
func (t *traceItem) Title() string       { return t.key }
func (t *traceItem) Description() string {
	status := "stopped"
	if t.tracer != nil {
		if t.tracer.Running() {
			status = "running"
		} else if t.tracer.Planned() {
			status = "planned"
		}
	} else if time.Now().Before(t.cfg.Start) {
		status = "planned"
	}
	var parts []string
	counts := t.counts
	if t.tracer != nil {
		counts = t.tracer.Counts()
	} else if !t.loaded {
		if c, err := tracerLoadCounts(t.cfg.Profile, t.cfg.Key, t.cfg.Topics); err == nil {
			t.counts = c
			t.loaded = true
			counts = c
		}
	}
	for _, tp := range t.cfg.Topics {
		parts = append(parts, fmt.Sprintf("%s:%d", tp, counts[tp]))
	}
	if len(parts) > 0 {
		status += " " + strings.Join(parts, " ")
	}
	var times []string
	if !t.cfg.Start.IsZero() {
		times = append(times, t.cfg.Start.Format(time.RFC3339))
	}
	if !t.cfg.End.IsZero() {
		times = append(times, t.cfg.End.Format(time.RFC3339))
	}
	if len(times) > 0 {
		status += " " + strings.Join(times, " -> ")
	}
	return status
}

// state groups state related to tracing functionality.
type State struct {
	list  list.Model
	items []*traceItem
	form  *traceForm
	*history.Component
	viewKey string
	hmodel  *histModel
}

// Component implements the traces interface for managing traces. It owns the
// tracing state but delegates broader navigation and history logging to the
// root model.
type Component struct {
	*State
	api     API
	store   Store
	actions map[string]KeyAction
}

type histModel struct {
	api       API
	prev, cur constants.AppMode
}

func (h *histModel) SetMode(mode history.Mode) tea.Cmd {
	if m, ok := mode.(constants.AppMode); ok {
		h.prev = h.cur
		h.cur = m
		switch m {
		case constants.ModeTracer:
			return h.api.SetModeTracer()
		case constants.ModeViewTrace:
			return h.api.SetModeViewTrace()
		}
	}
	return nil
}

func (h *histModel) PreviousMode() history.Mode { return h.prev }

func (h *histModel) CurrentMode() history.Mode { return h.cur }

func (h *histModel) SetModeTraceFilter() tea.Cmd {
	h.prev = h.cur
	h.cur = constants.ModeTraceFilter
	return h.api.SetModeTraceFilter()
}

func (h *histModel) SetFocus(id string) tea.Cmd { return h.api.SetFocus(id) }

func (h *histModel) Width() int { return h.api.Width() }

func (h *histModel) Height() int { return h.api.Height() }

func (h *histModel) OverlayHelp(v string) string { return h.api.OverlayHelp(v) }

func NewComponent(api API, ts State, store Store) *Component {
	hm := &histModel{api: api, cur: constants.ModeViewTrace, prev: constants.ModeTracer}
	ts.Component = history.NewComponent(hm, nil)
	ts.hmodel = hm
	c := &Component{State: &ts, api: api, store: store}
	c.actions = map[string]KeyAction{
		constants.KeyCtrlD: func(tea.KeyMsg) tea.Cmd {
			c.SavePlannedTraces()
			return tea.Quit
		},
		constants.KeyEsc: func(tea.KeyMsg) tea.Cmd {
			c.SavePlannedTraces()
			return c.api.SetModeClient()
		},
		constants.KeyA: func(tea.KeyMsg) tea.Cmd {
			profs := c.api.Profiles()
			opts := make([]string, len(profs))
			for i, p := range profs {
				opts[i] = p.Name
			}
			topics := c.api.SubscribedTopics()
			f := newTraceForm(opts, c.api.ActiveConnection(), topics)
			c.form = &f
			return tea.Batch(
				c.api.SetModeEditTrace(),
				c.api.SetFocus(IDForm),
				textinput.Blink,
			)
		},
		constants.KeyEnter: func(msg tea.KeyMsg) tea.Cmd {
			i := c.list.Index()
			if i >= 0 && i < len(c.items) {
				it := c.items[i]
				if it.tracer != nil && (it.tracer.Running() || it.tracer.Planned()) {
					c.stopTrace(i)
				} else {
					c.startTrace(i)
				}
			}
			return c.listUpdate(msg)
		},
		constants.KeyV: func(tea.KeyMsg) tea.Cmd {
			i := c.list.Index()
			if i >= 0 && i < len(c.items) {
				c.loadTraceMessages(i)
			}
			return nil
		},
		constants.KeyDelete: func(tea.KeyMsg) tea.Cmd {
			i := c.list.Index()
			if i >= 0 && i < len(c.items) {
				it := c.items[i]
				key := it.key
				cfg := it.cfg
				rf := func() tea.Cmd { return c.api.SetFocus(c.api.FocusedID()) }
				c.api.StartConfirm(
					fmt.Sprintf("Delete trace '%s'? [y/n]", key),
					"This also removes all stored data of this trace",
					rf,
					func() tea.Cmd {
						c.stopTrace(i)
						c.items = append(c.items[:i], c.items[i+1:]...)
						items := make([]list.Item, len(c.items))
						for idx, itm := range c.items {
							items[idx] = itm
						}
						c.list.SetItems(items)
						if err := c.store.RemoveTrace(key); err != nil {
							c.api.LogHistory("", err.Error(), "log", false, err.Error())
						}
						if err := c.store.ClearData(cfg.Profile, key); err != nil {
							c.api.LogHistory("", err.Error(), "log", false, err.Error())
						}
						if c.anyTraceRunning() {
							return traceTicker()
						}
						return nil
					},
					nil,
				)
			}
			return nil
		},
		constants.KeyCtrlShiftUp: func(msg tea.KeyMsg) tea.Cmd {
			if c.api.TraceHeight() > 1 {
				c.api.SetTraceHeight(c.api.TraceHeight() - 1)
				c.list.SetSize(c.api.Width()-4, c.api.Height()-4)
			}
			return nil
		},
		constants.KeyCtrlShiftDown: func(msg tea.KeyMsg) tea.Cmd {
			c.api.SetTraceHeight(c.api.TraceHeight() + 1)
			c.list.SetSize(c.api.Width()-4, c.api.Height()-4)
			return nil
		},
	}
	return c
}

func (t *Component) Init() tea.Cmd { return nil }

func (t *Component) View() string { return t.viewTraces() }

func (t *Component) Focus() tea.Cmd { return nil }

func (t *Component) Blur() {}

// Focusables satisfies FocusableSet; the base model provides trace focusables.
func (t *Component) Focusables() map[string]focus.Focusable { return map[string]focus.Focusable{} }

// List exposes the trace configuration list model.
func (t *Component) List() *list.Model { return &t.list }

// ViewList exposes the trace message list model.
func (t *Component) ViewList() *list.Model { return t.Component.List() }

type traceTickMsg struct{}

// traceTicker schedules periodic refresh events while traces run.
func traceTicker() tea.Cmd {
	return tea.Tick(500*time.Millisecond, func(t time.Time) tea.Msg { return traceTickMsg{} })
}

func (t *Component) startFilter() tea.Cmd {
	idx := t.traceIndex(t.viewKey)
	var topics []string
	if idx >= 0 {
		topics = append(topics, t.items[idx].cfg.Topics...)
	}
	var topic, payload string
	var start, end time.Time
	if t.FilterQuery() != "" {
		ts, s, e, p := history.ParseQuery(t.FilterQuery())
		if len(ts) > 0 {
			topic = ts[0]
		}
		start, end, payload = s, e, p
	} else {
		end = time.Now()
		start = end.Add(-time.Hour)
	}
	hf := history.NewFilterForm(topics, topic, payload, start, end, t.ShowArchived())
	t.SetFilterForm(&hf)
	return t.hmodel.SetModeTraceFilter()
}

// Update manages the traces list and responds to key presses.
func (t *Component) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case traceTickMsg:
		// refresh
	case tea.KeyMsg:
		if act, ok := t.actions[msg.String()]; ok {
			return act(msg)
		}
	}
	return t.listUpdate(msg)
}

func (t *Component) listUpdate(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	t.list, cmd = t.list.Update(msg)
	if t.anyTraceRunning() {
		return tea.Batch(cmd, traceTicker())
	}
	return cmd
}

// UpdateForm handles input for the new trace form.
func (t *Component) UpdateForm(msg tea.Msg) tea.Cmd {
	if t.form == nil {
		return nil
	}
	if km, ok := msg.(tea.KeyMsg); ok {
		switch km.String() {
		case constants.KeyCtrlD:
			return tea.Quit
		case constants.KeyEsc:
			t.form = nil
			return t.api.SetModeTracer()
		}
		if t.api.FocusedID() != IDForm {
			return nil
		}
		if km.String() == constants.KeyEnter {
			form, err := t.form.Validate()
			if err != nil {
				t.form = &form
				return nil
			}
			t.form = &form
			cfg := t.form.Config()
			if cfg.Key == "" || len(cfg.Topics) == 0 || cfg.Profile == "" {
				t.form.errMsg = "key, profile and topics required"
				return nil
			}
			if cfg.Start.IsZero() {
				cfg.Start = time.Now().Round(time.Second)
				if tf, ok := t.form.Fields[idxTraceStart].(*ui.TextField); ok {
					tf.SetValue(cfg.Start.Format(time.RFC3339))
				}
			}
			if cfg.End.IsZero() {
				cfg.End = cfg.Start.Add(time.Hour)
				if tf, ok := t.form.Fields[idxTraceEnd].(*ui.TextField); ok {
					tf.SetValue(cfg.End.Format(time.RFC3339))
				}
			}
			if t.traceIndex(cfg.Key) >= 0 {
				t.form.errMsg = "trace key exists"
				return nil
			}
			p, err := connections.LoadProfile(cfg.Profile, "")
			if err != nil {
				t.form.errMsg = err.Error()
				return nil
			}
			if p.FromEnv {
				connections.ApplyEnvVars(p)
			}
			connections.ApplyDefaultPassword(p)
			client, err := t.api.NewClient(*p)
			if err != nil {
				t.form.errMsg = err.Error()
				return nil
			}
			client.Disconnect()
			newItem := &traceItem{key: cfg.Key, cfg: cfg}
			t.items = append(t.items, newItem)
			items := t.list.Items()
			items = append(items, newItem)
			t.list.SetItems(items)
			if err := addTrace(cfg); err != nil {
				t.form.errMsg = err.Error()
				return nil
			}
			t.form = nil
			return t.api.SetModeTracer()
		}
	} else if t.api.FocusedID() != IDForm {
		return nil
	}
	var cmd tea.Cmd
	f, cmd := t.form.Update(msg)
	t.form = &f
	return cmd
}

// UpdateView displays messages captured for a trace.
func (t *Component) UpdateView(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case constants.KeyEsc:
			return t.api.SetModeTracer()
		case constants.KeyCtrlD:
			return tea.Quit
		case constants.KeyCtrlShiftUp:
			if t.api.TraceHeight() > 1 {
				t.api.SetTraceHeight(t.api.TraceHeight() - 1)
				t.Component.List().SetSize(t.api.Width()-4, t.api.TraceHeight())
			}
			return nil
		case constants.KeyCtrlShiftDown:
			t.api.SetTraceHeight(t.api.TraceHeight() + 1)
			t.Component.List().SetSize(t.api.Width()-4, t.api.TraceHeight())
			return nil
		case constants.KeySlash:
			return t.startFilter()
		}
	}
	return t.Component.Update(msg)
}
