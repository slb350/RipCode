package styles

import (
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPalette_HasSemanticColors(t *testing.T) {
	p := DefaultPalette

	// All palette fields must be non-empty hex colors
	assert.NotEmpty(t, p.Background)
	assert.NotEmpty(t, p.BackgroundPanel)
	assert.NotEmpty(t, p.BackgroundElement)
	assert.NotEmpty(t, p.Text)
	assert.NotEmpty(t, p.TextMuted)
	assert.NotEmpty(t, p.Border)
	assert.NotEmpty(t, p.Primary)
	assert.NotEmpty(t, p.Secondary)
	assert.NotEmpty(t, p.Error)
	assert.NotEmpty(t, p.Success)
	assert.NotEmpty(t, p.Warning)
}

func TestNewTheme_FromPalette(t *testing.T) {
	theme := NewTheme(DefaultPalette)
	require.NotNil(t, theme)

	// Theme should expose pre-built styles
	assert.NotNil(t, theme.TextStyle)
	assert.NotNil(t, theme.TextMutedStyle)
	assert.NotNil(t, theme.ErrorStyle)
	assert.NotNil(t, theme.SuccessStyle)
	assert.NotNil(t, theme.PrimaryStyle)
	assert.NotNil(t, theme.SecondaryStyle)
	assert.NotNil(t, theme.StatusBarStyle)
	assert.NotNil(t, theme.BorderStyle)
}

func TestTheme_ModeColor_Build(t *testing.T) {
	theme := NewTheme(DefaultPalette)
	c := theme.ModeColor("build")
	assert.Equal(t, lipgloss.Color(DefaultPalette.Primary), c)
}

func TestTheme_ModeColor_Plan(t *testing.T) {
	theme := NewTheme(DefaultPalette)
	c := theme.ModeColor("plan")
	assert.Equal(t, lipgloss.Color(DefaultPalette.Secondary), c)
}

func TestTheme_ModeColor_Unknown(t *testing.T) {
	theme := NewTheme(DefaultPalette)
	c := theme.ModeColor("unknown")
	// Falls back to Primary
	assert.Equal(t, lipgloss.Color(DefaultPalette.Primary), c)
}

func TestDefaultTheme_Exists(t *testing.T) {
	require.NotNil(t, DefaultTheme)
	assert.Equal(t, DefaultPalette, DefaultTheme.Palette)
}

// Backward compatibility: old package-level vars must still work
func TestBackwardCompat_Colors(t *testing.T) {
	assert.NotNil(t, ColorBlue)
	assert.NotNil(t, ColorGreen)
	assert.NotNil(t, ColorGray)
	assert.NotNil(t, ColorOrange)
	assert.NotNil(t, ColorRed)
	assert.NotNil(t, ColorDimGray)
	assert.NotNil(t, ColorBgDark)
	assert.NotNil(t, ColorBorder)
}

func TestBackwardCompat_Styles(t *testing.T) {
	// These should render without panicking
	_ = User.Render("test")
	_ = Assistant.Render("test")
	_ = Tool.Render("test")
	_ = ToolCmd.Render("test")
	_ = Error.Render("test")
	_ = Muted.Render("test")
	_ = StatusBar.Render("test")
	_ = Border.Render("test")
}
