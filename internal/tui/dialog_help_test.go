package tui

import (
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/stephenbrandon/ripcode/internal/tui/components"
	"github.com/stretchr/testify/assert"
)

func TestApp_HelpDialog_OpensWithSlashHelp(t *testing.T) {
	a := makeSessionApp(t)
	model, _ := a.Update(components.InputSubmitMsg{Value: "/help"})
	a = model.(App)
	assert.True(t, a.helpDialogOpen)
}

func TestApp_HelpDialog_ShowsCommands(t *testing.T) {
	a := makeSessionApp(t)
	model, _ := a.Update(components.InputSubmitMsg{Value: "/help"})
	a = model.(App)
	view := a.View()
	assert.Contains(t, view.Content, "/new")
	assert.Contains(t, view.Content, "/models")
}

func TestApp_HelpDialog_ShowsKeybinds(t *testing.T) {
	a := makeSessionApp(t)
	model, _ := a.Update(components.InputSubmitMsg{Value: "/help"})
	a = model.(App)
	// Switch to keybinds tab
	model, _ = a.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	a = model.(App)
	view := a.View()
	assert.Contains(t, view.Content, "Ctrl+A")
}

func TestApp_HelpDialog_FilterReducesResults(t *testing.T) {
	a := makeSessionApp(t)
	model, _ := a.Update(components.InputSubmitMsg{Value: "/help"})
	a = model.(App)
	// Type to filter
	model, _ = a.Update(tea.KeyPressMsg{Text: "m"})
	a = model.(App)
	model, _ = a.Update(tea.KeyPressMsg{Text: "o"})
	a = model.(App)
	model, _ = a.Update(tea.KeyPressMsg{Text: "d"})
	a = model.(App)
	view := a.View()
	assert.Contains(t, view.Content, "/model")
}

func TestApp_HelpDialog_EscapeCloses(t *testing.T) {
	a := makeSessionApp(t)
	model, _ := a.Update(components.InputSubmitMsg{Value: "/help"})
	a = model.(App)
	assert.True(t, a.helpDialogOpen)
	model, _ = a.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	a = model.(App)
	assert.False(t, a.helpDialogOpen)
}

func TestApp_HelpDialog_TabSwitchesSections(t *testing.T) {
	a := makeSessionApp(t)
	model, _ := a.Update(components.InputSubmitMsg{Value: "/help"})
	a = model.(App)
	assert.Equal(t, 0, a.helpDialogTab)
	model, _ = a.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	a = model.(App)
	assert.Equal(t, 1, a.helpDialogTab)
	model, _ = a.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	a = model.(App)
	assert.Equal(t, 0, a.helpDialogTab)
}

func TestApp_HelpDialog_EnterDoesNotCrash(t *testing.T) {
	a := makeSessionApp(t)
	model, _ := a.Update(components.InputSubmitMsg{Value: "/help"})
	a = model.(App)
	model, _ = a.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	a = model.(App)
	assert.False(t, a.helpDialogOpen)
}

func TestApp_HelpDialog_ClosesOtherDialogs(t *testing.T) {
	a := makeSessionApp(t)
	a.commandOpen = true
	model, _ := a.Update(components.InputSubmitMsg{Value: "/help"})
	a = model.(App)
	assert.True(t, a.helpDialogOpen)
	assert.False(t, a.commandOpen)
}
