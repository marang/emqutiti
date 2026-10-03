package history

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/marang/emqutiti/ui"
)

// Scroll delegates mouse wheel handling to the configured scroller.
func (h *Component) Scroll(msg tea.MouseMsg) tea.Cmd { return h.sc.Scroll(msg) }

// CanScroll reports whether the configured scroller can scroll.
func (h *Component) CanScroll() bool { return h.sc.CanScroll() }

// HandleSelection updates history selection based on index and shift key.
func (h *Component) HandleSelection(idx int, shift bool) {
	if idx < 0 || idx >= len(h.items) {
		return
	}
	h.list.Select(idx)
	if shift {
		if h.selectionAnchor == -1 {
			h.SetSelectionAnchor(h.list.Index())
			if h.selectionAnchor >= 0 && h.selectionAnchor < len(h.items) {
				v := true
				h.items[h.selectionAnchor].IsSelected = &v
			}
		}
		// Mouse ranges add to existing marks; keyboard ranges can still contract.
		h.captureSelectionBaseline()
		h.updateSelectionRange(idx)
	} else {
		for i := range h.items {
			h.items[i].IsSelected = nil
		}
		h.SetSelectionAnchor(-1)
	}
}

// HandleClick selects a rendered row at client-content coordinates, including
// the two-column inset and the optional filter summary above the list.
func (h *Component) HandleClick(msg tea.MouseMsg, top, vpYOffset int) {
	y := msg.Y - top + vpYOffset
	if h.filterQuery != "" {
		y--
	}
	idx := ui.ListRowAt(h.list, msg.X-2, y, 2, 0)
	if idx >= 0 {
		h.HandleSelection(idx, msg.Shift)
	}
}

// UpdateSelectionRange selects history entries from the anchor to idx.
func (h *Component) UpdateSelectionRange(idx int) { h.updateSelectionRange(idx) }

func (h *Component) updateSelectionRange(idx int) {
	if idx < 0 || idx >= len(h.items) || h.selectionAnchor < 0 || h.selectionAnchor >= len(h.items) {
		return
	}
	start := h.selectionAnchor
	end := idx
	if start > end {
		start, end = end, start
	}
	for i := range h.items {
		h.items[i].IsSelected = nil
		if h.selectionBaseline[selectionKey(h.items[i])] {
			v := true
			h.items[i].IsSelected = &v
		}
	}
	for i := start; i <= end && i < len(h.items); i++ {
		v := true
		h.items[i].IsSelected = &v
	}
}

func selectionKey(item Item) Item {
	item.Timestamp = item.Timestamp.Round(0).UTC()
	item.IsSelected, item.IsMarkedForDeletion = nil, nil
	return item
}

func (h *Component) captureSelectionBaseline() {
	h.selectionBaseline = make(map[Item]bool)
	if h.selectionAnchor < 0 {
		return
	}
	for _, item := range h.items {
		if item.IsSelected != nil && *item.IsSelected {
			h.selectionBaseline[selectionKey(item)] = true
		}
	}
}
