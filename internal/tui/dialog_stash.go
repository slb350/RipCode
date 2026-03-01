package tui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
)

func (a App) handleStashDialogKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	entries := a.stash.List()

	switch {
	case msg.Code == tea.KeyEscape:
		a.stashDialogOpen = false
		a.input.Focus()
		return a, nil

	case msg.Code == tea.KeyEnter:
		if a.stashDialogSelect < len(entries) {
			// Restore selected entry to input and remove from stash
			entry := entries[len(entries)-1-a.stashDialogSelect] // displayed newest-first
			a.input.SetValue(entry.Content)
			a.stash.Delete(entry.ID)
			a.warnOnErr(persistStash(a.stash), "stash")
		}
		a.stashDialogOpen = false
		a.input.Focus()
		return a, nil

	case msg.Code == tea.KeyUp:
		if a.stashDialogSelect > 0 {
			a.stashDialogSelect--
		}
		return a, nil

	case msg.Code == tea.KeyDown:
		if a.stashDialogSelect < len(entries)-1 {
			a.stashDialogSelect++
		}
		return a, nil

	case msg.Mod == tea.ModCtrl && msg.Code == 'd':
		if a.stashDialogSelect < len(entries) {
			entry := entries[len(entries)-1-a.stashDialogSelect]
			a.stash.Delete(entry.ID)
			a.warnOnErr(persistStash(a.stash), "stash")
			remaining := a.stash.List()
			if a.stashDialogSelect >= len(remaining) && a.stashDialogSelect > 0 {
				a.stashDialogSelect--
			}
			if len(remaining) == 0 {
				a.stashDialogOpen = false
				a.input.Focus()
			}
		}
		return a, nil

	default:
		return a, nil
	}
}

func (a App) renderStashDialog() string {
	var sb strings.Builder
	sb.WriteString("Stash (Enter restore, Ctrl+D delete, Esc close)\n")

	entries := a.stash.List()
	if len(entries) == 0 {
		sb.WriteString("\n  Empty stash")
		return sb.String()
	}

	// Display newest first
	for i := len(entries) - 1; i >= 0; i-- {
		idx := len(entries) - 1 - i
		marker := "  "
		if idx == a.stashDialogSelect {
			marker = "> "
		}
		content := entries[i].Content
		if len(content) > 50 {
			content = content[:47] + "..."
		}
		sb.WriteString(fmt.Sprintf("\n%s\"%s\"", marker, content))
	}

	return sb.String()
}
