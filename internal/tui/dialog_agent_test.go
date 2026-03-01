package tui

import (
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/stretchr/testify/assert"
)

func TestAgentDialog_ListsAllAgents(t *testing.T) {
	a := makeSessionApp(t)
	a.agentDialogOpen = true
	rendered := a.renderAgentDialog()
	assert.Contains(t, rendered, "build")
	assert.Contains(t, rendered, "plan")
}

func TestAgentDialog_FilterByQuery(t *testing.T) {
	a := makeSessionApp(t)
	a.agentDialogOpen = true
	a.agentDialogQuery = "plan"
	rendered := a.renderAgentDialog()
	assert.Contains(t, rendered, "plan")
	assert.NotContains(t, rendered, "> build")
}

func TestAgentDialog_Enter_SwitchesAgent(t *testing.T) {
	a := makeSessionApp(t)
	a.agentDialogOpen = true
	a.agentDialogSelect = 1 // plan is second
	model, _ := a.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	a = model.(App)
	assert.False(t, a.agentDialogOpen)
	assert.Equal(t, "plan", a.agent.Name)
}

func TestAgentDialog_Escape_Closes(t *testing.T) {
	a := makeSessionApp(t)
	a.agentDialogOpen = true
	model, _ := a.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	a = model.(App)
	assert.False(t, a.agentDialogOpen)
}

func TestAgentDialog_ShowsNativeIndicator(t *testing.T) {
	a := makeSessionApp(t)
	a.agentDialogOpen = true
	rendered := a.renderAgentDialog()
	assert.Contains(t, rendered, "[native]")
}

func TestAgentDialog_ShowsCurrentAgentMarker(t *testing.T) {
	a := makeSessionApp(t)
	a.agentDialogOpen = true
	rendered := a.renderAgentDialog()
	assert.Contains(t, rendered, "● build", "current agent should have marker")
}
