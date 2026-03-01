package components

import (
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/stephenbrandon/ripcode/internal/tui/styles"
)

// SessionFooter displays bottom session metadata/status.
type SessionFooter struct {
	width     int
	workDir   string
	streaming bool
	connected bool
}

// NewSessionFooter creates a new footer component.
func NewSessionFooter() SessionFooter {
	return SessionFooter{
		workDir:   ".",
		connected: false,
	}
}

// SetSize updates footer width.
func (f *SessionFooter) SetSize(width int) {
	f.width = width
}

// SetWorkDir updates displayed working directory.
func (f *SessionFooter) SetWorkDir(dir string) {
	if strings.TrimSpace(dir) == "" {
		f.workDir = "."
		return
	}
	f.workDir = dir
}

// SetStreaming toggles running state.
func (f *SessionFooter) SetStreaming(streaming bool) {
	f.streaming = streaming
}

// SetConnected indicates whether a provider is configured.
func (f *SessionFooter) SetConnected(connected bool) {
	f.connected = connected
}

// View renders footer row.
func (f SessionFooter) View() string {
	if f.width == 0 {
		return ""
	}

	left := f.workDir
	right := "/help"

	if !f.connected {
		right = "Set OPENROUTER_API_KEY to connect"
	} else if f.streaming {
		right = "● Running · Esc interrupt"
	} else {
		right = "/models · /help · ^B sidebar"
	}

	gap := f.width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 1 {
		right = "/help"
		gap = f.width - lipgloss.Width(left) - lipgloss.Width(right)
	}
	if gap < 0 {
		gap = 0
	}

	return styles.Muted.Render(left) + strings.Repeat(" ", gap) + styles.Muted.Render(right)
}
