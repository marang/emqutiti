package emqutiti

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestNegotiatedKeyboardPreservesLegacyKeys(t *testing.T) {
	for _, test := range []struct {
		raw string
		key tea.KeyMsg
	}{
		{"\x1b[100;5u", tea.KeyMsg{Type: tea.KeyCtrlD}},
		{"\x1b[98;5u", tea.KeyMsg{Type: tea.KeyCtrlB}},
		{"\x1b[99;5u", tea.KeyMsg{Type: tea.KeyCtrlC}},
		{"\x1b[108;5u", tea.KeyMsg{Type: tea.KeyCtrlL}},
		{"\x1b[27u", tea.KeyMsg{Type: tea.KeyEsc}},
		{"\x1b[9;2u", tea.KeyMsg{Type: tea.KeyShiftTab}},
		{"\x1b[127;5u", tea.KeyMsg{Type: tea.KeyCtrlH}},
		{"\x1b[127;2u", tea.KeyMsg{Type: tea.KeyBackspace}},
		{"\x1b[127;3u", tea.KeyMsg{Type: tea.KeyBackspace, Alt: true}},
		{"\x1b[114;3u", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}, Alt: true}},
		{"\x1b[114::114;3u", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}, Alt: true}},
		{"\x1b[97::97;5u", tea.KeyMsg{Type: tea.KeyCtrlA}},
		{"\x1b[97:65;4u", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'A'}, Alt: true}},
		{"\x1b[44:60;4u", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'<'}, Alt: true}},
		{"\x1b[46:62:46;4u", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'>'}, Alt: true}},
		{"\x1b[228:196;4u", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'\u00c4'}, Alt: true}},
		{"\x1b[97;4u", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'A'}, Alt: true}},
		{"\x1b[228;3u", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'\u00e4'}, Alt: true}},
		{"\x1b[13;2u", tea.KeyMsg{Type: tea.KeyEnter}},
		{"\x1b[57414u", tea.KeyMsg{Type: tea.KeyEnter}},
		{"\x1b[57400u", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'1'}}},
		{"\x1b[57413u", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'+'}}},
		{"\x1b[57417u", tea.KeyMsg{Type: tea.KeyLeft}},
		{"\x1b[57417;6u", tea.KeyMsg{Type: tea.KeyCtrlShiftLeft}},
		{"\x1b[57418u", tea.KeyMsg{Type: tea.KeyRight}},
		{"\x1b[57419u", tea.KeyMsg{Type: tea.KeyUp}},
		{"\x1b[57420u", tea.KeyMsg{Type: tea.KeyDown}},
		{"\x1b[57421u", tea.KeyMsg{Type: tea.KeyPgUp}},
		{"\x1b[57422u", tea.KeyMsg{Type: tea.KeyPgDown}},
		{"\x1b[57423u", tea.KeyMsg{Type: tea.KeyHome}},
		{"\x1b[57424u", tea.KeyMsg{Type: tea.KeyEnd}},
		{"\x1b[57425u", tea.KeyMsg{Type: tea.KeyInsert}},
		{"\x1b[57426u", tea.KeyMsg{Type: tea.KeyDelete}},
		{"\x1b[100;197u", tea.KeyMsg{Type: tea.KeyCtrlD}},
	} {
		t.Run(fmt.Sprintf("%q", test.raw), func(t *testing.T) {
			if got, ok := decodeKittyKey([]byte(test.raw)); !ok || !reflect.DeepEqual(got, test.key) {
				t.Fatalf("decoded key=%#v, valid=%v; want %#v", got, ok, test.key)
			}
			input := test.raw + "\x04"
			want := []tea.KeyMsg{test.key}
			if test.key.Type == tea.KeyCtrlD {
				// The capture harness consumes Ctrl+D as its quit command.
				want = nil
			}
			for split := 1; split < len(input); split++ {
				r := &ctrlEnterChunkReader{chunks: [][]byte{[]byte(input[:split]), []byte(input[split:])}}
				if got := ctrlEnterInputKeys(captureCtrlEnterInput(t, r)); !reflect.DeepEqual(got, want) {
					t.Fatalf("split %d: keys=%#v, want %#v", split, got, test.key)
				}
			}
		})
	}
	for r := 'a'; r <= 'z'; r++ {
		raw := fmt.Sprintf("\x1b[%d;5u\x04", r)
		want := tea.KeyMsg{Type: tea.KeyType(r - 'a' + 1)}
		if got, ok := decodeKittyKey([]byte(strings.TrimSuffix(raw, "\x04"))); !ok || !reflect.DeepEqual(got, want) {
			t.Fatalf("Ctrl+%c: decoded key=%#v", r, got)
		}
		keys := []tea.KeyMsg{want}
		if r == 'd' {
			keys = nil
		}
		if got := ctrlEnterInputKeys(captureCtrlEnterInput(t, strings.NewReader(raw))); !reflect.DeepEqual(got, keys) {
			t.Fatalf("Ctrl+%c: keys=%#v", r, got)
		}
	}
}

func TestNegotiatedKeyboardRejectsUnrepresentableAndMalformedKeys(t *testing.T) {
	for _, raw := range []string{
		"\x1b[100;6u", "\x1b[100;9u", "\x1b[100;0u", "\x1b[100;257u",
		"\x1b[100;;5u", "\x1b[100;5;100u", "\x1b[100;5:3u", "\x1b[100;5:2u",
		"\x1b[100;5\x00u", "\x1b[99999999;5u", "\x1b[?100;5u",
		"\x1b[100;5 u", "\x1b[100;u", "\x1b[;5u", "\x1b[1114112;3u",
		"\x1b[55296;3u", "\x1b[57358u", "\x1b[13;7u", "\x1b[13;5u",
		"\x1b[97:;4u", "\x1b[97::;4u", "\x1b[97:::97;3u", "\x1b[97:65:97:65;4u",
		"\x1b[97:65;4:1u", "\x1b[97:65;4;65u", "\x1b[97::97;u",
		"\x1b[100:68;6u", "\x1b[100:68;10u", "\x1b[100:68;3u",
		"\x1b[13:13;5u", "\x1b[100:13;3u", "\x1b[100::13;5u",
		"\x1b[97:55296;4u", "\x1b[97:1114112;4u", "\x1b[97:57358;4u",
	} {
		if msg, ok := decodeKittyKey([]byte(raw)); ok {
			t.Fatalf("%q was downgraded to an unrelated key: %#v", raw, msg)
		}
	}
}

func TestNegotiatedKeyboardShiftAltMessageNavigation(t *testing.T) {
	for _, tc := range []struct {
		name, legacy, negotiated string
		start, want              int
	}{
		{"input begin", "\x1b<", "\x1b[44:60;4u", 2, 0},
		{"input end", "\x1b>", "\x1b[46:62;4u", 0, 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, raw := range []string{tc.legacy, tc.negotiated} {
				m := reviewFixture(t, 80, 24)
				m.SetFocus(idMessage)
				m.message.SetPayload("one\ntwo\nthree")
				if tc.start == 0 {
					m.Update(tea.KeyMsg{Type: tea.KeyCtrlHome})
				}
				if got := m.message.Input().Line(); got != tc.start {
					t.Fatalf("initial line=%d, want %d", got, tc.start)
				}
				captureCtrlEnterInputWithUpdate(t, strings.NewReader(raw+"\x04"), func(msg tea.Msg) tea.Cmd {
					_, cmd := m.Update(msg)
					return cmd
				})
				if got := m.message.Input().Line(); got != tc.want || m.message.Input().Value() != "one\ntwo\nthree" {
					t.Fatalf("%q: line=%d, want %d; draft=%q", raw, got, tc.want, m.message.Input().Value())
				}
			}
		})
	}
}

func TestNegotiatedKeyboardKeepsControlsInsidePaste(t *testing.T) {
	payload := "\x1b[100;5u\x1b[27u\x1b[114;3u\x1b[57414;5u\x1b[44:60;4u\x1b[100::100;5u"
	input := "\x1b[200~" + payload + "\x1b[201~\x04"
	for split := 1; split < len(input); split++ {
		r := &ctrlEnterChunkReader{chunks: [][]byte{[]byte(input[:split]), []byte(input[split:])}}
		assertCtrlEnterProtectedPaste(t, captureCtrlEnterInput(t, r), payload)
	}
}
