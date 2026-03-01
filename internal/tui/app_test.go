package tui

import (
	"context"
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

func TestApp_CtrlCQuits(t *testing.T) {
	app := NewApp()
	_, cmd := app.Update(tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl})
	assert.NotNil(t, cmd)
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
	view := a.View()
	assert.Contains(t, view.Content, "Available models")
	assert.Contains(t, view.Content, "anthropic/claude-sonnet-4")
	assert.Contains(t, view.Content, "openai/gpt-4o")

	// Second /models claude — cache hit, should NOT produce a Cmd.
	model, cmd = a.Update(components.InputSubmitMsg{Value: "/models claude"})
	a = model.(App)
	assert.Nil(t, cmd, "cache hit must not produce a command")
	assert.Equal(t, 1, p.calls, "provider should not be called again")
	view = a.View()
	assert.Contains(t, view.Content, "Filtered models")
	assert.Contains(t, view.Content, "anthropic/claude-sonnet-4")
}

func TestApp_ModelsCommand_PrefixMatchRejectsInvalid(t *testing.T) {
	app := NewApp()
	app.SetProvider(&modelListProvider{})
	app.SetRegistry(tool.NewRegistry())
	app.SetSession(session.New(t.TempDir()))
	app.SetAgent(agent.BuildAgent())

	model, _ := app.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	a := model.(App)

	// "/modelsxyz" should NOT route to models handler — it should go
	// through normal submit, which adds a user chat entry and starts streaming.
	model, _ = a.Update(components.InputSubmitMsg{Value: "/modelsxyz"})
	a = model.(App)
	assert.True(t, a.streaming, "/modelsxyz must not route to models handler")
}
