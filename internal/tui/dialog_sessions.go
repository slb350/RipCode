package tui

import (
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/stephenbrandon/ripcode/internal/store"
	"github.com/stephenbrandon/ripcode/internal/tui/components"
)

func (a *App) loadSessions() tea.Cmd {
	return func() tea.Msg {
		summaries, err := store.List()
		if err != nil {
			return SessionsLoadedMsg{Err: err}
		}
		return SessionsLoadedMsg{Sessions: summaries}
	}
}

func (a App) handleSessionsDialogKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if a.sessionsDialogConfirm {
		return a.handleSessionsDeleteConfirm(msg)
	}

	switch {
	case msg.Code == tea.KeyEscape:
		a.sessionsDialogOpen = false
		a.input.Focus()
		return a, nil

	case msg.Code == tea.KeyEnter:
		filtered := a.filteredSessions()
		if len(filtered) == 0 || a.sessionsDialogSelect >= len(filtered) {
			return a, nil
		}
		entry := filtered[a.sessionsDialogSelect]
		return a.resumeSession(entry.ID)

	case msg.Code == tea.KeyUp:
		if a.sessionsDialogSelect > 0 {
			a.sessionsDialogSelect--
		}
		return a, nil

	case msg.Code == tea.KeyDown:
		filtered := a.filteredSessions()
		if a.sessionsDialogSelect < len(filtered)-1 {
			a.sessionsDialogSelect++
		}
		return a, nil

	case msg.Mod == tea.ModCtrl && msg.Code == 'd':
		filtered := a.filteredSessions()
		if len(filtered) > 0 {
			a.sessionsDialogConfirm = true
		}
		return a, nil

	case msg.Code == tea.KeyBackspace:
		a.sessionsDialogQuery = backspaceRune(a.sessionsDialogQuery)
		a.sessionsDialogSelect = 0
		return a, nil

	default:
		if msg.Text != "" {
			a.sessionsDialogQuery += msg.Text
			a.sessionsDialogSelect = 0
		}
		return a, nil
	}
}

func (a App) handleSessionsDeleteConfirm(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case msg.Code == tea.KeyEscape:
		a.sessionsDialogConfirm = false
		return a, nil

	case msg.Code == tea.KeyEnter:
		filtered := a.filteredSessions()
		if a.sessionsDialogSelect < len(filtered) {
			entry := filtered[a.sessionsDialogSelect]
			_ = store.Delete(entry.ID)
			// Remove from cached entries
			for i, e := range a.sessionsDialogEntries {
				if e.ID == entry.ID {
					a.sessionsDialogEntries = append(a.sessionsDialogEntries[:i], a.sessionsDialogEntries[i+1:]...)
					break
				}
			}
			if a.sessionsDialogSelect >= len(a.filteredSessions()) && a.sessionsDialogSelect > 0 {
				a.sessionsDialogSelect--
			}
		}
		a.sessionsDialogConfirm = false
		return a, nil

	default:
		return a, nil
	}
}

func (a App) filteredSessions() []store.SessionSummary {
	if a.sessionsDialogQuery == "" {
		return a.sessionsDialogEntries
	}
	q := strings.ToLower(a.sessionsDialogQuery)
	var out []store.SessionSummary
	for _, e := range a.sessionsDialogEntries {
		if strings.Contains(strings.ToLower(e.Title), q) ||
			strings.Contains(strings.ToLower(e.WorkDir), q) {
			out = append(out, e)
		}
	}
	return out
}

func (a App) resumeSession(id string) (tea.Model, tea.Cmd) {
	loaded, err := store.Load(id)
	if err != nil {
		a.sessionsDialogOpen = false
		a.input.Focus()
		id := a.toasts.Show("Failed to load session: "+err.Error(), components.ToastError, 3*time.Second)
		return a, func() tea.Msg {
			time.Sleep(3 * time.Second)
			return ToastDismissMsg{ID: id}
		}
	}

	a.session = loaded
	a.sessionsDialogOpen = false
	a.input.Focus()

	// Rebuild chat from session records
	a.rebuildChatFromSession()

	// Update status bar
	title := loaded.Title
	if title == "" {
		title = shortSessionTitle(loaded.ID)
	}
	a.statusbar.SetTitle(title)
	a.statusbar.SetTokens(loaded.Tokens.Input + loaded.Tokens.Output)

	toastID := a.toasts.Show("Session resumed", components.ToastSuccess, 3*time.Second)
	return a, func() tea.Msg {
		time.Sleep(3 * time.Second)
		return ToastDismissMsg{ID: toastID}
	}
}

func (a *App) rebuildChatFromSession() {
	a.chat.Clear()
	if a.session == nil {
		return
	}
	for _, rec := range a.session.Messages {
		switch rec.Message.Role {
		case "user":
			a.chat.AddEntry(components.ChatEntry{
				Role:    "user",
				Content: rec.Message.Content,
			})
		case "assistant":
			a.chat.AddEntry(components.ChatEntry{
				Role:    "assistant",
				Content: rec.Message.Content,
			})
		case "tool":
			a.chat.AddEntry(components.ChatEntry{
				Role:       "tool",
				Content:    rec.Message.Content,
				ToolID:     rec.Message.ToolCallID,
				ToolStatus: "success",
			})
		}
	}
}

func sessionDateGroup(t, today time.Time) string {
	yesterday := today.Add(-24 * time.Hour)
	d := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())

	switch {
	case !d.Before(today):
		return "Today"
	case !d.Before(yesterday):
		return "Yesterday"
	default:
		return t.Format("Jan 2, 2006")
	}
}

func (a App) renderSessionsDialog() string {
	var sb strings.Builder
	sb.WriteString("Sessions (type to filter, Esc close)\n")

	if !a.sessionsDialogLoaded {
		sb.WriteString("\n  Loading...")
		return sb.String()
	}

	filtered := a.filteredSessions()
	if len(filtered) == 0 {
		sb.WriteString("\n  No sessions found")
		return sb.String()
	}

	if a.sessionsDialogConfirm && a.sessionsDialogSelect < len(filtered) {
		entry := filtered[a.sessionsDialogSelect]
		title := entry.Title
		if title == "" {
			title = entry.ID
		}
		sb.WriteString(fmt.Sprintf("\n  Delete \"%s\"?  [Enter] confirm  [Esc] cancel", title))
		return sb.String()
	}

	// Render with date group headers
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	lastGroup := ""
	for i, entry := range filtered {
		group := sessionDateGroup(entry.UpdatedAt, today)
		if group != lastGroup {
			sb.WriteString(fmt.Sprintf("\n  -- %s --", group))
			lastGroup = group
		}
		marker := "  "
		if i == a.sessionsDialogSelect {
			marker = "> "
		}
		title := entry.Title
		if title == "" {
			title = entry.ID
		}
		sb.WriteString(fmt.Sprintf("\n%s%-30s  %d msgs", marker, title, entry.MessageCount))
	}

	// Footer with selected session details
	if a.sessionsDialogSelect < len(filtered) {
		sel := filtered[a.sessionsDialogSelect]
		sb.WriteString(fmt.Sprintf("\n\n  %s  %d msgs  %s",
			sel.UpdatedAt.Format("3:04 PM"),
			sel.MessageCount,
			sel.WorkDir))
	}

	sb.WriteString("\n  [Enter] resume  [Ctrl+D] delete  [Esc] close")
	return sb.String()
}
