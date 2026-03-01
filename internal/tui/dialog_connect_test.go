package tui

import (
	"os"
	"path/filepath"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/stephenbrandon/ripcode/internal/tui/components"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApp_ConnectCommand_OpensDialog(t *testing.T) {
	a := makeSessionApp(t)
	model, _ := a.Update(components.InputSubmitMsg{Value: "/connect"})
	a = model.(App)
	assert.True(t, a.connectDialog.open)
}

func TestApp_ConnectDialog_EscCloses(t *testing.T) {
	a := makeSessionApp(t)
	model, _ := a.Update(components.InputSubmitMsg{Value: "/connect"})
	a = model.(App)
	model, _ = a.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	a = model.(App)
	assert.False(t, a.connectDialog.open)
}

func TestConnectCommand_OpensDialog(t *testing.T) {
	a := makeSessionApp(t)
	model, _ := a.Update(components.InputSubmitMsg{Value: "/connect"})
	a = model.(App)
	assert.True(t, a.connectDialog.open)
}

func TestConnectDialog_AcceptsTextInput(t *testing.T) {
	a := makeSessionApp(t)
	a.connectDialog.open = true
	model, _ := a.Update(tea.KeyPressMsg{Code: 's', Text: "s"})
	a = model.(App)
	assert.Equal(t, "s", a.connectDialog.input)
}

func TestConnectDialog_Escape_Closes(t *testing.T) {
	a := makeSessionApp(t)
	a.connectDialog.open = true
	a.connectDialog.input = "some-key"
	model, _ := a.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	a = model.(App)
	assert.False(t, a.connectDialog.open)
	assert.Equal(t, "", a.connectDialog.input)
}

func TestConnectDialog_ShowsCurrentStatus(t *testing.T) {
	a := makeSessionApp(t)
	a.connectDialog.open = true
	rendered := a.renderConnectDialog()
	assert.Contains(t, rendered, "connected")
}

func TestConnectDialog_EmptyInput_ShowsError(t *testing.T) {
	a := makeSessionApp(t)
	a.connectDialog.open = true
	a.connectDialog.input = ""
	model, cmd := a.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	a = model.(App)
	assert.True(t, a.connectDialog.open, "dialog should stay open on empty input")
	assert.NotNil(t, cmd, "should show toast")
}

func TestConnectDialog_Save_ShowsRestartRequiredAndPersistsKey(t *testing.T) {
	a := makeSessionApp(t)
	a.connectDialog.open = true
	a.connectDialog.input = "sk-test-key"

	model, cmd := a.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	a = model.(App)

	assert.NotNil(t, cmd)
	assert.False(t, a.connectDialog.open)
	toast := a.toasts.Current()
	require.NotNil(t, toast)
	assert.Contains(t, toast.Message, "Restart")

	envPath := filepath.Join(a.session.WorkDir, ".env")
	data, err := os.ReadFile(envPath)
	require.NoError(t, err)
	assert.Contains(t, string(data), "OPENROUTER_API_KEY")
	assert.Contains(t, string(data), "sk-test-key")
}
