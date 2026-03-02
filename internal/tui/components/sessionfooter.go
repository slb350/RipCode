package components

import (
	"fmt"
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
	mcpCount  int
	lspCount  int
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

// SetMCPCount sets the number of enabled MCP servers.
func (f *SessionFooter) SetMCPCount(n int) {
	f.mcpCount = n
}

// SetLSPCount sets the number of enabled LSP clients.
func (f *SessionFooter) SetLSPCount(n int) {
	f.lspCount = n
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
		var parts []string
		if f.mcpCount > 0 {
			parts = append(parts, fmt.Sprintf("⊙ %d", f.mcpCount))
		}
		if f.lspCount > 0 {
			parts = append(parts, fmt.Sprintf("• %d", f.lspCount))
		}
		parts = append(parts, "/help", "^B sidebar")
		right = strings.Join(parts, " · ")
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
