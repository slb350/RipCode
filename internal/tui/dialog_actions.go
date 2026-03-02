package tui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/atotto/clipboard"

	"github.com/stephenbrandon/ripcode/internal/provider"
	"github.com/stephenbrandon/ripcode/internal/store"
	"github.com/stephenbrandon/ripcode/internal/tui/components"
)

// realClipboard wraps atotto/clipboard for production use.
type realClipboard struct{}

func (realClipboard) WriteAll(text string) error { return clipboard.WriteAll(text) }

// actionsForRole returns the available actions for a chat entry role.
// Returns nil for roles that don't support actions (system, complete).
func actionsForRole(role string) []string {
	switch role {
	case components.RoleUser:
		return []string{"Copy", "Revert to here", "Fork from here"}
	case components.RoleAssistant:
		return []string{"Copy", "Fork from here"}
	case components.RoleTool:
		return []string{"Copy output"}
	case components.RoleError:
		return []string{"Copy"}
	default:
		return nil
	}
}

func (a App) handleMessageActionsDialogKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	actions := actionsForRole(a.messageActions.entryRole)

	switch {
	case msg.Code == tea.KeyEscape:
		a.messageActions.open = false
		a.input.Focus()
		return a, nil

	case msg.Code == tea.KeyUp:
		if len(actions) == 0 {
			return a, nil
		}
		a.messageActions.selected--
		if a.messageActions.selected < 0 {
			a.messageActions.selected = len(actions) - 1
		}
		return a, nil

	case msg.Code == tea.KeyDown:
		if len(actions) == 0 {
			return a, nil
		}
		a.messageActions.selected++
		if a.messageActions.selected >= len(actions) {
			a.messageActions.selected = 0
		}
		return a, nil

	case msg.Code == tea.KeyEnter:
		if len(actions) == 0 {
			a.messageActions.open = false
			a.input.Focus()
			return a, nil
		}
		sel := clamp(a.messageActions.selected, 0, len(actions)-1)
		action := actions[sel]
		return a.executeMessageAction(action)

	default:
		return a, nil
	}
}

func (a App) executeMessageAction(action string) (tea.Model, tea.Cmd) {
	a.messageActions.open = false
	a.input.Focus()

	entries := a.chat.Entries()
	idx := a.messageActions.messageIdx
	if idx < 0 || idx >= len(entries) {
		return a, a.ShowToast("Invalid message index", components.ToastError)
	}
	entry := entries[idx]

	switch action {
	case "Copy", "Copy output":
		text := entry.CopyableContent()
		if text == "" {
			text = entry.Content
		}
		if a.clipboard == nil {
			return a, a.ShowToast("Clipboard not available", components.ToastError)
		}
		if err := a.clipboard.WriteAll(text); err != nil {
			return a, a.ShowToast("Copy failed: "+err.Error(), components.ToastError)
		}
		return a, a.ShowToast("Copied to clipboard", components.ToastSuccess)

	case "Revert to here":
		if a.session == nil {
			return a, a.ShowToast("No active session", components.ToastError)
		}
		if !a.session.CanUndo() {
			return a, a.ShowToast("Nothing to revert", components.ToastWarning)
		}
		// Compute target length: keep the clicked message and its full turn
		// (i.e. user + following assistant/tool messages before the next user).
		records := a.session.Records()
		targetLen := idx + 1
		for j := targetLen; j < len(records); j++ {
			if records[j].Message.Role == provider.RoleUser {
				break
			}
			targetLen++
		}
		// Revert exchanges until the session length matches.
		for a.session.Len() > targetLen && a.session.CanUndo() {
			if _, err := a.session.Revert(); err != nil {
				break
			}
		}
		a.rebuildChatFromSession()
		a.chat.AddEntry(components.ChatEntry{
			Role:    components.RoleSystem,
			Content: "--- reverted ---",
		})
		a.warnOnErr(store.Save(a.session), "session")
		return a, a.ShowToast("Reverted to message", components.ToastInfo)

	case "Fork from here":
		if a.session == nil {
			return a, a.ShowToast("No active session", components.ToastError)
		}
		// Find the session record index for this chat entry.
		// The chat entry idx maps to session records; we fork up to idx.
		forkIdx := idx
		if forkIdx >= a.session.Len() {
			forkIdx = a.session.Len() - 1
		}
		if forkIdx < 0 {
			return a, a.ShowToast("Fork failed: invalid index", components.ToastError)
		}
		forked, err := a.session.Fork(forkIdx)
		if err != nil {
			return a, a.ShowToast("Fork failed: "+err.Error(), components.ToastError)
		}
		if err := store.Save(a.session); err != nil {
			return a, a.ShowToast("Fork failed: could not save session", components.ToastError)
		}
		if err := store.Save(forked); err != nil {
			return a, a.ShowToast("Fork failed: could not save fork", components.ToastError)
		}
		a.session = forked
		a.rebuildChatFromSession()
		a.statusbar.SetTitle(shortSessionTitle(forked.ID))
		a.statusbar.SetTokens(0)
		return a, a.ShowToast("Forked session", components.ToastSuccess)

	default:
		store.LogErrorf("dialog_actions: unknown action %q", action)
		return a, nil
	}
}

func (a App) renderMessageActionsDialog() string {
	actions := actionsForRole(a.messageActions.entryRole)
	header := fmt.Sprintf("Message actions (Enter select, Esc close)")

	items := make([]pickerItem, len(actions))
	for i, act := range actions {
		items[i] = pickerItem{Label: act}
	}
	return renderPickerList(header, items, a.messageActions.selected, 8)
}

// chatBounds returns the Y range of the chat area (top inclusive, bottom exclusive).
// Uses package-level layout constants from app.go.
func (a App) chatBounds() (top, bottom int) {
	top = layoutStatusH
	if toastView := a.toasts.View(); toastView != "" {
		top += strings.Count(toastView, "\n") + 1
	}
	bottom = a.height - layoutInputH - layoutFooterH
	return
}
