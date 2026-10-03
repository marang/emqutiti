package history

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/marang/emqutiti/internal/clipboardutil"
)

func TestFormatDetailPayloadJSON(t *testing.T) {
	payload := `{"foo":"bar","nested":{"a":1,"b":[true,false]}}`
	var expected bytes.Buffer
	if err := json.Indent(&expected, []byte(payload), "", "  "); err != nil {
		t.Fatalf("unexpected indent error: %v", err)
	}

	got := FormatDetailPayload(payload)
	if got != expected.String() {
		t.Fatalf("formatted payload mismatch\nexpected:\n%s\n\ngot:\n%s", expected.String(), got)
	}
}

func TestDetailFitsAndFullPayloadScrolls(t *testing.T) {
	for _, size := range dialogSizes {
		t.Run(fmt.Sprintf("%dx%d", size.width, size.height), func(t *testing.T) {
			h := NewComponent(&sizedModel{width: size.width, height: size.height}, nil)
			payload := strings.Repeat("payload line\n", 60) + strings.Repeat("x", 200) + "TAIL"
			h.SetDetailItem(Item{Payload: payload})
			h.Detail().Width, h.Detail().Height = size.width-4, size.height-4
			h.Detail().SetContent(payload)
			view := h.ViewDetail()
			if w, ht := lipgloss.Width(view), lipgloss.Height(view); w > size.width || ht != size.height {
				t.Fatalf("detail rendered %dx%d, want width <= %d and height %d", w, ht, size.width, size.height)
			}
			h.Detail().GotoBottom()
			view = h.ViewDetail()
			if !strings.Contains(view, "TAIL") {
				t.Fatal("full payload tail is not reachable by scrolling")
			}
			if !strings.Contains(view, "[esc]") || !strings.Contains(view, "copy") {
				t.Fatal("detail decisions disappeared at the bottom")
			}
		})
	}
}

func TestFormatDetailPayloadNonJSON(t *testing.T) {
	payload := "not json\ntext"
	if got := FormatDetailPayload(payload); got != payload {
		t.Fatalf("expected non-JSON payload to remain unchanged, got %q", got)
	}
}

func TestUpdateDetailCopyPayload(t *testing.T) {
	originalCopy := clipboardutil.Copy
	t.Cleanup(func() { clipboardutil.Copy = originalCopy })
	var copied string
	clipboardutil.Copy = func(s string) error {
		copied = s
		return nil
	}

	payload := `{"foo":"bar","nested":{"a":1,"b":[true,false]}}`
	h := NewComponent(stubModel{}, nil)
	h.SetDetailItem(Item{Payload: payload})
	h.Detail().SetContent(FormatDetailPayload(payload))

	h.UpdateDetail(tea.KeyMsg{Type: tea.KeyCtrlC})

	expected := FormatDetailPayload(payload)
	if copied != expected {
		t.Fatalf("expected payload copy:\n%s\n\ngot:\n%s", expected, copied)
	}
	items := h.Items()
	if len(items) != 1 {
		t.Fatalf("expected copy log entry, got %d items", len(items))
	}
	if items[0].Payload != "Copied detail payload" {
		t.Fatalf("expected copy log payload, got %q", items[0].Payload)
	}
}
