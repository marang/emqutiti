package connections

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/marang/emqutiti/ui"
)

func assertBrokerFormBounds(t *testing.T, f *Form, width, height int) string {
	t.Helper()
	view := f.View()
	if got := lipgloss.Height(view); got != height {
		t.Fatalf("height=%d want=%d\n%s", got, height, ansi.Strip(view))
	}
	for y, line := range strings.Split(view, "\n") {
		if got := lipgloss.Width(line); got > width {
			t.Fatalf("row %d width=%d exceeds %d: %q", y, got, width, ansi.Strip(line))
		}
	}
	if last := strings.Split(ansi.Strip(view), "\n")[height-1]; last != formActions {
		t.Fatalf("footer=%q want=%q", last, formActions)
	}
	return ansi.Strip(view)
}

func TestBrokerFormViewportFitsAndTabsReachEveryField(t *testing.T) {
	for _, size := range []struct{ width, height int }{{58, 37}, {78, 21}, {58, 17}, {38, 13}} {
		t.Run(fmt.Sprintf("%dx%d", size.width, size.height), func(t *testing.T) {
			f := NewForm(Profile{
				Name:            strings.Repeat("broker", 30),
				Username:        strings.Repeat("user", 30),
				Host:            strings.Repeat("host\u754c\u0065\u0301", 30),
				CACertPath:      strings.Repeat("/certificate", 30),
				LastWillPayload: strings.Repeat("long payload", 30),
			}, -1)
			before, err := f.Profile()
			if err != nil {
				t.Fatal(err)
			}
			f.SetSize(size.width, size.height)
			for i, fd := range formFields {
				if f.Focus != i {
					t.Fatalf("focus=%d want=%d", f.Focus, i)
				}
				view := assertBrokerFormBounds(t, &f, size.width, size.height)
				if !strings.Contains(view, fd.label+":") {
					t.Fatalf("focused field %s hidden\n%s", fd.key, view)
				}
				f, _ = f.Update(tea.KeyMsg{Type: tea.KeyTab})
			}
			if f.Focus != 0 {
				t.Fatalf("Tab did not wrap: focus=%d", f.Focus)
			}
			for _, size := range []struct{ width, height int }{{38, 13}, {78, 21}, {58, 37}} {
				f.SetSize(size.width, size.height)
				assertBrokerFormBounds(t, &f, size.width, size.height)
			}
			after, err := f.Profile()
			if err != nil || !reflect.DeepEqual(before, after) {
				t.Fatalf("values changed during resize/navigation: before=%#v after=%#v err=%v", before, after, err)
			}
		})
	}
}

func TestBrokerFormEnvTogglePreservesViewportAndReadOnlyValues(t *testing.T) {
	t.Setenv("EMQUTITI_ENV_HOST", "envhost")
	f := NewForm(Profile{Name: "env", LastWillPayload: "keep this"}, -1)
	f.SetSize(38, 13)
	f, _ = f.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	assertBrokerFormBounds(t, &f, 38, 13)
	if !f.fromEnv || f.Fields[fieldIndex["Host"]].Value() != "envhost" || !f.Fields[fieldIndex["Host"]].ReadOnly() {
		t.Fatal("env toggle did not load read-only values")
	}
	f, _ = f.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	view := assertBrokerFormBounds(t, &f, 38, 13)
	if !strings.Contains(view, "Last Will Payload:") {
		t.Fatalf("read-only last field not reachable\n%s", view)
	}
	if f.Fields[fieldIndex["LastWillPayload"]].Value() != "keep this" {
		t.Fatal("env toggle lost existing values")
	}
	f.Focus = 0
	f.ApplyFocus()
	f, _ = f.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	assertBrokerFormBounds(t, &f, 38, 13)
	if f.fromEnv || f.Fields[fieldIndex["Host"]].ReadOnly() || f.Fields[fieldIndex["Host"]].Value() != "envhost" {
		t.Fatal("manual mode lost loaded values or stayed read-only")
	}
}

func TestBrokerFormMouseGeometryAfterScrollAndOptions(t *testing.T) {
	for _, focus := range []string{"FromEnv", "Schema", "LastWillQos"} {
		t.Run(focus, func(t *testing.T) {
			f := NewForm(Profile{}, -1)
			f.SetSize(38, 13)
			f.Focus = fieldIndex[focus]
			f.ApplyFocus()
			f.View()
			for range len(f.rows) {
				f, _ = f.Update(tea.MouseMsg{X: 1, Y: 1, Action: tea.MouseActionPress, Button: tea.MouseButtonWheelDown})
			}
			if f.offset == 0 {
				t.Fatal("mouse wheel did not scroll")
			}
			view := assertBrokerFormBounds(t, &f, 38, 13)
			last := fieldIndex["LastWillPayload"]
			clicked := false
			for y, line := range strings.Split(view, "\n") {
				if strings.Contains(line, formFields[last].label+":") {
					f, _ = f.Update(tea.MouseMsg{X: 3, Y: y, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft})
					clicked = true
					break
				}
			}
			if !clicked || f.Focus != last {
				t.Fatalf("last field click not routed: clicked=%v focus=%d\n%s", clicked, f.Focus, view)
			}
			assertBrokerFormBounds(t, &f, 38, 13)
		})
	}
}

func TestBrokerFormMouseSelectOptionsAndFollowingField(t *testing.T) {
	f := NewForm(Profile{}, -1)
	f.SetSize(38, 13)
	f.Focus = fieldIndex["Schema"]
	f.ApplyFocus()
	f.View()
	optionY := -1
	for row, hit := range f.rows {
		if hit.field == f.Focus && hit.option == 5 {
			optionY = row - f.offset
		}
	}
	if optionY < 0 || optionY >= f.height-1 {
		t.Fatalf("last schema option hidden at y=%d", optionY)
	}
	f, _ = f.Update(tea.MouseMsg{X: 24, Y: optionY, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft})
	if got := f.Fields[fieldIndex["Schema"]].Value(); got != "mqtts" {
		t.Fatalf("option click selected %q want mqtts", got)
	}
	f, _ = f.Update(tea.MouseMsg{X: 24, Y: optionY, Action: tea.MouseActionPress, Button: tea.MouseButtonWheelDown})
	view := f.View()
	for y, line := range strings.Split(ansi.Strip(view), "\n") {
		if strings.Contains(line, "Host:") {
			f, _ = f.Update(tea.MouseMsg{X: 24, Y: y, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft})
			if f.Focus != fieldIndex["Host"] {
				t.Fatalf("click after options focused %d want Host", f.Focus)
			}
			return
		}
	}
	t.Fatalf("Host not reachable after options\n%s", ansi.Strip(view))
}

func TestBrokerFormPaddingAndOutsideClicksDoNotToggle(t *testing.T) {
	f := NewForm(Profile{}, -1)
	f.SetSize(38, 13)
	f.View()
	for _, point := range []struct{ x, y int }{{1, 0}, {-1, 1}, {38, 1}, {1, -1}, {1, 12}, {1, 14}} {
		f, _ = f.Update(tea.MouseMsg{X: point.x, Y: point.y, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft})
		if f.Fields[fieldIndex["FromEnv"]].(*ui.CheckField).Bool() {
			t.Fatalf("outside/padding click at %v toggled FromEnv", point)
		}
	}
}

func TestBrokerFormValidationRevealsFieldAndPreservesValues(t *testing.T) {
	f := NewForm(Profile{Name: "draft", LastWillPayload: "keep this"}, -1)
	f.SetSize(38, 13)
	f.Fields[fieldIndex["ReceiveMaximum"]].(*ui.TextField).SetValue("invalid")
	_, err := f.Validate()
	if err == nil || f.Focus != fieldIndex["ReceiveMaximum"] {
		t.Fatalf("invalid field not focused: focus=%d err=%v", f.Focus, err)
	}
	view := assertBrokerFormBounds(t, &f, 38, 13)
	if !strings.Contains(view, "Receive Maximum:") || !strings.Contains(view, "Enter a whole number.") {
		t.Fatalf("validation not visible\n%s", view)
	}
	if f.Fields[fieldIndex["LastWillPayload"]].Value() != "keep this" || f.Fields[fieldIndex["Name"]].Value() != "draft" {
		t.Fatal("validation lost draft")
	}
	f.Fields[fieldIndex["ReceiveMaximum"]].(*ui.TextField).SetValue("42")
	p, err := f.Validate()
	if err != nil || p.ReceiveMaximum != 42 || len(f.validation) != 0 {
		t.Fatalf("corrected value not accepted: profile=%#v err=%v", p, err)
	}
}

func TestBrokerFormKeyboardScrollAndFooterActions(t *testing.T) {
	f := NewForm(Profile{}, -1)
	f.SetSize(38, 13)
	f.View()
	f, _ = f.Update(tea.KeyMsg{Type: tea.KeyCtrlDown})
	if f.offset != 1 || f.Focus != 0 {
		t.Fatalf("Ctrl+Down offset=%d focus=%d", f.offset, f.Focus)
	}
	f, _ = f.Update(tea.KeyMsg{Type: tea.KeyCtrlUp})
	if f.offset != 0 || f.Focus != 0 {
		t.Fatalf("Ctrl+Up offset=%d focus=%d", f.offset, f.Focus)
	}
	for _, tc := range []struct {
		x, y int
		want FormAction
	}{{0, 12, SaveForm}, {14, 12, CancelForm}, {12, 12, NoFormAction}, {0, 11, NoFormAction}, {38, 12, NoFormAction}} {
		if got := f.ActionAt(tc.x, tc.y); got != tc.want {
			t.Errorf("ActionAt(%d,%d)=%v want=%v", tc.x, tc.y, got, tc.want)
		}
	}
	f.CycleFocus(tea.KeyMsg{Type: tea.KeyShiftTab})
	f.ApplyFocus()
	view := assertBrokerFormBounds(t, &f, 38, 13)
	if !strings.Contains(view, "Last Will Payload:") {
		t.Fatalf("direct CycleFocus did not reveal last field\n%s", view)
	}
}
