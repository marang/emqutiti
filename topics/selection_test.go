package topics

import (
	"fmt"
	"testing"
)

func TestRebuildActiveTopicListPreservesClientSelection(t *testing.T) {
	for _, pane := range []int{0, 1} {
		for _, subscribed := range []bool{false, true} {
			t.Run(fmt.Sprintf("pane=%d/subscribed=%t", pane, subscribed), func(t *testing.T) {
				c := newTestComponent()
				c.Items = []Item{{Name: "df", Subscribed: true}, {Name: "testx", Subscribed: subscribed}}
				c.RebuildActiveTopicList()
				c.SetActivePane(pane)
				c.SetSelected(1)
				c.SortTopics()
				if c.Items[c.Selected()].Name != "testx" {
					t.Fatal("sorting changed selected topic")
				}
				c.RebuildActiveTopicList()
				selected := c.Selected()
				if selected < 0 || c.Items[selected].Name != "testx" {
					t.Fatalf("rebuilding manager list changed client selection: %d", selected)
				}
			})
		}
	}
}

func TestRebuildActiveTopicListKeepsManagerSelectionInActivePane(t *testing.T) {
	c, _ := managerComponent()
	c.list.Select(1)
	c.syncSelection()
	c.Items[1].Subscribed = false
	c.SortTopics()
	c.RebuildActiveTopicList()
	if selected := c.Selected(); selected < 0 || !c.Items[selected].Subscribed || c.Items[selected].Name != c.list.SelectedItem().(Item).Name {
		t.Fatalf("manager selection left the active subscribed pane: %d", selected)
	}
}
