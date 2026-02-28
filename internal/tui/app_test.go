package tui

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stephenbrandon/ripcode/internal/agent"
	"github.com/stretchr/testify/assert"
)

func TestNewApp(t *testing.T) {
	app := NewApp()
	assert.False(t, app.ready)
	assert.False(t, app.streaming)
}

func TestApp_WindowSize(t *testing.T) {
	app := NewApp()
	model, _ := app.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	a := model.(App)

	assert.True(t, a.ready)
	assert.Equal(t, 80, a.width)
	assert.Equal(t, 24, a.height)
}

func TestApp_View_NotReady(t *testing.T) {
	app := NewApp()
	view := app.View()
	assert.Contains(t, view.Content, "Initializing")
}

func TestApp_View_Ready(t *testing.T) {
	app := NewApp()
	model, _ := app.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	a := model.(App)

	view := a.View()
	assert.Contains(t, view.Content, "ripcode")
	assert.True(t, view.AltScreen)
}

func TestApp_EscQuits(t *testing.T) {
	app := NewApp()
	_, cmd := app.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	assert.NotNil(t, cmd)
}

func TestApp_CtrlCQuits(t *testing.T) {
	app := NewApp()
	_, cmd := app.Update(tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl})
	assert.NotNil(t, cmd)
}

func TestApp_AgentEventDone(t *testing.T) {
	app := NewApp()
	app.streaming = true

	model, _ := app.Update(AgentEventMsg{
		Event: agent.Event{Type: agent.EventDone},
	})
	a := model.(App)

	assert.False(t, a.streaming)
}

func TestApp_AgentEventError(t *testing.T) {
	app := NewApp()
	app.streaming = true

	model, _ := app.Update(AgentEventMsg{
		Event: agent.Event{
			Type:  agent.EventError,
			Error: assert.AnError,
		},
	})
	a := model.(App)

	assert.False(t, a.streaming)
}
