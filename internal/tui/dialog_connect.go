package tui

import (
	"os"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/stephenbrandon/ripcode/internal/config"
	"github.com/stephenbrandon/ripcode/internal/tui/components"
)

func (a App) handleConnectDialogKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case msg.Code == tea.KeyEscape:
		a.connectDialogOpen = false
		a.connectDialogInput = ""
		a.input.Focus()
		return a, nil

	case msg.Code == tea.KeyEnter:
		key := strings.TrimSpace(a.connectDialogInput)
		if key == "" {
			return a, a.ShowToast("API key cannot be empty", components.ToastError)
		}
		// Save the key
		workDir := ""
		if a.session != nil {
			workDir = a.session.WorkDir
		}
		if workDir == "" {
			workDir, _ = os.Getwd()
		}
		if err := config.SaveAPIKey(workDir, key); err != nil {
			return a, a.ShowToast("Save failed: "+err.Error(), components.ToastError)
		}
		a.connectDialogOpen = false
		a.connectDialogInput = ""
		a.input.Focus()
		return a, a.ShowToast("API key saved", components.ToastSuccess)

	case msg.Code == tea.KeyBackspace:
		a.connectDialogInput = backspaceRune(a.connectDialogInput)
		return a, nil

	default:
		if msg.Text != "" {
			a.connectDialogInput += msg.Text
		}
		return a, nil
	}
}

func (a App) renderConnectDialog() string {
	var sb strings.Builder
	sb.WriteString("Provider Connection (Enter save, Esc close)\n")
	sb.WriteString("\n  Provider: OpenRouter")
	if a.provider != nil {
		sb.WriteString(" (connected)")
	} else {
		sb.WriteString(" (not connected)")
	}
	sb.WriteString("\n  Model: " + a.model)
	sb.WriteString("\n\n  API Key: ")
	if a.connectDialogInput == "" {
		sb.WriteString("(type key here)")
	} else {
		sb.WriteString(strings.Repeat("*", len(a.connectDialogInput)))
	}
	sb.WriteString("\n\n  Enter an OpenRouter API key to save to .env")
	return sb.String()
}
