package tui

import (
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/stephenbrandon/ripcode/internal/tui/components"
	"github.com/stretchr/testify/assert"
)

func TestApp_ConnectCommand_OpensDialog(t *testing.T) {
	a := makeSessionApp(t)
	model, _ := a.Update(components.InputSubmitMsg{Value: "/connect"})
	a = model.(App)
	assert.True(t, a.connectDialogOpen)
}

func TestApp_ConnectDialog_EscCloses(t *testing.T) {
	a := makeSessionApp(t)
	model, _ := a.Update(components.InputSubmitMsg{Value: "/connect"})
	a = model.(App)
	model, _ = a.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	a = model.(App)
	assert.False(t, a.connectDialogOpen)
}

func TestConnectCommand_OpensDialog(t *testing.T) {
	a := makeSessionApp(t)
	model, _ := a.Update(components.InputSubmitMsg{Value: "/connect"})
	a = model.(App)
	assert.True(t, a.connectDialogOpen)
}

func TestConnectDialog_AcceptsTextInput(t *testing.T) {
	a := makeSessionApp(t)
	a.connectDialogOpen = true
	model, _ := a.Update(tea.KeyPressMsg{Code: 's', Text: "s"})
	a = model.(App)
	assert.Equal(t, "s", a.connectDialogInput)
}

func TestConnectDialog_Escape_Closes(t *testing.T) {
	a := makeSessionApp(t)
	a.connectDialogOpen = true
	a.connectDialogInput = "some-key"
	model, _ := a.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	a = model.(App)
	assert.False(t, a.connectDialogOpen)
	assert.Equal(t, "", a.connectDialogInput)
}

func TestConnectDialog_ShowsCurrentStatus(t *testing.T) {
	a := makeSessionApp(t)
	a.connectDialogOpen = true
	rendered := a.renderConnectDialog()
	assert.Contains(t, rendered, "connected")
}

func TestConnectDialog_EmptyInput_ShowsError(t *testing.T) {
	a := makeSessionApp(t)
	a.connectDialogOpen = true
	a.connectDialogInput = ""
	model, cmd := a.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	a = model.(App)
	assert.True(t, a.connectDialogOpen, "dialog should stay open on empty input")
	assert.NotNil(t, cmd, "should show toast")
}
