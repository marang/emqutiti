package emqutiti

import (
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/marang/emqutiti/constants"
)

func initUI(order []string) uiState {
	vp := viewport.New(0, 0)
	fm := make(map[string]int, len(order))
	for i, id := range order {
		fm[id] = i
	}
	return uiState{
		focusIndex:  0,
		modeStack:   []constants.AppMode{constants.ModeClient},
		width:       0,
		height:      0,
		viewport:    vp,
		elemPos:     map[string]int{},
		focusOrder:  order,
		focusMap:    fm,
		focusMemory: map[constants.AppMode]int{},
	}
}

func initLayout() layoutConfig {
	return layoutConfig{
		message: boxConfig{height: 6},
		history: boxConfig{height: 10},
		topics:  boxConfig{height: 1},
		trace:   boxConfig{height: 10},
	}
}
