package traces

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/marang/emqutiti/ui"
)

type traceForm struct {
	ui.Form
	errMsg string
}

const (
	idxTraceKey = iota
	idxTraceProfile
	idxTraceTopics
	idxTraceStart
	idxTraceEnd
)

// newTraceForm builds a form for creating or editing a trace.
func newTraceForm(profiles []string, current string, topics []string) traceForm {
	keyField := ui.NewTextField("", "Key")
	profileField, err := ui.NewSelectField(current, profiles)
	if err != nil {
		profileField = &ui.SelectField{}
	}
	topicsField := ui.NewTextField(strings.Join(topics, ","), "Topics")
	startField := ui.NewTextField("", "Start (YYYY-MM-DDTHH:MM:SSZ)", ui.WithRFC3339())
	endField := ui.NewTextField("", "End (YYYY-MM-DDTHH:MM:SSZ)", ui.WithRFC3339())
	fields := []ui.Field{keyField, profileField, topicsField, startField, endField}
	tf := traceForm{Form: ui.Form{Fields: fields, Focus: 0}}
	if err != nil {
		tf.errMsg = err.Error()
	}
	tf.ApplyFocus()
	return tf
}

// Init implements tea.Model for the trace form.
func (f traceForm) Init() tea.Cmd { return textinput.Blink }

// Update handles user input and updates form fields.
func (f traceForm) Update(msg tea.Msg) (traceForm, tea.Cmd) {
	var cmd tea.Cmd
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
		cmd = f.Fields[f.Focus].Update(msg)
	}
	return f, cmd
}

func (f traceForm) Validate() (traceForm, error) {
	f.errMsg = ""
	startStr := strings.TrimSpace(f.Fields[idxTraceStart].Value())
	endStr := strings.TrimSpace(f.Fields[idxTraceEnd].Value())
	startTime, err := ui.ParseRFC3339(startStr)
	if err != nil {
		f.errMsg = fmt.Sprintf("start %s", err.Error())
		return f, err
	}
	endTime, err := ui.ParseRFC3339(endStr)
	if err != nil {
		f.errMsg = fmt.Sprintf("end %s", err.Error())
		return f, err
	}
	if !startTime.IsZero() {
		if tf, ok := f.Fields[idxTraceStart].(*ui.TextField); ok {
			tf.SetValue(startTime.Format(time.RFC3339))
		}
	}
	if !endTime.IsZero() {
		if tf, ok := f.Fields[idxTraceEnd].(*ui.TextField); ok {
			tf.SetValue(endTime.Format(time.RFC3339))
		}
	}
	if !startTime.IsZero() && !endTime.IsZero() && startTime.After(endTime) {
		err := fmt.Errorf("end must be after start")
		f.errMsg = err.Error()
		return f, err
	}
	return f, nil
}

// View renders the form interface.
func (f traceForm) View() string {
	labels := []string{"Key", "Profile", "Topics", "Start", "End"}
	var b strings.Builder
	for i, fld := range f.Fields {
		label := labels[i]
		if i == f.Focus {
			label = ui.FocusedStyle.Render(label)
		}
		b.WriteString(label + ": " + fld.View() + "\n")
		if sf, ok := fld.(*ui.SelectField); ok && f.IsFocused(i) {
			if opts := sf.OptionsView(); opts != "" {
				b.WriteString(opts + "\n")
			}
		}
	}
	if f.errMsg != "" {
		b.WriteString("\n" + ui.ErrorStyle.Render(f.errMsg))
	}
	b.WriteString("\n" + ui.InfoStyle.Render("[enter] save  [esc] cancel"))
	return b.String()
}

// Config returns the tracer configuration from the form values.
func (f traceForm) Config() TracerConfig {
	vals := make([]string, len(f.Fields))
	for i, fld := range f.Fields {
		vals[i] = fld.Value()
	}
	cfg := TracerConfig{}
	cfg.Key = strings.TrimSpace(vals[idxTraceKey])
	cfg.Profile = vals[idxTraceProfile]
	if t := strings.TrimSpace(vals[idxTraceTopics]); t != "" {
		parts := strings.Split(t, ",")
		for i := range parts {
			parts[i] = strings.TrimSpace(parts[i])
		}
		cfg.Topics = parts
	}
	if s := strings.TrimSpace(vals[idxTraceStart]); s != "" {
		if tm, err := time.Parse(time.RFC3339, s); err == nil {
			cfg.Start = tm
		}
	}
	if e := strings.TrimSpace(vals[idxTraceEnd]); e != "" {
		if tm, err := time.Parse(time.RFC3339, e); err == nil {
			cfg.End = tm
		}
	}
	return cfg
}
