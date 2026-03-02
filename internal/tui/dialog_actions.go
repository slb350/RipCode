package tui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/atotto/clipboard"

	"github.com/stephenbrandon/ripcode/internal/provider"
	"github.com/stephenbrandon/ripcode/internal/session"
	"github.com/stephenbrandon/ripcode/internal/store"
	"github.com/stephenbrandon/ripcode/internal/tui/components"
)

// Action label constants used in actionsForRole and executeMessageAction.
const (
	actionCopy       = "Copy"
	actionCopyOutput = "Copy output"
	actionRevert     = "Revert to here"
	actionFork       = "Fork from here"
)

// realClipboard wraps atotto/clipboard for production use.
type realClipboard struct{}

func (realClipboard) WriteAll(text string) error { return clipboard.WriteAll(text) }

// actionsForRole returns the available actions for a chat entry role.
// Returns nil for roles that don't support actions (system, complete).
func actionsForRole(role string) []string {
	switch role {
	case components.RoleUser:
		return []string{actionCopy, actionRevert, actionFork}
	case components.RoleAssistant:
		return []string{actionCopy, actionFork}
	case components.RoleTool:
		return []string{actionCopyOutput}
	case components.RoleError:
		return []string{actionCopy}
	default:
		return nil
	}
}

// endOfTurn returns the index of the last record in the turn starting at startIdx.
// A turn is the record at startIdx plus all following non-user records.
func endOfTurn(records []session.MessageRecord, startIdx int) int {
	end := startIdx
	for j := startIdx + 1; j < len(records); j++ {
		if records[j].Message.Role == provider.RoleUser {
			break
		}
		end = j
	}
	return end
}

func sessionRoleForChatRole(role string) (provider.Role, bool) {
	switch role {
	case components.RoleUser:
		return provider.RoleUser, true
	case components.RoleAssistant:
		return provider.RoleAssistant, true
	case components.RoleTool:
		return provider.RoleTool, true
	default:
		return "", false
	}
}

// sessionRecordIndexForChatIndex maps a rendered chat entry index to a concrete
// session record index, skipping non-session chat rows (system/complete/error).
func (a App) sessionRecordIndexForChatIndex(chatIdx int) (int, bool) {
	if a.session == nil {
		return 0, false
	}
	entries := a.chat.Entries()
	if chatIdx < 0 || chatIdx >= len(entries) {
		return 0, false
	}

	records := a.session.Records()
	recIdx := 0
	for i, entry := range entries {
		wantRole, ok := sessionRoleForChatRole(entry.Role)
		if !ok {
			continue
		}
		for recIdx < len(records) && records[recIdx].Message.Role != wantRole {
			recIdx++
		}
		if recIdx >= len(records) {
			return 0, false
		}
		if i == chatIdx {
			return recIdx, true
		}
		recIdx++
	}
	return 0, false
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
	case actionCopy, actionCopyOutput:
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

	case actionRevert:
		if a.session == nil {
			return a, a.ShowToast("No active session", components.ToastError)
		}
		if !a.session.CanUndo() {
			return a, a.ShowToast("Nothing to revert", components.ToastWarning)
		}
		recordIdx, ok := a.sessionRecordIndexForChatIndex(idx)
		if !ok {
			return a, a.ShowToast("Revert failed: message not found", components.ToastError)
		}
		// Keep the clicked message and its full turn (user + following
		// assistant/tool messages before the next user).
		records := a.session.Records()
		targetLen := endOfTurn(records, recordIdx) + 1
		// Revert exchanges until the session length matches.
		partial := false
		for a.session.Len() > targetLen && a.session.CanUndo() {
			if _, err := a.session.Revert(); err != nil {
				store.LogErrorf("dialog_actions: revert loop error at session len %d (target %d): %v", a.session.Len(), targetLen, err)
				partial = true
				break
			}
		}
		a.rebuildChatFromSession()
		a.chat.AddEntry(components.ChatEntry{
			Role:    components.RoleSystem,
			Content: "--- reverted ---",
		})
		a.warnOnErr(store.Save(a.session), "session")
		if partial {
			return a, a.ShowToast("Partially reverted (error occurred)", components.ToastWarning)
		}
		return a, a.ShowToast("Reverted to message", components.ToastInfo)

	case actionFork:
		if a.session == nil {
			return a, a.ShowToast("No active session", components.ToastError)
		}
		recordIdx, ok := a.sessionRecordIndexForChatIndex(idx)
		if !ok {
			return a, a.ShowToast("Fork failed: message not found", components.ToastError)
		}
		// Include the full clicked turn up to (but not including) the next user.
		records := a.session.Records()
		forkIdx := endOfTurn(records, recordIdx)
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
