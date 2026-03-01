package tui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
)

func (a App) handleStatusDialogKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case msg.Code == tea.KeyEscape, msg.Code == tea.KeyEnter:
		a.statusDialog.open = false
		a.input.Focus()
		return a, nil
	default:
		return a, nil
	}
}

func (a App) renderStatusDialog() string {
	var sb strings.Builder
	sb.WriteString("Status                                    esc\n")

	modelName := a.model
	if modelName == "" {
		modelName = "(not set)"
	}
	agentName := a.agent.Name
	if agentName == "" {
		agentName = "build"
	}
	providerName := "(none)"
	if a.provider != nil {
		providerName = a.provider.Name()
	}

	sb.WriteString("\nSystem")
	sb.WriteString(fmt.Sprintf("\n  Model       %s", modelName))
	sb.WriteString(fmt.Sprintf("\n  Agent       %s", agentName))
	sb.WriteString(fmt.Sprintf("\n  Provider    %s", providerName))

	sb.WriteString("\n\nSession")
	msgCount := 0
	tokIn := 0
	tokOut := 0
	workDir := "(not set)"
	if a.session != nil {
		msgCount = a.session.Len()
		tokIn = a.session.Tokens.Input
		tokOut = a.session.Tokens.Output
		workDir = a.session.WorkDir
	}
	sb.WriteString(fmt.Sprintf("\n  Messages    %d messages", msgCount))
	sb.WriteString(fmt.Sprintf("\n  Tokens      %s in / %s out", formatNumber(tokIn), formatNumber(tokOut)))
	sb.WriteString(fmt.Sprintf("\n  WorkDir     %s", workDir))

	sb.WriteString("\n\nMCP Servers")
	sb.WriteString("\n  (none connected)")

	sb.WriteString("\n\nLSP Clients")
	sb.WriteString("\n  (none connected)")

	sb.WriteString("\n\nFormatters")
	sb.WriteString("\n  (none configured)")

	return sb.String()
}
