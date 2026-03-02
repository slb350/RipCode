package tui

import (
	"github.com/stephenbrandon/ripcode/internal/agent"
	"github.com/stephenbrandon/ripcode/internal/provider"
	"github.com/stephenbrandon/ripcode/internal/store"
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

// ShellResultMsg carries the result of an async shell command.
type ShellResultMsg struct {
	Command string
	Output  string
	Error   string
}

// SessionsLoadedMsg carries the result of an async session list fetch.
type SessionsLoadedMsg struct {
	Sessions  []store.SessionSummary
	Corrupted []string
	Err       error
}
