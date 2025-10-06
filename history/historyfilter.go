package history

import (
	"fmt"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/marang/emqutiti/ui"
)

const (
	idxFilterTopic = iota
	idxFilterPayload
	idxFilterStart
	idxFilterEnd
	idxFilterArchived
)

const dateFormatPlaceholder = "YYYY-MM-DDTHH:MM:SSZ or +HH:MM"

// historyFilterForm captures filter inputs for history searches.
type historyFilterForm struct {
	ui.Form
	topic    *ui.SuggestField
	payload  *ui.TextField
	start    *ui.TextField
	end      *ui.TextField
	archived *ui.CheckField
	errMsg   string
}

// Topic returns the topic field.
func (f *historyFilterForm) Topic() *ui.SuggestField { return f.topic }

// Payload returns the payload field.
func (f *historyFilterForm) Payload() *ui.TextField { return f.payload }

// Start returns the start time field.
func (f *historyFilterForm) Start() *ui.TextField { return f.start }

// End returns the end time field.
func (f *historyFilterForm) End() *ui.TextField { return f.end }

// Archived returns the archived checkbox field.
func (f *historyFilterForm) Archived() *ui.CheckField { return f.archived }

// newHistoryFilterForm builds a form with optional prefilled values.
// Start and end remain blank when zero, allowing searches across all time.
func newHistoryFilterForm(topics []string, topic, payload string, start, end time.Time, archived bool) historyFilterForm {
	sort.Strings(topics)
	tf := ui.NewSuggestField(topics, "topic")
	tf.SetValue(topic)

	pf := ui.NewTextField("", "text contains")
	pf.SetValue(payload)

	sf := ui.NewTextField("", fmt.Sprintf("Start (%s)", dateFormatPlaceholder), ui.WithRFC3339())
	if !start.IsZero() {
		sf.SetValue(start.Format(time.RFC3339))
	}

	ef := ui.NewTextField("", fmt.Sprintf("End (%s)", dateFormatPlaceholder), ui.WithRFC3339())
	if !end.IsZero() {
		ef.SetValue(end.Format(time.RFC3339))
	}

	af := ui.NewCheckField(archived)

	f := historyFilterForm{
		Form:     ui.Form{Fields: []ui.Field{tf, pf, sf, ef, af}},
		topic:    tf,
		payload:  pf,
		start:    sf,
		end:      ef,
		archived: af,
	}
	f.ApplyFocus()
	return f
}

// NewFilterForm builds a history filter form with optional prefilled values.
func NewFilterForm(topics []string, topic, payload string, start, end time.Time, archived bool) historyFilterForm {
	return newHistoryFilterForm(topics, topic, payload, start, end, archived)
}

// Update handles focus cycling and topic completion.
func (f historyFilterForm) Update(msg tea.Msg) (historyFilterForm, tea.Cmd) {
	var cmd tea.Cmd
	f.errMsg = ""
	switch m := msg.(type) {
	case tea.KeyMsg:
		if c, ok := f.Fields[f.Focus].(ui.KeyConsumer); ok && c.WantsKey(m) {
			cmd = f.Fields[f.Focus].Update(msg)
		} else {
			f.CycleFocus(m)
			if len(f.Fields) > 0 {
				cmd = f.Fields[f.Focus].Update(msg)
			}
		}
	case tea.MouseMsg:
		if m.Action == tea.MouseActionPress && m.Button == tea.MouseButtonLeft {
			if m.Y >= 1 && m.Y-1 < len(f.Fields) {
				f.Focus = m.Y - 1
			}
		}
		if len(f.Fields) > 0 {
			cmd = f.Fields[f.Focus].Update(msg)
		}
	}
	f.ApplyFocus()
	return f, cmd
}

// View renders the filter fields with labels.
func (f historyFilterForm) View() string {
	line := fmt.Sprintf("Topic: %s", f.topic.View())
	lines := []string{line}
	if sugg := f.topic.SuggestionsView(); sugg != "" {
		lines = append(lines, sugg)
	}
	lines = append(lines,
		"",
		fmt.Sprintf("Text:  %s", f.payload.View()),
		"",
		fmt.Sprintf("Start: %s", f.start.View()),
		"",
		fmt.Sprintf("End:   %s", f.end.View()),
		"",
		fmt.Sprintf("Archived: %s", f.archived.View()),
	)
	if f.errMsg != "" {
		lines = append(lines, "", ui.ErrorStyle.Render(f.errMsg))
	}
	return strings.Join(lines, "\n")
}

func (f historyFilterForm) Validate() (historyFilterForm, error) {
	startTime, err := ui.ParseRFC3339(f.start.Value())
	if err != nil {
		f.errMsg = fmt.Sprintf("Start %s", err.Error())
		return f, err
	}
	endTime, err := ui.ParseRFC3339(f.end.Value())
	if err != nil {
		f.errMsg = fmt.Sprintf("End %s", err.Error())
		return f, err
	}
	if !startTime.IsZero() {
		f.start.SetValue(startTime.Format(time.RFC3339))
	}
	if !endTime.IsZero() {
		f.end.SetValue(endTime.Format(time.RFC3339))
	}
	if !startTime.IsZero() && !endTime.IsZero() && startTime.After(endTime) {
		err := fmt.Errorf("End must be after start")
		f.errMsg = err.Error()
		return f, err
	}
	return f, nil
}

// query builds a history search string.
func (f historyFilterForm) query() string {
	var parts []string
	if v := f.topic.Value(); v != "" {
		parts = append(parts, "topic="+v)
	}
	if v := f.payload.Value(); v != "" {
		parts = append(parts, "payload="+v)
	}
	if v := f.start.Value(); v != "" {
		parts = append(parts, "start="+v)
	}
	if v := f.end.Value(); v != "" {
		parts = append(parts, "end="+v)
	}
	return strings.Join(parts, " ")
}
