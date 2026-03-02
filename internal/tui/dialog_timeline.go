package tui

import (
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/stephenbrandon/ripcode/internal/provider"
	"github.com/stephenbrandon/ripcode/internal/tui/components"
	"github.com/stephenbrandon/ripcode/internal/tui/styles"
)

// Timeline entry status constants.
const (
	timelineStatusOK          = "ok"
	timelineStatusInterrupted = "interrupted"
)

type timelineEntry struct {
	ID       string
	Content  string
	Time     time.Time
	Tokens   int           // total tokens for this exchange
	Tools    int           // tool call count
	Duration time.Duration // assistant response duration
	Status   string        // timelineStatusOK, timelineStatusInterrupted
}

func (a App) timelineEntries() []timelineEntry {
	if a.session == nil {
		return nil
	}
	records := a.session.Records()
	var entries []timelineEntry
	for i, rec := range records {
		if rec.Message.Role != provider.RoleUser {
			continue
		}
		content := rec.Message.Content
		if len(content) > 60 {
			content = content[:57] + "..."
		}
		entry := timelineEntry{
			ID:      rec.ID,
			Content: content,
			Time:    rec.CreatedAt,
		}
		// Scan forward for assistant metadata
		for j := i + 1; j < len(records); j++ {
			r := records[j]
			if r.Message.Role == provider.RoleUser {
				break
			}
			if r.Meta != nil {
				entry.Tokens = r.Meta.InputTokens + r.Meta.OutputTokens
				entry.Duration = r.Meta.Duration
				if r.Meta.FinishReason == "length" {
					entry.Status = timelineStatusInterrupted
				}
			}
			entry.Tools += len(r.Message.ToolCalls)
		}
		if entry.Status == "" {
			entry.Status = timelineStatusOK
		}
		entries = append(entries, entry)
	}
	return entries
}

func (a App) handleTimelineDialogKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case msg.Code == tea.KeyEscape:
		a.timelineDialog.open = false
		a.input.Focus()
		return a, nil

	case msg.Code == tea.KeyEnter:
		entries := a.timelineEntries()
		if a.timelineDialog.selected < len(entries) {
			// Scroll chat to the position of this user message
			a.scrollToUserMessage(a.timelineDialog.selected)
		}
		a.timelineDialog.open = false
		a.input.Focus()
		return a, nil

	case msg.Code == tea.KeyUp:
		if a.timelineDialog.selected > 0 {
			a.timelineDialog.selected--
		}
		return a, nil

	case msg.Code == tea.KeyDown:
		entries := a.timelineEntries()
		if a.timelineDialog.selected < len(entries)-1 {
			a.timelineDialog.selected++
		}
		return a, nil

	default:
		return a, nil
	}
}

func (a *App) scrollToUserMessage(idx int) {
	if linePos, ok := a.chat.LineOffsetForUserMessage(idx); ok {
		a.chat.SetScrollPos(linePos)
	}
}

func (a App) renderTimelineDialog() string {
	muted := styles.DefaultTheme.TextMutedStyle

	var sb strings.Builder
	sb.WriteString("Timeline (Enter jump, Esc close)\n")

	entries := a.timelineEntries()
	if len(entries) == 0 {
		sb.WriteString("\n  No user messages")
		return sb.String()
	}

	for i, entry := range entries {
		marker := "  "
		if i == a.timelineDialog.selected {
			marker = "> "
		}
		sb.WriteString(fmt.Sprintf("\n%s%s  %s", marker, entry.Time.Format("15:04"), entry.Content))

		// Badge line
		var badges []string
		if entry.Tokens > 0 {
			badges = append(badges, fmt.Sprintf("⊙ %s", components.FormatTokens(entry.Tokens)))
		}
		if entry.Tools > 0 {
			badges = append(badges, fmt.Sprintf("⚡ %d tools", entry.Tools))
		}
		if entry.Duration > 0 {
			badges = append(badges, fmt.Sprintf("%.1fs", entry.Duration.Seconds()))
		}
		if entry.Status == timelineStatusInterrupted {
			badges = append(badges, "⚠ interrupted")
		}
		if len(badges) > 0 {
			indent := "        " // 8 spaces: 2 marker + 5 time + 2 spaces - 1
			sb.WriteString("\n" + indent + muted.Render(strings.Join(badges, " · ")))
		}
	}

	return sb.String()
}
