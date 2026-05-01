package emqutiti

import (
	"fmt"
	"strings"
	"testing"

	"github.com/marang/emqutiti/topics"
)

func TestRenderTopicChipsEmpty(t *testing.T) {
	chips := renderTopicChips(nil, 0, -1, 80)
	if len(chips) != 0 {
		t.Fatalf("expected 0 chips, got %d", len(chips))
	}
}

func TestRenderTopicChipsLarge(t *testing.T) {
	items := make([]topics.Item, 100)
	for i := range items {
		items[i] = topics.Item{Name: fmt.Sprintf("t%d", i)}
	}
	chips := renderTopicChips(items, 50, -1, 80)
	if len(chips) != len(items) {
		t.Fatalf("expected %d chips, got %d", len(items), len(chips))
	}
}

func TestRenderTopicChipsDoNotPrefixModes(t *testing.T) {
	items := []topics.Item{
		{Name: "read", Subscribed: true},
		{Name: "write", Publish: true},
		{Name: "both", Subscribed: true, Publish: true},
		{Name: "off"},
	}
	chips := renderTopicChips(items, 0, -1, 80)
	for i, prefix := range []string{"r read", "w write", "rw both", "x off"} {
		if strings.Contains(chips[i], prefix) {
			t.Fatalf("chip %d should not include mode prefix %q: %q", i, prefix, chips[i])
		}
	}
}

func TestLayoutTopicViewportEmpty(t *testing.T) {
	m, _ := initialModel(nil)
	m.ui.width = 80
	content, bounds, boxH, infoH, scroll := m.layoutTopicViewport(nil)
	if content == "" {
		t.Fatalf("expected content with info lines")
	}
	if len(bounds) != 0 {
		t.Fatalf("expected no bounds, got %d", len(bounds))
	}
	if boxH <= 0 || infoH != 1 {
		t.Fatalf("unexpected box or info height")
	}
	if scroll >= 0 {
		t.Fatalf("expected negative scroll for empty content")
	}
}

func TestLayoutTopicViewportLarge(t *testing.T) {
	m, _ := initialModel(nil)
	m.ui.width = 80
	items := make([]topics.Item, 200)
	for i := range items {
		items[i] = topics.Item{Name: fmt.Sprintf("t%d", i), Subscribed: true}
	}
	chips := renderTopicChips(items, 0, -1, m.ui.width-4)
	content, bounds, _, _, scroll := m.layoutTopicViewport(chips)
	if content == "" {
		t.Fatalf("expected content for large list")
	}
	if len(bounds) == 0 {
		t.Fatalf("expected bounds for large list")
	}
	if scroll < 0 {
		t.Fatalf("expected non-negative scroll for large list")
	}
}

func TestBuildTopicBoxesEmpty(t *testing.T) {
	m, _ := initialModel(nil)
	m.ui.width = 80
	topicsBox, _ := m.buildTopicBoxes("content", 1, 2, -1)
	if !strings.Contains(topicsBox, "Topics 0/0") {
		t.Fatalf("expected label 'Topics 0/0', got %q", topicsBox)
	}
}

func TestBuildTopicBoxesLarge(t *testing.T) {
	m, _ := initialModel(nil)
	m.ui.width = 80
	items := make([]topics.Item, 50)
	for i := range items {
		items[i] = topics.Item{Name: fmt.Sprintf("t%d", i), Subscribed: i%2 == 0}
	}
	m.topics.Items = items
	topicsBox, _ := m.buildTopicBoxes("content", 1, 2, 0)
	if !strings.Contains(topicsBox, "Topics 25/50") {
		t.Fatalf("expected label 'Topics 25/50', got %q", topicsBox)
	}
}
