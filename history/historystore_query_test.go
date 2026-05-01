package history

import (
	"reflect"
	"testing"
	"time"
)

func TestParseQuery(t *testing.T) {
	cases := []struct {
		query   string
		topics  []string
		start   string
		end     string
		payload string
	}{
		{
			query:   "",
			topics:  nil,
			start:   "",
			end:     "",
			payload: "",
		},
		{
			query:   "topic=a start=2025-07-28T18:25:21Z hello world",
			topics:  []string{"a"},
			start:   "2025-07-28T18:25:21Z",
			end:     "",
			payload: "hello world",
		},
		{
			query:   "start=2025-07-28T18:25:21Z end=2025-07-28T20:00:00Z payload=foo",
			topics:  nil,
			start:   "2025-07-28T18:25:21Z",
			end:     "2025-07-28T20:00:00Z",
			payload: "foo",
		},
		{
			query:   "foo bar topic=a,b end=2025-07-28T21:00:00Z",
			topics:  []string{"a", "b"},
			start:   "",
			end:     "2025-07-28T21:00:00Z",
			payload: "foo bar",
		},
		{
			query:   "payload=first topic=c start=2025-07-28T18:25:21Z second",
			topics:  []string{"c"},
			start:   "2025-07-28T18:25:21Z",
			end:     "",
			payload: "first second",
		},
	}

	for _, c := range cases {
		gotTopics, gotStart, gotEnd, gotPayload := ParseQuery(c.query)

		if !reflect.DeepEqual(gotTopics, c.topics) {
			t.Errorf("topics mismatch for %q: %v != %v", c.query, gotTopics, c.topics)
		}

		wantStart, _ := time.Parse(time.RFC3339, c.start)
		wantEnd, _ := time.Parse(time.RFC3339, c.end)

		if !gotStart.Equal(wantStart) {
			if !(gotStart.IsZero() && c.start == "") {
				t.Errorf("start time mismatch for %q: %v != %v", c.query, gotStart, wantStart)
			}
		}

		if !gotEnd.Equal(wantEnd) {
			if !(gotEnd.IsZero() && c.end == "") {
				t.Errorf("end time mismatch for %q: %v != %v", c.query, gotEnd, wantEnd)
			}
		}

		if gotPayload != c.payload {
			t.Errorf("payload mismatch for %q: %q != %q", c.query, gotPayload, c.payload)
		}
	}
}

func TestApplyFilterArchived(t *testing.T) {
	hs := &store{}
	ts := time.Now()
	if err := hs.Append(Message{Timestamp: ts, Topic: "t1", Payload: "active", Kind: "pub", Retained: false}); err != nil {
		t.Fatalf("Append failed: %v", err)
	}
	if err := hs.Append(Message{Timestamp: ts.Add(time.Second), Topic: "t2", Payload: "arch", Kind: "pub", Archived: true, Retained: false}); err != nil {
		t.Fatalf("Append failed: %v", err)
	}

	items, _ := ApplyFilter("", hs, false)
	if len(items) != 1 || items[0].Archived {
		t.Fatalf("expected 1 unarchived item, got %v", items)
	}

	items, _ = ApplyFilter("", hs, true)
	if len(items) != 1 || !items[0].Archived {
		t.Fatalf("expected 1 archived item, got %v", items)
	}
}

func TestAppendWhileFilteredUsesFilteredStoreResults(t *testing.T) {
	hs := &store{}
	h := NewComponent(stubModel{}, hs)
	h.filterQuery = "payload=match"
	ts := time.Now()
	if err := hs.Append(Message{Timestamp: ts, Topic: "t1", Payload: "match", Kind: "pub"}); err != nil {
		t.Fatalf("Append failed: %v", err)
	}

	h.Append("t1", "match", "pub", false, "match")

	if got := len(h.Items()); got != 2 {
		t.Fatalf("expected two matching items without duplicate append, got %d: %#v", got, h.Items())
	}
	if h.Items()[0].Timestamp.Equal(h.Items()[1].Timestamp) {
		t.Fatalf("expected distinct stored matching items, got duplicate timestamps")
	}
}

func TestAppendWhileFilteredExcludesNonMatchingMessage(t *testing.T) {
	hs := &store{}
	h := NewComponent(stubModel{}, hs)
	h.filterQuery = "payload=match"

	h.Append("t1", "miss", "pub", false, "miss")

	if got := len(h.Items()); got != 0 {
		t.Fatalf("expected non-matching append hidden by active filter, got %d: %#v", got, h.Items())
	}
}
