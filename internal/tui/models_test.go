package tui

import (
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/stephenbrandon/ripcode/internal/agent"
	"github.com/stephenbrandon/ripcode/internal/provider"
	"github.com/stephenbrandon/ripcode/internal/session"
	"github.com/stephenbrandon/ripcode/internal/tool"
	"github.com/stephenbrandon/ripcode/internal/tui/components"
	"github.com/stretchr/testify/assert"
)

func TestApp_ModelsCommand_AsyncFetchAndCache(t *testing.T) {
	p := &modelListProvider{
		models: []provider.ModelInfo{
			{ID: "anthropic/claude-sonnet-4", Name: "Claude Sonnet 4"},
			{ID: "openai/gpt-4o", Name: "GPT-4o"},
		},
	}

	app := NewApp()
	app.SetProvider(p)
	app.SetRegistry(tool.NewRegistry())
	app.SetSession(session.New(t.TempDir()))
	app.SetAgent(agent.BuildAgent())
	app.SetModel("glm-5")

	model, _ := app.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	a := model.(App)

	// First /models — cache miss, should return a Cmd (async fetch).
	model, cmd := a.Update(components.InputSubmitMsg{Value: "/models"})
	a = model.(App)
	assert.NotNil(t, cmd, "cache miss must return a fetch command")
	assert.Equal(t, 0, p.calls, "provider should not be called synchronously")

	// Execute the command to simulate the async fetch completing.
	msg := cmd()
	loadedMsg, ok := msg.(ModelsLoadedMsg)
	assert.True(t, ok, "cmd must produce ModelsLoadedMsg")
	assert.Nil(t, loadedMsg.Err)
	assert.Len(t, loadedMsg.Models, 2)
	assert.Equal(t, 1, p.calls, "provider called once by the async cmd")

	// Feed the message back into Update.
	model, cmd = a.Update(loadedMsg)
	a = model.(App)
	assert.Nil(t, cmd, "loaded message should not produce another cmd")
	assert.True(t, a.modelsLoaded, "cache should be populated")
	assert.True(t, a.modelDialogOpen, "/models should open the model picker dialog")
	view := a.View()
	assert.Contains(t, view.Content, "Select model")
	assert.Contains(t, view.Content, "anthropic/claude-sonnet-4")
	assert.Contains(t, view.Content, "openai/gpt-4o")

	// Second /models claude — cache hit, should NOT produce a Cmd.
	model, cmd = a.Update(components.InputSubmitMsg{Value: "/models claude"})
	a = model.(App)
	assert.Nil(t, cmd, "cache hit must not produce a command")
	assert.Equal(t, 1, p.calls, "provider should not be called again")
	assert.True(t, a.modelDialogOpen, "cache hit should reopen model picker")
	assert.Equal(t, "claude", a.modelDialogQuery)
	view = a.View()
	assert.Contains(t, view.Content, "filter: claude")
	assert.Contains(t, view.Content, "anthropic/claude-sonnet-4")
}

func TestApp_ModelSlashCommand_SetsProviderModel(t *testing.T) {
	p := &modelListProvider{}
	app := NewApp()
	app.SetProvider(p)
	app.SetRegistry(tool.NewRegistry())
	app.SetSession(session.New(t.TempDir()))
	app.SetAgent(agent.BuildAgent())
	app.SetModel("glm-5")

	model, _ := app.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	a := model.(App)

	model, _ = a.Update(components.InputSubmitMsg{Value: "/model openai/gpt-4o"})
	a = model.(App)

	assert.Equal(t, "openai/gpt-4o", p.model)
	assert.Equal(t, "gpt-4o", a.model)
	assert.Contains(t, a.View().Content, `Model switched to "openai/gpt-4o".`)
}

func TestSwitchModel_AddsToRecent(t *testing.T) {
	a := makeSessionApp(t)
	a.switchModel("anthropic/claude-4")
	assert.Len(t, a.modelPrefs.Recent, 1)
	assert.Equal(t, "anthropic/claude-4", a.modelPrefs.Recent[0].ModelID)
}

func TestApp_SwitchModel_ClearsIncompatibleVariant(t *testing.T) {
	a := makeSessionApp(t)
	a.fullModelID = "anthropic/claude-sonnet-4-thinking"
	a.activeVariant = "high"
	// Switch to a model without variants
	a.modelsCache = []provider.ModelInfo{{ID: "openai/gpt-4o", Name: "GPT-4o"}}
	a.modelsLoaded = true
	a.switchModel("openai/gpt-4o")
	assert.Equal(t, "", a.activeVariant)
}

func TestFormatContextLength_EdgeCases(t *testing.T) {
	assert.Equal(t, "1M", formatContextLength(1000000))
	assert.Equal(t, "1M", formatContextLength(999999))
	assert.Equal(t, "200K", formatContextLength(200000))
	assert.Equal(t, "", formatContextLength(0))
}

func TestFilterModels_ExactMatch_RanksFirst(t *testing.T) {
	models := []provider.ModelInfo{
		{ID: "openai/gpt-4o", Name: "GPT-4o"},
		{ID: "anthropic/claude-4", Name: "Claude 4"},
		{ID: "openai/gpt-4o-mini", Name: "GPT-4o Mini"},
	}
	result := filterModels(models, "gpt-4o")
	assert.NotEmpty(t, result)
	assert.Equal(t, "openai/gpt-4o", result[0].ID)
}

func TestFilterModels_FuzzyMatch_FindsPartial(t *testing.T) {
	models := []provider.ModelInfo{
		{ID: "anthropic/claude-sonnet-4", Name: "Claude Sonnet 4"},
		{ID: "openai/gpt-4o", Name: "GPT-4o"},
	}
	result := filterModels(models, "clde")
	if assert.NotEmpty(t, result, "fuzzy search should match 'clde' to 'claude'") {
		assert.Equal(t, "anthropic/claude-sonnet-4", result[0].ID)
	}
}

func TestFilterModels_CaseInsensitive(t *testing.T) {
	models := []provider.ModelInfo{
		{ID: "anthropic/claude-4", Name: "Claude 4"},
	}
	result := filterModels(models, "CLAUDE")
	assert.Len(t, result, 1)
}

func TestFilterModels_EmptyQuery_ReturnsAll(t *testing.T) {
	models := []provider.ModelInfo{
		{ID: "a/m1", Name: "M1"},
		{ID: "b/m2", Name: "M2"},
	}
	result := filterModels(models, "")
	assert.Len(t, result, 2)
}

func TestFilterModels_NoMatch_ReturnsEmpty(t *testing.T) {
	models := []provider.ModelInfo{
		{ID: "anthropic/claude-4", Name: "Claude 4"},
	}
	result := filterModels(models, "zzzznotamodel")
	assert.Empty(t, result)
}

func TestFilterModels_MatchesBothIDAndName(t *testing.T) {
	models := []provider.ModelInfo{
		{ID: "x/hidden-model", Name: "Visible Name"},
	}
	// Match by name
	result := filterModels(models, "Visible")
	assert.Len(t, result, 1)
	// Match by ID
	result = filterModels(models, "hidden")
	assert.Len(t, result, 1)
}
