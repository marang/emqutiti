package emqutiti

import (
	"context"
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/marang/emqutiti/constants"
)

func ctrlEnterInputKeys(messages []tea.Msg) []tea.KeyMsg {
	var keys []tea.KeyMsg
	for _, msg := range messages {
		if key, ok := msg.(tea.KeyMsg); ok {
			keys = append(keys, key)
		}
	}
	return keys
}

func ctrlEnterAtomicInputKeys(messages []tea.Msg) []tea.KeyMsg {
	var result []tea.KeyMsg
	for _, key := range ctrlEnterInputKeys(messages) {
		// Tea batches adjacent ordinary runes within a read. Frame boundaries
		// may split that batch without changing the text or key modifiers.
		if key.Type == tea.KeyRunes && !key.Alt && !key.Paste {
			for _, r := range key.Runes {
				result = append(result, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
			}
		} else {
			result = append(result, key)
		}
	}
	return result
}

func captureCtrlEnterBaselineInput(t *testing.T, input string) []tea.Msg {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	m := &ctrlEnterCaptureModel{}
	p := tea.NewProgram(m, tea.WithInput(strings.NewReader(input)), tea.WithOutput(io.Discard), tea.WithContext(ctx), tea.WithoutRenderer(), tea.WithoutSignalHandler())
	if _, err := p.Run(); err != nil {
		t.Fatal(err)
	}
	return m.messages
}

func TestCtrlEnterFramingNestedPasteEverySplit(t *testing.T) {
	for _, prefix := range []string{"\x1b]x", "\x1bPx", "\x1bO", "\x1b\x1bO"} {
		t.Run(prefix, func(t *testing.T) {
			input := prefix + "\x1b[200~abc\x1b[201~\x04"
			want := ctrlEnterInputKeys(captureCtrlEnterBaselineInput(t, input))
			for split := 1; split < len(input); split++ {
				r := &ctrlEnterChunkReader{chunks: [][]byte{[]byte(input[:split]), []byte(input[split:])}}
				messages := captureCtrlEnterInput(t, r)
				if got := ctrlEnterInputKeys(messages); !reflect.DeepEqual(got, want) {
					t.Fatalf("split %d: keys = %#v, baseline = %#v", split, got, want)
				}
				if len(ctrlEnterEvents(messages)) != 0 || shiftedSpaceEvents(messages) != 0 {
					t.Fatal("nested paste became an application shortcut")
				}
			}
		})
	}
}

func ctrlEnterTeaKeyCorpus(t *testing.T) []string {
	t.Helper()
	pc := reflect.ValueOf(tea.NewProgram).Pointer()
	source, _ := runtime.FuncForPC(pc).FileLine(pc)
	file, err := parser.ParseFile(token.NewFileSet(), filepath.Join(filepath.Dir(source), "key.go"), nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	// Read the pinned dependency's actual map using Go's parser, rather than
	// maintaining a second handwritten terminal-key compatibility table.
	corpus := make(map[string]bool)
	entries := 0
	ast.Inspect(file, func(node ast.Node) bool {
		decl, ok := node.(*ast.ValueSpec)
		if !ok || len(decl.Names) != 1 || decl.Names[0].Name != "sequences" {
			return true
		}
		for _, element := range decl.Values[0].(*ast.CompositeLit).Elts {
			entry := element.(*ast.KeyValueExpr)
			raw, err := strconv.Unquote(entry.Key.(*ast.BasicLit).Value)
			if err != nil {
				t.Fatal(err)
			}
			entries++
			corpus[raw] = true
			alt := false
			for _, field := range entry.Value.(*ast.CompositeLit).Elts {
				field := field.(*ast.KeyValueExpr)
				if field.Key.(*ast.Ident).Name == "Alt" {
					alt = field.Value.(*ast.Ident).Name == "true"
				}
			}
			if !alt {
				corpus["\x1b"+raw] = true
			}
		}
		return false
	})
	if entries != 142 {
		t.Fatalf("Tea compatibility corpus changed: got %d sequences, want pinned 142", entries)
	}
	var result []string
	for raw := range corpus {
		result = append(result, raw)
	}
	sort.Strings(result)
	return result
}

func TestCtrlEnterFramingTeaKeyCorpusEverySplit(t *testing.T) {
	for _, raw := range ctrlEnterTeaKeyCorpus(t) {
		t.Run(strconv.Quote(raw), func(t *testing.T) {
			input := raw + "\x04"
			want := captureCtrlEnterBaselineInput(t, input)
			for split := 1; split < len(input); split++ {
				r := &ctrlEnterChunkReader{chunks: [][]byte{[]byte(input[:split]), []byte(input[split:])}}
				got := captureCtrlEnterInput(t, r)
				if !reflect.DeepEqual(got, want) {
					t.Fatalf("split %d: messages = %#v, baseline = %#v", split, got, want)
				}
			}
		})
	}
}

func assertCtrlEnterProtectedPaste(t *testing.T, messages []tea.Msg, payload string) {
	t.Helper()
	var pastes []string
	for _, key := range ctrlEnterInputKeys(messages) {
		if key.Paste {
			pastes = append(pastes, string(key.Runes))
		}
	}
	if !reflect.DeepEqual(pastes, []string{payload}) || len(ctrlEnterEvents(messages)) != 0 || shiftedSpaceEvents(messages) != 0 {
		t.Fatalf("unprotected paste: pastes = %#v, CtrlEnter = %#v, ShiftSpace = %d", pastes, ctrlEnterEvents(messages), shiftedSpaceEvents(messages))
	}
}

func TestCtrlEnterFramingBareControlProtectedPasteEverySplit(t *testing.T) {
	payload := "abc\x1b[13;5u\x1b[32;2u"
	for _, prefix := range []string{"\x1bO", "\x1b\x1bO", "\x1bP", "\x1b]"} {
		t.Run(strconv.Quote(prefix), func(t *testing.T) {
			input := prefix + "\x1b[200~" + payload + "\x1b[201~\x04"
			for split := 1; split < len(input); split++ {
				r := &ctrlEnterChunkReader{chunks: [][]byte{[]byte(input[:split]), []byte(input[split:])}}
				assertCtrlEnterProtectedPaste(t, captureCtrlEnterInput(t, r), payload)
			}
		})
	}
}

func TestCtrlEnterFramingCoreMessagesMatchTea(t *testing.T) {
	for _, raw := range []string{
		"\x1b[A", "\x1b[B", "\x1b[C", "\x1b[D", "\x1b[1;6A", "\x1b[1;6B",
		"\x1b[15~", "\x1b[15;3~", "\x1b[17~", "\x1b[23~", "\x1b[24~",
		"\x1b[Z", "\t", "\r", "\n", "\x13", "\x05", "\x1bx", "\x1b ",
		"\x1b[<0;12;4M", "\x1b[<0;12;4m", "\x1b[M !!",
		"\x1b[O", "\x1bOF", "\x1bOH", "\x1bOX",
		"\xc3\xa4", "\x1b\xc3\xa4", "\xf0\x9f\x98\x80", "\x1b\xf0\x9f\x98\x80",
	} {
		t.Run(raw, func(t *testing.T) {
			input := raw + "\x04"
			want := captureCtrlEnterBaselineInput(t, input)
			for split := 1; split < len(input); split++ {
				r := &ctrlEnterChunkReader{chunks: [][]byte{[]byte(input[:split]), []byte(input[split:])}}
				got := captureCtrlEnterInput(t, r)
				if !reflect.DeepEqual(got, want) {
					t.Fatalf("split %d: messages = %#v, baseline = %#v", split, got, want)
				}
			}
		})
	}
}

func TestCtrlEnterFramingIncompleteX10DoesNotEatQuit(t *testing.T) {
	for _, raw := range []string{"\x1b[M", "\x1b[Ma"} {
		t.Run(strconv.Quote(raw), func(t *testing.T) {
			input := raw + "\x04"
			want := ctrlEnterInputKeys(captureCtrlEnterBaselineInput(t, input))
			got := ctrlEnterInputKeys(captureCtrlEnterInput(t, strings.NewReader(input)))
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("keys = %#v, baseline = %#v", got, want)
			}
		})
	}
}

func TestCtrlEnterFramingStandaloneFocusReports(t *testing.T) {
	for _, test := range []struct {
		raw  string
		want tea.Msg
	}{{"\x1b[I", tea.FocusMsg{}}, {"\x1b[O", tea.BlurMsg{}}} {
		t.Run(strconv.Quote(test.raw), func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			r := framingCtrlEnterReader(strings.NewReader(test.raw))
			defer r.close()
			m := &ctrlEnterCaptureModel{apply: func(tea.Msg) tea.Cmd { return tea.Quit }}
			p := tea.NewProgram(m, tea.WithInput(r), tea.WithFilter(r.filter), tea.WithOutput(io.Discard), tea.WithContext(ctx), tea.WithoutRenderer(), tea.WithoutSignalHandler())
			if _, err := p.Run(); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(m.messages, []tea.Msg{test.want}) {
				t.Fatalf("messages = %#v, want %#v", m.messages, test.want)
			}
		})
	}
}

func TestCtrlEnterFramingBorrowedDiagnosticBeforeTrailingEscape(t *testing.T) {
	for _, prefix := range []string{"\x1b[M", "\x1b[O", "\x1b[["} {
		t.Run(strconv.Quote(prefix), func(t *testing.T) {
			step := 0
			r := newCtrlEnterInputReader(func(dst []byte, _ bool) (int, error) {
				step++
				switch step {
				case 1:
					return copy(dst, prefix+"\x1b"), nil
				case 2:
					return 0, errCtrlEnterInputTimeout
				case 3:
					return copy(dst, "\x04"), nil
				default:
					return 0, io.EOF
				}
			})
			defer r.close()
			buf := make([]byte, 256)
			if n, err := r.Read(buf); err != nil || string(buf[:n]) != prefix+"\x1b" {
				t.Fatalf("partial report = %q, %v", buf[:n], err)
			}
			if !r.waitAck {
				t.Fatal("borrowed CSI can be overwritten before filter copies it")
			}
			diagnostic := captureCtrlEnterBaselineInput(t, prefix+"\x04")[0]
			copied := r.filter(nil, diagnostic)
			if !reflect.DeepEqual(copied, diagnostic) {
				t.Fatalf("diagnostic changed: %#v", copied)
			}
			if n, err := r.Read(buf); err != nil || string(buf[:n]) != "\x04" {
				t.Fatalf("ordered key after ack = %q, %v", buf[:n], err)
			}
		})
	}
}

func TestCtrlEnterFramingSuffixUTF8EverySplit(t *testing.T) {
	for _, prefix := range []string{"\x1bO", "\x1b\x1bO", "\x1b[O", "\x1b\x1b[O", "\x1b[[", "\x1b\x1b[["} {
		for _, runeBytes := range []string{"\xc3\xa4", "\xf0\x9f\x98\x80"} {
			input := prefix + runeBytes + "\x04"
			want := ctrlEnterAtomicInputKeys(captureCtrlEnterBaselineInput(t, input))
			for split := 1; split < len(input); split++ {
				r := &ctrlEnterChunkReader{chunks: [][]byte{[]byte(input[:split]), []byte(input[split:])}}
				got := ctrlEnterAtomicInputKeys(captureCtrlEnterInput(t, r))
				if !reflect.DeepEqual(got, want) {
					t.Fatalf("%q split %d: keys = %#v, baseline = %#v", input, split, got, want)
				}
			}
		}
	}
}

func ctrlEnterUnicodeEditorFixture(t *testing.T) *model {
	t.Helper()
	m := reviewFixture(t, 80, 24)
	m.ui.modeStack = []constants.AppMode{constants.ModeClient}
	m.message.SetPayload("seed")
	m.message.Input().CursorEnd()
	m.SetFocus(idMessage)
	return m
}

func TestCtrlEnterFramingSuffixUTF8RootEditorEverySplit(t *testing.T) {
	input := "\x1bO\xc3\xa4\x04"
	for split := 1; split < len(input); split++ {
		m := ctrlEnterUnicodeEditorFixture(t)
		r := &ctrlEnterChunkReader{chunks: [][]byte{[]byte(input[:split]), []byte(input[split:])}}
		messages := captureCtrlEnterInputWithUpdate(t, r, func(msg tea.Msg) tea.Cmd {
			_, cmd := m.Update(msg)
			return cmd
		})
		if got := m.message.Input().Value(); got != "seedO\xc3\xa4" || m.pendingPublishes() != 0 || len(ctrlEnterEvents(messages)) != 0 {
			t.Fatalf("split %d: editor draft = %q; pending = %d", split, got, m.pendingPublishes())
		}
	}
}

type ctrlEnterScriptReader func([]byte) (int, error)

func (r ctrlEnterScriptReader) Read(dst []byte) (int, error) { return r(dst) }

func TestCtrlEnterFramingSuffixUTF8Timeout(t *testing.T) {
	for _, prefix := range []string{"\x1bO", "\x1b\x1bO", "\x1b[O", "\x1b[["} {
		for _, runeBytes := range []string{"\xc3\xa4", "\xf0\x9f\x98\x80"} {
			want := ctrlEnterAtomicInputKeys(captureCtrlEnterBaselineInput(t, prefix+runeBytes+"\x04"))
			for split := 0; split < len(runeBytes); split++ {
				step := 0
				source := ctrlEnterScriptReader(func(dst []byte) (int, error) {
					step++
					switch step {
					case 1:
						return copy(dst, prefix+runeBytes[:split]), nil
					case 2:
						return 0, errCtrlEnterInputTimeout
					case 3:
						return copy(dst, runeBytes[split:]+"\x04"), nil
					default:
						return 0, io.EOF
					}
				})
				got := ctrlEnterAtomicInputKeys(captureCtrlEnterInput(t, source))
				if !reflect.DeepEqual(got, want) {
					t.Fatalf("%q rune split %d: keys = %#v, baseline = %#v", prefix+runeBytes, split, got, want)
				}
			}
		}
	}
}

func TestCtrlEnterFramingPendingRuneCloseAndEOF(t *testing.T) {
	for _, sourceErr := range []error{errCtrlEnterInputTimeout, io.EOF} {
		step := 0
		r := newCtrlEnterInputReader(func(dst []byte, _ bool) (int, error) {
			step++
			if step == 1 {
				return copy(dst, "\xc3"), nil
			}
			return 0, sourceErr
		})
		buf := make([]byte, 256)
		n, err := r.Read(buf)
		if errors.Is(sourceErr, io.EOF) {
			if n != 1 || err != nil || buf[0] != 0xc3 {
				t.Fatalf("truncated rune at EOF = %d, %v", n, err)
			}
		} else if n != 0 || err != nil || !r.pendingRune {
			t.Fatalf("pending rune = %d, %v, pending=%v", n, err, r.pendingRune)
		}
		r.close()
		if n, err := r.Read(buf); n != 0 || !errors.Is(err, io.EOF) {
			t.Fatalf("close pending rune = %d, %v", n, err)
		}
	}
}

func ctrlEnterSS3Cases() []struct {
	raw string
	key tea.KeyMsg
} {
	var cases []struct {
		raw string
		key tea.KeyMsg
	}
	for _, key := range []struct {
		final byte
		type_ tea.KeyType
	}{
		{'A', tea.KeyUp}, {'B', tea.KeyDown}, {'C', tea.KeyRight}, {'D', tea.KeyLeft},
		{'P', tea.KeyF1}, {'Q', tea.KeyF2}, {'R', tea.KeyF3}, {'S', tea.KeyF4},
	} {
		for _, alt := range []bool{false, true} {
			raw := "\x1bO" + string(key.final)
			if alt {
				raw = "\x1b" + raw
			}
			cases = append(cases, struct {
				raw string
				key tea.KeyMsg
			}{raw, tea.KeyMsg{Type: key.type_, Alt: alt}})
		}
	}
	return cases
}

func assertCtrlEnterNestedPaste(t *testing.T, messages []tea.Msg, payload string) {
	t.Helper()
	var pastes []string
	for _, key := range ctrlEnterInputKeys(messages) {
		if key.Paste {
			pastes = append(pastes, string(key.Runes))
		}
	}
	if !reflect.DeepEqual(pastes, []string{payload}) {
		t.Fatalf("pastes = %#v, want original payload %q", pastes, payload)
	}
	if got := ctrlEnterEvents(messages); !reflect.DeepEqual(got, []ctrlEnterMsg{{press: true}}) || shiftedSpaceEvents(messages) != 0 {
		t.Fatalf("paste/normal input routing changed: CtrlEnter = %#v, ShiftSpace = %d", got, shiftedSpaceEvents(messages))
	}
}

func TestCtrlEnterFramingNestedPasteAroundFrameLimit(t *testing.T) {
	payload := "abc\x1b[13;5u\x1b[32;2u"
	for _, control := range []string{"\x1b]", "\x1bP"} {
		for length := 120; length <= 130; length++ {
			input := control + strings.Repeat("x", length) + "\x1b[200~" + payload + "\x1b[201~\x1b[13;5u\x04"
			messages := captureCtrlEnterInput(t, strings.NewReader(input))
			assertCtrlEnterNestedPaste(t, messages, payload)
		}
	}
}

func TestCtrlEnterFramingSS3EverySplit(t *testing.T) {
	for _, test := range ctrlEnterSS3Cases() {
		t.Run(test.key.String(), func(t *testing.T) {
			input := test.raw + "\x04"
			for split := 1; split < len(input); split++ {
				r := &ctrlEnterChunkReader{chunks: [][]byte{[]byte(input[:split]), []byte(input[split:])}}
				messages := captureCtrlEnterInput(t, r)
				if got := ctrlEnterInputKeys(messages); !reflect.DeepEqual(got, []tea.KeyMsg{test.key}) {
					t.Fatalf("split %d: keys = %#v, want %#v", split, got, test.key)
				}
				if len(ctrlEnterEvents(messages)) != 0 || shiftedSpaceEvents(messages) != 0 {
					t.Fatal("SS3 became an application shortcut")
				}
			}
		})
	}
}
