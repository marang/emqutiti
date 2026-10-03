package history

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/marang/emqutiti/constants"
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
	topic     *ui.SuggestField
	payload   *ui.TextField
	start     *ui.TextField
	end       *ui.TextField
	archived  *ui.CheckField
	errMsg    string
	width     int
	height    int
	viewport  viewport.Model
	lastFocus int
}

type filterRow struct {
	text       string
	field      int
	suggestion string
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
		viewport: viewport.New(0, 0),
	}
	f.ApplyFocus()
	return f
}

// SetSize fits the filter to the full terminal dimensions, including its box.
func (f *historyFilterForm) SetSize(width, height int) {
	changed := f.width != width || f.height != height
	f.width, f.height = width, height
	if width <= 0 || height <= 0 {
		return
	}
	innerWidth := max(1, f.boxWidth()-6)
	for _, field := range []*ui.TextField{f.topic.TextField, f.payload, f.start, f.end} {
		field.Width = max(1, innerWidth-6-lipgloss.Width(field.Prompt))
	}
	f.viewport.Width = innerWidth
	f.syncViewport(changed || f.Focus != f.lastFocus)
}

func (f historyFilterForm) boxWidth() int {
	return max(10, min(max(32, f.width/2), f.width-2))
}

func (f historyFilterForm) verticalPadding() int {
	if f.height < 6 {
		return 0
	}
	return 1
}

func (f *historyFilterForm) syncViewport(followFocus bool) []filterRow {
	rows := f.rows()
	f.viewport.Height = min(len(rows), max(1, f.height-2-2*f.verticalPadding()))
	lines := make([]string, len(rows))
	for i, row := range rows {
		lines[i] = row.text
	}
	f.viewport.SetContent(strings.Join(lines, "\n"))
	f.viewport.SetYOffset(f.viewport.YOffset)
	if followFocus {
		for i, row := range rows {
			if row.field != f.Focus || row.suggestion != "" {
				continue
			}
			if i < f.viewport.YOffset {
				f.viewport.SetYOffset(i)
			} else if i >= f.viewport.YOffset+f.viewport.Height {
				f.viewport.SetYOffset(i - f.viewport.Height + 1)
			}
			break
		}
	}
	f.lastFocus = f.Focus
	return rows
}

// contentOrigin includes centering, the border, and the existing form padding.
func (f historyFilterForm) contentOrigin() (int, int) {
	boxHeight := f.viewport.Height + 2 + 2*f.verticalPadding()
	return max(0, (f.width-f.boxWidth())/2) + 3,
		max(0, (f.height-boxHeight)/2) + 1 + f.verticalPadding()
}

// NewFilterForm builds a history filter form with optional prefilled values.
func NewFilterForm(topics []string, topic, payload string, start, end time.Time, archived bool) historyFilterForm {
	return newHistoryFilterForm(topics, topic, payload, start, end, archived)
}

// Update handles focus cycling and topic completion.
func (f historyFilterForm) Update(msg tea.Msg) (historyFilterForm, tea.Cmd) {
	var cmd tea.Cmd
	followFocus := false
	switch m := msg.(type) {
	case tea.KeyMsg:
		switch m.String() {
		case constants.KeyPgUp, constants.KeyPgDown:
			f.syncViewport(false)
			f.viewport, cmd = f.viewport.Update(m)
			return f, cmd
		case constants.KeyCtrlUp, constants.KeyCtrlK:
			f.syncViewport(false)
			f.viewport.ScrollUp(1)
			return f, nil
		case constants.KeyCtrlDown, constants.KeyCtrlJ:
			f.syncViewport(false)
			f.viewport.ScrollDown(1)
			return f, nil
		}
		f.errMsg = ""
		followFocus = true
		if c, ok := f.Fields[f.Focus].(ui.KeyConsumer); ok && c.WantsKey(m) {
			cmd = f.Fields[f.Focus].Update(msg)
		} else {
			f.CycleFocus(m)
			if len(f.Fields) > 0 {
				cmd = f.Fields[f.Focus].Update(msg)
			}
		}
	case tea.MouseMsg:
		rows := f.syncViewport(false)
		x, y := f.contentOrigin()
		if m.X < x || m.X >= x+f.viewport.Width || m.Y < y || m.Y >= y+f.viewport.Height {
			return f, nil
		}
		if m.Button == tea.MouseButtonWheelUp || m.Button == tea.MouseButtonWheelDown {
			f.viewport, cmd = f.viewport.Update(m)
			return f, cmd
		}
		if m.Action != tea.MouseActionPress || m.Button != tea.MouseButtonLeft {
			return f, nil
		}
		row := rows[m.Y-y+f.viewport.YOffset]
		if row.field < 0 {
			return f, nil
		}
		f.Focus = row.field
		followFocus = true
		f.ApplyFocus()
		if row.suggestion != "" {
			f.topic.SetValue(row.suggestion)
			f.topic.CursorEnd()
		}
		cmd = f.Fields[f.Focus].Update(msg)
	}
	f.ApplyFocus()
	if f.width > 0 && f.height > 0 {
		f.syncViewport(followFocus || f.Focus != f.lastFocus)
	}
	return f, cmd
}

func (f historyFilterForm) rows() []filterRow {
	width := f.viewport.Width
	rows := []filterRow{{text: fmt.Sprintf("Topic: %s", f.topic.View()), field: idxFilterTopic}}
	if sugg := f.topic.SuggestionsView(); sugg != "" {
		for _, suggestion := range strings.Split(sugg, "\n") {
			text := suggestion
			if width > 0 {
				text = ansi.WrapWc(text, width, "")
			}
			for _, line := range strings.Split(text, "\n") {
				rows = append(rows, filterRow{text: line, field: idxFilterTopic, suggestion: ansi.Strip(suggestion)})
			}
		}
	}
	for _, field := range []filterRow{
		{text: fmt.Sprintf("Text:  %s", f.payload.View()), field: idxFilterPayload},
		{text: fmt.Sprintf("Start: %s", f.start.View()), field: idxFilterStart},
		{text: fmt.Sprintf("End:   %s", f.end.View()), field: idxFilterEnd},
		{text: fmt.Sprintf("Archived: %s", f.archived.View()), field: idxFilterArchived},
	} {
		rows = append(rows, filterRow{field: -1}, field)
	}
	if f.errMsg != "" {
		rows = append(rows, filterRow{field: -1})
		errText := ui.ErrorStyle.Render(f.errMsg)
		if width > 0 {
			errText = ansi.WrapWc(errText, width, "")
		}
		for _, line := range strings.Split(errText, "\n") {
			rows = append(rows, filterRow{text: line, field: -1})
		}
	}
	if width > 0 {
		for i := range rows {
			rows[i].text = ansi.TruncateWc(rows[i].text, width, "")
		}
	}
	return rows
}

// View renders the filter fields with labels in their scrollable area.
func (f *historyFilterForm) View() string {
	if f.width > 0 && f.height > 0 {
		f.syncViewport(false)
		return f.viewport.View()
	}
	rows := f.rows()
	lines := make([]string, len(rows))
	for i, row := range rows {
		lines[i] = row.text
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
