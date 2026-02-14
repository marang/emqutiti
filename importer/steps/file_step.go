package steps

import (
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/marang/emqutiti/ui"
)

// FileStep handles selecting the file to import.
type FileStep struct{ *Base }

func NewFileStep(b *Base) *FileStep {
	b.Current = File
	b.File.Focus()
	return &FileStep{Base: b}
}

func (s *FileStep) Update(msg tea.Msg) (Step, tea.Cmd) {
	var cmd tea.Cmd
	s.File, cmd = s.File.Update(msg)
	if km, ok := msg.(tea.KeyMsg); ok && (km.Type == tea.KeyEnter || km.Type == tea.KeyCtrlN) {
		path := strings.TrimSpace(s.File.Value())
		if path == "" {
			return s, cmd
		}
		rows, err := ReadFile(path)
		if err != nil {
			s.File.SetValue(path + " (" + err.Error() + ")")
			return s, cmd
		}
		if len(rows) == 0 {
			s.File.SetValue(path + " (no data)")
			return s, cmd
		}
		s.Rows = rows
		s.Headers = s.Headers[:0]
		for k := range rows[0] {
			s.Headers = append(s.Headers, k)
		}
		sort.Strings(s.Headers)
		var fields []ui.Field
		for _, k := range s.Headers {
			fi := ui.NewTextField(k, "")
			if v, ok := s.Prefs.Mapping[k]; ok {
				fi.SetValue(v)
			}
			fields = append(fields, fi)
		}
		if len(fields) > 0 {
			s.Form = ui.Form{Fields: fields, Focus: 0}
			s.Form.ApplyFocus()
		}
		return NewMapStep(s.Base), cmd
	}
	return s, cmd
}

func (s *FileStep) View(bw, _ int) string {
	content := s.File.View() + "\n[enter] load file  [ctrl+n] next"
	return ui.LegendBox(content, "Import", bw, 0, ui.ColBlue, true, -1)
}
