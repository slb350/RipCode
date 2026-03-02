package tui

import (
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/stephenbrandon/ripcode/internal/agent"
	"github.com/stephenbrandon/ripcode/internal/session"
	"github.com/stephenbrandon/ripcode/internal/tool"
	"github.com/stretchr/testify/assert"
)

func TestApp_CommandPalette_ShowsCategories(t *testing.T) {
	app := NewApp()
	app.SetProvider(&modelListProvider{})
	app.SetRegistry(tool.NewRegistry())
	app.SetSession(session.New(t.TempDir()))
	app.SetAgent(agent.BuildAgent())

	model, _ := app.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	a := model.(App)
	a.state = StateSession

	model, _ = a.Update(tea.KeyPressMsg{Code: 'p', Mod: tea.ModCtrl})
	a = model.(App)

	view := a.View()
	assert.Contains(t, view.Content, "Session")
	assert.Contains(t, view.Content, "View")
	assert.Contains(t, view.Content, "System")
}

func TestApp_CommandPalette_ShowsKeybindLabels(t *testing.T) {
	app := NewApp()
	app.SetProvider(&modelListProvider{})
	app.SetRegistry(tool.NewRegistry())
	app.SetSession(session.New(t.TempDir()))
	app.SetAgent(agent.BuildAgent())

	model, _ := app.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	a := model.(App)
	a.state = StateSession

	model, _ = a.Update(tea.KeyPressMsg{Code: 'p', Mod: tea.ModCtrl})
	a = model.(App)

	view := a.View()
	assert.Contains(t, view.Content, "Ctrl+B")
}

func TestApp_CommandPalette_ShowsSuggestedSection(t *testing.T) {
	app := NewApp()
	app.SetProvider(&modelListProvider{})
	app.SetRegistry(tool.NewRegistry())
	app.SetSession(session.New(t.TempDir()))
	app.SetAgent(agent.BuildAgent())

	model, _ := app.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	a := model.(App)
	a.state = StateSession

	model, _ = a.Update(tea.KeyPressMsg{Code: 'p', Mod: tea.ModCtrl})
	a = model.(App)

	view := a.View()
	assert.Contains(t, view.Content, "Suggested")
}

func TestCommandPalette_ShowsF2Keybind(t *testing.T) {
	a := makeSessionApp(t)
	cmd := a.cmdRegistry.Get("recent-model")
	assert.NotNil(t, cmd, "recent-model command should be registered")
	assert.Equal(t, "F2", cmd.Keybind)
}

func TestCommandPalette_ShowsCtrlTKeybind(t *testing.T) {
	a := makeSessionApp(t)
	cmd := a.cmdRegistry.Get("variant")
	assert.NotNil(t, cmd, "variant command should be registered")
	assert.Equal(t, "Ctrl+T", cmd.Keybind)
}
