package tui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
)

type statusItem struct {
	Name    string
	Enabled bool
	Detail  string
}

func renderStatusSection(sb *strings.Builder, title string, items []statusItem) {
	sb.WriteString("\n\n")
	sb.WriteString(title)
	if len(items) == 0 {
		sb.WriteString("\n  (none connected)")
		return
	}
	enabled := 0
	for _, item := range items {
		state := "disabled"
		if item.Enabled {
			state = "enabled"
			enabled++
		}
		if item.Detail == "" {
			sb.WriteString(fmt.Sprintf("\n  %s (%s)", item.Name, state))
		} else {
			sb.WriteString(fmt.Sprintf("\n  %s (%s) — %s", item.Name, state, item.Detail))
		}
	}
	sb.WriteString(fmt.Sprintf("\n  %d/%d enabled", enabled, len(items)))
}

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

	var mcpItems []statusItem
	if a.mcpConfig != nil {
		for _, srv := range a.mcpConfig.Servers {
			detail := srv.Command
			if detail == "" {
				detail = srv.URL
			}
			mcpItems = append(mcpItems, statusItem{srv.Name, srv.Enabled, detail})
		}
	}
	renderStatusSection(&sb, "MCP Servers", mcpItems)

	var lspItems []statusItem
	if a.lspConfig != nil {
		for _, cl := range a.lspConfig.Clients {
			lspItems = append(lspItems, statusItem{cl.Name, cl.Enabled, cl.Root})
		}
	}
	renderStatusSection(&sb, "LSP Clients", lspItems)

	sb.WriteString("\n\nFormatters")
	sb.WriteString("\n  (none configured)")

	return sb.String()
}
