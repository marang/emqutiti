package emqutiti

import (
	"bytes"
	"crypto/rand"
	"errors"
	"io"
	"math/big"
	"reflect"
	"sync"
	"unicode/utf8"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/ansi/parser"
)

var errCtrlEnterInputTimeout = errors.New("extended-key input timeout")

const ctrlEnterPasteStart = "\x1b[200~"

// ctrlEnterInputReader frames short terminal events before Bubble Tea's
// 256-byte reader, which otherwise treats arbitrary short reads as boundaries.
// next must bound continuation reads; an idle ESC must still reach the UI.
// Original input bytes are preserved. Internal diagnostic delimiters are
// removed by filter; no physical keys or out-of-band event sends are used.
type ctrlEnterInputReader struct {
	next        func([]byte, bool) (int, error)
	input       []byte
	inputErr    error
	frame       []byte
	parser      *ansi.Parser
	paste       bool
	pasteMatch  int
	marker      []byte
	current     []byte
	ack         chan struct{}
	done        chan struct{}
	closeOnce   sync.Once
	waitAck     bool
	allowKey    bool
	directAck   string
	runeAfter   bool
	pendingRune bool
}

func newCtrlEnterInputReader(next func([]byte, bool) (int, error)) *ctrlEnterInputReader {
	p := ansi.NewParser()
	p.SetDataSize(1)
	// A private diagnostic CSI is a frame delimiter, not a physical key alias.
	// Its random nonce prevents terminal input from spoofing the handshake.
	nonce := new(big.Int).SetBytes([]byte(rand.Text())).String()
	return &ctrlEnterInputReader{
		next: next, parser: p, marker: []byte("\x1b[?" + nonce + "z"),
		ack: make(chan struct{}, 1), done: make(chan struct{}),
	}
}

func (r *ctrlEnterInputReader) close() {
	r.closeOnce.Do(func() { close(r.done) })
}

// filter runs serially in Tea's update loop. Hold the next read until the
// delimiter is consumed so Tea cannot overwrite a borrowed unknown-CSI slice.
// Only the exact original frame can become a key event, not a CSI nested in an
// unknown/malformed control string. Unrelated diagnostics retain their type.
func (r *ctrlEnterInputReader) filter(_ tea.Model, msg tea.Msg) tea.Msg {
	raw, ok := ctrlEnterCSIBytes(msg)
	if !ok {
		return msg
	}
	if bytes.Equal(raw, r.marker) {
		r.ack <- struct{}{}
		return nil
	}
	if r.allowKey && bytes.Equal(raw, r.current) {
		if event, ok := normalizeCtrlEnterMsg(msg).(ctrlEnterMsg); ok {
			return event
		}
		if event, ok := decodeShiftedSpace(raw); ok {
			return event
		}
		if key, ok := decodeKittyKey(raw); ok {
			return key
		}
	}
	v := reflect.ValueOf(msg)
	copy := reflect.MakeSlice(v.Type(), v.Len(), v.Len())
	reflect.Copy(copy, v)
	if r.directAck != "" && bytes.Equal(raw, []byte(r.directAck)) {
		// A truncated legacy event cannot safely take a delimiter. Its sole
		// borrowed diagnostic acknowledges the frame after copying; its
		// remaining bytes cannot contain another CSI.
		r.ack <- struct{}{}
	}
	return copy.Interface()
}

func (r *ctrlEnterInputReader) readByte(continuation bool) (byte, error) {
	for len(r.input) == 0 {
		if r.inputErr != nil {
			return 0, r.inputErr
		}
		var buf [256]byte
		n, err := r.next(buf[:], continuation)
		if n < 0 || n > len(buf) {
			return 0, errors.New("invalid terminal read count")
		}
		if n == 0 && err == nil {
			return 0, io.ErrNoProgress
		}
		r.input = append(r.input[:0], buf[:n]...)
		if errors.Is(err, errCtrlEnterInputTimeout) {
			if n == 0 {
				return 0, err
			}
		} else {
			r.inputErr = err
		}
	}
	b := r.input[0]
	r.input = r.input[1:]
	return b, nil
}

func (r *ctrlEnterInputReader) Read(dst []byte) (int, error) {
	if len(dst) == 0 {
		return 0, nil
	}
	select {
	case <-r.done:
		return 0, io.EOF
	default:
	}
	if r.waitAck {
		select {
		case <-r.ack:
			r.waitAck = false
		case <-r.done:
			return 0, io.EOF
		}
	}
	if len(r.frame) == 0 || r.pendingRune {
		r.directAck = ""
		r.runeAfter = false
		r.pendingRune = false
		fresh := r.parser.State() == parser.GroundState
		if err := r.readFrame(); err != nil {
			return 0, err
		}
		if r.pendingRune {
			// Keep at most one unfinished UTF-8 rune until the next read.
			// Tea accepts an empty read; cancelreader can poll/cancel again.
			return 0, nil
		}
		// Focus reports require an exact buffer match in Tea, and their
		// value messages do not borrow the read buffer or need an ack.
		focusReport := !r.runeAfter && (bytes.Equal(r.frame, []byte("\x1b[I")) || bytes.Equal(r.frame, []byte("\x1b[O")))
		switch {
		case len(r.frame) < 6 && bytes.HasPrefix(r.frame, []byte("\x1b[M")):
			r.directAck = "\x1b[M"
		case bytes.Equal(r.frame, []byte("\x1b[O\x1b")), bytes.Equal(r.frame, []byte("\x1b[[\x1b")):
			r.directAck = string(r.frame[:3])
		}
		if !r.paste && !focusReport && (r.directAck != "" || r.frame[len(r.frame)-1] != '\x1b' && bytes.ContainsRune(r.frame, '\x1b')) {
			r.current = append(r.current[:0], r.frame...)
			r.allowKey = fresh && r.parser.State() == parser.GroundState
			if r.directAck == "" {
				r.frame = append(r.frame, r.marker...)
			}
			r.waitAck = true
		}
	}
	// Never fragment a keyboard/paste marker because of the consumer's buffer.
	if len(dst) < len(r.frame) {
		// The frame has not been delivered, so its delimiter cannot be acked.
		r.waitAck = false
		return 0, io.ErrShortBuffer
	}
	if len(r.current) > 0 && (r.directAck != "" || bytes.HasSuffix(r.frame, r.marker)) {
		r.waitAck = true
	}
	n := copy(dst, r.frame)
	r.frame = r.frame[:0]
	return n, nil
}

func (r *ctrlEnterInputReader) readFrame() error {
	const frameLimit = 128
	const pasteEnd = "\x1b[201~"
	// Keep an opener atomic even when a malformed control-string prefix puts
	// it across the chunk boundary. The extension is at most five bytes.
	for len(r.frame) < frameLimit || (!r.paste && len(r.frame) < frameLimit+len(ctrlEnterPasteStart)-1 && ctrlEnterPasteStartPending(r.frame)) {
		b, err := r.readByte(len(r.frame) > 0)
		if err != nil {
			if len(r.frame) > 0 {
				if errors.Is(err, errCtrlEnterInputTimeout) && r.frame[0] != '\x1b' && !utf8.FullRune(r.frame) {
					r.pendingRune = true
					return nil
				}
				r.parser.Reset()
				return nil // deliver bytes before an accompanying error, including EOF
			}
			return err
		}
		r.frame = append(r.frame, b)
		if r.paste {
			if b == pasteEnd[r.pasteMatch] {
				r.pasteMatch++
			} else if b == pasteEnd[0] {
				r.pasteMatch = 1
			} else {
				r.pasteMatch = 0
			}
			if r.pasteMatch == len(pasteEnd) {
				r.paste, r.pasteMatch = false, 0
				return nil
			}
			// Bubble Tea accumulates these bounded chunks into one paste.
			continue
		}
		if r.frame[0] != '\x1b' && r.parser.State() == parser.GroundState {
			if utf8.FullRune(r.frame) {
				return nil
			}
			continue
		}
		r.parser.Advance(b)
		// Tea observes paste markers even within malformed control strings.
		// Its paste boundary is independent of the ANSI parser's state.
		if bytes.HasSuffix(r.frame, []byte(ctrlEnterPasteStart)) {
			r.paste = true
			r.parser.Reset()
			return nil
		}
		if r.parser.State() == parser.GroundState {
			// Tea's SS3 and legacy CSI-O/Linux-console keys carry one more
			// byte than ANSI recognizes. Their Alt forms prefix another ESC.
			if ctrlEnterNeedsKeySuffix(r.frame) {
				b, err := r.readByte(true)
				if err == nil {
					if b >= utf8.RuneSelf {
						// A non-key rune belongs to the next frame, including
						// all of its continuation bytes.
						r.input = append([]byte{b}, r.input...)
						r.runeAfter = true
						return nil
					}
					r.frame = append(r.frame, b)
					// A malformed SS3 may end where a new escape starts.
					// Keep that escape attached to its following sequence.
					if b == '\x1b' {
						r.parser.Advance(b)
						continue
					}
				}
			}
			// X10 mouse's three coordinates are not part of the ANSI CSI.
			if bytes.Equal(r.frame, []byte("\x1b[M")) {
				for range 3 {
					b, err := r.readByte(true)
					if err != nil {
						return nil
					}
					r.frame = append(r.frame, b)
				}
			}
			return nil
		}
	}
	return nil
}

func ctrlEnterNeedsKeySuffix(frame []byte) bool {
	if bytes.HasPrefix(frame, []byte("\x1b\x1b")) {
		frame = frame[1:]
	}
	return bytes.Equal(frame, []byte("\x1bO")) ||
		bytes.Equal(frame, []byte("\x1b[O")) ||
		bytes.Equal(frame, []byte("\x1b[["))
}

func ctrlEnterPasteStartPending(frame []byte) bool {
	for length := 1; length < len(ctrlEnterPasteStart); length++ {
		if bytes.HasSuffix(frame, []byte(ctrlEnterPasteStart[:length])) {
			return true
		}
	}
	return false
}
