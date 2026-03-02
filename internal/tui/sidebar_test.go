package tui

import (
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/stephenbrandon/ripcode/internal/agent"
	"github.com/stephenbrandon/ripcode/internal/session"
	"github.com/stephenbrandon/ripcode/internal/store"
	"github.com/stephenbrandon/ripcode/internal/tool"
	"github.com/stephenbrandon/ripcode/internal/tui/components"
	"github.com/stretchr/testify/assert"
)

func TestApp_Sidebar_VisibleOnWideLayout_AndTogglesWithCtrlB(t *testing.T) {
	workDir := t.TempDir()
	app := NewApp()
	app.SetProvider(&modelListProvider{})
	app.SetRegistry(tool.NewRegistry())
	app.SetSession(session.New(workDir))
	app.SetAgent(agent.BuildAgent())
	app.SetModel("glm-5")

	model, _ := app.Update(tea.WindowSizeMsg{Width: 140, Height: 30})
	a := model.(App)
	a.state = StateSession

	view := a.View().Content
	assert.Contains(t, view, "Tools")
	assert.Contains(t, view, "Context")

	model, _ = a.Update(tea.KeyPressMsg{Code: 'b', Mod: tea.ModCtrl})
	a = model.(App)

	view = a.View().Content
	assert.NotContains(t, view, "▾ Tools")
}

func TestApp_SidebarSlashCommand_TogglesSidebar(t *testing.T) {
	app := NewApp()
	app.SetProvider(&modelListProvider{})
	app.SetRegistry(tool.NewRegistry())
	app.SetSession(session.New(t.TempDir()))
	app.SetAgent(agent.BuildAgent())

	model, _ := app.Update(tea.WindowSizeMsg{Width: 140, Height: 30})
	a := model.(App)
	a.state = StateSession
	assert.False(t, a.sidebarHidden)

	model, _ = a.Update(components.InputSubmitMsg{Value: "/sidebar"})
	a = model.(App)
	assert.True(t, a.sidebarHidden)
	assert.Contains(t, a.View().Content, "Sidebar hidden.")
}

func TestApp_Sidebar_NarrowCtrlB_OpensOverlayAndEscCloses(t *testing.T) {
	app := NewApp()
	app.SetProvider(&modelListProvider{})
	app.SetRegistry(tool.NewRegistry())
	app.SetSession(session.New(t.TempDir()))
	app.SetAgent(agent.BuildAgent())

	model, _ := app.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	a := model.(App)
	a.state = StateSession

	assert.False(t, a.sidebarOverlayActive())
	assert.NotContains(t, a.View().Content, "Sidebar overlay (Esc close)")

	model, _ = a.Update(tea.KeyPressMsg{Code: 'b', Mod: tea.ModCtrl})
	a = model.(App)

	assert.True(t, a.sidebarOverlayActive())
	view := a.View().Content
	assert.Contains(t, view, "Sidebar overlay (Esc close)")
	assert.Contains(t, view, "Session")

	model, cmd := a.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	a = model.(App)

	assert.Nil(t, cmd, "Esc should close overlay, not quit app")
	assert.False(t, a.sidebarOverlayActive())
	assert.NotContains(t, a.View().Content, "Sidebar overlay (Esc close)")
}

func TestApp_SidebarSlashCommand_OnNarrow_TogglesOverlay(t *testing.T) {
	app := NewApp()
	app.SetProvider(&modelListProvider{})
	app.SetRegistry(tool.NewRegistry())
	app.SetSession(session.New(t.TempDir()))
	app.SetAgent(agent.BuildAgent())

	model, _ := app.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	a := model.(App)
	a.state = StateSession

	model, _ = a.Update(components.InputSubmitMsg{Value: "/sidebar"})
	a = model.(App)
	assert.True(t, a.sidebarOverlayActive())
	assert.Contains(t, a.View().Content, "Sidebar overlay (Esc close)")

	model, _ = a.Update(components.InputSubmitMsg{Value: "/sidebar"})
	a = model.(App)
	assert.False(t, a.sidebarOverlayActive())
	assert.Contains(t, a.View().Content, "Sidebar hidden.")
}

func TestApp_SidebarOverlay_MouseClickOutside_Closes(t *testing.T) {
	app := NewApp()
	app.SetProvider(&modelListProvider{})
	app.SetRegistry(tool.NewRegistry())
	app.SetSession(session.New(t.TempDir()))
	app.SetAgent(agent.BuildAgent())

	model, _ := app.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	a := model.(App)
	a.state = StateSession

	model, _ = a.Update(tea.KeyPressMsg{Code: 'b', Mod: tea.ModCtrl})
	a = model.(App)
	assert.True(t, a.sidebarOverlayActive())

	x, y, w, h := a.sidebarOverlayPanelRect()

	// Click inside panel should keep overlay open.
	model, _ = a.Update(tea.MouseClickMsg{
		X:      x + min(1, w-1),
		Y:      y + min(1, h-1),
		Button: tea.MouseLeft,
	})
	a = model.(App)
	assert.True(t, a.sidebarOverlayActive())

	// Click outside panel should close overlay.
	model, _ = a.Update(tea.MouseClickMsg{
		X:      0,
		Y:      0,
		Button: tea.MouseLeft,
	})
	a = model.(App)
	assert.False(t, a.sidebarOverlayActive())
}

// --- Sidebar Section tests ---

func TestSidebar_ContextSection_Rendered(t *testing.T) {
	a := makeSidebarApp(t)
	sidebar := a.renderSidebar()
	assert.Contains(t, sidebar, "Context")
	assert.Contains(t, sidebar, "tokens")
}

func TestSidebar_MCPSection_ShowsServers(t *testing.T) {
	a := makeSidebarApp(t)
	a.mcpConfig = &store.MCPConfig{
		Servers: []store.MCPServer{
			{Name: "github", Command: "gh mcp", Enabled: true},
			{Name: "disabled-srv", Command: "cmd", Enabled: false},
		},
	}
	sidebar := a.renderSidebar()
	assert.Contains(t, sidebar, "MCP")
	assert.Contains(t, sidebar, "github")
	assert.Contains(t, sidebar, "disabled-srv")
}

func TestSidebar_LSPSection_ShowsClients(t *testing.T) {
	a := makeSidebarApp(t)
	a.lspConfig = &store.LSPConfig{
		Clients: []store.LSPClient{
			{Name: "gopls", Root: "/project", Enabled: true},
		},
	}
	sidebar := a.renderSidebar()
	assert.Contains(t, sidebar, "LSP")
	assert.Contains(t, sidebar, "gopls")
}

func TestSidebar_TodoSection_ShowsItems(t *testing.T) {
	a := makeSidebarApp(t)
	td := tool.NewTodoTool()
	td.Execute(tool.Context{WorkDir: "."}, `{"action":"write","items":[{"subject":"Fix bug","status":"pending"}]}`)
	a.todoTool = td
	sidebar := a.renderSidebar()
	assert.Contains(t, sidebar, "Todo")
	assert.Contains(t, sidebar, "Fix bug")
}

func TestSidebar_ModifiedFiles_ShowsFiles(t *testing.T) {
	a := makeSidebarApp(t)
	a.modifiedFiles.add("/tmp/foo.go")
	a.modifiedFiles.add("/tmp/bar.go")
	sidebar := a.renderSidebar()
	assert.Contains(t, sidebar, "Modified")
	assert.Contains(t, sidebar, "foo.go")
	assert.Contains(t, sidebar, "bar.go")
}

func TestSidebar_SectionCollapse_HidesContent(t *testing.T) {
	a := makeSidebarApp(t)
	a.uiPrefs = &store.UIPrefs{
		CollapsedSections: map[string]bool{"context": true},
	}
	sidebar := a.renderSidebar()
	assert.Contains(t, sidebar, "Context")
	assert.NotContains(t, sidebar, "tokens")
}

func TestSidebar_SectionCollapse_ShowsIndicator(t *testing.T) {
	a := makeSidebarApp(t)
	a.uiPrefs = &store.UIPrefs{
		CollapsedSections: map[string]bool{"context": true},
	}
	sidebar := a.renderSidebar()
	assert.Contains(t, sidebar, "▸")
}

func TestSidebar_SectionExpanded_ShowsIndicator(t *testing.T) {
	a := makeSidebarApp(t)
	sidebar := a.renderSidebar()
	assert.Contains(t, sidebar, "▾")
}

// --- Sidebar Overlay Keybind tests ---

func TestSidebarOverlay_NumberKeyTogglesSection(t *testing.T) {
	a := makeOverlayApp(t)

	// Press '1' to toggle context section
	model, _ := a.Update(tea.KeyPressMsg{Code: '1'})
	a = model.(App)

	assert.True(t, a.uiPrefs.IsCollapsed(sectionContext))

	// Press '1' again to toggle back
	model, _ = a.Update(tea.KeyPressMsg{Code: '1'})
	a = model.(App)

	assert.False(t, a.uiPrefs.IsCollapsed(sectionContext))
}

func TestSidebarOverlay_AllNumberKeys(t *testing.T) {
	a := makeOverlayApp(t)

	sections := []struct {
		key  rune
		name string
	}{
		{'1', sectionContext},
		{'2', sectionMCP},
		{'3', sectionLSP},
		{'4', sectionTodo},
		{'5', sectionModified},
		{'6', sectionTools},
	}

	for _, s := range sections {
		model, _ := a.Update(tea.KeyPressMsg{Code: s.key})
		a = model.(App)
		assert.True(t, a.uiPrefs.IsCollapsed(s.name), "section %q should be collapsed after pressing %c", s.name, s.key)
	}
}

func TestSidebarOverlay_DKeyDismissesGettingStarted(t *testing.T) {
	a := makeOverlayApp(t)

	model, _ := a.Update(tea.KeyPressMsg{Code: 'd'})
	a = model.(App)

	assert.True(t, a.uiPrefs.GettingStartedDismissed)
}

// --- Getting Started Card tests ---

func TestApp_GettingStarted_ShownForNewUser(t *testing.T) {
	a := makeSidebarApp(t)
	a.uiPrefs = &store.UIPrefs{GettingStartedDismissed: false}
	// No session history — sessionsDialogLoaded false by default
	sidebar := a.renderSidebar()
	assert.Contains(t, sidebar, "Getting Started")
}

func TestApp_GettingStarted_HiddenAfterDismiss(t *testing.T) {
	a := makeSidebarApp(t)
	a.uiPrefs = &store.UIPrefs{GettingStartedDismissed: true}
	sidebar := a.renderSidebar()
	assert.NotContains(t, sidebar, "Getting Started")
}

func TestApp_GettingStarted_DismissPersists(t *testing.T) {
	a := makeOverlayApp(t)
	a.uiPrefs = &store.UIPrefs{GettingStartedDismissed: false}

	// Press 'd' to dismiss
	model, _ := a.Update(tea.KeyPressMsg{Code: 'd'})
	a = model.(App)

	assert.True(t, a.uiPrefs.GettingStartedDismissed)
}

func TestApp_GettingStarted_HiddenWhenHasSessions(t *testing.T) {
	a := makeSidebarApp(t)
	a.uiPrefs = &store.UIPrefs{GettingStartedDismissed: false}
	a.sessionsDialog.loaded = true
	a.sessionsDialog.entries = []store.SessionSummary{
		{ID: "test", Title: "test"},
	}
	sidebar := a.renderSidebar()
	assert.NotContains(t, sidebar, "Getting Started")
}

func TestApp_GettingStarted_HiddenWhenSessionEntriesCached(t *testing.T) {
	a := makeSidebarApp(t)
	a.uiPrefs = &store.UIPrefs{GettingStartedDismissed: false}
	a.sessionsDialog.entries = []store.SessionSummary{
		{ID: "test", Title: "test"},
	}

	sidebar := a.renderSidebar()
	assert.NotContains(t, sidebar, "Getting Started")
}

func TestPanelRectFromRendered_ZeroDimensions(t *testing.T) {
	a := makeSidebarApp(t)
	x, y, w, h := a.panelRectFromRendered("")
	assert.Equal(t, 1, w, "width should be clamped to at least 1")
	assert.Equal(t, 1, h, "height should be clamped to at least 1")
	assert.GreaterOrEqual(t, x, 0)
	assert.GreaterOrEqual(t, y, 0)
}

func TestApp_Sidebar_ShowsTodoFromTool(t *testing.T) {
	a := makeSidebarApp(t)

	// Use SetTodoTool to set the tool reference
	td := tool.NewTodoTool()
	td.Execute(tool.Context{WorkDir: "."}, `{"action":"write","items":[{"subject":"Sidebar task","status":"pending"}]}`)
	a.todoTool = td

	sidebar := a.renderSidebar()
	assert.Contains(t, sidebar, "Todo")
	assert.Contains(t, sidebar, "Sidebar task")
}
