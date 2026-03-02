package tui

import (
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/stephenbrandon/ripcode/internal/agent"
	"github.com/stephenbrandon/ripcode/internal/session"
	"github.com/stephenbrandon/ripcode/internal/store"
	"github.com/stephenbrandon/ripcode/internal/tool"
	"github.com/stephenbrandon/ripcode/internal/tui/components"
	"github.com/stretchr/testify/assert"
)

func TestNewApp(t *testing.T) {
	app := NewApp()
	assert.False(t, app.ready)
	assert.False(t, app.streaming)
	assert.Empty(t, app.agent.Name, "NewApp should not set a default agent")
	assert.Equal(t, StateHome, app.state, "NewApp should start in StateHome")
}

func TestApp_SetAgent(t *testing.T) {
	app := NewApp()
	ag := agent.BuildAgent()
	app.SetAgent(ag)

	assert.Equal(t, "build", app.agent.Name)
	assert.NotEmpty(t, app.agent.SystemPrompt)
}

func TestApp_WindowSize(t *testing.T) {
	app := NewApp()
	model, _ := app.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	a := model.(App)

	assert.True(t, a.ready)
	assert.Equal(t, 80, a.width)
	assert.Equal(t, 24, a.height)
}

func TestApp_StartsInHomeState(t *testing.T) {
	app := NewApp()
	sess := &session.Session{WorkDir: "/tmp/project"}
	app.SetSession(sess)

	model, _ := app.Update(tea.WindowSizeMsg{Width: 80, Height: 30})
	a := model.(App)

	assert.Equal(t, StateHome, a.state)
	view := a.View()
	assert.Contains(t, view.Content, "██████╗", "home state should render logo")
	assert.Contains(t, view.Content, "ripcode")
}

func TestApp_HomeShowsLogo(t *testing.T) {
	app := NewApp()
	model, _ := app.Update(tea.WindowSizeMsg{Width: 80, Height: 30})
	a := model.(App)

	view := a.View()
	assert.Contains(t, view.Content, "██████╗")
	assert.Contains(t, view.Content, "code")
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

func TestApp_ShowToast_AddsToToastManager(t *testing.T) {
	app := NewApp()
	app.SetProvider(&modelListProvider{})
	app.SetRegistry(tool.NewRegistry())
	app.SetSession(session.New(t.TempDir()))
	app.SetAgent(agent.BuildAgent())

	model, _ := app.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	a := model.(App)
	a.state = StateSession

	cmd := a.ShowToast("Test toast", components.ToastInfo)
	assert.NotNil(t, cmd, "ShowToast should return a dismiss command")
	assert.NotNil(t, a.toasts.Current())
	assert.Equal(t, "Test toast", a.toasts.Current().Message)

	view := a.View()
	assert.Contains(t, view.Content, "Test toast")
}

func TestApp_ToastDismissMsg_DismissesMatchingID(t *testing.T) {
	app := NewApp()
	app.SetProvider(&modelListProvider{})
	app.SetRegistry(tool.NewRegistry())
	app.SetSession(session.New(t.TempDir()))
	app.SetAgent(agent.BuildAgent())

	model, _ := app.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	a := model.(App)
	a.state = StateSession

	id := a.toasts.Show("temp", components.ToastInfo, 3*time.Second)
	assert.NotNil(t, a.toasts.Current())

	model, _ = a.Update(ToastDismissMsg{ID: id})
	a = model.(App)
	assert.Nil(t, a.toasts.Current(), "matching dismiss should clear toast")
}

func TestApp_ToastDismissMsg_IgnoresMismatchedID(t *testing.T) {
	app := NewApp()
	app.SetProvider(&modelListProvider{})
	app.SetRegistry(tool.NewRegistry())
	app.SetSession(session.New(t.TempDir()))
	app.SetAgent(agent.BuildAgent())

	model, _ := app.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	a := model.(App)
	a.state = StateSession

	a.toasts.Show("first", components.ToastInfo, 3*time.Second)
	id2 := a.toasts.Show("second", components.ToastWarning, 3*time.Second)

	// Try to dismiss with stale ID
	model, _ = a.Update(ToastDismissMsg{ID: id2 - 1})
	a = model.(App)
	assert.NotNil(t, a.toasts.Current(), "stale ID should not dismiss")
	assert.Equal(t, "second", a.toasts.Current().Message)
}

func TestApp_SessionLayout_ShowsHeaderAndFooter(t *testing.T) {
	workDir := t.TempDir()
	app := NewApp()
	app.SetProvider(&modelListProvider{})
	app.SetRegistry(tool.NewRegistry())
	app.SetSession(session.New(workDir))
	app.SetAgent(agent.BuildAgent())
	app.SetModel("glm-5")

	model, _ := app.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	a := model.(App)
	a.state = StateSession

	view := a.View().Content
	assert.Contains(t, view, "# Session")
	assert.Contains(t, view, workDir)
	assert.Contains(t, view, "/help")
}

func TestApp_InputEmptyMode_NoPanic(t *testing.T) {
	i := components.NewInput()
	i.SetMode("")
	i.SetSize(80, 6)
	// This should not panic even with empty mode
	view := i.View()
	assert.Contains(t, view, "Build", "empty mode should fallback to Build")
}

func TestApp_SessionResetPreservesWorkDir(t *testing.T) {
	workDir := t.TempDir()
	sess := session.New(workDir)
	sess.AddUser("test")
	sess.AddTokens(100, 50)
	sess.SetSystemPrompt("you are helpful")
	oldID := sess.ID

	sess.Reset()

	assert.Equal(t, workDir, sess.WorkDir, "Reset should preserve WorkDir")
	assert.Empty(t, sess.Records(), "Reset should clear messages")
	assert.Equal(t, 0, sess.Tokens.Input, "Reset should clear tokens")
	assert.NotEqual(t, oldID, sess.ID, "Reset should generate new ID")
}

func TestApp_SetFullModelID_RestoresSavedVariant(t *testing.T) {
	t.Setenv("RIPCODE_DIR", t.TempDir())
	app := NewApp()
	p := &modelListProvider{}
	app.SetProvider(p)
	if app.modelPrefs == nil {
		app.modelPrefs = &store.ModelPrefs{}
	}
	app.modelPrefs.SetVariant("anthropic/claude-sonnet-4-thinking", "high")

	app.SetFullModelID("anthropic/claude-sonnet-4-thinking")

	assert.Equal(t, "high", app.activeVariant)
	assert.Equal(t, "high", p.reasoningEffort)
}

func TestApp_SetFullModelID_ClearsVariantWhenUnsupported(t *testing.T) {
	t.Setenv("RIPCODE_DIR", t.TempDir())
	app := NewApp()
	p := &modelListProvider{}
	app.SetProvider(p)
	app.activeVariant = "high"
	p.reasoningEffort = "high"

	app.SetFullModelID("openai/gpt-4o")

	assert.Equal(t, "", app.activeVariant)
	assert.Equal(t, "", p.reasoningEffort)
}
