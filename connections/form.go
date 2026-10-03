package connections

import (
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/marang/emqutiti/constants"
	"github.com/marang/emqutiti/ui"
)

// Form collects broker configuration fields.
type Form struct {
	ui.Form
	Index       int // -1 for new
	fromEnv     bool
	width       int
	height      int
	offset      int
	lastFocus   int
	revealFocus bool
	rows        []formRow
	validation  map[int]string
}

type formRow struct {
	text          string
	field, option int // -1 for non-field rows or non-option rows
}

// FormAction identifies a command in the form's fixed footer.
type FormAction int

const (
	NoFormAction FormAction = iota
	SaveForm
	CancelForm
)

const formActions = "[enter] save  [esc] cancel"

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
	{key: "FromEnv", label: "Load from env", placeholder: "Values from env", fieldType: ftBool},
	{key: "Name", label: "Name", placeholder: "Name", fieldType: ftText},
	{key: "Schema", label: "Schema", placeholder: "Schema", fieldType: ftSelect, options: []string{"tcp", "ssl", "ws", "wss", "mqtt", "mqtts"}},
	{key: "Host", label: "Host", placeholder: "Host", fieldType: ftText},
	{key: "Port", label: "Port", placeholder: "Port", fieldType: ftText},
	{key: "ClientID", label: "Client ID", placeholder: "Client ID", fieldType: ftText},
	{key: "RandomIDSuffix", label: "Random ID suffix", placeholder: "Random ID suffix", fieldType: ftBool},
	{key: "Username", label: "Username", placeholder: "Username", fieldType: ftText},
	{key: "Password", label: "Password", fieldType: ftPassword},
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

func envVarNames(prefix string) []string {
	rt := reflect.TypeOf(Profile{})
	var names []string
	for i := 0; i < rt.NumField(); i++ {
		tag := rt.Field(i).Tag.Get("env")
		if tag == "" {
			continue
		}
		names = append(names, prefix+strings.ToUpper(tag))
	}
	sort.Strings(names)
	return names
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
	cf := Form{Form: ui.Form{Fields: fields, Focus: 0}, Index: idx, fromEnv: p.FromEnv, lastFocus: -1}
	cf.ApplyFocus()
	return cf
}

// Init sets up the text input blink command.
func (f Form) Init() tea.Cmd {
	return textinput.Blink
}

// SetSize sets the available content area, excluding the surrounding box.
func (f *Form) SetSize(width, height int) {
	width, height = max(1, width), max(2, height)
	if f.width != width || f.height != height {
		f.width, f.height = width, height
		f.revealFocus = true
	}
}

// CycleFocus uses the shared tab order and reveals the newly focused field.
func (f *Form) CycleFocus(msg tea.KeyMsg) {
	f.Form.CycleFocus(msg)
	f.revealFocus = true
}

// ActionAt reports a footer command at content-local coordinates.
func (f *Form) ActionAt(x, y int) FormAction {
	footerY := len(f.rows) + 1
	if f.height > 0 {
		footerY = f.height - 1
	}
	if y != footerY || x < 0 || (f.width > 0 && x >= f.width) {
		return NoFormAction
	}
	switch {
	case x < len("[enter] save"):
		return SaveForm
	case x >= len("[enter] save  ") && x < len(formActions):
		return CancelForm
	default:
		return NoFormAction
	}
}

// Update handles keyboard and mouse events for the form.
func (f Form) Update(msg tea.Msg) (Form, tea.Cmd) {
	var cmds []tea.Cmd
	switch m := msg.(type) {
	case tea.KeyMsg:
		switch m.String() {
		case constants.KeyCtrlUp, constants.KeyCtrlK:
			f.scroll(-1)
			return f, nil
		case constants.KeyCtrlDown, constants.KeyCtrlJ:
			f.scroll(1)
			return f, nil
		}
		f.CycleFocus(m)
	case tea.MouseMsg:
		if m.X < 0 || m.Y < 0 || (f.width > 0 && m.X >= f.width) || (f.height > 0 && m.Y >= f.height-1) {
			return f, nil
		}
		switch m.Button {
		case tea.MouseButtonWheelUp:
			f.scroll(-1)
			return f, nil
		case tea.MouseButtonWheelDown:
			f.scroll(1)
			return f, nil
		}
		if m.Action != tea.MouseActionPress || m.Button != tea.MouseButtonLeft {
			return f, nil
		}
		f.View()
		row := m.Y + f.offset
		if row >= len(f.rows) || f.rows[row].field < 0 {
			return f, nil
		}
		hit := f.rows[row]
		f.Focus = hit.field
		f.revealFocus = true
		f.ApplyFocus()
		if hit.option >= 0 {
			f.Fields[hit.field].(*ui.SelectField).Index = hit.option
			return f, nil
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
		p, err := f.Validate()
		if err != nil {
			chk.SetBool(f.fromEnv)
			cmds = append(cmds, func() tea.Msg { return StatusMessage(err.Error()) })
		} else {
			width, height := f.width, f.height
			f = NewForm(p, f.Index)
			if width > 0 && height > 0 {
				f.SetSize(width, height)
			}
			f.Focus = idxFromEnv
			f.fromEnv = chk.Bool()
		}
	}
	return f, tea.Batch(cmds...)
}

// View renders the scrollable form with a fixed save/cancel footer.
func (f *Form) View() string {
	f.renderRows()
	if f.height == 0 {
		lines := make([]string, 0, len(f.rows)+2)
		for _, row := range f.rows {
			lines = append(lines, row.text)
		}
		return strings.Join(append(lines, "", ui.InfoStyle.Render(formActions)), "\n")
	}
	bodyHeight := f.height - 1
	if f.revealFocus || f.lastFocus != f.Focus {
		f.ensureFocusedVisible(bodyHeight)
	}
	f.offset = min(max(0, f.offset), max(0, len(f.rows)-bodyHeight))
	f.lastFocus, f.revealFocus = f.Focus, false
	lines := make([]string, bodyHeight, f.height)
	for y := range lines {
		if row := f.offset + y; row < len(f.rows) {
			lines[y] = f.rows[row].text
		}
	}
	footer := ansi.Truncate(ui.InfoStyle.PaddingLeft(0).Render(formActions), f.width, "")
	return strings.Join(append(lines, footer), "\n")
}

func (f *Form) renderRows() {
	f.rows = f.rows[:0]
	maxLabel := 0
	for _, fd := range formFields {
		if w := lipgloss.Width(fd.label); w > maxLabel {
			maxLabel = w
		}
	}
	idxFromEnv := fieldIndex["FromEnv"]
	idxName := fieldIndex["Name"]
	labelStyle := lipgloss.NewStyle().Width(maxLabel).Align(lipgloss.Right)
	inputWidth := f.width - maxLabel - 2
	stacked := f.width > 0 && inputWidth < 8
	if stacked {
		inputWidth = f.width
	}
	for i, in := range f.Fields {
		label := formFields[i].label
		if i == f.Focus {
			label = ui.FocusedStyle.Render(label)
		}
		if i == idxFromEnv {
			f.appendRows("", -1, -1)
		}
		if tf, ok := in.(*ui.TextField); ok && f.width > 0 {
			width := max(1, inputWidth-lipgloss.Width(tf.Prompt)-1)
			if tf.Width != width {
				tf.Width = width
				tf.SetCursor(tf.Position())
			}
		}
		indent := maxLabel + 2
		if stacked {
			f.appendRows(label+":", i, -1)
			f.appendRows(in.View(), i, -1)
			indent = 0
		} else {
			row := lipgloss.JoinHorizontal(lipgloss.Top, labelStyle.Render(label), ": ", in.View())
			f.appendRows(row, i, -1)
		}
		if sf, ok := in.(*ui.SelectField); ok && f.IsFocused(i) {
			if opts := sf.OptionsView(); opts != "" {
				for option, line := range strings.Split(opts, "\n") {
					f.appendRows(strings.Repeat(" ", indent)+line, i, option)
				}
			}
		}
		if message := f.validation[i]; message != "" {
			f.appendRows(ui.FormError.Render(message), i, -1)
		}
		if i == idxFromEnv {
			name := strings.TrimSpace(f.Fields[idxName].Value())
			prefix := "EMQUTITI_<NAME>_"
			if name != "" {
				prefix = EnvPrefix(name)
			}
			hintText := []string{
				"Toggle to load values from env vars: " + prefix + "<FIELD>.",
				"Turn off to edit manually.",
			}
			for _, line := range hintText {
				f.appendRows(ui.InfoStyle.Render(line), -1, -1)
			}
			f.appendRows("", -1, -1)
		}
	}
	if chk, ok := f.Fields[idxFromEnv].(*ui.CheckField); ok && chk.Bool() {
		prefix := EnvPrefix(f.Fields[idxName].Value())
		vars := envVarNames(prefix)
		if len(vars) == 0 {
			vars = []string{prefix + "<FIELD>"}
		}
		f.appendRows(ui.InfoStyle.Render("Values loaded from env vars: "+strings.Join(vars, ", ")), -1, -1)
	}
}

func (f *Form) appendRows(text string, field, option int) {
	if f.width > 0 {
		text = ansi.Wrap(text, f.width, "")
	}
	for _, line := range strings.Split(text, "\n") {
		f.rows = append(f.rows, formRow{text: line, field: field, option: option})
	}
}

func (f *Form) ensureFocusedVisible(height int) {
	first, last := -1, -1
	for i, row := range f.rows {
		if row.field == f.Focus {
			if first < 0 {
				first = i
			}
			last = i
		}
	}
	if first < 0 {
		return
	}
	if first < f.offset || last-first+1 > height {
		f.offset = first
	} else if last >= f.offset+height {
		f.offset = last - height + 1
	}
}

func (f *Form) scroll(delta int) {
	f.View()
	if f.height > 0 {
		f.offset = min(max(0, f.offset+delta), max(0, len(f.rows)-(f.height-1)))
	}
}

// Validate preserves input and reveals the first invalid numeric field.
func (f *Form) Validate() (Profile, error) {
	p, err := f.Profile()
	f.validation = make(map[int]string)
	if err == nil {
		return p, nil
	}
	rt := reflect.TypeOf(Profile{})
	for i, fd := range formFields {
		field, ok := rt.FieldByName(fd.key)
		if !ok || field.Type.Kind() != reflect.Int || f.Fields[i].Value() == "" {
			continue
		}
		if _, parseErr := strconv.Atoi(f.Fields[i].Value()); parseErr != nil {
			if len(f.validation) == 0 {
				f.Focus = i
			}
			f.validation[i] = "Enter a whole number."
		}
	}
	f.revealFocus = true
	f.ApplyFocus()
	return p, err
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
