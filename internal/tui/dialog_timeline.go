package tui

import (
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
)

type timelineEntry struct {
	ID      string
	Content string
	Time    time.Time
}

func (a App) timelineEntries() []timelineEntry {
	if a.session == nil {
		return nil
	}
	var entries []timelineEntry
	for _, rec := range a.session.Messages {
		if rec.Message.Role == "user" {
			content := rec.Message.Content
			if len(content) > 60 {
				content = content[:57] + "..."
			}
			entries = append(entries, timelineEntry{
				ID:      rec.ID,
				Content: content,
				Time:    rec.CreatedAt,
			})
		}
	}
	return entries
}

func (a App) handleTimelineDialogKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case msg.Code == tea.KeyEscape:
		a.timelineDialogOpen = false
		a.input.Focus()
		return a, nil

	case msg.Code == tea.KeyEnter:
		entries := a.timelineEntries()
		if a.timelineDialogSelect < len(entries) {
			// Scroll chat to the position of this user message
			a.scrollToUserMessage(a.timelineDialogSelect)
		}
		a.timelineDialogOpen = false
		a.input.Focus()
		return a, nil

	case msg.Code == tea.KeyUp:
		if a.timelineDialogSelect > 0 {
			a.timelineDialogSelect--
		}
		return a, nil

	case msg.Code == tea.KeyDown:
		entries := a.timelineEntries()
		if a.timelineDialogSelect < len(entries)-1 {
			a.timelineDialogSelect++
		}
		return a, nil

	default:
		return a, nil
	}
}

func (a *App) scrollToUserMessage(idx int) {
	userCount := 0
	linePos := 0
	for _, entry := range a.chat.Entries() {
		if entry.Role == "user" {
			if userCount == idx {
				a.chat.SetScrollPos(linePos)
				return
			}
			userCount++
		}
		linePos += 2 // approximate: entry + blank line
	}
}

func (a App) renderTimelineDialog() string {
	var sb strings.Builder
	sb.WriteString("Timeline (Enter jump, Esc close)\n")

	entries := a.timelineEntries()
	if len(entries) == 0 {
		sb.WriteString("\n  No user messages")
		return sb.String()
	}

	for i, entry := range entries {
		marker := "  "
		if i == a.timelineDialogSelect {
			marker = "> "
		}
		sb.WriteString(fmt.Sprintf("\n%s%s", marker, entry.Content))
	}

	return sb.String()
}
