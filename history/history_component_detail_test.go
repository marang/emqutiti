package history

import (
	"bytes"
	"encoding/json"
	"testing"
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
