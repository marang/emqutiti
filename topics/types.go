package topics

const (
	idTopicsSubscribed   = "topics-subscribed"
	idTopicsUnsubscribed = "topics-unsubscribed"
	idTopic              = "topic"
)

// Item represents a topic with subscription and publish state.
type Item struct {
	Name       string
	Subscribed bool
	Publish    bool
}

func (t Item) FilterValue() string { return t.Name }
func (t Item) Title() string       { return t.Name }
func (t Item) Description() string {
	status := "unsubscribed"
	if t.Subscribed {
		status = "subscribed"
	}
	if t.Publish {
		status += ", publish"
	}
	return status
}

type ChipBound struct {
	XPos, YPos    int
	Width, Height int
	Index         int
}

type paneManager interface{ SetActivePane(int) }

type paneState struct {
	sel     int
	page    int
	m       paneManager
	index   int
	focused bool
}

func (p *paneState) Focus() {
	p.focused = true
	if p.m != nil {
		p.m.SetActivePane(p.index)
	}
}

func (p *paneState) Blur() { p.focused = false }

func (p paneState) IsFocused() bool { return p.focused }

func (p paneState) View() string { return "" }

type topicsPanes struct {
	subscribed   paneState
	unsubscribed paneState
	active       int
}
