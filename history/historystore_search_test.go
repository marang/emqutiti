package history

import (
	"testing"
	"time"
)

// Test that HistoryStore.Search respects topic, time, and payload filters for
// active and archived messages.
func TestHistoryStoreSearch(t *testing.T) {
	now := time.Now()

	t.Run("active", func(t *testing.T) {
		hs := &store{}
		if err := hs.Append(Message{Timestamp: now.Add(-30 * time.Minute), Topic: "a", Payload: "foo", Kind: "pub", Retained: false}); err != nil {
			t.Fatalf("Append failed: %v", err)
		}
		if err := hs.Append(Message{Timestamp: now.Add(-2 * time.Hour), Topic: "b", Payload: "bar", Kind: "pub", Retained: false}); err != nil {
			t.Fatalf("Append failed: %v", err)
		}

		res := hs.Search(false, []string{"a"}, now.Add(-1*time.Hour), now, "")
		if len(res) != 1 || res[0].Topic != "a" {
			t.Fatalf("topic filter failed: %#v", res)
		}

		res = hs.Search(false, nil, now.Add(-1*time.Hour), now, "foo")
		if len(res) != 1 || res[0].Payload != "foo" {
			t.Fatalf("payload filter failed: %#v", res)
		}

		res = hs.Search(false, []string{"b"}, now.Add(-1*time.Hour), now, "")
		if len(res) != 0 {
			t.Fatalf("time filter failed: %#v", res)
		}
	})

	t.Run("archived", func(t *testing.T) {
		hs := &store{}
		if err := hs.Append(Message{Timestamp: now.Add(-30 * time.Minute), Topic: "a", Payload: "foo", Kind: "pub", Archived: true, Retained: false}); err != nil {
			t.Fatalf("Append failed: %v", err)
		}
		if err := hs.Append(Message{Timestamp: now.Add(-2 * time.Hour), Topic: "b", Payload: "bar", Kind: "pub", Archived: true, Retained: false}); err != nil {
			t.Fatalf("Append failed: %v", err)
		}

		res := hs.Search(true, []string{"a"}, now.Add(-1*time.Hour), now, "")
		if len(res) != 1 || res[0].Topic != "a" {
			t.Fatalf("topic filter failed: %#v", res)
		}

		res = hs.Search(true, nil, now.Add(-1*time.Hour), now, "foo")
		if len(res) != 1 || res[0].Payload != "foo" {
			t.Fatalf("payload filter failed: %#v", res)
		}

		res = hs.Search(true, []string{"b"}, now.Add(-1*time.Hour), now, "")
		if len(res) != 0 {
			t.Fatalf("time filter failed: %#v", res)
		}
	})
}

func TestFuzzyMatchTopic(t *testing.T) {
	tests := []struct {
		name     string
		topic    string
		patterns []string
		want     bool
	}{
		{
			name:     "empty patterns match all",
			topic:    "sensor/temperature",
			patterns: nil,
			want:     true,
		},
		{
			name:     "exact match",
			topic:    "sensor/temperature",
			patterns: []string{"sensor/temperature"},
			want:     true,
		},
		{
			name:     "fuzzy match partial",
			topic:    "sensor/temperature",
			patterns: []string{"temp"},
			want:     true,
		},
		{
			name:     "fuzzy match beginning",
			topic:    "sensor/temperature",
			patterns: []string{"sens"},
			want:     true,
		},
		{
			name:     "fuzzy match non-contiguous",
			topic:    "sensor/temperature",
			patterns: []string{"sntp"},
			want:     true,
		},
		{
			name:     "no match",
			topic:    "sensor/temperature",
			patterns: []string{"humidity"},
			want:     false,
		},
		{
			name:     "one of multiple patterns matches",
			topic:    "sensor/temperature",
			patterns: []string{"humidity", "temp"},
			want:     true,
		},
		{
			name:     "empty pattern in list ignored",
			topic:    "sensor/temperature",
			patterns: []string{"", "temp"},
			want:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := fuzzyMatchTopic(tt.topic, tt.patterns)
			if got != tt.want {
				t.Errorf("fuzzyMatchTopic(%q, %v) = %v, want %v", tt.topic, tt.patterns, got, tt.want)
			}
		})
	}
}

func TestHistoryStoreSearchFuzzy(t *testing.T) {
	now := time.Now()
	hs := &store{}

	// Add messages with various topics.
	topics := []string{
		"home/living-room/temperature",
		"home/living-room/humidity",
		"home/bedroom/temperature",
		"office/desk/light",
		"garage/door/status",
	}
	for i, topic := range topics {
		if err := hs.Append(Message{
			Timestamp: now.Add(-time.Duration(i) * time.Minute),
			Topic:     topic,
			Payload:   "test",
			Kind:      "pub",
		}); err != nil {
			t.Fatalf("Append failed: %v", err)
		}
	}

	t.Run("fuzzy matches temperature topics", func(t *testing.T) {
		res := hs.Search(false, []string{"temp"}, time.Time{}, time.Time{}, "")
		if len(res) != 2 {
			t.Fatalf("expected 2 temperature matches, got %d: %v", len(res), res)
		}
		for _, m := range res {
			if m.Topic != "home/living-room/temperature" && m.Topic != "home/bedroom/temperature" {
				t.Errorf("unexpected topic: %s", m.Topic)
			}
		}
	})

	t.Run("fuzzy matches living-room topics", func(t *testing.T) {
		res := hs.Search(false, []string{"living"}, time.Time{}, time.Time{}, "")
		if len(res) != 2 {
			t.Fatalf("expected 2 living-room matches, got %d: %v", len(res), res)
		}
	})

	t.Run("fuzzy matches with non-contiguous pattern", func(t *testing.T) {
		// "hlrt" should match "home/living-room/temperature"
		res := hs.Search(false, []string{"hlrt"}, time.Time{}, time.Time{}, "")
		if len(res) == 0 {
			t.Fatalf("expected fuzzy match for 'hlrt', got none")
		}
	})

	t.Run("exact match still works", func(t *testing.T) {
		res := hs.Search(false, []string{"garage/door/status"}, time.Time{}, time.Time{}, "")
		if len(res) != 1 || res[0].Topic != "garage/door/status" {
			t.Fatalf("exact match failed: %v", res)
		}
	})

	t.Run("no match returns empty", func(t *testing.T) {
		res := hs.Search(false, []string{"xyz123"}, time.Time{}, time.Time{}, "")
		if len(res) != 0 {
			t.Fatalf("expected no matches, got %d", len(res))
		}
	})
}
