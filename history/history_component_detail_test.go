package history

import (
	"bytes"
	"encoding/json"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

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
