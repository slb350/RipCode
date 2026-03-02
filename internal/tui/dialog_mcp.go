package tui

import (
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/stephenbrandon/ripcode/internal/tui/components"
)

func (a App) handleMCPDialogKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	serverCount := 0
	if a.mcpConfig != nil {
		serverCount = len(a.mcpConfig.Servers)
	}

	switch {
	case msg.Code == tea.KeyEscape || msg.Code == tea.KeyEnter:
		a.mcpDialog.open = false
		a.input.Focus()
		return a, nil

	case msg.Code == tea.KeyUp:
		if a.mcpDialog.selected > 0 {
			a.mcpDialog.selected--
		}
		return a, nil

	case msg.Code == tea.KeyDown:
		if a.mcpDialog.selected < serverCount-1 {
			a.mcpDialog.selected++
		}
		return a, nil

	case msg.Code == ' ':
		if a.mcpConfig != nil && a.mcpDialog.selected < serverCount {
			srv := a.mcpConfig.Servers[a.mcpDialog.selected]
			newState, found := a.mcpConfig.ToggleEnabled(srv.Name)
			if !found {
				a.toasts.Show(fmt.Sprintf("server %q not found", srv.Name), components.ToastWarning, 3*time.Second)
				return a, nil
			}
			a.warnOnErr(a.mcpConfig.Save(), "MCP config")
			a.footer.SetMCPCount(a.mcpConfig.CountEnabled())
			label := "disabled"
			if newState {
				label = "enabled"
			}
			a.toasts.Show(fmt.Sprintf("%s %s", srv.Name, label), components.ToastInfo, 3*time.Second)
		}
		return a, nil

	default:
		return a, nil
	}
}

func (a App) renderMCPDialog() string {
	var sb strings.Builder
	sb.WriteString("MCP Servers (Space toggle, Esc close)\n")

	if a.mcpConfig == nil || len(a.mcpConfig.Servers) == 0 {
		sb.WriteString("\n  No MCP servers configured")
		sb.WriteString("\n  Add servers to ~/.ripcode/state/mcp.json")
		return sb.String()
	}

	for i, srv := range a.mcpConfig.Servers {
		prefix := "  "
		if i == a.mcpDialog.selected {
			prefix = "> "
		}
		icon := enabledIcon(srv.Enabled)
		detail := srv.Command
		if detail == "" {
			detail = srv.URL
		}
		sb.WriteString(fmt.Sprintf("\n%s%s %s", prefix, icon, srv.Name))
		if detail != "" {
			sb.WriteString(fmt.Sprintf(" — %s", detail))
		}
	}

	sb.WriteString(fmt.Sprintf("\n\n  %d/%d enabled", a.mcpConfig.CountEnabled(), len(a.mcpConfig.Servers)))
	return sb.String()
}
