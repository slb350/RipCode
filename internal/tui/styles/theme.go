package styles

import "charm.land/lipgloss/v2"

// Colors
var (
	ColorBlue    = lipgloss.Color("33")
	ColorGreen   = lipgloss.Color("78")
	ColorGray    = lipgloss.Color("245")
	ColorOrange  = lipgloss.Color("214")
	ColorRed     = lipgloss.Color("196")
	ColorDimGray = lipgloss.Color("241")
	ColorBgDark  = lipgloss.Color("236")
	ColorBorder  = lipgloss.Color("240")
)

// Text styles
var (
	User      = lipgloss.NewStyle().Bold(true).Foreground(ColorBlue)
	Assistant = lipgloss.NewStyle().Bold(true).Foreground(ColorGreen)
	Tool      = lipgloss.NewStyle().Foreground(ColorGray)
	ToolCmd   = lipgloss.NewStyle().Foreground(ColorOrange).Bold(true)
	Error     = lipgloss.NewStyle().Foreground(ColorRed).Bold(true)
	Muted     = lipgloss.NewStyle().Foreground(ColorDimGray).Italic(true)
)

// Layout styles
var (
	StatusBar = lipgloss.NewStyle().Background(ColorBgDark).Padding(0, 1)
	Border    = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(ColorBorder)
)
