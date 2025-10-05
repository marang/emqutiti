package emqutiti

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	cfg "github.com/marang/emqutiti/cmd"
	connections "github.com/marang/emqutiti/connections"
	"github.com/marang/emqutiti/help"
	"github.com/marang/emqutiti/history"
	"github.com/marang/emqutiti/importer"
	"github.com/marang/emqutiti/importer/steps"
	"github.com/marang/emqutiti/proxy"
	"github.com/marang/emqutiti/traces"
)

type stubTraceStore struct {
	traces.Store
	hasData        bool
	hasDataErr     error
	addCfg         traces.TracerConfig
	addErr         error
	checkedProfile string
	checkedKey     string
}

func (s *stubTraceStore) LoadTraces() map[string]traces.TracerConfig      { return nil }
func (s *stubTraceStore) SaveTraces(map[string]traces.TracerConfig) error { return nil }
func (s *stubTraceStore) AddTrace(cfg traces.TracerConfig) error {
	s.addCfg = cfg
	return s.addErr
}
func (s *stubTraceStore) RemoveTrace(string) error                                { return nil }
func (s *stubTraceStore) Messages(string, string) ([]traces.TracerMessage, error) { return nil, nil }
func (s *stubTraceStore) HasData(profile, key string) (bool, error) {
	s.checkedProfile = profile
	s.checkedKey = key
	return s.hasData, s.hasDataErr
}
func (s *stubTraceStore) ClearData(string, string) error { return nil }
func (s *stubTraceStore) LoadCounts(string, string, []string) (map[string]int, error) {
	return nil, nil
}

type stubProgram struct{ run func() (tea.Model, error) }

func (s stubProgram) Run() (tea.Model, error) { return s.run() }

type stubMQTTClient struct{ disconnected bool }

func (s *stubMQTTClient) Publish(topic string, qos byte, retained bool, payload interface{}) error {
	return nil
}

func (s *stubMQTTClient) Disconnect() { s.disconnected = true }

type stubHistoryStore struct{ closed bool }

func (s *stubHistoryStore) Append(history.Message) error { return nil }
func (s *stubHistoryStore) Search(bool, []string, time.Time, time.Time, string) []history.Message {
	return nil
}
func (s *stubHistoryStore) Delete(string) error  { return nil }
func (s *stubHistoryStore) Archive(string) error { return nil }
func (s *stubHistoryStore) Count(bool) int       { return 0 }
func (s *stubHistoryStore) Close() error {
	s.closed = true
	return nil
}

func writeTempConfig(t *testing.T, contents string) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "config-*.toml")
	if err != nil {
		t.Fatalf("CreateTemp: %v", err)
	}
	if _, err := f.WriteString(contents); err != nil {
		t.Fatalf("WriteString: %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	return f.Name()
}

func TestRunTrace(t *testing.T) {
	st := &stubTraceStore{}
	called := false
	d := &appDeps{
		traceKey:    "k",
		traceTopics: "a,b",
		profileName: "p",
		traceStore:  st,
		traceRun: func(ctx context.Context, k, tp, pf, stt, end string) error {
			called = true
			if k != "k" || tp != "a,b" || pf != "p" {
				t.Fatalf("unexpected args %v %v %v", k, tp, pf)
			}
			return nil
		},
	}
	if err := runTrace(d); err != nil {
		t.Fatalf("runTrace error: %v", err)
	}
	if !called {
		t.Fatalf("traceRun not called")
	}
	if st.addCfg.Key != "k" || st.addCfg.Profile != "p" {
		t.Fatalf("unexpected cfg: %#v", st.addCfg)
	}
}

func TestRunTracePromptsForProfile(t *testing.T) {
	st := &stubTraceStore{}
	called := false
	d := &appDeps{
		traceKey:    "k",
		traceTopics: "topic",
		traceStore:  st,
		selectProfile: func(r io.Reader, w io.Writer, file string) (string, error) {
			if file != "cfg" {
				t.Fatalf("unexpected file %q", file)
			}
			if r == nil || w == nil {
				t.Fatalf("nil reader/writer provided")
			}
			return "chosen", nil
		},
		profileIn:  strings.NewReader(""),
		profileOut: io.Discard,
		configFile: "cfg",
		traceRun: func(ctx context.Context, key, topics, profile, start, end string) error {
			called = true
			if profile != "chosen" {
				t.Fatalf("expected profile 'chosen', got %q", profile)
			}
			return nil
		},
	}
	if err := runTrace(d); err != nil {
		t.Fatalf("runTrace error: %v", err)
	}
	if !called {
		t.Fatalf("traceRun not called")
	}
	if st.addCfg.Profile != "chosen" {
		t.Fatalf("expected stored profile 'chosen', got %q", st.addCfg.Profile)
	}
	if st.checkedProfile != "chosen" || st.checkedKey != "k" {
		t.Fatalf("unexpected HasData args %q %q", st.checkedProfile, st.checkedKey)
	}
}

func TestRunTraceEndPast(t *testing.T) {
	past := time.Now().Add(-time.Hour).Format(time.RFC3339)
	d := &appDeps{
		traceKey:    "k",
		traceTopics: "t",
		profileName: "p",
		traceEnd:    past,
		traceStore:  &stubTraceStore{},
		traceRun:    func(context.Context, string, string, string, string, string) error { return nil },
	}
	if err := runTrace(d); err == nil {
		t.Fatalf("expected error for past end time")
	}
}

func TestRunTraceTimeout(t *testing.T) {
	st := &stubTraceStore{}
	d := &appDeps{
		traceKey:    "k",
		traceTopics: "t",
		profileName: "p",
		traceStore:  st,
		traceRun: func(ctx context.Context, k, tp, pf, stt, end string) error {
			<-ctx.Done()
			return ctx.Err()
		},
		timeout: 10 * time.Millisecond,
	}
	err := runTrace(d)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected deadline exceeded, got %v", err)
	}
}

func TestMainDispatchImportFlags(t *testing.T) {
	orig := initProxy
	initProxy = func() (string, *proxy.Proxy) { return "", nil }
	called := false
	d := newAppDeps()
	d.runners["import"] = func(ad *appDeps) error {
		called = true
		if ad.importFile != "f" || ad.profileName != "pr" {
			t.Fatalf("unexpected params %v %v", ad.importFile, ad.profileName)
		}
		return nil
	}
	d.runners["trace"] = func(*appDeps) error { t.Fatalf("runTrace called"); return nil }
	d.runners["ui"] = func(*appDeps) error { t.Fatalf("runUI called"); return nil }
	runMain(d, cfg.AppConfig{ImportFile: "f", ProfileName: "pr"})
	if !called {
		t.Fatalf("runImport not called")
	}
	initProxy = orig
}

func TestMainDispatchTrace(t *testing.T) {
	orig := initProxy
	initProxy = func() (string, *proxy.Proxy) { return "", nil }
	called := false
	d := newAppDeps()
	d.runners["trace"] = func(ad *appDeps) error {
		called = true
		if ad.traceKey != "k" || ad.traceTopics != "t" {
			t.Fatalf("unexpected params %v %v", ad.traceKey, ad.traceTopics)
		}
		return nil
	}
	d.runners["import"] = func(*appDeps) error { t.Fatalf("runImport called"); return nil }
	d.runners["ui"] = func(*appDeps) error { t.Fatalf("runUI called"); return nil }
	runMain(d, cfg.AppConfig{TraceKey: "k", TraceTopics: "t"})
	if !called {
		t.Fatalf("runTrace not called")
	}
	initProxy = orig
}

func TestMainDispatchUI(t *testing.T) {
	orig := initProxy
	initProxy = func() (string, *proxy.Proxy) { return "", nil }
	called := false
	d := newAppDeps()
	d.runners["ui"] = func(*appDeps) error { called = true; return nil }
	d.runners["import"] = func(*appDeps) error { t.Fatalf("runImport called"); return nil }
	d.runners["trace"] = func(*appDeps) error { t.Fatalf("runTrace called"); return nil }
	runMain(d, cfg.AppConfig{})
	if !called {
		t.Fatalf("runUI not called")
	}
	initProxy = orig
}

func TestRunImport(t *testing.T) {
	t.Setenv("EMQUTITI_DEFAULT_PASSWORD", "pw")

	client := &stubMQTTClient{}
	d := &appDeps{
		importFile:  "file",
		profileName: "pr",
		loadProfile: func(name, _ string) (*connections.Profile, error) {
			if name != "pr" {
				t.Fatalf("unexpected profile %s", name)
			}
			return &connections.Profile{}, nil
		},
		newMQTTClient: func(p connections.Profile, fn statusFunc) (mqttClient, error) {
			if p.Password != "pw" {
				t.Fatalf("expected password override, got %s", p.Password)
			}
			return client, nil
		},
		newImporter: func(cl steps.Publisher, path string) *importer.Model {
			if cl != client || path != "file" {
				t.Fatalf("unexpected importer args")
			}
			return nil
		},
		newProgram: func(m tea.Model, opts ...tea.ProgramOption) program {
			return stubProgram{run: func() (tea.Model, error) { return nil, nil }}
		},
	}
	if err := runImport(d); err != nil {
		t.Fatalf("runImport error: %v", err)
	}
	if !client.disconnected {
		t.Fatalf("client not disconnected")
	}
}

func TestListProfiles(t *testing.T) {
	cfgPath := writeTempConfig(t, `[[profiles]]
name = "local"
schema = "mqtt"
host = "localhost"
port = 1883

[[profiles]]
name = "remote"
schema = "ssl"
host = "example.com"
port = 8883
`)
	var buf bytes.Buffer
	if err := listProfiles(&buf, cfgPath); err != nil {
		t.Fatalf("listProfiles returned error: %v", err)
	}
	want := "local\tmqtt://localhost:1883\nremote\tssl://example.com:8883\n"
	if got := buf.String(); got != want {
		t.Fatalf("unexpected output:\n got %q\nwant %q", got, want)
	}
}

func TestListProfilesEmpty(t *testing.T) {
	cfgPath := writeTempConfig(t, "")
	var buf bytes.Buffer
	if err := listProfiles(&buf, cfgPath); err != nil {
		t.Fatalf("listProfiles returned error: %v", err)
	}
	want := "No profiles configured.\n"
	if got := buf.String(); got != want {
		t.Fatalf("unexpected output:\n got %q\nwant %q", got, want)
	}
}

func TestListProfilesMissingFile(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing.toml")
	if err := listProfiles(io.Discard, missing); err == nil {
		t.Fatalf("expected error when config file is missing")
	}
}

func TestRunUI(t *testing.T) {
	st := &stubHistoryStore{}
	d := &appDeps{
		initialModel: func(*connections.Connections) (*model, error) {
			return &model{help: &help.Component{}}, nil
		},
		newProgram: func(m tea.Model, opts ...tea.ProgramOption) program {
			return stubProgram{run: func() (tea.Model, error) {
				hc := history.NewComponent(nil, st)
				return &model{history: hc}, nil
			}}
		},
	}
	if err := runUI(d); err != nil {
		t.Fatalf("runUI error: %v", err)
	}
	if !st.closed {
		t.Fatalf("store not closed")
	}
}

func TestPromptProfileSelectionSingle(t *testing.T) {
	cfgPath := writeTempConfig(t, `[[profiles]]
name = "local"
schema = "mqtt"
host = "localhost"
port = 1883
`)
	var out bytes.Buffer
	name, err := promptProfileSelection(strings.NewReader("\n"), &out, cfgPath)
	if err != nil {
		t.Fatalf("promptProfileSelection error: %v", err)
	}
	if name != "local" {
		t.Fatalf("expected local, got %q", name)
	}
	if !strings.Contains(out.String(), "Using profile \"local\"") {
		t.Fatalf("expected confirmation in output, got %q", out.String())
	}
}

func TestPromptProfileSelectionMultiple(t *testing.T) {
	cfgPath := writeTempConfig(t, `default_profile = "remote"
[[profiles]]
name = "local"
schema = "mqtt"
host = "localhost"
port = 1883

[[profiles]]
name = "remote"
schema = "ssl"
host = "example.com"
port = 8883
`)
	input := strings.NewReader("2\n")
	var out bytes.Buffer
	name, err := promptProfileSelection(input, &out, cfgPath)
	if err != nil {
		t.Fatalf("promptProfileSelection error: %v", err)
	}
	if name != "remote" {
		t.Fatalf("expected remote, got %q", name)
	}
	if !strings.Contains(out.String(), "Select a connection profile:") {
		t.Fatalf("expected prompt output, got %q", out.String())
	}
}
