package tui

import (
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/stephenbrandon/ripcode/internal/session"
	"github.com/stephenbrandon/ripcode/internal/store"
	"github.com/stephenbrandon/ripcode/internal/tool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApp_FooterShowsMCPCount_FromConfig(t *testing.T) {
	t.Setenv("RIPCODE_DIR", t.TempDir())
	cfg := &store.MCPConfig{
		Servers: []store.MCPServer{
			{Name: "srv1", Enabled: true},
			{Name: "srv2", Enabled: true},
			{Name: "srv3", Enabled: false},
		},
	}
	require.NoError(t, cfg.Save())

	app := NewApp()
	app.SetProvider(&modelListProvider{})
	app.SetRegistry(tool.NewRegistry())
	app.SetSession(session.New(t.TempDir()))
	app.state = StateSession
	app.footer.SetWorkDir("/project")
	model, _ := app.Update(tea.WindowSizeMsg{Width: 160, Height: 40})
	a := model.(App)

	view := a.footer.View()
	assert.Contains(t, view, "⊙ 2")
}

func TestApp_FooterShowsLSPCount_FromConfig(t *testing.T) {
	t.Setenv("RIPCODE_DIR", t.TempDir())
	cfg := &store.LSPConfig{
		Clients: []store.LSPClient{
			{Name: "gopls", Enabled: true},
		},
	}
	require.NoError(t, cfg.Save())

	app := NewApp()
	app.SetProvider(&modelListProvider{})
	app.SetRegistry(tool.NewRegistry())
	app.SetSession(session.New(t.TempDir()))
	app.state = StateSession
	app.footer.SetWorkDir("/project")
	model, _ := app.Update(tea.WindowSizeMsg{Width: 160, Height: 40})
	a := model.(App)

	view := a.footer.View()
	assert.Contains(t, view, "• 1")
}

func TestApp_NewApp_LoadsConfigs(t *testing.T) {
	t.Setenv("RIPCODE_DIR", t.TempDir())

	app := NewApp()
	assert.NotNil(t, app.mcpConfig)
	assert.NotNil(t, app.lspConfig)
	assert.NotNil(t, app.uiPrefs)
}
