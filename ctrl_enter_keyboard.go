package emqutiti

import (
	"runtime"
	"unicode"
	"unicode/utf8"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

// Disambiguation also changes ordinary Ctrl/Alt keys, Escape and keypad input.
// Translate only representable keys, inside the framing reader's paste guard.
// In particular, never collapse unsupported modifiers into a plain shortcut.
func decodeKittyKey(raw []byte) (tea.Msg, bool) {
	if len(raw) < 4 || len(raw) > 32 || raw[0] != '\x1b' || raw[1] != '[' || raw[len(raw)-1] != 'u' {
		return nil, false
	}
	digits := 0
	for _, b := range raw[2 : len(raw)-1] {
		if b == ';' || b == ':' {
			digits = 0
		} else if b < '0' || b > '9' {
			return nil, false
		} else {
			digits++
			if digits > 7 {
				return nil, false
			}
		}
	}
	if digits == 0 {
		return nil, false
	}
	p := ansi.NewParser()
	_, _, n, state := ansi.DecodeSequence(raw, 0, p)
	params := p.Params()
	if n != len(raw) || state != 0 || p.Command() != int('u') || len(params) < 1 {
		return nil, false
	}
	code := params[0].Param(-1)
	keyParams := 1
	for keyParams < len(params) && params[keyParams-1].HasMore() {
		keyParams++
	}
	if keyParams > 3 || params[keyParams-1].HasMore() || len(params) > keyParams+1 {
		return nil, false
	}
	shifted := -1
	if keyParams > 1 {
		if !kittyTextCode(code) {
			return nil, false
		}
		shifted = params[1].Param(-1)
		if shifted == -1 && keyParams != 3 || shifted != -1 && !kittyTextCode(shifted) {
			return nil, false
		}
		if keyParams == 3 && !kittyTextCode(params[2].Param(-1)) {
			return nil, false
		}
	}
	encoded := 1
	if len(params) > keyParams {
		if params[keyParams].HasMore() {
			return nil, false
		}
		encoded = params[keyParams].Param(-1)
	}
	if encoded < 1 || encoded > 256 {
		return nil, false
	}
	modifiers := (encoded - 1) &^ (64 | 128)
	if shifted != -1 && modifiers&1 == 0 {
		return nil, false
	}
	// Report keypad text as its usual editor input rather than losing it now
	// that disambiguation distinguishes it from the main keyboard.
	if code >= 57399 && code <= 57408 {
		code = '0' + code - 57399
	} else if code >= 57409 && code <= 57416 {
		code = int([]rune{'.', '/', '*', '-', '+', '\r', '=', ','}[code-57409])
		if code == '\r' {
			if retained, ok := publishEnterModifiers(encoded, runtime.GOOS == "darwin"); ok {
				return ctrlEnterMsg{press: true, retained: retained}, true
			}
		}
	}
	if code >= 57417 && code <= 57426 {
		keys := [10][4]tea.KeyType{
			{tea.KeyLeft, tea.KeyShiftLeft, tea.KeyCtrlLeft, tea.KeyCtrlShiftLeft},
			{tea.KeyRight, tea.KeyShiftRight, tea.KeyCtrlRight, tea.KeyCtrlShiftRight},
			{tea.KeyUp, tea.KeyShiftUp, tea.KeyCtrlUp, tea.KeyCtrlShiftUp},
			{tea.KeyDown, tea.KeyShiftDown, tea.KeyCtrlDown, tea.KeyCtrlShiftDown},
			{tea.KeyPgUp, 0, tea.KeyCtrlPgUp, 0},
			{tea.KeyPgDown, 0, tea.KeyCtrlPgDown, 0},
			{tea.KeyHome, tea.KeyShiftHome, tea.KeyCtrlHome, tea.KeyCtrlShiftHome},
			{tea.KeyEnd, tea.KeyShiftEnd, tea.KeyCtrlEnd, tea.KeyCtrlShiftEnd},
			{tea.KeyInsert, 0, 0, 0},
			{tea.KeyDelete, 0, 0, 0},
		}
		if modifiers&^7 != 0 {
			return nil, false
		}
		index := (modifiers & 1) | ((modifiers & 4) >> 1)
		key := keys[code-57417][index]
		return tea.KeyMsg{Type: key, Alt: modifiers&2 != 0}, key != 0
	}
	key := tea.KeyMsg{Alt: modifiers&2 != 0}
	switch code {
	case 13:
		if modifiers&^3 != 0 {
			return nil, false
		}
		key.Type = tea.KeyEnter
	case 9:
		if modifiers&^3 != 0 {
			return nil, false
		}
		key.Type = tea.KeyTab
		if modifiers&1 != 0 {
			key.Type = tea.KeyShiftTab
		}
	case 27:
		if modifiers&^2 != 0 {
			return nil, false
		}
		key.Type = tea.KeyEsc
	case 127:
		if modifiers&^7 != 0 {
			return nil, false
		}
		key.Type = tea.KeyBackspace
		if modifiers&4 != 0 {
			key.Type = tea.KeyCtrlH
		}
	default:
		if modifiers&4 != 0 {
			if modifiers&^6 != 0 {
				return nil, false
			}
			if code >= 'a' && code <= 'z' {
				key.Type = tea.KeyType(code - 'a' + 1)
			} else {
				controls := map[int]tea.KeyType{
					' ': tea.KeyCtrlAt, '@': tea.KeyCtrlAt, '2': tea.KeyCtrlAt,
					'[': tea.KeyCtrlOpenBracket, '3': tea.KeyCtrlOpenBracket,
					'\\': tea.KeyCtrlBackslash, '4': tea.KeyCtrlBackslash,
					']': tea.KeyCtrlCloseBracket, '5': tea.KeyCtrlCloseBracket,
					'^': tea.KeyCtrlCaret, '6': tea.KeyCtrlCaret,
					'_': tea.KeyCtrlUnderscore, '/': tea.KeyCtrlUnderscore, '7': tea.KeyCtrlUnderscore,
					'?': tea.KeyCtrlQuestionMark, '8': tea.KeyCtrlQuestionMark,
				}
				var ok bool
				if key.Type, ok = controls[code]; !ok {
					return nil, false
				}
			}
		} else {
			if modifiers&^3 != 0 || !kittyTextCode(code) {
				return nil, false
			}
			if modifiers&1 != 0 {
				// Use the terminal's layout-aware Shift value. Without one,
				// letters can be uppercased but punctuation must not be guessed.
				if shifted == -1 {
					if !unicode.IsLetter(rune(code)) && code != ' ' {
						return nil, false
					}
					shifted = int(unicode.ToUpper(rune(code)))
				}
				code = shifted
			}
			key.Type, key.Runes = tea.KeyRunes, []rune{rune(code)}
			if code == ' ' {
				key.Type = tea.KeySpace
			}
		}
	}
	return key, true
}

func kittyTextCode(code int) bool {
	return utf8.ValidRune(rune(code)) && !unicode.IsControl(rune(code)) && (code < 57344 || code > 63743)
}
