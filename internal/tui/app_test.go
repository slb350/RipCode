package tui

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stephenbrandon/ripcode/internal/agent"
	"github.com/stephenbrandon/ripcode/internal/provider"
	"github.com/stephenbrandon/ripcode/internal/session"
	"github.com/stephenbrandon/ripcode/internal/tool"
	"github.com/stephenbrandon/ripcode/internal/tui/components"
	"github.com/stretchr/testify/assert"
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

func TestApp_EscQuits_InHome(t *testing.T) {
	app := NewApp()
	_, cmd := app.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	assert.NotNil(t, cmd)
}

func TestApp_CtrlCQuits_WhenInputEmpty(t *testing.T) {
	app := NewApp()
	_, cmd := app.Update(tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl})
	assert.NotNil(t, cmd, "ctrl+c with empty input should quit")
}

func TestApp_CtrlC_ClearsInputWhenNonEmpty(t *testing.T) {
	app := NewApp()
	app.SetProvider(&modelListProvider{})
	app.SetRegistry(tool.NewRegistry())
	app.SetSession(session.New(t.TempDir()))
	app.SetAgent(agent.BuildAgent())

	model, _ := app.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	a := model.(App)
	a.state = StateSession
	a.input.SetValue("some text")

	model, cmd := a.Update(tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl})
	a = model.(App)

	assert.Nil(t, cmd, "ctrl+c with non-empty input should not quit")
	assert.Equal(t, "", a.input.Value(), "ctrl+c should clear input")
}

func TestApp_CtrlC_QuitsWhenInputEmpty(t *testing.T) {
	app := NewApp()
	app.SetProvider(&modelListProvider{})
	app.SetRegistry(tool.NewRegistry())
	app.SetSession(session.New(t.TempDir()))
	app.SetAgent(agent.BuildAgent())

	model, _ := app.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	a := model.(App)
	a.state = StateSession
	// Input is empty

	_, cmd := a.Update(tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl})
	assert.NotNil(t, cmd, "ctrl+c with empty input should quit")
}

func TestApp_SubmitPrompt_AddedToHistory(t *testing.T) {
	app := NewApp()
	app.SetProvider(&modelListProvider{})
	app.SetRegistry(tool.NewRegistry())
	app.SetSession(session.New(t.TempDir()))
	app.SetAgent(agent.BuildAgent())

	model, _ := app.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	a := model.(App)

	// Submit a slash command (doesn't start streaming)
	model, _ = a.Update(components.InputSubmitMsg{Value: "/help"})
	a = model.(App)

	// Up arrow should recall "/help"
	a.input.SetValue("")
	model, _ = a.Update(tea.KeyPressMsg{Code: tea.KeyUp})
	a = model.(App)
	assert.Equal(t, "/help", a.input.Value())
}

func TestApp_UpArrow_AtFirstLine_RecallsPreviousPrompt(t *testing.T) {
	app := NewApp()
	app.SetProvider(&modelListProvider{})
	app.SetRegistry(tool.NewRegistry())
	app.SetSession(session.New(t.TempDir()))
	app.SetAgent(agent.BuildAgent())

	model, _ := app.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	a := model.(App)

	model, _ = a.Update(components.InputSubmitMsg{Value: "/help"})
	a = model.(App)
	model, _ = a.Update(components.InputSubmitMsg{Value: "/new"})
	a = model.(App)

	// Up at first line recalls "/new"
	model, _ = a.Update(tea.KeyPressMsg{Code: tea.KeyUp})
	a = model.(App)
	assert.Equal(t, "/new", a.input.Value())

	// Up again recalls "/help"
	model, _ = a.Update(tea.KeyPressMsg{Code: tea.KeyUp})
	a = model.(App)
	assert.Equal(t, "/help", a.input.Value())
}

func TestApp_DownArrow_AtLastLine_RecallsNextOrDraft(t *testing.T) {
	app := NewApp()
	app.SetProvider(&modelListProvider{})
	app.SetRegistry(tool.NewRegistry())
	app.SetSession(session.New(t.TempDir()))
	app.SetAgent(agent.BuildAgent())

	model, _ := app.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	a := model.(App)

	model, _ = a.Update(components.InputSubmitMsg{Value: "/help"})
	a = model.(App)

	// Type something as draft
	a.input.SetValue("my draft")

	// Up to recall "/help"
	model, _ = a.Update(tea.KeyPressMsg{Code: tea.KeyUp})
	a = model.(App)
	assert.Equal(t, "/help", a.input.Value())

	// Down to restore draft
	model, _ = a.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	a = model.(App)
	assert.Equal(t, "my draft", a.input.Value())
}

func TestApp_UpArrow_AtMiddleLine_MovesUpNormally(t *testing.T) {
	app := NewApp()
	app.SetProvider(&modelListProvider{})
	app.SetRegistry(tool.NewRegistry())
	app.SetSession(session.New(t.TempDir()))
	app.SetAgent(agent.BuildAgent())

	model, _ := app.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	a := model.(App)
	a.state = StateSession

	// Multi-line input: cursor on line 1 (not first line)
	a.input.SetValue("line1\nline2")
	// SetValue moves cursor to end (line 1, pos 5)
	// Up should move cursor to line 0, not history
	model, _ = a.Update(tea.KeyPressMsg{Code: tea.KeyUp})
	a = model.(App)
	assert.Equal(t, "line1\nline2", a.input.Value(), "should not change value")
}

func TestApp_DownArrow_AtMiddleLine_MovesDownNormally(t *testing.T) {
	app := NewApp()
	app.SetProvider(&modelListProvider{})
	app.SetRegistry(tool.NewRegistry())
	app.SetSession(session.New(t.TempDir()))
	app.SetAgent(agent.BuildAgent())

	model, _ := app.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	a := model.(App)
	a.state = StateSession

	// Multi-line input: cursor on line 0 (not last line)
	a.input.SetValue("line1\nline2")
	a.input.SetCursorOffset(3) // mid line 0
	// Down should move to line 1, not history
	model, _ = a.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	a = model.(App)
	assert.Equal(t, "line1\nline2", a.input.Value(), "should not change value")
}

func TestApp_HistoryNavigation_SavesAndRestoresDraft(t *testing.T) {
	app := NewApp()
	app.SetProvider(&modelListProvider{})
	app.SetRegistry(tool.NewRegistry())
	app.SetSession(session.New(t.TempDir()))
	app.SetAgent(agent.BuildAgent())

	model, _ := app.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	a := model.(App)

	model, _ = a.Update(components.InputSubmitMsg{Value: "/help"})
	a = model.(App)

	a.input.SetValue("draft text")

	// Up: saves draft, shows /help
	model, _ = a.Update(tea.KeyPressMsg{Code: tea.KeyUp})
	a = model.(App)
	assert.Equal(t, "/help", a.input.Value())

	// Down: restores draft
	model, _ = a.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	a = model.(App)
	assert.Equal(t, "draft text", a.input.Value())
}

func TestApp_ContentDelta_ContinuesListening(t *testing.T) {
	app := NewApp()
	ch := make(chan agent.Event, 1)
	app.streaming = true
	app.state = StateSession
	app.eventCh = ch

	model, cmd := app.Update(AgentEventMsg{
		Event: agent.Event{Type: agent.EventContentDelta, Content: "hello"},
	})
	a := model.(App)

	assert.True(t, a.streaming)
	assert.NotNil(t, cmd, "non-terminal event must return a listen command")
}

func TestApp_ToolStart_ContinuesListening(t *testing.T) {
	app := NewApp()
	ch := make(chan agent.Event, 1)
	app.streaming = true
	app.state = StateSession
	app.eventCh = ch

	model, cmd := app.Update(AgentEventMsg{
		Event: agent.Event{
			Type: agent.EventToolStart,
			Tool: &agent.ToolEvent{ID: "1", Name: "bash", Args: `{"command":"ls"}`},
		},
	})
	a := model.(App)

	assert.True(t, a.streaming)
	assert.NotNil(t, cmd, "non-terminal event must return a listen command")
}

func TestApp_ToolEnd_ContinuesListening(t *testing.T) {
	app := NewApp()
	ch := make(chan agent.Event, 1)
	app.streaming = true
	app.state = StateSession
	app.eventCh = ch

	model, cmd := app.Update(AgentEventMsg{
		Event: agent.Event{
			Type: agent.EventToolEnd,
			Tool: &agent.ToolEvent{ID: "1", Name: "bash", Output: "file.txt"},
		},
	})
	a := model.(App)

	assert.True(t, a.streaming)
	assert.NotNil(t, cmd, "non-terminal event must return a listen command")
}

func TestApp_AgentEventDone(t *testing.T) {
	app := NewApp()
	ch := make(chan agent.Event, 1)
	app.streaming = true
	app.state = StateSession
	app.eventCh = ch

	model, cmd := app.Update(AgentEventMsg{
		Event: agent.Event{Type: agent.EventDone},
	})
	a := model.(App)

	assert.False(t, a.streaming)
	assert.Nil(t, a.eventCh, "Done must clear the event channel")
	assert.Nil(t, cmd, "terminal event must not return a command")
}

func TestApp_AgentEventError(t *testing.T) {
	app := NewApp()
	ch := make(chan agent.Event, 1)
	app.streaming = true
	app.state = StateSession
	app.eventCh = ch

	model, cmd := app.Update(AgentEventMsg{
		Event: agent.Event{
			Type:  agent.EventError,
			Error: assert.AnError,
		},
	})
	a := model.(App)

	assert.False(t, a.streaming)
	assert.Nil(t, a.eventCh, "Error must clear the event channel")
	assert.Nil(t, cmd, "terminal event must not return a command")
}

func TestApp_EscCancel_ClearsChannel(t *testing.T) {
	app := NewApp()
	ch := make(chan agent.Event, 1)
	app.streaming = true
	app.state = StateSession
	app.eventCh = ch
	app.cancel = func() {} // no-op cancel

	model, _ := app.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	a := model.(App)

	assert.False(t, a.streaming)
	assert.Nil(t, a.eventCh, "Esc cancel must clear the event channel")
}

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

func TestApp_UnknownSlashCommand_ShowsError(t *testing.T) {
	app := NewApp()
	app.SetProvider(&modelListProvider{})
	app.SetRegistry(tool.NewRegistry())
	app.SetSession(session.New(t.TempDir()))
	app.SetAgent(agent.BuildAgent())

	model, _ := app.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	a := model.(App)

	// Unknown slash commands should be handled locally with an error.
	model, _ = a.Update(components.InputSubmitMsg{Value: "/modelsxyz"})
	a = model.(App)
	assert.False(t, a.streaming)
	assert.Contains(t, a.View().Content, "Unknown command")
}

func TestApp_AgentSlashCommand_SwitchesMode(t *testing.T) {
	app := NewApp()
	app.SetProvider(&modelListProvider{})
	app.SetRegistry(tool.NewRegistry())
	app.SetSession(session.New(t.TempDir()))
	app.SetAgent(agent.BuildAgent())

	model, _ := app.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	a := model.(App)

	model, _ = a.Update(components.InputSubmitMsg{Value: "/agent plan"})
	a = model.(App)

	assert.Equal(t, "plan", a.agent.Name)
	assert.Contains(t, a.View().Content, `Agent switched to "plan".`)
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
	assert.True(t, a.modelDialogOpen)

	model, _ = a.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	a = model.(App)

	assert.False(t, a.modelDialogOpen)
	assert.Equal(t, "openai/gpt-4o", p.model)
	assert.Equal(t, "gpt-4o", a.model)
	assert.Contains(t, a.View().Content, `Model switched to "openai/gpt-4o".`)
}

func TestApp_CommandPalette_OpensWithCtrlK(t *testing.T) {
	app := NewApp()
	app.SetProvider(&modelListProvider{})
	app.SetRegistry(tool.NewRegistry())
	app.SetSession(session.New(t.TempDir()))
	app.SetAgent(agent.BuildAgent())

	model, _ := app.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	a := model.(App)
	a.state = StateSession

	model, _ = a.Update(tea.KeyPressMsg{Code: 'k', Mod: tea.ModCtrl})
	a = model.(App)

	assert.True(t, a.commandOpen)
	assert.Contains(t, a.View().Content, "Commands (Ctrl+P/Ctrl+K")
}

func TestApp_CommandPalette_OpensWithCtrlP(t *testing.T) {
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

	assert.True(t, a.commandOpen)
	assert.Contains(t, a.View().Content, "Commands (Ctrl+P/Ctrl+K")
}

func TestApp_TabCyclesAgent(t *testing.T) {
	app := NewApp()
	app.SetProvider(&modelListProvider{})
	app.SetRegistry(tool.NewRegistry())
	app.SetSession(session.New(t.TempDir()))
	app.SetAgent(agent.BuildAgent())

	model, _ := app.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	a := model.(App)
	a.state = StateSession
	assert.Equal(t, "build", a.agent.Name)

	model, _ = a.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	a = model.(App)
	assert.Equal(t, "plan", a.agent.Name)

	model, _ = a.Update(tea.KeyPressMsg{Code: tea.KeyTab, Mod: tea.ModShift})
	a = model.(App)
	assert.Equal(t, "build", a.agent.Name)
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

func TestApp_InlineSlashAutocomplete_ExecutesModelsCommand(t *testing.T) {
	p := &modelListProvider{
		models: []provider.ModelInfo{
			{ID: "openai/gpt-4o", Name: "GPT-4o"},
		},
	}

	app := NewApp()
	app.SetProvider(p)
	app.SetRegistry(tool.NewRegistry())
	app.SetSession(session.New(t.TempDir()))
	app.SetAgent(agent.BuildAgent())
	app.modelsLoaded = true
	app.modelsCache = p.models

	model, _ := app.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	a := model.(App)
	a.state = StateSession

	model, _ = a.Update(tea.KeyPressMsg{Text: "/"})
	a = model.(App)
	model, _ = a.Update(tea.KeyPressMsg{Text: "m"})
	a = model.(App)
	model, _ = a.Update(tea.KeyPressMsg{Text: "o"})
	a = model.(App)
	model, _ = a.Update(tea.KeyPressMsg{Text: "d"})
	a = model.(App)

	assert.True(t, a.inlineOpen)
	assert.Equal(t, "/", a.inlineMode)

	model, cmd := a.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	a = model.(App)

	assert.Nil(t, cmd)
	assert.False(t, a.inlineOpen)
	assert.True(t, a.modelDialogOpen)
	assert.Contains(t, a.View().Content, "Select model")
}

func TestApp_InlineFileAutocomplete_InsertsMention(t *testing.T) {
	workDir := t.TempDir()
	err := os.WriteFile(filepath.Join(workDir, "main.go"), []byte("package main"), 0o644)
	assert.NoError(t, err)

	app := NewApp()
	app.SetSession(session.New(workDir))
	app.SetAgent(agent.BuildAgent())

	model, _ := app.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	a := model.(App)
	a.state = StateSession

	// Type "@" — triggers async file cache load.
	model, cmd := a.Update(tea.KeyPressMsg{Text: "@"})
	a = model.(App)

	// Execute the file cache command and feed result back.
	if cmd != nil {
		msg := cmd()
		model, _ = a.Update(msg)
		a = model.(App)
	}
	assert.True(t, a.fileCacheLoaded)

	// Continue typing "m" and "a".
	model, _ = a.Update(tea.KeyPressMsg{Text: "m"})
	a = model.(App)
	model, _ = a.Update(tea.KeyPressMsg{Text: "a"})
	a = model.(App)

	assert.True(t, a.inlineOpen)
	assert.Equal(t, "@", a.inlineMode)
	assert.Contains(t, a.View().Content, "main.go")

	model, _ = a.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	a = model.(App)

	assert.False(t, a.inlineOpen)
	assert.Equal(t, "@main.go ", a.input.Value())
}

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
	assert.Contains(t, view, "Recent tools")
	assert.Contains(t, view, "^B toggle sidebar")

	model, _ = a.Update(tea.KeyPressMsg{Code: 'b', Mod: tea.ModCtrl})
	a = model.(App)

	view = a.View().Content
	assert.NotContains(t, view, "Recent tools")
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

func TestApp_ClearCommand_ResetsSession(t *testing.T) {
	workDir := t.TempDir()
	app := NewApp()
	app.SetProvider(&modelListProvider{})
	app.SetRegistry(tool.NewRegistry())
	sess := session.New(workDir)
	sess.AddUser("hello")
	sess.AddTokens(500, 200)
	app.SetSession(sess)
	app.SetAgent(agent.BuildAgent())

	model, _ := app.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	a := model.(App)

	oldID := a.session.ID
	model, _ = a.Update(components.InputSubmitMsg{Value: "/clear"})
	a = model.(App)

	assert.Empty(t, a.session.Messages, "/clear should reset session messages")
	assert.Equal(t, 0, a.session.Tokens.Input, "/clear should reset token count")
	assert.Equal(t, 0, a.session.Tokens.Output)
	assert.NotEqual(t, oldID, a.session.ID, "/clear should generate new session ID")
	assert.Contains(t, a.View().Content, "Conversation cleared.")
}

func TestApp_NewCommand_ResetsSession(t *testing.T) {
	workDir := t.TempDir()
	app := NewApp()
	app.SetProvider(&modelListProvider{})
	app.SetRegistry(tool.NewRegistry())
	sess := session.New(workDir)
	sess.AddUser("hello")
	sess.AddTokens(1000, 300)
	app.SetSession(sess)
	app.SetAgent(agent.BuildAgent())

	model, _ := app.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	a := model.(App)

	model, _ = a.Update(components.InputSubmitMsg{Value: "/new"})
	a = model.(App)

	assert.Empty(t, a.session.Messages)
	assert.Equal(t, 0, a.session.Tokens.Input)
	assert.Contains(t, a.View().Content, "Conversation cleared.")
}

func TestApp_ExitCommand_Quits(t *testing.T) {
	app := NewApp()
	app.SetProvider(&modelListProvider{})
	app.SetRegistry(tool.NewRegistry())
	app.SetSession(session.New(t.TempDir()))
	app.SetAgent(agent.BuildAgent())

	model, _ := app.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	a := model.(App)

	_, cmd := a.Update(components.InputSubmitMsg{Value: "/exit"})
	assert.NotNil(t, cmd, "/exit should return a quit command")

	_, cmd = a.Update(components.InputSubmitMsg{Value: "/quit"})
	assert.NotNil(t, cmd, "/quit should return a quit command")

	_, cmd = a.Update(components.InputSubmitMsg{Value: "/q"})
	assert.NotNil(t, cmd, "/q should return a quit command")
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
	assert.True(t, a.modelDialogOpen)
	assert.Equal(t, 0, a.modelDialogSelect)

	// Down arrow moves selection
	model, _ = a.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	a = model.(App)
	assert.Equal(t, 1, a.modelDialogSelect)

	// Down again
	model, _ = a.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	a = model.(App)
	assert.Equal(t, 2, a.modelDialogSelect)

	// Down wraps to 0
	model, _ = a.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	a = model.(App)
	assert.Equal(t, 0, a.modelDialogSelect)

	// Up wraps to last
	model, _ = a.Update(tea.KeyPressMsg{Code: tea.KeyUp})
	a = model.(App)
	assert.Equal(t, 2, a.modelDialogSelect)

	// Esc closes dialog
	model, _ = a.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	a = model.(App)
	assert.False(t, a.modelDialogOpen)
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
	assert.Empty(t, sess.Messages, "Reset should clear messages")
	assert.Equal(t, 0, sess.Tokens.Input, "Reset should clear tokens")
	assert.NotEqual(t, oldID, sess.ID, "Reset should generate new ID")
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
