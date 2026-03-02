package tui

import (
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/stephenbrandon/ripcode/internal/agent"
	"github.com/stephenbrandon/ripcode/internal/tui/components"
)

func (a App) filteredAgents() []agent.AgentInfo {
	all := agent.AllAgents()
	q := strings.ToLower(strings.TrimSpace(a.agentDialog.query))
	if q == "" {
		return all
	}
	var out []agent.AgentInfo
	for _, ag := range all {
		if strings.Contains(strings.ToLower(ag.Name), q) || strings.Contains(strings.ToLower(ag.Description), q) {
			out = append(out, ag)
		}
	}
	return out
}

func (a App) handleAgentDialogKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case msg.Code == tea.KeyEscape:
		a.agentDialog.open = false
		a.agentDialog.query = ""
		a.agentDialog.selected = 0
		a.input.Focus()
		return a, nil

	case msg.Code == tea.KeyEnter:
		agents := a.filteredAgents()
		if len(agents) == 0 {
			a.agentDialog.open = false
			a.input.Focus()
			return a, nil
		}
		sel := clamp(a.agentDialog.selected, 0, len(agents)-1)
		selected := agents[sel]
		a.agentDialog.open = false
		a.agentDialog.query = ""
		a.agentDialog.selected = 0
		a.input.Focus()
		switch selected.Name {
		case agent.NameBuild:
			a.SetAgent(agent.BuildAgent())
		case agent.NamePlan:
			a.SetAgent(agent.PlanAgent())
		}
		return a, a.ShowToast("Agent: "+selected.Name, components.ToastSuccess)

	case msg.Code == tea.KeyUp:
		agents := a.filteredAgents()
		if len(agents) == 0 {
			return a, nil
		}
		a.agentDialog.selected--
		if a.agentDialog.selected < 0 {
			a.agentDialog.selected = len(agents) - 1
		}
		return a, nil

	case msg.Code == tea.KeyDown:
		agents := a.filteredAgents()
		if len(agents) == 0 {
			return a, nil
		}
		a.agentDialog.selected++
		if a.agentDialog.selected >= len(agents) {
			a.agentDialog.selected = 0
		}
		return a, nil

	case msg.Code == tea.KeyBackspace:
		a.agentDialog.query = backspaceRune(a.agentDialog.query)
		a.agentDialog.selected = 0
		return a, nil

	default:
		if msg.Text != "" {
			a.agentDialog.query += msg.Text
			a.agentDialog.selected = 0
		}
		return a, nil
	}
}

func (a App) renderAgentDialog() string {
	agents := a.filteredAgents()
	query := strings.TrimSpace(a.agentDialog.query)
	header := "Select agent (Enter choose, Esc close)"
	if query != "" {
		header += " - filter: " + query
	}

	items := make([]pickerItem, len(agents))
	for i, ag := range agents {
		label := ag.Name
		if ag.Name == a.agent.Name {
			label = "● " + label
		}
		if ag.Native {
			label += "  [native]"
		}
		items[i] = pickerItem{Label: label, Description: ag.Description}
	}
	return renderPickerList(header, items, a.agentDialog.selected, 9)
}
