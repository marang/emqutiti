package ui

import "github.com/charmbracelet/lipgloss"

var (
	FocusedStyle = lipgloss.NewStyle().Foreground(ColPink)
	BlurredStyle = lipgloss.NewStyle().Foreground(ColGray)
	CursorStyle  = FocusedStyle
	NoCursor     = lipgloss.NewStyle()

	Chip                = lipgloss.NewStyle().Padding(0, 1).MarginRight(1).Border(lipgloss.NormalBorder()).BorderForeground(ColBlue)
	ChipFocused         = Chip.BorderTopForeground(ColPink).BorderLeftForeground(ColPink).Foreground(ColPink)
	ChipHovered         = Chip
	ChipInactive        = Chip.BorderForeground(ColGray).Foreground(ColGray)
	ChipInactiveFocused = ChipInactive.BorderTopForeground(ColPink).BorderLeftForeground(ColPink).Foreground(ColPink)
	ChipInactiveHovered = ChipInactive
	ChipPublish         = Chip.BorderForeground(ColBlue).Background(ColBlue).Foreground(ColWhite).BorderStyle(lipgloss.InnerHalfBlockBorder())
	ChipPublishFocused  = ChipPublish.BorderTopForeground(ColPink).BorderLeftForeground(ColPink)
	ChipPublishHovered  = ChipPublish

	InfoStyle       = lipgloss.NewStyle().Foreground(ColBlue).PaddingLeft(1)
	ErrorStyle      = lipgloss.NewStyle().Foreground(ColWarn).PaddingLeft(1)
	InfoSubtleStyle = lipgloss.NewStyle().Foreground(ColGray).PaddingLeft(1)

	HelpStyle        = lipgloss.NewStyle().Foreground(ColCyan)
	HelpFocused      = HelpStyle.Foreground(ColDarkGray).Background(ColPink)
	HelpHovered      = HelpStyle.Foreground(ColDarkGray).Background(ColCyan)
	ContextHelpStyle = lipgloss.NewStyle().Foreground(ColWhite).PaddingLeft(1)
	ContextHelpLead  = lipgloss.NewStyle().Foreground(ColCyan)

	// Form field styles
	FormLabel        = lipgloss.NewStyle().Foreground(ColBlue).Bold(true)
	FormLabelFocused = lipgloss.NewStyle().Foreground(ColPink).Bold(true)
	FormHelp         = lipgloss.NewStyle().Foreground(ColGray).Italic(true)
	FormError        = lipgloss.NewStyle().Foreground(ColWarn)

	// Focus indicators
	ReadOnlyIndicator = lipgloss.NewStyle().Foreground(ColGray).Italic(true)
)
