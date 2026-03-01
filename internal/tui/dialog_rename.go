package tui

import (
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/stephenbrandon/ripcode/internal/tui/components"
)

func (a App) handleRenameDialogKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case msg.Code == tea.KeyEscape:
		a.renameDialogOpen = false
		a.input.Focus()
		return a, nil

	case msg.Code == tea.KeyEnter:
		if strings.TrimSpace(a.renameDialogValue) == "" {
			// Stay open — empty title not allowed
			return a, nil
		}
		a.renameDialogOpen = false
		a.input.Focus()
		if a.session != nil {
			a.session.Title = a.renameDialogValue
			a.statusbar.SetTitle(a.renameDialogValue)
		}
		id := a.toasts.Show(fmt.Sprintf("Renamed to \"%s\"", a.renameDialogValue), components.ToastSuccess, 3*time.Second)
		return a, func() tea.Msg {
			time.Sleep(3 * time.Second)
			return ToastDismissMsg{ID: id}
		}

	case msg.Code == tea.KeyBackspace:
		a.renameDialogValue = backspaceRune(a.renameDialogValue)
		return a, nil

	default:
		if msg.Text != "" {
			a.renameDialogValue += msg.Text
		}
		return a, nil
	}
}

func (a App) renderRenameDialog() string {
	var sb strings.Builder
	sb.WriteString("Rename session                            esc\n")
	sb.WriteString(fmt.Sprintf("\n  > %s█", a.renameDialogValue))
	sb.WriteString("\n\n  [Enter] save  [Esc] cancel")
	return sb.String()
}
