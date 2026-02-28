package tui

import "github.com/stephenbrandon/ripcode/internal/agent"

// AgentEventMsg wraps an agent loop event for the TUI.
type AgentEventMsg struct {
	Event agent.Event
}

// ModeChangeMsg signals a mode change.
type ModeChangeMsg struct {
	Mode agent.Mode
}
