package traces

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/marang/emqutiti/connections"
)

type reportTestAPI struct {
	testAPI
	client Client
}

func (a *reportTestAPI) NewClient(connections.Profile) (Client, error) { return a.client, nil }

type reportTestStore struct {
	noopStore
	hasData bool
	cleared int
}

func (s *reportTestStore) HasData(string, string) (bool, error) { return s.hasData, nil }
func (s *reportTestStore) ClearData(string, string) error       { s.cleared++; return nil }

func reportTestComponent(t *testing.T, client Client) (*Component, *reportTestAPI, *reportTestStore) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("EMQUTITI_HOME", dir)
	if err := os.WriteFile(filepath.Join(dir, "config.toml"), []byte("[[profiles]]\nname = 'report-test'\nhost = 'localhost'\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	previousAddr := proxyAddr
	SetProxyAddr("127.0.0.1:1")
	t.Cleanup(func() { SetProxyAddr(previousAddr) })
	api := &reportTestAPI{testAPI: testAPI{focused: IDList}, client: client}
	store := &reportTestStore{}
	item := &traceItem{key: "report-test", cfg: TracerConfig{Key: "report-test", Profile: "report-test", Topics: []string{"topic"}, Start: time.Now().Add(-time.Second)}, loaded: true}
	c := NewComponent(api, State{items: []*traceItem{item}}, store)
	c.list.SetItems([]list.Item{item})
	t.Cleanup(func() { c.stopTrace(0) })
	return c, api, store
}

func runTraceReportCommand(t *testing.T, cmd tea.Cmd) ReportMsg {
	t.Helper()
	if cmd == nil {
		t.Fatal("trace did not return a report listener")
	}
	result := make(chan tea.Msg, 1)
	go func() { result <- cmd() }()
	select {
	case msg := <-result:
		if batch, ok := msg.(tea.BatchMsg); ok {
			for _, child := range batch {
				if child != nil {
					return runTraceReportCommand(t, child)
				}
			}
		}
		report, ok := msg.(ReportMsg)
		if !ok {
			t.Fatalf("listener returned %T, want ReportMsg", msg)
		}
		return report
	case <-time.After(2 * time.Second):
		t.Fatal("report listener did not terminate")
		return ReportMsg{}
	}
}

func TestTraceReportSubscriptionFailureReachesHistory(t *testing.T) {
	c, api, _ := reportTestComponent(t, &failingSubscriptionClient{newFakeClient()})
	cmd := c.Update(tea.KeyMsg{Type: tea.KeyEnter})
	tr := c.items[0].tracer
	if tr == nil {
		t.Fatal("start action did not install a tracer")
	}
	select {
	case <-tr.done:
	case <-time.After(2 * time.Second):
		t.Fatal("failing subscription did not finish")
	}
	msg := runTraceReportCommand(t, cmd)
	if msg.Err == nil || !strings.Contains(msg.Err.Error(), "subscribe topic: subscription denied") {
		t.Fatalf("listener lost the subscription failure: %+v", msg)
	}
	next := c.HandleReport(msg)
	if len(api.logs) != 1 || !strings.Contains(api.logs[0], "report-test") || !strings.Contains(api.logs[0], "subscription denied") {
		t.Fatalf("runtime failure did not reach UI history: %v", api.logs)
	}
	if cmd := c.HandleReport(runTraceReportCommand(t, next)); cmd != nil {
		t.Fatal("completed tracer kept rearming its listener")
	}
}

func TestTraceReportOverwriteConfirmationReturnsListener(t *testing.T) {
	c, api, store := reportTestComponent(t, &failingSubscriptionClient{newFakeClient()})
	store.hasData = true
	c.startTrace(0)
	if api.confirm == nil || c.items[0].tracer != nil {
		t.Fatal("overwrite did not wait for confirmation")
	}
	msg := runTraceReportCommand(t, api.confirm())
	c.HandleReport(msg)
	if store.cleared != 1 || len(api.logs) != 1 || !strings.Contains(api.logs[0], "subscription denied") {
		t.Fatalf("confirmation lost its listener: clears=%d logs=%v", store.cleared, api.logs)
	}
}

func TestTraceReportStopWithoutErrorTerminatesListener(t *testing.T) {
	client := newFakeClient()
	c, api, _ := reportTestComponent(t, client)
	cmd := c.forceStartTrace(0)
	select {
	case <-client.subCh:
	case <-time.After(2 * time.Second):
		t.Fatal("tracer did not subscribe")
	}
	c.stopTrace(0)
	msg := runTraceReportCommand(t, cmd)
	if msg.Err != nil || c.HandleReport(msg) != nil || len(api.logs) != 0 {
		t.Fatalf("clean stop produced a failure or rearmed listener: %+v logs=%v", msg, api.logs)
	}
}

func TestTraceReportCompletionDrainsPendingFailure(t *testing.T) {
	for i := 0; i < 50; i++ {
		tr := newTracer(TracerConfig{Key: "fatal"}, newFakeClient())
		tr.done = make(chan struct{})
		tr.reportErr(errors.New("fatal subscription error"))
		close(tr.done)
		msg := runTraceReportCommand(t, listenTraceReports("fatal", tr))
		if msg.Err == nil || msg.Err.Error() != "fatal subscription error" {
			t.Fatal("completion discarded a pending fatal report")
		}
	}
}

func TestTraceReportStaleOrDeletedInstanceIgnored(t *testing.T) {
	c, api := traceTestComponent()
	old := newTracer(TracerConfig{Key: "first"}, newFakeClient())
	current := newTracer(TracerConfig{Key: "first"}, newFakeClient())
	c.items[0].tracer = current
	for _, msg := range []ReportMsg{
		{Key: "first", Tracer: old, Err: errors.New("old failure")},
		{Key: "deleted", Tracer: current, Err: errors.New("deleted failure")},
	} {
		if c.HandleReport(msg) != nil || len(api.logs) != 0 {
			t.Fatal("stale report was logged or rearmed")
		}
	}
	c.items = nil
	if c.HandleReport(ReportMsg{Key: "first", Tracer: current, Err: errors.New("deleted failure")}) != nil || len(api.logs) != 0 {
		t.Fatal("deleted trace report was logged or rearmed")
	}
}

func TestTraceReportRearmsForOngoingStoreErrors(t *testing.T) {
	c, api := traceTestComponent()
	tr := newTracer(TracerConfig{Key: "first"}, newFakeClient())
	tr.done = make(chan struct{})
	c.items[0].tracer = tr
	cmd := listenTraceReports("first", tr)
	for _, text := range []string{"write failure one", "write failure two"} {
		tr.reportErr(errors.New(text))
		cmd = c.HandleReport(runTraceReportCommand(t, cmd))
		if cmd == nil || !strings.Contains(api.logs[len(api.logs)-1], text) {
			t.Fatalf("store error did not rearm listener: %v", api.logs)
		}
	}
	close(tr.done)
	if c.HandleReport(runTraceReportCommand(t, cmd)) != nil || len(api.logs) != 2 {
		t.Fatal("finished store-error listener did not terminate")
	}
}
