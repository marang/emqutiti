package emqutiti

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/marang/emqutiti/topics"
)

func TestSelectedTopicWrapsLongName(t *testing.T) {
	longName := strings.Repeat("x", 50)
	chips := renderTopicChips([]topics.Item{{Name: longName, Subscribed: true}}, 0, -1, 20)
	stripped := ansi.Strip(chips[0])
	if strings.Contains(stripped, "…") {
		t.Fatalf("expected no ellipsis for selected topic")
	}
	if strings.Count(stripped, "x") != len(longName) {
		t.Fatalf("expected full topic name rendered")
	}
}

func TestUnselectedTopicTruncated(t *testing.T) {
	longName := strings.Repeat("x", 50)
	items := []topics.Item{{Name: "short", Subscribed: true}, {Name: longName, Subscribed: true}}
	chips := renderTopicChips(items, 0, -1, 20)
	if !strings.Contains(ansi.Strip(chips[1]), "…") {
		t.Fatalf("expected truncated chip for unselected topic")
	}
}

func TestSelectedTopicWrapsWide(t *testing.T) {
	longName := strings.Repeat("x", maxTopicChipWidth+10)
	chips := renderTopicChips([]topics.Item{{Name: longName, Subscribed: true}}, 0, -1, 120)
	stripped := ansi.Strip(chips[0])
	if strings.Contains(stripped, "…") {
		t.Fatalf("expected no ellipsis for selected topic")
	}
	if strings.Count(stripped, "x") != len(longName) {
		t.Fatalf("expected full topic name rendered")
	}
}
