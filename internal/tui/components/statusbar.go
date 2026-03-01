package components

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/stephenbrandon/ripcode/internal/tui/styles"
)

// StatusBar displays the session header information.
type StatusBar struct {
	width        int
	title        string
	model        string
	mode         string
	tokens       int
	spinning     bool
	leaderHint   string
	variantBadge string
}

// NewStatusBar creates a new status bar.
func NewStatusBar() StatusBar {
	return StatusBar{
		title: "Session",
		model: "claude-sonnet-4",
		mode:  "build",
	}
}

// SetSize updates the status bar width.
func (s *StatusBar) SetSize(width int) {
	s.width = width
}

// SetTitle updates the displayed session title.
func (s *StatusBar) SetTitle(title string) {
	if strings.TrimSpace(title) == "" {
		s.title = "Session"
		return
	}
	s.title = title
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

// SetLeaderHint sets a leader key hint to display in the status bar.
func (s *StatusBar) SetLeaderHint(hint string) {
	s.leaderHint = hint
}

// SetVariantBadge sets the variant badge text to display in the status bar.
func (s *StatusBar) SetVariantBadge(badge string) {
	s.variantBadge = badge
}

// View renders the status bar.
func (s StatusBar) View() string {
	if s.width == 0 {
		return ""
	}

	left := " #" + " " + s.title
	if s.spinning {
		left += " ●"
	}

	right := fmt.Sprintf("%s · %s", s.mode, s.model)
	if s.tokens > 0 {
		right += fmt.Sprintf(" · %s tokens", FormatTokens(s.tokens))
	}
	if s.variantBadge != "" {
		right += " · " + s.variantBadge
	}
	if s.leaderHint != "" {
		right += " · " + s.leaderHint
	}
	right = " " + right

	gap := s.width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 0 {
		gap = 0
	}

	bar := left + strings.Repeat(" ", gap) + right
	return styles.StatusBar.Width(s.width).Render(bar)
}

// FormatTokens formats a token count for display (e.g. "1.5K", "2.3M").
func FormatTokens(n int) string {
	if n >= 1_000_000 {
		return fmt.Sprintf("%.1fM", float64(n)/1_000_000)
	}
	if n >= 1_000 {
		return fmt.Sprintf("%.1fK", float64(n)/1_000)
	}
	return fmt.Sprintf("%d", n)
}
