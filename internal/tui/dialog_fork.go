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
		a.forkDialog.open = false
		a.input.Focus()
		return a, nil

	case msg.Code == tea.KeyEnter:
		entries := a.timelineEntries()
		if a.forkDialog.selected >= len(entries) {
			a.forkDialog.open = false
			a.input.Focus()
			return a, nil
		}
		// Find the message index in session for this user message
		entry := entries[a.forkDialog.selected]
		forkIdx := -1
		recs := a.session.Records()
		for i, rec := range recs {
			if rec.ID == entry.ID {
				// Include the next assistant response if available
				if i+1 < len(recs) && recs[i+1].Message.Role == "assistant" {
					forkIdx = i + 1
				} else {
					forkIdx = i
				}
				break
			}
		}
		if forkIdx < 0 {
			a.forkDialog.open = false
			a.input.Focus()
			return a, a.ShowToast("Fork failed: message not found", components.ToastError)
		}
		forked, err := a.session.Fork(forkIdx)
		if err != nil {
			a.forkDialog.open = false
			a.input.Focus()
			return a, a.ShowToast("Fork failed: "+err.Error(), components.ToastError)
		}
		// Save the current session before switching
		if err := store.Save(a.session); err != nil {
			a.forkDialog.open = false
			a.input.Focus()
			return a, a.ShowToast("Fork failed: could not save session", components.ToastError)
		}
		// Switch to forked session
		a.session = forked
		a.rebuildChatFromSession()
		a.statusbar.SetTitle(shortSessionTitle(forked.ID))
		a.statusbar.SetTokens(0)
		a.forkDialog.open = false
		a.input.Focus()
		return a, a.ShowToast("Forked session", components.ToastSuccess)

	case msg.Code == tea.KeyUp:
		if a.forkDialog.selected > 0 {
			a.forkDialog.selected--
		}
		return a, nil

	case msg.Code == tea.KeyDown:
		entries := a.timelineEntries()
		if a.forkDialog.selected < len(entries)-1 {
			a.forkDialog.selected++
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
		if i == a.forkDialog.selected {
			marker = "> "
		}
		sb.WriteString(fmt.Sprintf("\n%s%s", marker, entry.Content))
	}

	return sb.String()
}
