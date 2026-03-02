package tui

import (
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/stephenbrandon/ripcode/internal/agent"
	"github.com/stephenbrandon/ripcode/internal/provider"
	"github.com/stephenbrandon/ripcode/internal/session"
	"github.com/stephenbrandon/ripcode/internal/store"
	"github.com/stephenbrandon/ripcode/internal/tool"
	"github.com/stephenbrandon/ripcode/internal/tui/components"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApp_ForkDialog_OpensWithSlashFork(t *testing.T) {
	a := makeSessionAppWithHistory(t)
	model, _ := a.Update(components.InputSubmitMsg{Value: "/fork"})
	a = model.(App)
	assert.True(t, a.forkDialog.open)
}

func TestApp_ForkDialog_EscCloses(t *testing.T) {
	a := makeSessionAppWithHistory(t)
	model, _ := a.Update(components.InputSubmitMsg{Value: "/fork"})
	a = model.(App)
	assert.True(t, a.forkDialog.open)
	model, _ = a.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	a = model.(App)
	assert.False(t, a.forkDialog.open)
}

func TestApp_ForkDialog_ArrowNavigates(t *testing.T) {
	a := makeSessionAppWithHistory(t)
	model, _ := a.Update(components.InputSubmitMsg{Value: "/fork"})
	a = model.(App)
	model, _ = a.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	a = model.(App)
	assert.Equal(t, 1, a.forkDialog.selected)
}

func TestApp_ForkDialog_ShowsUserMessages(t *testing.T) {
	a := makeSessionAppWithHistory(t)
	model, _ := a.Update(components.InputSubmitMsg{Value: "/fork"})
	a = model.(App)
	assert.True(t, a.forkDialog.open)
	// Should have entries for user messages (uses timelineEntries)
	assert.NotEmpty(t, a.timelineEntries())
}

func TestApp_ForkDialog_EmptySession_ShowsWarning(t *testing.T) {
	a := makeSessionApp(t)
	a.session.ClearMessages() // empty
	a.chat.Clear()
	model, _ := a.Update(components.InputSubmitMsg{Value: "/fork"})
	a = model.(App)
	assert.False(t, a.forkDialog.open)
	toast := a.toasts.Current()
	assert.NotNil(t, toast)
	assert.Contains(t, toast.Message, "Nothing to fork")
}

func TestApp_ForkDialog_EnterCreatesForkedSession(t *testing.T) {
	a := makeSessionAppWithHistory(t)
	origID := a.session.ID
	model, _ := a.Update(components.InputSubmitMsg{Value: "/fork"})
	a = model.(App)
	// Select first user message and press Enter
	model, _ = a.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	a = model.(App)
	assert.False(t, a.forkDialog.open)
	assert.NotEqual(t, origID, a.session.ID)
	assert.Equal(t, origID, a.session.ParentID)
	// Forked at first user message — should include user + assistant = 2 messages
	assert.Len(t, a.session.Records(), 2)
}

func TestApp_ForkDialog_ClosesOtherDialogs(t *testing.T) {
	a := makeSessionAppWithHistory(t)
	a.helpDialog.open = true
	model, _ := a.Update(components.InputSubmitMsg{Value: "/fork"})
	a = model.(App)
	assert.True(t, a.forkDialog.open)
	assert.False(t, a.helpDialog.open)
}

func TestApp_ForkDialog_EnterPersistsForkedSession(t *testing.T) {
	a := makeSessionAppWithHistory(t)
	model, _ := a.Update(components.InputSubmitMsg{Value: "/fork"})
	a = model.(App)
	model, _ = a.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	a = model.(App)

	_, err := store.Load(a.session.ID)
	require.NoError(t, err, "forked session should be saved immediately")
}

func TestApp_ForkDialog_IncludesFullToolExchangeInFork(t *testing.T) {
	t.Setenv("RIPCODE_DIR", t.TempDir())
	app := NewApp()
	app.SetProvider(&modelListProvider{})
	app.SetRegistry(tool.NewRegistry())
	sess := session.New(t.TempDir())
	call := provider.ToolCall{ID: "call-1", Name: "read", Args: `{"file_path":"main.go"}`}
	sess.AddUser("inspect file")
	sess.AddAssistant("", []provider.ToolCall{call}, nil)
	sess.AddToolResult(call.ID, "contents")
	sess.AddAssistant("done", nil, nil)
	sess.AddUser("next question")
	sess.AddAssistant("next answer", nil, nil)
	app.SetSession(sess)
	app.SetAgent(agent.BuildAgent())
	app.state = StateSession
	model, _ := app.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	a := model.(App)
	a.rebuildChatFromSession()

	model, _ = a.Update(components.InputSubmitMsg{Value: "/fork"})
	a = model.(App)
	model, _ = a.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	a = model.(App)

	recs := a.session.Records()
	require.Len(t, recs, 4, "fork should include user turn, tool result, and final assistant")
	assert.Equal(t, provider.RoleTool, recs[2].Message.Role)
	assert.Equal(t, provider.RoleAssistant, recs[3].Message.Role)
}
