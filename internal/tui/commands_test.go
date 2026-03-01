package tui

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRegistry_ConnectCommand_Registered(t *testing.T) {
	a := makeSessionApp(t)
	cmd := a.cmdRegistry.Get("connect")
	assert.NotNil(t, cmd)
	assert.Equal(t, CategorySystem, cmd.Category)
}

func TestRegistry_AgentCommand_HasLeaderKeybind(t *testing.T) {
	a := makeSessionApp(t)
	cmd := a.cmdRegistry.Get("agent")
	assert.NotNil(t, cmd)
	assert.Equal(t, "ctrl+x a", cmd.Keybind)
}

func TestRegistry_ModelPickerEnhancements_InPalette(t *testing.T) {
	a := makeSessionApp(t)
	// Models command should exist
	cmd := a.cmdRegistry.Get("models")
	assert.NotNil(t, cmd)
	assert.Equal(t, CategoryAgent, cmd.Category)
}
