package traces

import (
	"github.com/marang/emqutiti/ui"
)

// ViewForm renders the form for new traces.
func (t *Component) ViewForm() string {
	if t.form == nil {
		return ""
	}
	t.SetSize(t.api.Width(), t.api.Height())
	t.api.ResetElemPos()
	focused := t.api.FocusedID() == IDForm
	if t.form != nil {
		if focused {
			t.form.ApplyFocus()
		} else {
			for _, fld := range t.form.Fields {
				fld.Blur()
			}
		}
	}
	content := t.form.View()
	view := ui.LegendBox(content, "New Trace", t.api.Width()-2, 0, ui.ColBlue, focused, -1)
	return t.api.OverlayHelp(view)
}
