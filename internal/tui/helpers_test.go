package tui

import (
	"context"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/stephenbrandon/ripcode/internal/agent"
	"github.com/stephenbrandon/ripcode/internal/provider"
	"github.com/stephenbrandon/ripcode/internal/session"
	"github.com/stephenbrandon/ripcode/internal/store"
	"github.com/stephenbrandon/ripcode/internal/tool"
	"github.com/stephenbrandon/ripcode/internal/tui/components"
)

type modelListProvider struct {
	models []provider.ModelInfo
	calls  int
	model  string
}

func (m *modelListProvider) Name() string { return "mock" }

func (m *modelListProvider) Chat(_ context.Context, _ []provider.Message, _ []provider.ToolDef) (<-chan provider.StreamEvent, error) {
	ch := make(chan provider.StreamEvent)
	close(ch)
	return ch, nil
}

func (m *modelListProvider) ListModels(_ context.Context) ([]provider.ModelInfo, error) {
	m.calls++
	return m.models, nil
}

func (m *modelListProvider) SetModel(model string) {
	m.model = model
}

func makeSessionApp(t *testing.T) App {
	t.Helper()
	t.Setenv("RIPCODE_DIR", t.TempDir())
	app := NewApp()
	app.SetProvider(&modelListProvider{})
	app.SetRegistry(tool.NewRegistry())
	app.SetSession(session.New(t.TempDir()))
	app.SetAgent(agent.BuildAgent())
	app.state = StateSession // must be set before WindowSizeMsg for layout
	model, _ := app.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	a := model.(App)
	// Add enough entries for scrolling
	for i := 0; i < 30; i++ {
		a.chat.AddEntry(components.ChatEntry{Role: "user", Content: "message"})
	}
	return a
}

func makeSessionAppWithHistory(t *testing.T) App {
	t.Helper()
	t.Setenv("RIPCODE_DIR", t.TempDir())
	app := NewApp()
	app.SetProvider(&modelListProvider{})
	app.SetRegistry(tool.NewRegistry())
	sess := session.New(t.TempDir())
	sess.AddUser("first question")
	sess.AddAssistant("first answer", nil, nil)
	sess.AddUser("second question")
	sess.AddAssistant("second answer", nil, nil)
	app.SetSession(sess)
	app.SetAgent(agent.BuildAgent())
	app.state = StateSession
	model, _ := app.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	a := model.(App)
	// Rebuild chat from session
	a.rebuildChatFromSession()
	return a
}

func makeSidebarApp(t *testing.T) App {
	t.Helper()
	t.Setenv("RIPCODE_DIR", t.TempDir())
	app := NewApp()
	app.SetProvider(&modelListProvider{})
	app.SetRegistry(tool.NewRegistry())
	app.SetSession(session.New(t.TempDir()))
	app.SetAgent(agent.BuildAgent())
	app.state = StateSession
	app.uiPrefs = &store.UIPrefs{}
	app.mcpConfig = &store.MCPConfig{}
	app.lspConfig = &store.LSPConfig{}
	model, _ := app.Update(tea.WindowSizeMsg{Width: 160, Height: 40})
	return model.(App)
}

func makeOverlayApp(t *testing.T) App {
	t.Helper()
	t.Setenv("RIPCODE_DIR", t.TempDir())
	app := NewApp()
	app.SetProvider(&modelListProvider{})
	app.SetRegistry(tool.NewRegistry())
	app.SetSession(session.New(t.TempDir()))
	app.SetAgent(agent.BuildAgent())
	app.state = StateSession
	app.uiPrefs = &store.UIPrefs{}
	app.mcpConfig = &store.MCPConfig{}
	app.lspConfig = &store.LSPConfig{}
	// Use narrow width so overlay mode activates (< 120)
	model, _ := app.Update(tea.WindowSizeMsg{Width: 80, Height: 30})
	a := model.(App)
	a.sidebarOverlay = true
	return a
}
