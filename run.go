package emqutiti

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	connections "github.com/marang/emqutiti/connections"
	history "github.com/marang/emqutiti/history"

	tea "github.com/charmbracelet/bubbletea"

	cfg "github.com/marang/emqutiti/cmd"
	"github.com/marang/emqutiti/constants"
	"github.com/marang/emqutiti/importer"
	"github.com/marang/emqutiti/importer/steps"
	"github.com/marang/emqutiti/traces"
)

type importerTeaModel struct{ *importer.Model }

func (m importerTeaModel) Init() tea.Cmd { return m.Model.Init() }

func (m importerTeaModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmd := m.Model.Update(msg)
	return m, cmd
}

func (m importerTeaModel) View() string { return m.Model.View() }

var (
	importFile  string
	profileName string
	traceKey    string
	traceTopics string
	traceStart  string
	traceEnd    string
)

type program interface{ Run() (tea.Model, error) }

type mqttClient interface {
	steps.Publisher
	Disconnect()
}

type ModeRunner func(*appDeps) error

type appDeps struct {
	importFile  string
	profileName string
	traceKey    string
	traceTopics string
	traceStart  string
	traceEnd    string

	traceStore traces.Store
	traceRun   func(context.Context, string, string, string, string, string) error

	loadProfile   func(string, string) (*connections.Profile, error)
	newMQTTClient func(connections.Profile, statusFunc) (mqttClient, error)
	newImporter   func(steps.Publisher, string) *importer.Model
	initialModel  func(*connections.Connections) (*model, error)
	newProgram    func(tea.Model, ...tea.ProgramOption) program
	selectProfile func(io.Reader, io.Writer, string) (string, error)
	profileIn     io.Reader
	profileOut    io.Writer
	configFile    string

	runners map[string]ModeRunner

	proxyAddr string
	timeout   time.Duration
}

func newAppDeps() *appDeps {
	d := &appDeps{
		traceStore:    traces.FileStore{},
		traceRun:      traces.Run,
		loadProfile:   connections.LoadProfile,
		newMQTTClient: func(p connections.Profile, fn statusFunc) (mqttClient, error) { return NewMQTTClient(p, fn) },
		newImporter:   importer.New,
		initialModel:  initialModel,
		newProgram: func(m tea.Model, opts ...tea.ProgramOption) program {
			return tea.NewProgram(m, opts...)
		},
		selectProfile: promptProfileSelection,
		profileIn:     os.Stdin,
		profileOut:    os.Stdout,
	}
	d.runners = map[string]ModeRunner{
		"trace":  runTrace,
		"import": runImport,
		"ui":     runUI,
	}
	return d
}

// Main sets up dependencies and launches the UI or other modes based on cfg.
func Main(c cfg.AppConfig) {
	if c.ListProfiles {
		if err := listProfiles(os.Stdout, ""); err != nil {
			log.Fatalf("Error listing profiles: %v", err)
		}
		return
	}
	d := newAppDeps()
	runMain(d, c)
}

func runMain(d *appDeps, c cfg.AppConfig) {
	importFile = c.ImportFile
	profileName = c.ProfileName
	traceKey = c.TraceKey
	traceTopics = c.TraceTopics
	traceStart = c.TraceStart
	traceEnd = c.TraceEnd

	d.importFile = c.ImportFile
	d.profileName = c.ProfileName
	d.traceKey = c.TraceKey
	d.traceTopics = c.TraceTopics
	d.traceStart = c.TraceStart
	d.traceEnd = c.TraceEnd
	d.timeout = c.Timeout

	addr, _ := initProxy()
	history.SetProxyAddr(addr)
	traces.SetProxyAddr(addr)
	d.proxyAddr = addr

	mode := "ui"
	if d.traceKey != "" {
		mode = "trace"
	} else if d.importFile != "" {
		mode = "import"
	}

	if runner, ok := d.runners[mode]; ok {
		if err := runner(d); err != nil {
			if mode == "ui" {
				log.Fatalf("Error running program: %v", err)
			}
			log.Println(err)
		}
	}
}

func listProfiles(w io.Writer, file string) error {
	cfg, err := connections.LoadConfig(file)
	if err != nil {
		return err
	}
	if len(cfg.Profiles) == 0 {
		_, err := fmt.Fprintln(w, "No profiles configured.")
		return err
	}
	for _, p := range cfg.Profiles {
		if _, err := fmt.Fprintf(w, "%s\t%s\n", p.Name, p.BrokerURL()); err != nil {
			return err
		}
	}
	return nil
}

func promptProfileSelection(r io.Reader, w io.Writer, file string) (string, error) {
	cfg, err := connections.LoadConfig(file)
	if err != nil {
		return "", err
	}
	if len(cfg.Profiles) == 0 {
		return "", fmt.Errorf("no connection profiles configured")
	}
	if len(cfg.Profiles) == 1 {
		if _, err := fmt.Fprintf(w, "Using profile %q (%s)\n", cfg.Profiles[0].Name, cfg.Profiles[0].BrokerURL()); err != nil {
			return "", err
		}
		return cfg.Profiles[0].Name, nil
	}
	defIdx := -1
	if cfg.DefaultProfile != "" {
		for i := range cfg.Profiles {
			if cfg.Profiles[i].Name == cfg.DefaultProfile {
				defIdx = i
				break
			}
		}
	}
	fmt.Fprintln(w, "Select a connection profile:")
	for i, p := range cfg.Profiles {
		marker := " "
		if i == defIdx {
			marker = "*"
		}
		if _, err := fmt.Fprintf(w, "  %s %d) %s\t%s\n", marker, i+1, p.Name, p.BrokerURL()); err != nil {
			return "", err
		}
	}
	if defIdx >= 0 {
		fmt.Fprintf(w, "Press Enter to use the default (%s).\n", cfg.Profiles[defIdx].Name)
	} else {
		fmt.Fprintf(w, "Press Enter to use the first profile (%s).\n", cfg.Profiles[0].Name)
	}
	fmt.Fprint(w, "> ")
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		choice := strings.TrimSpace(scanner.Text())
		if choice == "" {
			if defIdx >= 0 {
				return cfg.Profiles[defIdx].Name, nil
			}
			return cfg.Profiles[0].Name, nil
		}
		if idx, err := strconv.Atoi(choice); err == nil {
			idx--
			if idx >= 0 && idx < len(cfg.Profiles) {
				return cfg.Profiles[idx].Name, nil
			}
		} else {
			for _, p := range cfg.Profiles {
				if p.Name == choice {
					return p.Name, nil
				}
			}
		}
		fmt.Fprintln(w, "Invalid selection. Enter a number or profile name:")
		fmt.Fprint(w, "> ")
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	return "", fmt.Errorf("profile selection aborted")
}

func runTrace(d *appDeps) error {
	ctx := context.Background()
	if d.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, d.timeout)
		defer cancel()
	}
	if d.profileName == "" {
		if d.selectProfile == nil {
			return fmt.Errorf("no profile selector configured")
		}
		name, err := d.selectProfile(d.profileIn, d.profileOut, d.configFile)
		if err != nil {
			return err
		}
		if name == "" {
			return fmt.Errorf("no connection profile selected")
		}
		d.profileName = name
	}
	tlist := strings.Split(d.traceTopics, ",")
	for i := range tlist {
		tlist[i] = strings.TrimSpace(tlist[i])
	}
	var start, end time.Time
	var err error
	if d.traceStart != "" {
		start, err = time.Parse(time.RFC3339, d.traceStart)
		if err != nil {
			return fmt.Errorf("invalid trace start time: %w", err)
		}
	}
	if d.traceEnd != "" {
		end, err = time.Parse(time.RFC3339, d.traceEnd)
		if err != nil {
			return fmt.Errorf("invalid trace end time: %w", err)
		}
		if end.Before(time.Now()) {
			return fmt.Errorf("trace end time already passed")
		}
	}
	exists, err := d.traceStore.HasData(d.profileName, d.traceKey)
	if err != nil {
		return fmt.Errorf("trace data check failed: %w", err)
	}
	if exists {
		return fmt.Errorf("trace key already exists")
	}
	cfg := traces.TracerConfig{
		Profile: d.profileName,
		Topics:  tlist,
		Start:   start,
		End:     end,
		Key:     d.traceKey,
	}
	if err := d.traceStore.AddTrace(cfg); err != nil {
		return err
	}
	return d.traceRun(ctx, d.traceKey, d.traceTopics, d.profileName, d.traceStart, d.traceEnd)
}

// runImport launches the interactive import wizard using the provided file
// path and profile name.
func runImport(d *appDeps) error {
	p, err := d.loadProfile(d.profileName, "")
	if err != nil {
		return fmt.Errorf("error loading profile: %w", err)
	}
	connections.ApplyDefaultPassword(p)

	client, err := d.newMQTTClient(*p, nil)
	if err != nil {
		return fmt.Errorf("connect error: %w", err)
	}
	defer client.Disconnect()

	w := d.newImporter(client, d.importFile)
	prog := d.newProgram(importerTeaModel{w}, tea.WithAltScreen())
	if _, err := prog.Run(); err != nil {
		return fmt.Errorf("import error: %w", err)
	}
	return nil
}

func runUI(d *appDeps) error {
	initial, err := d.initialModel(nil)
	if err != nil {
		log.Printf("Warning: %v", err)
	}
	log.SetFlags(0)
	log.SetOutput(initial.logs)
	stop := startProxyStatusLogger(d.proxyAddr)
	defer stop()
	_ = initial.SetMode(constants.ModeConnections)
	p := d.newProgram(initial, tea.WithMouseAllMotion(), tea.WithAltScreen())
	finalModel, err := p.Run()
	if err != nil {
		return err
	}
	if m, ok := finalModel.(*model); ok {
		if st := m.history.Store(); st != nil {
			st.Close()
		}
	}
	return nil
}
