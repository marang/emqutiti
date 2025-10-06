package connections

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/marang/emqutiti/ui"
)

// Form collects broker configuration fields.
type Form struct {
	ui.Form
	Index   int  // -1 for new
	fromEnv bool // current state of env loading
}

type fieldType int

const (
	ftText fieldType = iota
	ftPassword
	ftBool
	ftSelect
)

type fieldDef struct {
	key, label, placeholder string
	fieldType               fieldType
	options                 []string
}

var formFields = []fieldDef{
	{key: "Name", label: "Name", placeholder: "Name", fieldType: ftText},
	{key: "Schema", label: "Schema", placeholder: "Schema", fieldType: ftSelect, options: []string{"tcp", "ssl", "ws", "wss", "mqtt", "mqtts"}},
	{key: "Host", label: "Host", placeholder: "Host", fieldType: ftText},
	{key: "Port", label: "Port", placeholder: "Port", fieldType: ftText},
	{key: "ClientID", label: "Client ID", placeholder: "Client ID", fieldType: ftText},
	{key: "RandomIDSuffix", label: "Random ID suffix", placeholder: "Random ID suffix", fieldType: ftBool},
	{key: "Username", label: "Username", placeholder: "Username", fieldType: ftText},
	{key: "Password", label: "Password", fieldType: ftPassword},
	{key: "FromEnv", label: "Load from env", placeholder: "Values from env", fieldType: ftBool},
	{key: "SSL", label: "SSL/TLS", placeholder: "SSL/TLS", fieldType: ftBool},
	{key: "SkipTLSVerify", label: "Skip TLS verify", placeholder: "Skip TLS verify", fieldType: ftBool},
	{key: "CACertPath", label: "CA Cert Path", placeholder: "CA Cert Path", fieldType: ftText},
	{key: "ClientCertPath", label: "Client Cert Path", placeholder: "Client Cert Path", fieldType: ftText},
	{key: "ClientKeyPath", label: "Client Key Path", placeholder: "Client Key Path", fieldType: ftText},
	{key: "MQTTVersion", label: "MQTT Version", placeholder: "MQTT Version", fieldType: ftSelect, options: []string{"3", "4", "5"}},
	{key: "ConnectTimeout", label: "Connect Timeout (s)", placeholder: "Connect Timeout (s)", fieldType: ftText},
	{key: "KeepAlive", label: "Keep Alive (s)", placeholder: "Keep Alive (s)", fieldType: ftText},
	{key: "QoS", label: "QoS", placeholder: "QoS", fieldType: ftSelect, options: []string{"0", "1", "2"}},
	{key: "AutoReconnect", label: "Auto Reconnect", placeholder: "Auto Reconnect", fieldType: ftBool},
	{key: "ReconnectPeriod", label: "Reconnect Period (s)", placeholder: "Reconnect Period (s)", fieldType: ftText},
	{key: "CleanStart", label: "Clean Start", placeholder: "Clean Start", fieldType: ftBool},
	{key: "SessionExpiry", label: "Session Expiry (s)", placeholder: "Session Expiry (s)", fieldType: ftText},
	{key: "ReceiveMaximum", label: "Receive Maximum", placeholder: "Receive Maximum", fieldType: ftText},
	{key: "MaximumPacketSize", label: "Maximum Packet Size", placeholder: "Maximum Packet Size", fieldType: ftText},
	{key: "TopicAliasMaximum", label: "Topic Alias Maximum", placeholder: "Topic Alias Maximum", fieldType: ftText},
	{key: "RequestResponseInfo", label: "Request Response Info", placeholder: "Request Response Info", fieldType: ftBool},
	{key: "RequestProblemInfo", label: "Request Problem Info", placeholder: "Request Problem Info", fieldType: ftBool},
	{key: "LastWillEnabled", label: "Use Last Will", placeholder: "Use Last Will", fieldType: ftBool},
	{key: "LastWillTopic", label: "Last Will Topic", placeholder: "Last Will Topic", fieldType: ftText},
	{key: "LastWillQos", label: "Last Will QoS", placeholder: "Last Will QoS", fieldType: ftSelect, options: []string{"0", "1", "2"}},
	{key: "LastWillRetain", label: "Last Will Retain", placeholder: "Last Will Retain", fieldType: ftBool},
	{key: "LastWillPayload", label: "Last Will Payload", placeholder: "Last Will Payload", fieldType: ftText},
}

var fieldIndex = func() map[string]int {
	m := make(map[string]int, len(formFields))
	for i, fd := range formFields {
		m[fd.key] = i
	}
	return m
}()

// NewForm builds a form populated from the given profile.
// idx is -1 when creating a new profile.
func NewForm(p Profile, idx int) Form {
	if p.FromEnv {
		ApplyEnvVars(&p)
	}
	pwKey := ""
	if p.Name != "" && p.Username != "" {
		pwKey = fmt.Sprintf("keyring:emqutiti-%s/%s", p.Name, p.Username)
	}
	rv := reflect.ValueOf(p)
	fields := make([]ui.Field, len(formFields))
	for i, fd := range formFields {
		placeholder := fd.placeholder
		if fd.key == "Password" && pwKey != "" {
			placeholder = pwKey
		}
		fv := rv.FieldByName(fd.key)
		var strVal string
		var boolVal bool
		switch fv.Kind() {
		case reflect.String:
			strVal = fv.String()
		case reflect.Int:
			strVal = fmt.Sprintf("%d", fv.Int())
		case reflect.Bool:
			boolVal = fv.Bool()
			strVal = fmt.Sprintf("%v", boolVal)
		}
		switch fd.fieldType {
		case ftBool:
			fields[i] = ui.NewCheckField(boolVal)
		case ftSelect:
			sf, err := ui.NewSelectField(strVal, fd.options)
			if err != nil {
				sf = &ui.SelectField{}
			}
			fields[i] = sf
		case ftPassword:
			fields[i] = ui.NewTextField(strVal, placeholder, ui.WithMask())
		default:
			fields[i] = ui.NewTextField(strVal, placeholder)
		}
	}
	if p.FromEnv {
		idxName := fieldIndex["Name"]
		idxFromEnv := fieldIndex["FromEnv"]
		for i, fld := range fields {
			if i == idxName || i == idxFromEnv {
				continue
			}
			fld.SetReadOnly(true)
		}
	}
	cf := Form{Form: ui.Form{Fields: fields, Focus: 0}, Index: idx, fromEnv: p.FromEnv}
	cf.ApplyFocus()
	return cf
}

// Init sets up the text input blink command.
func (f Form) Init() tea.Cmd {
	return textinput.Blink
}

// Update handles keyboard and mouse events for the form.
func (f Form) Update(msg tea.Msg) (Form, tea.Cmd) {
	var cmds []tea.Cmd
	switch m := msg.(type) {
	case tea.KeyMsg:
		f.CycleFocus(m)
	case tea.MouseMsg:
		if m.Action == tea.MouseActionPress && m.Button == tea.MouseButtonLeft {
			if m.Y >= 1 && m.Y-1 < len(f.Fields) {
				f.Focus = m.Y - 1
			}
		}
	}
	f.ApplyFocus()
	if len(f.Fields) > 0 {
		if cmd := f.Fields[f.Focus].Update(msg); cmd != nil {
			cmds = append(cmds, cmd)
		}
	}
	idxFromEnv := fieldIndex["FromEnv"]
	if chk, ok := f.Fields[idxFromEnv].(*ui.CheckField); ok && chk.Bool() != f.fromEnv {
		p, err := f.Profile()
		if err != nil {
			chk.SetBool(f.fromEnv)
			cmds = append(cmds, func() tea.Msg { return StatusMessage(err.Error()) })
		} else {
			f = NewForm(p, f.Index)
			f.Focus = idxFromEnv
			f.fromEnv = chk.Bool()
		}
	}
	return f, tea.Batch(cmds...)
}

// View renders the form with labels in a left column and inputs on the right.
func (f Form) View() string {
	var rows []string
	maxLabel := 0
	for _, fd := range formFields {
		if w := lipgloss.Width(fd.label); w > maxLabel {
			maxLabel = w
		}
	}
	labelStyle := lipgloss.NewStyle().Width(maxLabel).Align(lipgloss.Right)
	for i, in := range f.Fields {
		label := formFields[i].label
		if i == f.Focus {
			label = ui.FocusedStyle.Render(label)
		}
		left := labelStyle.Render(label)
		row := lipgloss.JoinHorizontal(lipgloss.Top, left, ": ", in.View())
		rows = append(rows, row)
		if sf, ok := in.(*ui.SelectField); ok && f.IsFocused(i) {
			if opts := sf.OptionsView(); opts != "" {
				indent := lipgloss.NewStyle().MarginLeft(maxLabel + 2).Render(opts)
				rows = append(rows, indent)
			}
		}
	}
	idxFromEnv := fieldIndex["FromEnv"]
	idxName := fieldIndex["Name"]
	if chk, ok := f.Fields[idxFromEnv].(*ui.CheckField); ok && chk.Bool() {
		prefix := EnvPrefix(f.Fields[idxName].Value())
		rows = append(rows, ui.InfoStyle.Render("Values loaded from env vars: "+prefix+"<FIELD>"))
	}
	rows = append(rows, "", ui.InfoStyle.Render("[enter] save  [esc] cancel"))
	return strings.Join(rows, "\n")
}

// Profile builds a Profile struct from the form values.
// It returns a populated Profile and any validation errors encountered
// while parsing numeric or boolean fields.
func (f Form) Profile() (Profile, error) {
	p := Profile{}
	var errs []string
	rv := reflect.ValueOf(&p).Elem()
	for i, fd := range formFields {
		field := rv.FieldByName(fd.key)
		val := f.Fields[i].Value()
		switch field.Kind() {
		case reflect.String:
			field.SetString(val)
		case reflect.Int:
			if val == "" {
				field.SetInt(0)
				continue
			}
			iv, err := strconv.Atoi(val)
			if err != nil {
				errs = append(errs, fmt.Sprintf("%s: %v", fd.label, err))
				continue
			}
			field.SetInt(int64(iv))
		case reflect.Bool:
			bv, err := strconv.ParseBool(val)
			if err != nil {
				errs = append(errs, fmt.Sprintf("%s: %v", fd.label, err))
				continue
			}
			field.SetBool(bv)
		}
	}
	if len(errs) > 0 {
		return p, errors.New(strings.Join(errs, "; "))
	}
	return p, nil
}
