package tui

import (
	"testing"
	"time"

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

func TestApp_SessionsDialog_OpensWithSlashSessions(t *testing.T) {
	app := makeSessionApp(t)
	model, _ := app.Update(components.InputSubmitMsg{Value: "/sessions"})
	a := model.(App)
	assert.True(t, a.sessionsDialog.open)
}

func TestApp_SessionsDialog_EscCloses(t *testing.T) {
	app := makeSessionApp(t)
	model, _ := app.Update(components.InputSubmitMsg{Value: "/sessions"})
	a := model.(App)
	model, _ = a.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	a = model.(App)
	assert.False(t, a.sessionsDialog.open)
}

func TestApp_SessionsDialog_FilterByTitle(t *testing.T) {
	app := makeSessionApp(t)
	model, _ := app.Update(components.InputSubmitMsg{Value: "/sessions"})
	a := model.(App)
	// Type a filter query
	model, _ = a.Update(tea.KeyPressMsg{Text: "a"})
	a = model.(App)
	model, _ = a.Update(tea.KeyPressMsg{Text: "b"})
	a = model.(App)
	assert.Equal(t, "ab", a.sessionsDialog.query)
}

func TestApp_SessionsDialog_ArrowNavigates(t *testing.T) {
	app := makeSessionApp(t)
	model, _ := app.Update(components.InputSubmitMsg{Value: "/sessions"})
	a := model.(App)
	// Load some sessions into the cache
	model, _ = a.Update(SessionsLoadedMsg{Sessions: []store.SessionSummary{
		{ID: "s1", Title: "first"},
		{ID: "s2", Title: "second"},
	}})
	a = model.(App)
	model, _ = a.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	a = model.(App)
	assert.Equal(t, 1, a.sessionsDialog.selected)
}

func TestApp_SessionsDialog_ClosesOtherDialogs(t *testing.T) {
	app := makeSessionApp(t)
	// Open help dialog first
	model, _ := app.Update(components.InputSubmitMsg{Value: "/help"})
	a := model.(App)
	assert.True(t, a.helpDialog.open)
	// Now open sessions
	a.helpDialog.open = false
	model, _ = a.Update(components.InputSubmitMsg{Value: "/sessions"})
	a = model.(App)
	assert.True(t, a.sessionsDialog.open)
	assert.False(t, a.helpDialog.open)
}

func TestApp_SessionsDialog_ShowsSessionEntries(t *testing.T) {
	app := makeSessionApp(t)
	model, _ := app.Update(components.InputSubmitMsg{Value: "/sessions"})
	a := model.(App)
	// Simulate sessions loaded
	model, _ = a.Update(SessionsLoadedMsg{Sessions: []store.SessionSummary{
		{ID: "s1", Title: "my project", MessageCount: 12},
	}})
	a = model.(App)
	assert.True(t, a.sessionsDialog.loaded)
	assert.Len(t, a.sessionsDialog.entries, 1)
	assert.Equal(t, "my project", a.sessionsDialog.entries[0].Title)
}

func TestApp_SessionsDialog_EmptyList(t *testing.T) {
	app := makeSessionApp(t)
	model, _ := app.Update(components.InputSubmitMsg{Value: "/sessions"})
	a := model.(App)
	model, _ = a.Update(SessionsLoadedMsg{Sessions: nil})
	a = model.(App)
	assert.True(t, a.sessionsDialog.loaded)
	assert.Empty(t, a.sessionsDialog.entries)
}

func TestApp_SessionsDialog_CtrlD_EntersDeleteConfirm(t *testing.T) {
	app := makeSessionApp(t)
	model, _ := app.Update(components.InputSubmitMsg{Value: "/sessions"})
	a := model.(App)
	model, _ = a.Update(SessionsLoadedMsg{Sessions: []store.SessionSummary{
		{ID: "s1", Title: "first"},
	}})
	a = model.(App)
	model, _ = a.Update(tea.KeyPressMsg{Mod: tea.ModCtrl, Code: 'd'})
	a = model.(App)
	assert.True(t, a.sessionsDialog.confirm)
}

func TestApp_SessionsDialog_DeleteConfirm_Esc_Cancels(t *testing.T) {
	app := makeSessionApp(t)
	model, _ := app.Update(components.InputSubmitMsg{Value: "/sessions"})
	a := model.(App)
	model, _ = a.Update(SessionsLoadedMsg{Sessions: []store.SessionSummary{
		{ID: "s1", Title: "first"},
	}})
	a = model.(App)
	model, _ = a.Update(tea.KeyPressMsg{Mod: tea.ModCtrl, Code: 'd'})
	a = model.(App)
	assert.True(t, a.sessionsDialog.confirm)
	model, _ = a.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	a = model.(App)
	assert.False(t, a.sessionsDialog.confirm)
	assert.True(t, a.sessionsDialog.open) // still open
}

func TestApp_SessionsDialog_BackspaceDeletesQuery(t *testing.T) {
	app := makeSessionApp(t)
	model, _ := app.Update(components.InputSubmitMsg{Value: "/sessions"})
	a := model.(App)
	model, _ = a.Update(tea.KeyPressMsg{Text: "a"})
	a = model.(App)
	model, _ = a.Update(tea.KeyPressMsg{Text: "b"})
	a = model.(App)
	assert.Equal(t, "ab", a.sessionsDialog.query)
	model, _ = a.Update(tea.KeyPressMsg{Code: tea.KeyBackspace})
	a = model.(App)
	assert.Equal(t, "a", a.sessionsDialog.query)
}

func TestApp_SessionsDialog_RenderShowsDateGroups(t *testing.T) {
	app := makeSessionApp(t)
	model, _ := app.Update(components.InputSubmitMsg{Value: "/sessions"})
	a := model.(App)

	now := time.Now()
	yesterday := now.Add(-24 * time.Hour)
	older := now.Add(-72 * time.Hour)

	model, _ = a.Update(SessionsLoadedMsg{Sessions: []store.SessionSummary{
		{ID: "s1", Title: "today session", UpdatedAt: now, MessageCount: 5},
		{ID: "s2", Title: "yesterday session", UpdatedAt: yesterday, MessageCount: 3},
		{ID: "s3", Title: "older session", UpdatedAt: older, MessageCount: 1},
	}})
	a = model.(App)

	rendered := a.renderSessionsDialog()
	assert.Contains(t, rendered, "Today")
	assert.Contains(t, rendered, "Yesterday")
	// Older date should show as formatted date
	assert.Contains(t, rendered, older.Format("Jan 2"))
}

func TestApp_SessionsDialog_RenderShowsTimeFooter(t *testing.T) {
	app := makeSessionApp(t)
	model, _ := app.Update(components.InputSubmitMsg{Value: "/sessions"})
	a := model.(App)

	now := time.Now()
	model, _ = a.Update(SessionsLoadedMsg{Sessions: []store.SessionSummary{
		{ID: "s1", Title: "test", UpdatedAt: now, CreatedAt: now, MessageCount: 5, WorkDir: "/tmp"},
	}})
	a = model.(App)

	rendered := a.renderSessionsDialog()
	assert.Contains(t, rendered, "5 msgs")
	assert.Contains(t, rendered, "/tmp")
}

func TestApp_ResumeSession_ReappliesSystemPrompt(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("RIPCODE_DIR", dir)

	// Create and save a session
	sess := session.New(t.TempDir())
	sess.AddUser("hello")
	sess.AddAssistant("world", nil, nil)
	require.NoError(t, store.Save(sess))

	// Create app manually (don't use makeSessionApp which overrides RIPCODE_DIR)
	app := NewApp()
	app.SetProvider(&modelListProvider{})
	app.SetRegistry(tool.NewRegistry())
	app.SetSession(session.New(t.TempDir()))
	app.SetAgent(agent.BuildAgent())
	app.state = StateSession
	model, _ := app.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	a := model.(App)

	// Resume the saved session
	model, _ = a.resumeSession(sess.ID)
	a = model.(App)

	// The loaded session should have the system prompt reapplied
	require.Equal(t, sess.ID, a.session.ID, "should have switched to loaded session")
	history := a.session.History()
	require.NotEmpty(t, history)
	assert.Equal(t, provider.RoleSystem, history[0].Role, "resumed session should have system prompt")
	assert.NotEmpty(t, history[0].Content)
}

func TestSessionDateGroup_Today(t *testing.T) {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	assert.Equal(t, "Today", sessionDateGroup(now, today))
}

func TestSessionDateGroup_Yesterday(t *testing.T) {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	yesterday := now.Add(-24 * time.Hour)
	assert.Equal(t, "Yesterday", sessionDateGroup(yesterday, today))
}

func TestSessionDateGroup_OlderDate(t *testing.T) {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	older := now.Add(-72 * time.Hour)
	result := sessionDateGroup(older, today)
	assert.Contains(t, result, older.Format("Jan 2"))
}
