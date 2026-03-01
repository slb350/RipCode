package tui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/stephenbrandon/ripcode/internal/store"
	"github.com/stephenbrandon/ripcode/internal/tui/components"
)

func (a App) handleForkDialogKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case msg.Code == tea.KeyEscape:
		a.forkDialogOpen = false
		a.input.Focus()
		return a, nil

	case msg.Code == tea.KeyEnter:
		entries := a.timelineEntries()
		if a.forkDialogSelect >= len(entries) {
			a.forkDialogOpen = false
			a.input.Focus()
			return a, nil
		}
		// Find the message index in session for this user message
		entry := entries[a.forkDialogSelect]
		forkIdx := -1
		for i, rec := range a.session.Messages {
			if rec.ID == entry.ID {
				// Include the next assistant response if available
				if i+1 < len(a.session.Messages) && a.session.Messages[i+1].Message.Role == "assistant" {
					forkIdx = i + 1
				} else {
					forkIdx = i
				}
				break
			}
		}
		if forkIdx < 0 {
			a.forkDialogOpen = false
			a.input.Focus()
			return a, a.ShowToast("Fork failed: message not found", components.ToastError)
		}
		forked, err := a.session.Fork(forkIdx)
		if err != nil {
			a.forkDialogOpen = false
			a.input.Focus()
			return a, a.ShowToast("Fork failed: "+err.Error(), components.ToastError)
		}
		// Save the current session before switching
		_ = store.Save(a.session)
		// Switch to forked session
		a.session = forked
		a.rebuildChatFromSession()
		a.statusbar.SetTitle(shortSessionTitle(forked.ID))
		a.statusbar.SetTokens(0)
		a.forkDialogOpen = false
		a.input.Focus()
		return a, a.ShowToast("Forked session", components.ToastSuccess)

	case msg.Code == tea.KeyUp:
		if a.forkDialogSelect > 0 {
			a.forkDialogSelect--
		}
		return a, nil

	case msg.Code == tea.KeyDown:
		entries := a.timelineEntries()
		if a.forkDialogSelect < len(entries)-1 {
			a.forkDialogSelect++
		}
		return a, nil

	default:
		return a, nil
	}
}

func (a App) renderForkDialog() string {
	var sb strings.Builder
	sb.WriteString("Fork from message (Enter fork, Esc close)\n")

	entries := a.timelineEntries()
	if len(entries) == 0 {
		sb.WriteString("\n  No user messages")
		return sb.String()
	}

	for i, entry := range entries {
		marker := "  "
		if i == a.forkDialogSelect {
			marker = "> "
		}
		sb.WriteString(fmt.Sprintf("\n%s%s", marker, entry.Content))
	}

	return sb.String()
}
