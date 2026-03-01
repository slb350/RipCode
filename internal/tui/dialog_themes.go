package tui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/stephenbrandon/ripcode/internal/tui/components"
)

func (a App) handleThemesDialogKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case msg.Code == tea.KeyEscape:
		a.themesDialogOpen = false
		a.input.Focus()
		return a, nil
	case msg.Code == tea.KeyEnter:
		a.themesDialogOpen = false
		a.input.Focus()
		return a, a.ShowToast("Theme applied", components.ToastSuccess)
	case msg.Code == tea.KeyUp:
		if a.themesDialogSelect > 0 {
			a.themesDialogSelect--
		}
		return a, nil
	case msg.Code == tea.KeyDown:
		if a.themesDialogSelect < 2 {
			a.themesDialogSelect++
		}
		return a, nil
	default:
		return a, nil
	}
}

func (a App) renderThemesDialog() string {
	themes := []string{"Default", "Dark", "Light"}
	var sb strings.Builder
	sb.WriteString("Themes (Enter select, Esc close)\n")
	for i, name := range themes {
		marker := "  "
		if i == a.themesDialogSelect {
			marker = "> "
		}
		sb.WriteString(fmt.Sprintf("\n%s%s", marker, name))
	}
	return sb.String()
}
