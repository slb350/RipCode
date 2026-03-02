package tui

import (
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/stephenbrandon/ripcode/internal/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApp_MCPDialog_OpensWithCommand(t *testing.T) {
	a := makeSidebarApp(t)
	a.mcpConfig = &store.MCPConfig{
		Servers: []store.MCPServer{
			{Name: "srv1", Command: "cmd1", Enabled: true},
		},
	}

	cmd := a.cmdRegistry.Get("mcp")
	require.NotNil(t, cmd)
	cmd.Handler(&a)

	assert.True(t, a.mcpDialog.open)
}

func TestApp_MCPDialog_ShowsServers(t *testing.T) {
	a := makeSidebarApp(t)
	a.mcpConfig = &store.MCPConfig{
		Servers: []store.MCPServer{
			{Name: "github-mcp", Command: "gh mcp", Enabled: true},
			{Name: "api-srv", URL: "http://localhost", Enabled: false},
		},
	}
	a.mcpDialog.open = true

	view := a.renderMCPDialog()
	assert.Contains(t, view, "github-mcp")
	assert.Contains(t, view, "api-srv")
}

func TestApp_MCPDialog_SpaceToggle(t *testing.T) {
	a := makeSidebarApp(t)
	a.mcpConfig = &store.MCPConfig{
		Servers: []store.MCPServer{
			{Name: "srv1", Command: "cmd1", Enabled: true},
		},
	}
	a.mcpDialog.open = true
	a.mcpDialog.selected = 0

	model, _ := a.Update(tea.KeyPressMsg{Code: ' '})
	a = model.(App)

	assert.False(t, a.mcpConfig.Servers[0].Enabled)
}

func TestApp_MCPDialog_Navigate(t *testing.T) {
	a := makeSidebarApp(t)
	a.mcpConfig = &store.MCPConfig{
		Servers: []store.MCPServer{
			{Name: "srv1", Command: "cmd1", Enabled: true},
			{Name: "srv2", Command: "cmd2", Enabled: false},
		},
	}
	a.mcpDialog.open = true
	a.mcpDialog.selected = 0

	model, _ := a.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	a = model.(App)

	assert.Equal(t, 1, a.mcpDialog.selected)
}

func TestApp_MCPDialog_EscapeCloses(t *testing.T) {
	a := makeSidebarApp(t)
	a.mcpConfig = &store.MCPConfig{}
	a.mcpDialog.open = true

	model, _ := a.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	a = model.(App)

	assert.False(t, a.mcpDialog.open)
}

func TestApp_MCPDialog_ToastOnToggle(t *testing.T) {
	a := makeSidebarApp(t)
	a.mcpConfig = &store.MCPConfig{
		Servers: []store.MCPServer{
			{Name: "srv1", Command: "cmd1", Enabled: true},
		},
	}
	a.mcpDialog.open = true
	a.mcpDialog.selected = 0

	model, _ := a.Update(tea.KeyPressMsg{Code: ' '})
	a = model.(App)

	toastView := a.toasts.View()
	assert.Contains(t, toastView, "srv1")
}
