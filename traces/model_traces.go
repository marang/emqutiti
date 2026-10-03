package traces

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"

	connections "github.com/marang/emqutiti/connections"
	"github.com/marang/emqutiti/history"
)

// forceStartTrace launches the tracer at index without checking existing data.
func (t *Component) forceStartTrace(index int) tea.Cmd {
	if index < 0 || index >= len(t.items) {
		return nil
	}
	item := t.items[index]
	p, err := connections.LoadProfile(item.cfg.Profile, "")
	if err != nil {
		t.api.LogHistory("", err.Error(), "log", false, err.Error())
		return nil
	}
	if p.FromEnv {
		connections.ApplyEnvVars(p)
	}
	connections.ApplyDefaultPassword(p)
	client, err := t.api.NewClient(*p)
	if err != nil {
		t.api.LogHistory("", err.Error(), "log", false, err.Error())
		return nil
	}
	tr := newTracer(item.cfg, client)
	if err := tr.Start(); err != nil {
		t.api.LogHistory("", err.Error(), "log", false, err.Error())
		client.Disconnect()
		return nil
	}
	item.tracer = tr
	if err := t.store.AddTrace(item.cfg); err != nil {
		t.api.LogHistory("", err.Error(), "log", false, err.Error())
	}
	return listenTraceReports(item.key, tr)
}

// startTrace starts the tracer at index, prompting if data already exists.
func (t *Component) startTrace(index int) tea.Cmd {
	if index < 0 || index >= len(t.items) {
		return nil
	}
	item := t.items[index]
	if !item.cfg.End.IsZero() && time.Now().After(item.cfg.End) {
		t.api.LogHistory("", fmt.Sprintf("trace '%s' already finished", item.key), "log", false, fmt.Sprintf("trace '%s' already finished", item.key))
		return nil
	}
	exists, err := t.store.HasData(item.cfg.Profile, item.key)
	if err == nil && exists {
		focused := t.api.FocusedID()
		rf := func() tea.Cmd { return t.api.SetFocus(focused) }
		t.api.StartConfirm(fmt.Sprintf("Overwrite trace '%s'? [y/n]", item.key), "existing trace data will be removed", rf, func() tea.Cmd {
			index := t.traceIndex(item.key)
			if index < 0 || t.items[index] != item {
				return nil
			}
			if err := t.store.ClearData(item.cfg.Profile, item.key); err != nil {
				t.api.LogHistory("", err.Error(), "log", false, err.Error())
				return nil
			}
			return t.forceStartTrace(index)
		}, nil)
		return nil
	}
	return t.forceStartTrace(index)
}

// stopTrace stops a running tracer at the given index.
func (t *Component) stopTrace(index int) {
	if index < 0 || index >= len(t.items) {
		return
	}
	if tr := t.items[index].tracer; tr != nil {
		tr.Stop()
	}
}

// anyTraceRunning reports whether any tracer is currently active or planned.
func (t *Component) anyTraceRunning() bool {
	for i := range t.items {
		if tr := t.items[i].tracer; tr != nil && (tr.Running() || tr.Planned()) {
			return true
		}
	}
	return false
}

// traceIndex returns the index of the trace with the given key or -1.
func (t *Component) traceIndex(key string) int {
	for i, it := range t.items {
		if it.key == key {
			return i
		}
	}
	return -1
}

// savePlannedTraces persists trace configurations for later sessions.
func (t *Component) SavePlannedTraces() {
	data := map[string]TracerConfig{}
	for _, it := range t.items {
		if it.tracer != nil {
			data[it.key] = it.tracer.Config()
		} else {
			data[it.key] = it.cfg
		}
	}
	if err := saveTraces(data); err != nil {
		t.api.LogHistory("", err.Error(), "log", false, err.Error())
	}
}

// loadTraceMessages loads messages for the trace at index and shows them.
func (t *Component) loadTraceMessages(index int) {
	if index < 0 || index >= len(t.items) {
		return
	}
	it := t.items[index]
	msgs, err := tracerMessages(it.cfg.Profile, it.key)
	if err != nil {
		t.api.LogHistory("", err.Error(), "log", false, err.Error())
		return
	}
	histItems := make([]history.Item, len(msgs))
	listItems := make([]list.Item, len(msgs))
	hmsgs := make([]history.Message, len(msgs))
	for i, mmsg := range msgs {
		hi := history.Item{Timestamp: mmsg.Timestamp, Topic: mmsg.Topic, Payload: mmsg.Payload, Kind: mmsg.Kind, Retained: mmsg.Retained}
		histItems[i] = hi
		listItems[i] = hi
		hmsgs[i] = history.Message{Timestamp: mmsg.Timestamp, Topic: mmsg.Topic, Payload: mmsg.Payload, Kind: mmsg.Kind, Archived: false, Retained: mmsg.Retained}
	}
	t.Component.SetItems(histItems)
	t.Component.List().SetItems(listItems)
	t.Component.SetStore(newMemStore(hmsgs))
	t.Component.List().SetSize(t.api.Width()-4, t.api.TraceHeight())
	t.viewKey = it.key
	_ = t.api.SetModeViewTrace()
}
