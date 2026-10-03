package ui

import "github.com/charmbracelet/lipgloss"

var (
	TextMuted = lipgloss.AdaptiveColor{Light: "240", Dark: "248"}
	TextMain  = lipgloss.AdaptiveColor{Light: "235", Dark: "255"}

	FocusedStyle = lipgloss.NewStyle().Foreground(ColPink)
	BlurredStyle = lipgloss.NewStyle().Foreground(TextMuted)
	CursorStyle  = FocusedStyle
	NoCursor     = lipgloss.NewStyle()

	Chip                = lipgloss.NewStyle().Padding(0, 1).MarginRight(1).Border(lipgloss.NormalBorder()).BorderForeground(TextMuted).Foreground(TextMain)
	ChipFocused         = Chip.BorderForeground(ColPink)
	ChipHovered         = Chip
	ChipInactive        = Chip
	ChipInactiveFocused = ChipFocused
	ChipInactiveHovered = ChipInactive
	ChipPublish         = Chip.Background(ColPink).Foreground(ColBlack).BorderStyle(lipgloss.InnerHalfBlockBorder()).BorderForeground(ColPink).BorderBackground(ColPink)
	ChipPublishFocused  = ChipPublish
	ChipPublishHovered  = ChipPublish

	InfoStyle       = lipgloss.NewStyle().Foreground(ColBlue).PaddingLeft(1)
	ErrorStyle      = lipgloss.NewStyle().Foreground(ColWarn).PaddingLeft(1)
	InfoSubtleStyle = lipgloss.NewStyle().Foreground(TextMuted).PaddingLeft(1)

	HelpStyle        = lipgloss.NewStyle().Foreground(ColCyan)
	HelpFocused      = HelpStyle.Foreground(ColDarkGray).Background(ColPink)
	HelpHovered      = HelpStyle.Foreground(ColDarkGray).Background(ColCyan)
	ContextHelpStyle = lipgloss.NewStyle().Foreground(TextMain).PaddingLeft(1)
	ContextHelpLead  = lipgloss.NewStyle().Foreground(ColCyan)

	// Form field styles
	FormLabel        = lipgloss.NewStyle().Foreground(ColBlue).Bold(true)
	FormLabelFocused = lipgloss.NewStyle().Foreground(ColPink).Bold(true)
	FormHelp         = lipgloss.NewStyle().Foreground(TextMuted).Italic(true)
	FormError        = lipgloss.NewStyle().Foreground(ColWarn)

	// Focus indicators
	ReadOnlyIndicator = lipgloss.NewStyle().Foreground(TextMuted).Italic(true)
)
