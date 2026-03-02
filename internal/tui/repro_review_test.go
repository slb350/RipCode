package tui

import (
	"errors"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stephenbrandon/ripcode/internal/agent"
	"github.com/stephenbrandon/ripcode/internal/session"
	"github.com/stephenbrandon/ripcode/internal/store"
	"github.com/stephenbrandon/ripcode/internal/tool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReview_EventError_PersistsSession(t *testing.T) {
	t.Setenv("RIPCODE_DIR", t.TempDir())

	app := NewApp()
	app.SetProvider(&modelListProvider{})
	app.SetRegistry(tool.NewRegistry())
	sess := session.New(t.TempDir())
	sess.AddUser("hello")
	app.SetSession(sess)
	app.SetAgent(agent.BuildAgent())
	app.state = StateSession
	model, _ := app.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	a := model.(App)

	model, _ = a.Update(AgentEventMsg{Event: agent.Event{Type: agent.EventError, Error: errors.New("boom")}})
	a = model.(App)
	_ = a

	loaded, err := store.Load(sess.ID)
	require.NoError(t, err, "session should be persisted on terminal error")
	require.NotNil(t, loaded)
	assert.Equal(t, sess.ID, loaded.ID)
}

func TestReview_ResumeSession_ClearsSessionScopedSidebarState(t *testing.T) {
	t.Setenv("RIPCODE_DIR", t.TempDir())

	app := NewApp()
	app.SetProvider(&modelListProvider{})
	app.SetRegistry(tool.NewRegistry())
	app.SetSession(session.New(t.TempDir()))
	app.SetAgent(agent.BuildAgent())
	app.state = StateSession
	model, _ := app.Update(tea.WindowSizeMsg{Width: 120, Height: 30})
	a := model.(App)

	// Seed session-scoped sidebar state from previous session.
	a.modifiedFiles.add("/tmp/old.go")
	a.toolpanel.AddEvent(agent.ToolEvent{ID: "tool-1", Name: "write", Output: "ok"})
	require.NotEmpty(t, a.toolpanel.Events())

	// Save another session and resume it.
	sess := session.New(t.TempDir())
	sess.AddUser("hello")
	sess.AddAssistant("world", nil, nil)
	require.NoError(t, store.Save(sess))

	model, _ = a.resumeSession(sess.ID)
	a = model.(App)

	assert.Empty(t, a.modifiedFiles.paths(), "resuming should clear modified files from previous session")
	assert.Empty(t, a.toolpanel.Events(), "resuming should clear tool event history from previous session")
}
