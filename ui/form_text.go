package ui

import (
	"fmt"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/marang/emqutiti/constants"
)

// TextField wraps a text input with optional read-only behaviour.
type TextField struct {
	textinput.Model
	readOnly bool
	filter   func(string) string
}

type TextFieldOption func(*textFieldOptions)

type textFieldOptions struct {
	mask      bool
	charLimit int
	width     int
	validator textinput.ValidateFunc
	filter    func(string) string
}

const defaultFieldWidth = 20

// NewTextField creates a TextField with the given value and placeholder.
func NewTextField(value, placeholder string, opts ...TextFieldOption) *TextField {
	cfg := textFieldOptions{}
	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}

	ti := textinput.New()
	ti.Placeholder = placeholder
	if cfg.mask {
		ti.EchoMode = textinput.EchoPassword
	}
	if cfg.charLimit > 0 {
		ti.CharLimit = cfg.charLimit
	}
	if cfg.validator != nil {
		ti.Validate = cfg.validator
	}

	tf := &TextField{Model: ti, filter: cfg.filter}
	tf.SetValue(value)

	width := cfg.width
	placeholderWidth := utf8.RuneCountInString(placeholder)
	valueWidth := utf8.RuneCountInString(tf.Value())
	if width < placeholderWidth {
		width = placeholderWidth
	}
	if width < valueWidth {
		width = valueWidth
	}
	if width < cfg.charLimit {
		width = cfg.charLimit
	}
	if width < defaultFieldWidth {
		width = defaultFieldWidth
	}
	tf.Width = width

	return tf
}

// WithMask configures the text field to hide characters as they are typed.
func WithMask() TextFieldOption {
	return func(cfg *textFieldOptions) {
		cfg.mask = true
	}
}

// WithCharLimit constrains the number of characters the field accepts.
func WithCharLimit(limit int) TextFieldOption {
	return func(cfg *textFieldOptions) {
		cfg.charLimit = limit
	}
}

// WithWidth sets an explicit field width in runes.
func WithWidth(width int) TextFieldOption {
	return func(cfg *textFieldOptions) {
		cfg.width = width
	}
}

// WithValidator applies a validation function that runs as the value changes.
func WithValidator(fn textinput.ValidateFunc) TextFieldOption {
	return func(cfg *textFieldOptions) {
		cfg.validator = fn
	}
}

const (
	RFC3339CharLimit     = len(time.RFC3339)
	rfc3339ExampleZ      = "2025-03-10T12:45:00Z"
	rfc3339ExampleOffset = "2025-03-10T12:45:00+02:00"
	dateTimePrefixLength = len("2006-01-02T15:04:05")
)

// WithRFC3339 enforces RFC3339-friendly behaviour on the field.
func WithRFC3339() TextFieldOption {
	return func(cfg *textFieldOptions) {
		cfg.validator = RFC3339Validator()
		cfg.charLimit = RFC3339CharLimit
		cfg.filter = sanitizeRFC3339
	}
}

// RFC3339Validator accepts partial RFC3339 text comprised of digits and valid
// separators. Once the value is long enough to parse, it must be a valid
// RFC3339 timestamp.
func RFC3339Validator() textinput.ValidateFunc {
	return func(input string) error {
		trimmed := strings.TrimSpace(input)
		if trimmed == "" {
			return nil
		}
		if !rfc3339Charset(trimmed) {
			return fmt.Errorf("use digits and separators like %s", rfc3339ExampleZ)
		}
		if len(trimmed) >= len("2006-01-02T15:04:05") {
			if _, err := time.Parse(time.RFC3339, strings.ToUpper(trimmed)); err != nil {
				return fmt.Errorf("use RFC3339 like %s or %s", rfc3339ExampleZ, rfc3339ExampleOffset)
			}
		}
		return nil
	}
}

func sanitizeRFC3339(value string) string {
	result := make([]rune, 0, len(value))
	for _, raw := range value {
		r := raw
		if unicode.IsLower(r) {
			switch r {
			case 't':
				r = 'T'
			case 'z':
				r = 'Z'
			default:
				r = unicode.ToUpper(r)
			}
		}
		if unicode.IsSpace(r) {
			continue
		}
		candidate := append(append([]rune(nil), result...), r)
		if isValidRFC3339Prefix(candidate) {
			result = candidate
		}
	}
	return string(result)
}

func isValidRFC3339Prefix(runes []rune) bool {
	if len(runes) > RFC3339CharLimit {
		return false
	}
	for i, r := range runes {
		if i >= dateTimePrefixLength {
			break
		}
		switch i {
		case 0, 1, 2, 3, 5, 6, 8, 9, 11, 12, 14, 15, 17, 18:
			if !unicode.IsDigit(r) {
				return false
			}
		case 4, 7:
			if r != '-' {
				return false
			}
		case 10:
			if r != 'T' {
				return false
			}
		case 13, 16:
			if r != ':' {
				return false
			}
		default:
			return false
		}
	}
	if len(runes) <= dateTimePrefixLength {
		return true
	}
	tail := string(runes[dateTimePrefixLength:])
	return validRFC3339Tail(tail)
}

func validRFC3339Tail(tail string) bool {
	if tail == "" {
		return true
	}
	remaining := tail
	if remaining[0] == '.' {
		i := 1
		for i < len(remaining) && isDigitByte(remaining[i]) {
			i++
		}
		if i == 1 {
			// only the decimal point so far
			if len(remaining) == 1 {
				return true
			}
		} else {
			if len(remaining) == i {
				return true
			}
		}
		remaining = remaining[i:]
		if remaining == "" {
			return true
		}
	}
	if remaining == "" {
		return true
	}
	switch remaining[0] {
	case 'Z':
		return len(remaining) == 1
	case '+', '-':
		rest := remaining[1:]
		if len(rest) == 0 {
			return true
		}
		if !isDigitByte(rest[0]) {
			return false
		}
		if len(rest) == 1 {
			return true
		}
		if !isDigitByte(rest[1]) {
			return false
		}
		if len(rest) == 2 {
			return true
		}
		if rest[2] != ':' {
			return false
		}
		if len(rest) == 3 {
			return true
		}
		if !isDigitByte(rest[3]) {
			return false
		}
		if len(rest) == 4 {
			return true
		}
		if !isDigitByte(rest[4]) {
			return false
		}
		return len(rest) == 5
	default:
		return false
	}
}

func isDigitByte(b byte) bool { return b >= '0' && b <= '9' }

func rfc3339Charset(s string) bool {
	for _, r := range s {
		switch {
		case unicode.IsDigit(r):
			continue
		case r == '-' || r == ':' || r == '+' || r == '.' || r == 'T' || r == 't' || r == 'Z' || r == 'z':
			continue
		default:
			return false
		}
	}
	return true
}

// ParseRFC3339 normalises and parses an RFC3339 timestamp. Empty strings
// return the zero time.
func ParseRFC3339(value string) (time.Time, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return time.Time{}, nil
	}
	parsed, err := time.Parse(time.RFC3339, strings.ToUpper(trimmed))
	if err != nil {
		return time.Time{}, fmt.Errorf("use RFC3339 like %s or %s", rfc3339ExampleZ, rfc3339ExampleOffset)
	}
	return parsed, nil
}

// SetValue applies the optional filter before storing the value.
func (t *TextField) SetValue(v string) {
	if t.filter != nil {
		v = t.filter(v)
	}
	t.Model.SetValue(v)
}

// SetReadOnly marks the field read only and blurs it when activated.
func (t *TextField) SetReadOnly(ro bool) {
	t.readOnly = ro
	if ro {
		t.Blur()
	}
}

// ReadOnly reports whether the field is read only.
func (t *TextField) ReadOnly() bool { return t.readOnly }

// Update forwards messages to the text input unless the field is read only.
func (t *TextField) Update(msg tea.Msg) tea.Cmd {
	if t.readOnly {
		return nil
	}
	var cmd tea.Cmd
	t.Model, cmd = t.Model.Update(msg)
	if t.filter != nil {
		filtered := t.filter(t.Model.Value())
		if filtered != t.Model.Value() {
			t.Model.SetValue(filtered)
		}
	}
	return cmd
}

// Value returns the text content of the field.
func (t *TextField) Value() string { return t.Model.Value() }

// Err exposes the current validation error, if any.
func (t *TextField) Err() error { return t.Model.Err }

func (t *TextField) Focus() {
	if !t.readOnly {
		t.Model.Focus()
	}
}

func (t *TextField) Blur() { t.Model.Blur() }

func (t *TextField) View() string {
	if t.readOnly {
		return BlurredStyle.Render(t.Model.View())
	}
	return t.Model.View()
}

// WantsKey reports whether the field wants to handle navigation keys itself
// instead of letting the form cycle focus. Plain "j" and "k" are treated as
// normal input so users can type them without jumping to another field.
func (t *TextField) WantsKey(k tea.KeyMsg) bool {
	switch k.String() {
	case constants.KeyJ, constants.KeyK:
		return true
	default:
		return false
	}
}
