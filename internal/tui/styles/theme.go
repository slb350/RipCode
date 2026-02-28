package styles

import (
	"image/color"

	"charm.land/lipgloss/v2"
)

// Palette defines the semantic hex colors for the entire UI.
type Palette struct {
	Background        string // app background
	BackgroundPanel   string // elevated panels (user messages)
	BackgroundElement string // interactive elements (focused input)
	Text              string // primary text
	TextMuted         string // secondary/hint text
	Border            string // borders and dividers
	Primary           string // build mode accent (peach)
	Secondary         string // plan mode accent (blue)
	Error             string // error states
	Success           string // success states
	Warning           string // warning states
}

// DefaultPalette is the standard dark theme palette.
var DefaultPalette = Palette{
	Background:        "#0a0a0a",
	BackgroundPanel:   "#141414",
	BackgroundElement: "#1e1e1e",
	Text:              "#eeeeee",
	TextMuted:         "#808080",
	Border:            "#484848",
	Primary:           "#fab283",
	Secondary:         "#5c9cf5",
	Error:             "#e06c75",
	Success:           "#7fd88f",
	Warning:           "#f5a742",
}

// Theme wraps a Palette with pre-built lipgloss styles.
type Theme struct {
	Palette Palette

	// Text role styles
	TextStyle      lipgloss.Style
	TextMutedStyle lipgloss.Style
	ErrorStyle     lipgloss.Style
	SuccessStyle   lipgloss.Style
	WarningStyle   lipgloss.Style
	PrimaryStyle   lipgloss.Style
	SecondaryStyle lipgloss.Style

	// Layout styles
	StatusBarStyle lipgloss.Style
	BorderStyle    lipgloss.Style
}

// NewTheme creates a Theme from a Palette.
func NewTheme(p Palette) *Theme {
	return &Theme{
		Palette: p,

		TextStyle:      lipgloss.NewStyle().Foreground(lipgloss.Color(p.Text)),
		TextMutedStyle: lipgloss.NewStyle().Foreground(lipgloss.Color(p.TextMuted)),
		ErrorStyle:     lipgloss.NewStyle().Foreground(lipgloss.Color(p.Error)).Bold(true),
		SuccessStyle:   lipgloss.NewStyle().Foreground(lipgloss.Color(p.Success)),
		WarningStyle:   lipgloss.NewStyle().Foreground(lipgloss.Color(p.Warning)),
		PrimaryStyle:   lipgloss.NewStyle().Foreground(lipgloss.Color(p.Primary)).Bold(true),
		SecondaryStyle: lipgloss.NewStyle().Foreground(lipgloss.Color(p.Secondary)).Bold(true),

		StatusBarStyle: lipgloss.NewStyle().Background(lipgloss.Color(p.BackgroundPanel)).Padding(0, 1),
		BorderStyle:    lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color(p.Border)),
	}
}

// ModeColor returns the accent color for the given mode name.
func (t *Theme) ModeColor(mode string) color.Color {
	switch mode {
	case "plan":
		return lipgloss.Color(t.Palette.Secondary)
	default:
		return lipgloss.Color(t.Palette.Primary)
	}
}

// DefaultTheme is the package-level theme instance.
var DefaultTheme = NewTheme(DefaultPalette)

// --- Backward-compatible exports ---
// These alias into DefaultTheme so existing code/tests don't break.

// Colors (mapped from hex palette to lipgloss.Color)
var (
	ColorBlue    = lipgloss.Color(DefaultPalette.Secondary)
	ColorGreen   = lipgloss.Color(DefaultPalette.Success)
	ColorGray    = lipgloss.Color(DefaultPalette.TextMuted)
	ColorOrange  = lipgloss.Color(DefaultPalette.Primary)
	ColorRed     = lipgloss.Color(DefaultPalette.Error)
	ColorDimGray = lipgloss.Color(DefaultPalette.TextMuted)
	ColorBgDark  = lipgloss.Color(DefaultPalette.BackgroundPanel)
	ColorBorder  = lipgloss.Color(DefaultPalette.Border)
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
