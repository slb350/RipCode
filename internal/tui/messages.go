package tui

import (
	"github.com/stephenbrandon/ripcode/internal/agent"
	"github.com/stephenbrandon/ripcode/internal/provider"
)

// AgentEventMsg wraps an agent loop event for the TUI.
type AgentEventMsg struct {
	Event agent.Event
}

// ModeChangeMsg signals a mode change.
type ModeChangeMsg struct {
	Mode agent.Mode
}

// ModelsLoadedMsg carries the result of an async model list fetch.
type ModelsLoadedMsg struct {
	Models []provider.ModelInfo
	Err    error
	Query  string
}

// FileCacheLoadedMsg carries the result of an async file cache scan.
type FileCacheLoadedMsg struct {
	Files []string
}

// ToastDismissMsg is sent after a toast's timer expires.
type ToastDismissMsg struct {
	ID int64
}
