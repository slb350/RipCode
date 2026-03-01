package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/stephenbrandon/ripcode/internal/agent"
	"github.com/stephenbrandon/ripcode/internal/provider"
	"github.com/stephenbrandon/ripcode/internal/session"
	"github.com/stephenbrandon/ripcode/internal/store"
	"github.com/stephenbrandon/ripcode/internal/tool"
	"github.com/stephenbrandon/ripcode/internal/tui/components"
	"github.com/stretchr/testify/assert"
)

func TestApp_ModelsDialog_SelectsModelWithEnter(t *testing.T) {
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
	app.SetModel("claude-sonnet-4")
	app.modelsCache = p.models
	app.modelsLoaded = true

	model, _ := app.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	a := model.(App)
	a.state = StateSession

	model, _ = a.Update(components.InputSubmitMsg{Value: "/models gpt"})
	a = model.(App)
	assert.True(t, a.modelDialog.open)

	model, _ = a.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	a = model.(App)

	assert.False(t, a.modelDialog.open)
	assert.Equal(t, "openai/gpt-4o", p.model)
	assert.Equal(t, "gpt-4o", a.model)
	assert.Contains(t, a.View().Content, `Model switched to "openai/gpt-4o".`)
}

func TestApp_ModelsDialog_KeyboardNavigation(t *testing.T) {
	p := &modelListProvider{
		models: []provider.ModelInfo{
			{ID: "a/model-a", Name: "Model A"},
			{ID: "b/model-b", Name: "Model B"},
			{ID: "c/model-c", Name: "Model C"},
		},
	}
	app := NewApp()
	app.SetProvider(p)
	app.SetRegistry(tool.NewRegistry())
	app.SetSession(session.New(t.TempDir()))
	app.SetAgent(agent.BuildAgent())
	app.modelsCache = p.models
	app.modelsLoaded = true

	model, _ := app.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	a := model.(App)
	a.state = StateSession

	// Open model dialog
	model, _ = a.Update(components.InputSubmitMsg{Value: "/models"})
	a = model.(App)
	assert.True(t, a.modelDialog.open)
	assert.Equal(t, 0, a.modelDialog.selected)

	// Down arrow moves selection
	model, _ = a.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	a = model.(App)
	assert.Equal(t, 1, a.modelDialog.selected)

	// Down again
	model, _ = a.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	a = model.(App)
	assert.Equal(t, 2, a.modelDialog.selected)

	// Down wraps to 0
	model, _ = a.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	a = model.(App)
	assert.Equal(t, 0, a.modelDialog.selected)

	// Up wraps to last
	model, _ = a.Update(tea.KeyPressMsg{Code: tea.KeyUp})
	a = model.(App)
	assert.Equal(t, 2, a.modelDialog.selected)

	// Esc closes dialog
	model, _ = a.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	a = model.(App)
	assert.False(t, a.modelDialog.open)
}

func TestModelDialog_CtrlF_TogglesFavorite_ShowsToast(t *testing.T) {
	a := makeSessionApp(t)
	a.modelsCache = []provider.ModelInfo{
		{ID: "anthropic/claude-4", Name: "Claude 4"},
		{ID: "openai/gpt-4o", Name: "GPT-4o"},
	}
	a.modelsLoaded = true
	a = a.openModelDialog("")
	// Toggle favorite with ctrl+f
	model, cmd := a.Update(tea.KeyPressMsg{Code: 'f', Mod: tea.ModCtrl})
	a = model.(App)
	assert.True(t, a.modelPrefs.IsFavorite(store.ModelRef{ProviderID: "anthropic", ModelID: "anthropic/claude-4"}))
	assert.NotNil(t, cmd, "should return toast cmd")
}

func TestModelDialog_CtrlF_UnfavoriteShowsToast(t *testing.T) {
	a := makeSessionApp(t)
	a.modelsCache = []provider.ModelInfo{
		{ID: "anthropic/claude-4", Name: "Claude 4"},
	}
	a.modelsLoaded = true
	a.modelPrefs.ToggleFavorite(store.ModelRef{ProviderID: "anthropic", ModelID: "anthropic/claude-4"})
	a = a.openModelDialog("")
	model, cmd := a.Update(tea.KeyPressMsg{Code: 'f', Mod: tea.ModCtrl})
	a = model.(App)
	assert.False(t, a.modelPrefs.IsFavorite(store.ModelRef{ProviderID: "anthropic", ModelID: "anthropic/claude-4"}))
	assert.NotNil(t, cmd)
}

func TestModelDialog_FavoriteIndicator_StarPrefix(t *testing.T) {
	a := makeSessionApp(t)
	a.modelsCache = []provider.ModelInfo{
		{ID: "anthropic/claude-4", Name: "Claude 4"},
	}
	a.modelsLoaded = true
	a.modelPrefs.ToggleFavorite(store.ModelRef{ProviderID: "anthropic", ModelID: "anthropic/claude-4"})
	a = a.openModelDialog("")
	rendered := a.renderModelDialog()
	assert.Contains(t, rendered, "★")
}

func TestModelDialog_FavoritesSection_ShownFirst(t *testing.T) {
	a := makeSessionApp(t)
	a.modelsCache = []provider.ModelInfo{
		{ID: "openai/gpt-4o", Name: "GPT-4o"},
		{ID: "anthropic/claude-4", Name: "Claude 4"},
	}
	a.modelsLoaded = true
	a.modelPrefs.ToggleFavorite(store.ModelRef{ProviderID: "anthropic", ModelID: "anthropic/claude-4"})
	a = a.openModelDialog("")
	rendered := a.renderModelDialog()
	// Claude should appear before GPT since it's a favorite
	claudeIdx := strings.Index(rendered, "claude-4")
	gptIdx := strings.Index(rendered, "gpt-4o")
	assert.Greater(t, gptIdx, claudeIdx, "favorites should appear first")
}

func TestModelDialog_FavoriteToggle_PersistsToStore(t *testing.T) {
	a := makeSessionApp(t)
	a.modelsCache = []provider.ModelInfo{
		{ID: "anthropic/claude-4", Name: "Claude 4"},
	}
	a.modelsLoaded = true
	a = a.openModelDialog("")
	model, _ := a.Update(tea.KeyPressMsg{Code: 'f', Mod: tea.ModCtrl})
	a = model.(App)
	// Load prefs from disk to verify persistence
	loaded, err := store.LoadModelPrefs()
	assert.NoError(t, err)
	assert.True(t, loaded.IsFavorite(store.ModelRef{ProviderID: "anthropic", ModelID: "anthropic/claude-4"}))
}

func TestModelDialog_FavoriteToggle_SelectionStaysOnSameModel(t *testing.T) {
	a := makeSessionApp(t)
	a.modelsCache = []provider.ModelInfo{
		{ID: "openai/gpt-4o", Name: "GPT-4o"},
		{ID: "anthropic/claude-4", Name: "Claude 4"},
	}
	a.modelsLoaded = true
	a = a.openModelDialog("")
	// Move selection to claude (index 1)
	a.modelDialog.selected = 1
	model, _ := a.Update(tea.KeyPressMsg{Code: 'f', Mod: tea.ModCtrl})
	a = model.(App)
	// After toggling favorite, claude should be in favorites (first), selection should still point at it
	displayed := a.filteredModelsDialog()
	if assert.NotEmpty(t, displayed) {
		assert.Equal(t, "anthropic/claude-4", displayed[a.modelDialog.selected].ID)
	}
}

func TestModelDialog_CtrlF_WhenNoModelsLoaded_NoOp(t *testing.T) {
	a := makeSessionApp(t)
	a.modelsCache = nil
	a.modelsLoaded = true
	a = a.openModelDialog("")
	model, cmd := a.Update(tea.KeyPressMsg{Code: 'f', Mod: tea.ModCtrl})
	a = model.(App)
	assert.True(t, a.modelDialog.open, "dialog should remain open")
	assert.Nil(t, cmd)
}

func TestModelDialog_CtrlA_OpensProviderFilter(t *testing.T) {
	a := makeSessionApp(t)
	a.modelsCache = []provider.ModelInfo{
		{ID: "anthropic/claude-4", Name: "Claude 4"},
		{ID: "openai/gpt-4o", Name: "GPT-4o"},
	}
	a.modelsLoaded = true
	a = a.openModelDialog("")
	model, _ := a.Update(tea.KeyPressMsg{Code: 'a', Mod: tea.ModCtrl})
	a = model.(App)
	assert.True(t, a.modelDialog.providerMode)
}

func TestProviderFilter_ListsUniqueProviders(t *testing.T) {
	a := makeSessionApp(t)
	a.modelsCache = []provider.ModelInfo{
		{ID: "anthropic/claude-4", Name: "Claude 4"},
		{ID: "anthropic/claude-3", Name: "Claude 3"},
		{ID: "openai/gpt-4o", Name: "GPT-4o"},
	}
	a.modelsLoaded = true
	providers := a.uniqueProviders()
	assert.Contains(t, providers, "anthropic")
	assert.Contains(t, providers, "openai")
	assert.Len(t, providers, 2)
}

func TestProviderFilter_SelectProvider_FiltersModelList(t *testing.T) {
	a := makeSessionApp(t)
	a.modelsCache = []provider.ModelInfo{
		{ID: "anthropic/claude-4", Name: "Claude 4"},
		{ID: "openai/gpt-4o", Name: "GPT-4o"},
	}
	a.modelsLoaded = true
	a = a.openModelDialog("")
	a.modelDialog.providerFilter = "anthropic"
	models := a.filteredModelsDialog()
	assert.Len(t, models, 1)
	assert.Equal(t, "anthropic/claude-4", models[0].ID)
}

func TestProviderFilter_Escape_ReturnsToFullList(t *testing.T) {
	a := makeSessionApp(t)
	a.modelsCache = []provider.ModelInfo{
		{ID: "anthropic/claude-4", Name: "Claude 4"},
		{ID: "openai/gpt-4o", Name: "GPT-4o"},
	}
	a.modelsLoaded = true
	a = a.openModelDialog("")
	a.modelDialog.providerFilter = "anthropic"
	// Press ctrl+a to toggle to provider mode, then escape should clear filter
	a.modelDialog.providerMode = true
	model, _ := a.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	a = model.(App)
	assert.False(t, a.modelDialog.providerMode)
	assert.Equal(t, "", a.modelDialog.providerFilter)
}

func TestProviderFilter_SelectAll_ShowsAllModels(t *testing.T) {
	a := makeSessionApp(t)
	a.modelsCache = []provider.ModelInfo{
		{ID: "anthropic/claude-4", Name: "Claude 4"},
		{ID: "openai/gpt-4o", Name: "GPT-4o"},
	}
	a.modelsLoaded = true
	a = a.openModelDialog("")
	a.modelDialog.providerFilter = "" // empty = all providers
	models := a.filteredModelsDialog()
	assert.Len(t, models, 2)
}

func TestModelDialog_ProviderSections_GroupedByProvider(t *testing.T) {
	a := makeSessionApp(t)
	a.modelsCache = []provider.ModelInfo{
		{ID: "anthropic/claude-4", Name: "Claude 4"},
		{ID: "openai/gpt-4o", Name: "GPT-4o"},
		{ID: "anthropic/claude-3", Name: "Claude 3"},
	}
	a.modelsLoaded = true
	a = a.openModelDialog("")
	rendered := a.renderModelDialog()
	assert.Contains(t, rendered, "anthropic")
	assert.Contains(t, rendered, "openai")
}

func TestModelDialog_ProviderSections_AlphabeticalOrder(t *testing.T) {
	a := makeSessionApp(t)
	a.modelsCache = []provider.ModelInfo{
		{ID: "openai/gpt-4o", Name: "GPT-4o"},
		{ID: "anthropic/claude-4", Name: "Claude 4"},
	}
	a.modelsLoaded = true
	a = a.openModelDialog("")
	rendered := a.renderModelDialog()
	anthIdx := strings.Index(rendered, "── anthropic")
	openIdx := strings.Index(rendered, "── openai")
	if anthIdx >= 0 && openIdx >= 0 {
		assert.Less(t, anthIdx, openIdx, "anthropic should appear before openai")
	}
}

func TestModelDialog_FreeBadge_ShownForFreeModels(t *testing.T) {
	a := makeSessionApp(t)
	a.modelsCache = []provider.ModelInfo{
		{ID: "free/model", Name: "Free Model", Pricing: nil},
	}
	a.modelsLoaded = true
	a = a.openModelDialog("")
	rendered := a.renderModelDialog()
	assert.Contains(t, rendered, "[free]")
}

func TestModelDialog_FreeBadge_NotShownForPaidModels(t *testing.T) {
	a := makeSessionApp(t)
	a.modelsCache = []provider.ModelInfo{
		{ID: "anthropic/claude-4", Name: "Claude 4", Pricing: &provider.ModelPricing{PromptPerMillion: 3.0, CompletionPerMillion: 15.0}},
	}
	a.modelsLoaded = true
	a = a.openModelDialog("")
	rendered := a.renderModelDialog()
	assert.NotContains(t, rendered, "[free]")
}

func TestModelDialog_Filtering_FlatList_NoSections(t *testing.T) {
	a := makeSessionApp(t)
	a.modelsCache = []provider.ModelInfo{
		{ID: "anthropic/claude-4", Name: "Claude 4"},
		{ID: "openai/gpt-4o", Name: "GPT-4o"},
	}
	a.modelsLoaded = true
	a = a.openModelDialog("")
	a.modelDialog.query = "claude"
	rendered := a.renderModelDialog()
	assert.NotContains(t, rendered, "── anthropic", "sections should not appear when filtering")
}

func TestModelDialog_ContextLengthShown(t *testing.T) {
	a := makeSessionApp(t)
	a.modelsCache = []provider.ModelInfo{
		{ID: "anthropic/claude-4", Name: "Claude 4", ContextLength: 200000},
	}
	a.modelsLoaded = true
	a = a.openModelDialog("")
	rendered := a.renderModelDialog()
	assert.Contains(t, rendered, "200K")
}

func TestModelDialog_RecentsSection_ShownAfterFavorites(t *testing.T) {
	a := makeSessionApp(t)
	a.modelsCache = []provider.ModelInfo{
		{ID: "anthropic/claude-4", Name: "Claude 4"},
		{ID: "openai/gpt-4o", Name: "GPT-4o"},
		{ID: "meta/llama-3", Name: "Llama 3"},
	}
	a.modelsLoaded = true
	// Set up one favorite and one recent
	a.modelPrefs.ToggleFavorite(store.ModelRef{ProviderID: "meta", ModelID: "meta/llama-3"})
	a.modelPrefs.AddRecent(store.ModelRef{ProviderID: "openai", ModelID: "openai/gpt-4o"})
	a = a.openModelDialog("")
	displayed := a.filteredModelsDialog()
	// Favorites first (llama), then recents (gpt-4o), then rest (claude)
	if assert.Len(t, displayed, 3) {
		assert.Equal(t, "meta/llama-3", displayed[0].ID, "favorite should be first")
		assert.Equal(t, "openai/gpt-4o", displayed[1].ID, "recent should be second")
		assert.Equal(t, "anthropic/claude-4", displayed[2].ID, "rest should be last")
	}
}

func TestModelDialog_EmptyCache_NavigationSafe(t *testing.T) {
	a := makeSessionApp(t)
	a.modelsCache = nil
	a.modelsLoaded = true
	a = a.openModelDialog("")

	// Up/down with empty cache should not panic
	model, cmd := a.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	a = model.(App)
	assert.Nil(t, cmd)

	model, cmd = a.Update(tea.KeyPressMsg{Code: tea.KeyUp})
	a = model.(App)
	assert.Nil(t, cmd)

	// Enter with empty cache should close dialog
	model, cmd = a.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	a = model.(App)
	assert.False(t, a.modelDialog.open)
	assert.Nil(t, cmd)
}

func TestApp_F2_CyclesRecentModel_Forward(t *testing.T) {
	a := makeSessionApp(t)
	a.fullModelID = "anthropic/claude-4"
	a.modelPrefs.AddRecent(store.ModelRef{ProviderID: "openai", ModelID: "openai/gpt-4o"})
	a.modelPrefs.AddRecent(store.ModelRef{ProviderID: "anthropic", ModelID: "anthropic/claude-4"})
	model, _ := a.Update(tea.KeyPressMsg{Code: tea.KeyF2})
	a = model.(App)
	assert.Equal(t, "openai/gpt-4o", a.fullModelID)
}

func TestApp_ShiftF2_CyclesRecentModel_Reverse(t *testing.T) {
	a := makeSessionApp(t)
	a.fullModelID = "anthropic/claude-4"
	a.modelPrefs.AddRecent(store.ModelRef{ProviderID: "openai", ModelID: "openai/gpt-4o"})
	a.modelPrefs.AddRecent(store.ModelRef{ProviderID: "anthropic", ModelID: "anthropic/claude-4"})
	model, _ := a.Update(tea.KeyPressMsg{Code: tea.KeyF2, Mod: tea.ModShift})
	a = model.(App)
	assert.Equal(t, "openai/gpt-4o", a.fullModelID)
}

func TestApp_F2_NoRecents_ShowsToast(t *testing.T) {
	a := makeSessionApp(t)
	model, cmd := a.Update(tea.KeyPressMsg{Code: tea.KeyF2})
	a = model.(App)
	assert.NotNil(t, cmd, "should show toast when no recents")
}

func TestApp_F2_SwitchesModelAndShowsToast(t *testing.T) {
	a := makeSessionApp(t)
	a.fullModelID = "anthropic/claude-4"
	a.modelPrefs.AddRecent(store.ModelRef{ProviderID: "anthropic", ModelID: "anthropic/claude-4"})
	a.modelPrefs.AddRecent(store.ModelRef{ProviderID: "openai", ModelID: "openai/gpt-4o"})
	model, cmd := a.Update(tea.KeyPressMsg{Code: tea.KeyF2})
	_ = model.(App)
	assert.NotNil(t, cmd, "should return toast cmd")
}

func TestApp_CtrlT_CyclesVariant(t *testing.T) {
	a := makeSessionApp(t)
	a.model = "claude-sonnet-4-thinking"
	a.fullModelID = "anthropic/claude-sonnet-4-thinking"
	model, _ := a.Update(tea.KeyPressMsg{Code: 't', Mod: tea.ModCtrl})
	a = model.(App)
	assert.Equal(t, "low", a.activeVariant)
}

func TestApp_CtrlT_NoVariants_ShowsToast(t *testing.T) {
	a := makeSessionApp(t)
	a.fullModelID = "openai/gpt-4o"
	model, cmd := a.Update(tea.KeyPressMsg{Code: 't', Mod: tea.ModCtrl})
	a = model.(App)
	assert.Equal(t, "", a.activeVariant)
	assert.NotNil(t, cmd)
}

func TestApp_CtrlT_ShowsToastWithVariantName(t *testing.T) {
	a := makeSessionApp(t)
	a.fullModelID = "anthropic/claude-sonnet-4-thinking"
	model, cmd := a.Update(tea.KeyPressMsg{Code: 't', Mod: tea.ModCtrl})
	a = model.(App)
	assert.NotNil(t, cmd, "should show toast for variant change")
	assert.Equal(t, "low", a.activeVariant)
}

func TestApp_VariantBadge_ShownInStatusBar(t *testing.T) {
	a := makeSessionApp(t)
	a.fullModelID = "anthropic/claude-sonnet-4-thinking"
	a.activeVariant = "high"
	a.statusbar.SetVariantBadge("[thinking:high]")
	view := a.View()
	assert.Contains(t, view.Content, "[thinking:high]")
}

func TestApp_VariantBadge_HiddenWhenNone(t *testing.T) {
	a := makeSessionApp(t)
	a.activeVariant = ""
	a.statusbar.SetVariantBadge("")
	view := a.View()
	assert.NotContains(t, view.Content, "[thinking:")
}

func TestApp_Variant_PersistsAcrossSessions(t *testing.T) {
	a := makeSessionApp(t)
	a.fullModelID = "anthropic/claude-sonnet-4-thinking"
	model, _ := a.Update(tea.KeyPressMsg{Code: 't', Mod: tea.ModCtrl})
	a = model.(App)
	// Check that the variant was persisted
	loaded, err := store.LoadModelPrefs()
	assert.NoError(t, err)
	assert.Equal(t, "low", loaded.GetVariant("anthropic/claude-sonnet-4-thinking"))
}

func TestApp_VariantCycle_SetsReasoningEffort(t *testing.T) {
	a := makeSessionApp(t)
	a.fullModelID = "anthropic/claude-sonnet-4-thinking"
	model, _ := a.Update(tea.KeyPressMsg{Code: 't', Mod: tea.ModCtrl})
	a = model.(App)
	p := a.provider.(*modelListProvider)
	assert.Equal(t, "low", p.reasoningEffort, "variant cycle should set reasoning effort on provider")
}

func TestApp_VariantCycle_ClearsReasoningEffort(t *testing.T) {
	a := makeSessionApp(t)
	a.fullModelID = "anthropic/claude-sonnet-4-thinking"
	// Cycle through low, medium, high, then back to none
	for i := 0; i < 4; i++ {
		m, _ := a.Update(tea.KeyPressMsg{Code: 't', Mod: tea.ModCtrl})
		a = m.(App)
	}
	p := a.provider.(*modelListProvider)
	assert.Equal(t, "", p.reasoningEffort, "cycling past last variant should clear reasoning effort")
}
