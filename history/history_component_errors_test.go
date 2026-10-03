package history

import (
	"bytes"
	"errors"
	"io"
	"log"
	"os"
	"strings"
	"testing"

	"github.com/marang/emqutiti/proxy"
)

type appendFailStore struct{ store }

func (*appendFailStore) Append(Message) error { return errors.New("storage unavailable") }

func TestAppendFailureLogsWithoutStdout(t *testing.T) {
	var logs bytes.Buffer
	previousLog := log.Writer()
	log.SetOutput(&logs)
	t.Cleanup(func() { log.SetOutput(previousLog) })
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	previousStdout := os.Stdout
	os.Stdout = w
	t.Cleanup(func() { os.Stdout = previousStdout; r.Close(); w.Close() })
	h := NewComponent(stubModel{}, &appendFailStore{})
	h.Append("topic", "payload", "pub", false, "")
	w.Close()
	os.Stdout = previousStdout
	output, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	if len(output) != 0 {
		t.Errorf("history failure wrote to stdout: %q", output)
	}
	want := "history append error: storage unavailable"
	if !strings.Contains(logs.String(), want) {
		t.Errorf("application log missing error: %q", logs.String())
	}
	if items := h.Items(); len(items) != 2 || items[1].Kind != "log" || items[1].Payload != want {
		t.Fatalf("history failure did not append a log item: %#v", items)
	}
	logs.Reset()
	h.SetShowArchived(true)
	h.Append("topic", "payload", "pub", false, "")
	if !strings.Contains(logs.String(), want) {
		t.Fatal("history failure hidden by archived view must still reach application log")
	}
}

func TestAppendLogTextPersistsThroughProxyReopen(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("EMQUTITI_HOME", dir)
	p, err := proxy.StartProxy("127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	previousAddr := proxyAddr
	SetProxyAddr(p.Addr())
	t.Cleanup(func() { p.Stop(); SetProxyAddr(previousAddr) })
	st, err := OpenStore("diagnostics")
	if err != nil {
		t.Fatal(err)
	}
	h := NewComponent(stubModel{}, st)
	want := "Publish failed for output: disconnected"
	h.Append("output", "", "log", false, want)
	filtered, _ := ApplyFilter("payload=Publish failed", st, false)
	if len(filtered) != 1 || filtered[0].Payload != want {
		t.Fatalf("persisted diagnostic lost after search: %+v", filtered)
	}
	if err := st.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := OpenStore("diagnostics")
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	loaded := NewComponent(stubModel{}, reopened)
	if items := loaded.Items(); len(items) != 1 || items[0].Payload != want {
		t.Fatalf("diagnostic lost after store reopen: %+v", items)
	}
}
