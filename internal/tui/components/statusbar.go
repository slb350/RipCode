package components

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/stephenbrandon/ripcode/internal/tui/styles"
)

// StatusBar displays model info, mode, and token count.
type StatusBar struct {
	width    int
	model    string
	mode     string
	tokens   int
	spinning bool
}

// NewStatusBar creates a new status bar.
func NewStatusBar() StatusBar {
	return StatusBar{
		model: "claude-sonnet-4",
		mode:  "build",
	}
}

// SetSize updates the status bar width.
func (s *StatusBar) SetSize(width int) {
	s.width = width
}

// SetModel updates the displayed model name.
func (s *StatusBar) SetModel(model string) {
	s.model = model
}

// SetMode updates the displayed mode.
func (s *StatusBar) SetMode(mode string) {
	s.mode = mode
}

// SetTokens updates the token count.
func (s *StatusBar) SetTokens(tokens int) {
	s.tokens = tokens
}

// SetSpinning controls the activity indicator.
func (s *StatusBar) SetSpinning(spinning bool) {
	s.spinning = spinning
}

// View renders the status bar.
func (s StatusBar) View() string {
	if s.width == 0 {
		return ""
	}

	left := " ripcode"
	if s.spinning {
		left += " ●"
	}

	right := fmt.Sprintf("%s │ %s", s.mode, s.model)
	if s.tokens > 0 {
		right += fmt.Sprintf(" │ %s tokens", formatTokens(s.tokens))
	}
	right += " "

	gap := s.width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 0 {
		gap = 0
	}

	bar := left + strings.Repeat(" ", gap) + right
	return styles.StatusBar.Width(s.width).Render(bar)
}

func formatTokens(n int) string {
	if n >= 1_000_000 {
		return fmt.Sprintf("%.1fM", float64(n)/1_000_000)
	}
	if n >= 1_000 {
		return fmt.Sprintf("%.1fK", float64(n)/1_000)
	}
	return fmt.Sprintf("%d", n)
}
