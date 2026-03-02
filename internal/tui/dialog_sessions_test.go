package tui

import (
	"encoding/json"
	"os"
	"path/filepath"
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

func TestApp_ResumeSession_PartialLoadStillResumes(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("RIPCODE_DIR", dir)

	// Create and save a valid session.
	sess := session.New(t.TempDir())
	sess.AddUser("hello")
	sess.AddAssistant("world", nil, nil)
	require.NoError(t, store.Save(sess))

	// Corrupt one message role so Load returns (session, error).
	path := filepath.Join(store.SessionsDir(), sess.ID+".json")
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	var raw map[string]any
	require.NoError(t, json.Unmarshal(data, &raw))
	msgs := raw["messages"].([]any)
	msgs[0].(map[string]any)["role"] = "bogus"
	data, err = json.MarshalIndent(raw, "", "  ")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, data, 0o644))

	app := NewApp()
	app.SetProvider(&modelListProvider{})
	app.SetRegistry(tool.NewRegistry())
	app.SetSession(session.New(t.TempDir()))
	app.SetAgent(agent.BuildAgent())
	app.state = StateSession
	model, _ := app.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	a := model.(App)

	model, _ = a.resumeSession(sess.ID)
	a = model.(App)

	require.NotNil(t, a.session)
	assert.Equal(t, sess.ID, a.session.ID)
	assert.Equal(t, 1, a.session.Len(), "should resume with valid records that remain")
	toast := a.toasts.Current()
	require.NotNil(t, toast)
	assert.Equal(t, components.ToastWarning, toast.Variant)
	assert.Contains(t, toast.Message, "invalid record")
}

func TestApp_SessionsDialog_DeleteConfirm_Enter_DeletesSession(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("RIPCODE_DIR", dir)

	// Create and save a session to disk
	sess := session.New(t.TempDir())
	sess.AddUser("hello")
	sess.AddAssistant("world", nil, nil)
	require.NoError(t, store.Save(sess))

	app := NewApp()
	app.SetProvider(&modelListProvider{})
	app.SetRegistry(tool.NewRegistry())
	app.SetSession(session.New(t.TempDir()))
	app.SetAgent(agent.BuildAgent())
	app.state = StateSession
	model, _ := app.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	a := model.(App)

	// Open sessions dialog
	model, _ = a.Update(components.InputSubmitMsg{Value: "/sessions"})
	a = model.(App)

	// Load the saved session
	model, _ = a.Update(SessionsLoadedMsg{Sessions: []store.SessionSummary{
		{ID: sess.ID, Title: "test session"},
	}})
	a = model.(App)
	require.Len(t, a.sessionsDialog.entries, 1)

	// Enter delete confirm mode
	model, _ = a.Update(tea.KeyPressMsg{Mod: tea.ModCtrl, Code: 'd'})
	a = model.(App)
	assert.True(t, a.sessionsDialog.confirm)

	// Confirm deletion with Enter
	model, _ = a.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	a = model.(App)
	assert.False(t, a.sessionsDialog.confirm)
	assert.Empty(t, a.sessionsDialog.entries, "session should be removed from entries")

	// Verify the session file was deleted from disk
	_, err := store.Load(sess.ID)
	assert.Error(t, err, "session file should be deleted from disk")
}

func TestApp_SessionsDialog_ReloadsOnReopen(t *testing.T) {
	app := makeSessionApp(t)
	// First open
	model, _ := app.Update(components.InputSubmitMsg{Value: "/sessions"})
	a := model.(App)
	model, _ = a.Update(SessionsLoadedMsg{Sessions: []store.SessionSummary{
		{ID: "s1", Title: "first"},
	}})
	a = model.(App)
	assert.True(t, a.sessionsDialog.loaded)
	assert.Len(t, a.sessionsDialog.entries, 1)

	// Close
	model, _ = a.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	a = model.(App)
	assert.False(t, a.sessionsDialog.open)

	// Reopen — loaded should be reset so data is refreshed
	model, cmd := a.Update(components.InputSubmitMsg{Value: "/sessions"})
	a = model.(App)
	assert.True(t, a.sessionsDialog.open)
	assert.False(t, a.sessionsDialog.loaded, "loaded should be false to trigger reload")
	assert.NotNil(t, cmd, "should return loadSessions cmd")
}

func TestApp_ResumeSession_UpdatesFooterWorkDir(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("RIPCODE_DIR", dir)

	// Create and save a session with a specific work dir
	workDir := t.TempDir()
	sess := session.New(workDir)
	sess.AddUser("hello")
	sess.AddAssistant("world", nil, nil)
	require.NoError(t, store.Save(sess))

	// Create app with a different work dir
	app := NewApp()
	app.SetProvider(&modelListProvider{})
	app.SetRegistry(tool.NewRegistry())
	originalDir := t.TempDir()
	app.SetSession(session.New(originalDir))
	app.SetAgent(agent.BuildAgent())
	app.state = StateSession
	model, _ := app.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	a := model.(App)

	// Resume the saved session
	model, _ = a.resumeSession(sess.ID)
	a = model.(App)

	// Footer should show the resumed session's work dir, not the original
	view := a.View()
	assert.Contains(t, view.Content, workDir, "footer should show resumed session's work dir")
	assert.NotContains(t, view.Content, originalDir, "footer should not show original work dir")
}

func TestApp_ResumeSession_ResetsFileCache(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("RIPCODE_DIR", dir)

	workDir := t.TempDir()
	sess := session.New(workDir)
	sess.AddUser("hello")
	sess.AddAssistant("world", nil, nil)
	require.NoError(t, store.Save(sess))

	app := NewApp()
	app.SetProvider(&modelListProvider{})
	app.SetRegistry(tool.NewRegistry())
	app.SetSession(session.New(t.TempDir()))
	app.SetAgent(agent.BuildAgent())
	app.state = StateSession
	model, _ := app.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	a := model.(App)

	// Simulate stale cache from previous session.
	a.fileCacheLoaded = true
	a.fileCacheLoading = true
	a.fileCache = []string{"old/path.go"}

	model, _ = a.resumeSession(sess.ID)
	a = model.(App)

	assert.False(t, a.fileCacheLoaded, "resume should invalidate old file cache")
	assert.False(t, a.fileCacheLoading, "resume should clear stale loading flag")
	assert.Empty(t, a.fileCache, "resume should clear stale cached paths")
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

func TestApp_SessionsLoadedMsg_WithError(t *testing.T) {
	app := makeSessionApp(t)
	model, _ := app.Update(components.InputSubmitMsg{Value: "/sessions"})
	a := model.(App)
	assert.True(t, a.sessionsDialog.open)
	// Send error
	model, cmd := a.Update(SessionsLoadedMsg{Err: assert.AnError})
	a = model.(App)
	assert.True(t, a.sessionsDialog.loaded, "loaded should still be set")
	assert.Empty(t, a.sessionsDialog.entries, "entries should remain empty on error")
	assert.NotNil(t, cmd, "should return toast cmd for error")
	toast := a.toasts.Current()
	assert.NotNil(t, toast, "should show error toast")
}

func TestApp_SessionsLoadedMsg_WithError_ClearsStaleEntries(t *testing.T) {
	app := makeSessionApp(t)
	model, _ := app.Update(components.InputSubmitMsg{Value: "/sessions"})
	a := model.(App)
	a.sessionsDialog.entries = []store.SessionSummary{
		{ID: "stale", Title: "stale"},
	}

	model, _ = a.Update(SessionsLoadedMsg{Err: assert.AnError})
	a = model.(App)

	assert.Empty(t, a.sessionsDialog.entries, "stale entries should be cleared on load error")
}
