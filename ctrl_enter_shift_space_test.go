package emqutiti

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/marang/emqutiti/constants"
	"github.com/marang/emqutiti/history"
)

func shiftedSpaceEvents(messages []tea.Msg) int {
	count := 0
	for _, msg := range messages {
		if _, ok := msg.(shiftedSpaceMsg); ok {
			count++
		}
	}
	return count
}

func TestShiftedSpaceActualTeaParserEverySplit(t *testing.T) {
	input := "\x1b[32;2u\x1b[13;5u \x04"
	for split := 1; split < len(input); split++ {
		r := &ctrlEnterChunkReader{chunks: [][]byte{[]byte(input[:split]), []byte(input[split:])}}
		messages := captureCtrlEnterInput(t, r)
		var order []string
		for _, msg := range messages {
			switch msg := msg.(type) {
			case shiftedSpaceMsg:
				order = append(order, "shift+space")
			case ctrlEnterMsg:
				order = append(order, "ctrl+enter")
			case tea.KeyMsg:
				order = append(order, msg.String())
			}
		}
		if got := strings.Join(order, "|"); got != "shift+space|ctrl+enter| " {
			t.Fatalf("split %d: %q", split, got)
		}
	}
}

func TestShiftedSpaceRejectsOtherEncodingsAndPaste(t *testing.T) {
	for _, raw := range []string{
		" ", "shift+space", "\x1b[32u", "\x1b[32;1u", "\x1b[32;3u", "\x1b[32;6u",
		"\x1b[32;2:1u", "\x1b[32;2:2u", "\x1b[32;2:3u", "\x1b[32;2;99u",
		"\x1b[32;2 u", "\x1b[27;2;32~",
		"\x1b]unknown\x1b[32;2u", "\x1bPunknown\x1b[32;2u",
	} {
		messages := captureCtrlEnterInput(t, strings.NewReader(raw+"\x04"))
		if count := shiftedSpaceEvents(messages); count != 0 {
			t.Fatalf("accepted %q: %d events", raw, count)
		}
	}
	if _, ok := decodeShiftedSpace([]byte("\x1b[32;2uX")); ok {
		t.Fatal("decoder accepted more than one frame")
	}
	input := "\x1b[200~ \x1b[32;2u shift+space\x1b[201~\x04"
	var chunks [][]byte
	for _, b := range []byte(input) {
		chunks = append(chunks, []byte{b})
	}
	if messages := captureCtrlEnterInput(t, &ctrlEnterChunkReader{chunks: chunks}); shiftedSpaceEvents(messages) != 0 {
		t.Fatal("paste became Shift+Space")
	}
}

func TestShiftedSpaceDispatchScopeAndToggle(t *testing.T) {
	m := reviewFixture(t, 80, 24)
	items := m.history.Items()
	for i := range items {
		items[i].IsSelected = nil
	}
	m.history.SetItems(items)
	m.history.List().Select(1)
	selected := func() bool {
		item := m.history.Items()[1]
		return item.IsSelected != nil && *item.IsSelected
	}
	renderedSelected := func() bool {
		t.Helper()
		if m.history.List().Index() != 1 {
			t.Fatal("list synchronization moved the current row")
		}
		item, ok := m.history.List().Items()[1].(history.Item)
		if !ok {
			t.Fatal("unexpected History list item type")
		}
		return item.IsSelected != nil && *item.IsSelected
	}
	toggle := func() {
		t.Helper()
		if _, handled := m.handleShiftedSpaceMsg(shiftedSpaceMsg{}); !handled {
			t.Fatal("typed event was not consumed")
		}
	}
	m.ui.modeStack = []constants.AppMode{constants.ModeConnections}
	m.SetFocus(idHistory)
	toggle()
	if selected() {
		t.Fatal("manager mode changed selection")
	}
	m.ui.modeStack = []constants.AppMode{constants.ModeClient}
	for _, focus := range []string{idMessage, idTopic, idTopics} {
		m.SetFocus(focus)
		toggle()
		if selected() || m.pendingPublishes() != 0 {
			t.Fatalf("focus %s changed selection or published", focus)
		}
	}
	m.SetFocus(idHistory)
	m.ui.panelResize = panelResizeState{id: idHistory}
	toggle()
	if selected() {
		t.Fatal("active drag changed selection")
	}
	m.ui.panelResize = panelResizeState{}
	m.history.SetShowArchived(true)
	toggle()
	if selected() {
		t.Fatal("archived History changed selection")
	}
	m.history.SetShowArchived(false)
	toggle()
	if !selected() || !renderedSelected() || m.history.SelectionAnchor() != 1 {
		t.Fatal("focused History did not toggle current row and anchor")
	}
	toggle()
	if selected() || renderedSelected() {
		t.Fatal("second press did not deselect current row")
	}
	if _, handled := m.handleShiftedSpaceMsg(tea.KeyMsg{Type: tea.KeySpace}); handled {
		t.Fatal("ordinary space was aliased")
	}
}
