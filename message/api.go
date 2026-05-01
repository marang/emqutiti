package message

// ID identifies the message input for focus management.
const ID = "message"

// Model defines the host model behavior required by the Component.
type Model interface {
	Width() int
	MessageHeight() int
	FocusedID() string
	HoveredID() string
	MessageTargetPreview() string
	OverlayHelp(view string) string
}
