package history

import (
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/viewport"
)

// List returns the history list model.
func (h *Component) List() *list.Model { return &h.list }

// Items returns the current history items.
func (h *Component) Items() []Item { return h.items }

// SetItems replaces the current history items.
func (h *Component) SetItems(items []Item) { h.items = items }

// PreserveSelection carries existing marks and the range anchor across a filter refresh.
func (h *Component) PreserveSelection(items []Item) {
	selected := make(map[Item]bool)
	var anchor Item
	hasAnchor := h.selectionAnchor >= 0 && h.selectionAnchor < len(h.items)
	if hasAnchor {
		anchor = selectionKey(h.items[h.selectionAnchor])
	}
	for _, item := range h.items {
		if item.IsSelected != nil && *item.IsSelected {
			selected[selectionKey(item)] = true
		}
	}
	h.selectionAnchor = -1
	for i := range items {
		key := selectionKey(items[i])
		if selected[key] {
			v := true
			items[i].IsSelected = &v
		}
		if hasAnchor && key == anchor {
			h.selectionAnchor = i
		}
	}
}

// Store returns the underlying history store.
func (h *Component) Store() Store { return h.store }

// SetStore sets the history store.
func (h *Component) SetStore(s Store) { h.store = s }

// Detail returns the detail viewport model.
func (h *Component) Detail() *viewport.Model { return &h.detail }

// DetailItem returns the item shown in the detail viewport.
func (h *Component) DetailItem() Item { return h.detailItem }

// SetDetailItem sets the item shown in the detail viewport.
func (h *Component) SetDetailItem(it Item) { h.detailItem = it }

// ShowArchived reports whether archived messages are displayed.
func (h *Component) ShowArchived() bool { return h.showArchived }

// SetShowArchived toggles display of archived messages.
func (h *Component) SetShowArchived(v bool) { h.showArchived = v }

// FilterForm returns the active history filter form.
func (h *Component) FilterForm() *historyFilterForm { return h.filterForm }

// SetFilterForm sets the active history filter form.
func (h *Component) SetFilterForm(f *historyFilterForm) { h.filterForm = f }

// FilterQuery returns the current history filter query.
func (h *Component) FilterQuery() string { return h.filterQuery }

// SetFilterQuery sets the history filter query.
func (h *Component) SetFilterQuery(q string) { h.filterQuery = q }

// SelectionAnchor returns the current selection anchor index.
func (h *Component) SelectionAnchor() int { return h.selectionAnchor }

// SetSelectionAnchor sets the selection anchor index.
func (h *Component) SetSelectionAnchor(i int) {
	h.selectionAnchor = i
	h.captureSelectionBaseline()
}
