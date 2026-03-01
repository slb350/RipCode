package tui

import (
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/stephenbrandon/ripcode/internal/store"
	"github.com/stephenbrandon/ripcode/internal/tui/components"
)

func (a App) handleRenameDialogKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case msg.Code == tea.KeyEscape:
		a.renameDialog.open = false
		a.input.Focus()
		return a, nil

	case msg.Code == tea.KeyEnter:
		if strings.TrimSpace(a.renameDialog.value) == "" {
			// Stay open — empty title not allowed
			return a, nil
		}
		a.renameDialog.open = false
		a.input.Focus()
		if a.session != nil {
			a.session.Title = a.renameDialog.value
			a.statusbar.SetTitle(a.renameDialog.value)
			a.warnOnErr(store.Save(a.session), "session")
		}
		id := a.toasts.Show(fmt.Sprintf("Renamed to \"%s\"", a.renameDialog.value), components.ToastSuccess, 3*time.Second)
		return a, func() tea.Msg {
			time.Sleep(3 * time.Second)
			return ToastDismissMsg{ID: id}
		}

	case msg.Code == tea.KeyBackspace:
		a.renameDialog.value = backspaceRune(a.renameDialog.value)
		return a, nil

	default:
		if msg.Text != "" {
			a.renameDialog.value += msg.Text
		}
		return a, nil
	}
}

func (a App) renderRenameDialog() string {
	var sb strings.Builder
	sb.WriteString("Rename session                            esc\n")
	sb.WriteString(fmt.Sprintf("\n  > %s█", a.renameDialog.value))
	sb.WriteString("\n\n  [Enter] save  [Esc] cancel")
	return sb.String()
}
