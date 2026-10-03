//go:build linux || darwin

package emqutiti

import (
	"errors"
	"os"
	"testing"
	"time"
)

func ctrlEnterPipeInput(t *testing.T) (*ctrlEnterTerminalInput, *os.File) {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { r.Close() })
	t.Cleanup(func() { w.Close() })
	return &ctrlEnterTerminalInput{File: r}, w
}

func TestCtrlEnterUnixInputDoesNotPrefetch(t *testing.T) {
	f, w := ctrlEnterPipeInput(t)
	if err := f.File.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write([]byte("ab")); err != nil {
		t.Fatal(err)
	}
	buf := make([]byte, 256)
	if n, err := f.readNext(buf, false); n != 1 || err != nil || buf[0] != 'a' {
		t.Fatalf("first read = %d, %v, %q", n, err, buf[:n])
	}
	// The second byte must remain readable on the underlying descriptor so
	// Tea's cancelreader can poll it without waiting for another physical key.
	if n, err := f.File.Read(buf); n != 1 || err != nil || buf[0] != 'b' {
		t.Fatalf("underlying read = %d, %v, %q", n, err, buf[:n])
	}
}

func TestCtrlEnterUnixInputContinuationDeadline(t *testing.T) {
	f, w := ctrlEnterPipeInput(t)
	buf := make([]byte, 256)
	start := time.Now()
	if n, err := f.readNext(buf, true); n != 0 || !errors.Is(err, errCtrlEnterInputTimeout) {
		t.Fatalf("idle continuation = %d, %v", n, err)
	}
	if f.deadline.IsZero() || time.Since(start) > time.Second {
		t.Fatalf("continuation deadline = %v after %v", f.deadline, time.Since(start))
	}
	deadline := f.deadline
	if _, err := w.Write([]byte("ab")); err != nil {
		t.Fatal(err)
	}
	if n, err := f.readNext(buf, true); n != 0 || !errors.Is(err, errCtrlEnterInputTimeout) || !f.deadline.Equal(deadline) {
		t.Fatalf("expired continuation = %d, %v, deadline %v", n, err, f.deadline)
	}
	if n, err := f.readNext(buf, false); n != 1 || err != nil || buf[0] != 'a' || !f.deadline.IsZero() {
		t.Fatalf("fresh read = %d, %v, deadline %v", n, err, f.deadline)
	}
	if n, err := f.readNext(buf, true); n != 1 || err != nil || buf[0] != 'b' || f.deadline.IsZero() {
		t.Fatalf("ready continuation = %d, %v, deadline %v", n, err, f.deadline)
	}
}
