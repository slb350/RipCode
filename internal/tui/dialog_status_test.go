package tui

import (
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/stephenbrandon/ripcode/internal/tui/components"
	"github.com/stretchr/testify/assert"
)

func TestApp_StatusDialog_OpensWithSlashStatus(t *testing.T) {
	a := makeSessionApp(t)
	model, _ := a.Update(components.InputSubmitMsg{Value: "/status"})
	a = model.(App)
	assert.True(t, a.statusDialog.open)
}

func TestApp_StatusDialog_ShowsModel(t *testing.T) {
	a := makeSessionApp(t)
	a.SetModel("glm-5")
	model, _ := a.Update(components.InputSubmitMsg{Value: "/status"})
	a = model.(App)
	view := a.View()
	assert.Contains(t, view.Content, "glm-5")
}

func TestApp_StatusDialog_ShowsAgent(t *testing.T) {
	a := makeSessionApp(t)
	model, _ := a.Update(components.InputSubmitMsg{Value: "/status"})
	a = model.(App)
	view := a.View()
	assert.Contains(t, view.Content, "build")
}

func TestApp_StatusDialog_ShowsTokenCount(t *testing.T) {
	a := makeSessionApp(t)
	a.session.AddTokens(4521, 2103)
	model, _ := a.Update(components.InputSubmitMsg{Value: "/status"})
	a = model.(App)
	view := a.View()
	assert.Contains(t, view.Content, "4,521")
}

func TestApp_StatusDialog_ShowsWorkDir(t *testing.T) {
	a := makeSessionApp(t)
	model, _ := a.Update(components.InputSubmitMsg{Value: "/status"})
	a = model.(App)
	view := a.View()
	assert.Contains(t, view.Content, a.session.WorkDir)
}

func TestApp_StatusDialog_ShowsMessageCount(t *testing.T) {
	a := makeSessionApp(t)
	a.session.AddUser("test")
	a.session.AddAssistant("reply", nil, nil)
	model, _ := a.Update(components.InputSubmitMsg{Value: "/status"})
	a = model.(App)
	view := a.View()
	assert.Contains(t, view.Content, "2 messages")
}

func TestApp_StatusDialog_EscapeCloses(t *testing.T) {
	a := makeSessionApp(t)
	model, _ := a.Update(components.InputSubmitMsg{Value: "/status"})
	a = model.(App)
	assert.True(t, a.statusDialog.open)
	model, _ = a.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	a = model.(App)
	assert.False(t, a.statusDialog.open)
}

func TestApp_StatusDialog_ClosesOtherDialogs(t *testing.T) {
	a := makeSessionApp(t)
	a.commandPalette.open = true
	model, _ := a.Update(components.InputSubmitMsg{Value: "/status"})
	a = model.(App)
	assert.True(t, a.statusDialog.open)
	assert.False(t, a.commandPalette.open)
}

func TestApp_StatusDialog_ShowsMCPSection(t *testing.T) {
	a := makeSessionApp(t)
	model, _ := a.Update(components.InputSubmitMsg{Value: "/status"})
	a = model.(App)
	rendered := a.renderStatusDialog()
	assert.Contains(t, rendered, "\n\nMCP Servers\n")
}

func TestApp_StatusDialog_ShowsLSPSection(t *testing.T) {
	a := makeSessionApp(t)
	model, _ := a.Update(components.InputSubmitMsg{Value: "/status"})
	a = model.(App)
	rendered := a.renderStatusDialog()
	assert.Contains(t, rendered, "\n\nLSP Clients\n")
}

func TestApp_StatusDialog_ShowsFormattersSection(t *testing.T) {
	a := makeSessionApp(t)
	model, _ := a.Update(components.InputSubmitMsg{Value: "/status"})
	a = model.(App)
	rendered := a.renderStatusDialog()
	assert.Contains(t, rendered, "\n\nFormatters\n")
}
