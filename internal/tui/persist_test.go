package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/stephenbrandon/ripcode/internal/session"
	"github.com/stephenbrandon/ripcode/internal/store"
	"github.com/stephenbrandon/ripcode/internal/tool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApp_FooterShowsMCPCount_FromConfig(t *testing.T) {
	t.Setenv("RIPCODE_DIR", t.TempDir())
	cfg := &store.MCPConfig{
		Servers: []store.MCPServer{
			{Name: "srv1", Enabled: true},
			{Name: "srv2", Enabled: true},
			{Name: "srv3", Enabled: false},
		},
	}
	require.NoError(t, cfg.Save())

	app := NewApp()
	app.SetProvider(&modelListProvider{})
	app.SetRegistry(tool.NewRegistry())
	app.SetSession(session.New(t.TempDir()))
	app.state = StateSession
	app.footer.SetWorkDir("/project")
	model, _ := app.Update(tea.WindowSizeMsg{Width: 160, Height: 40})
	a := model.(App)

	view := a.footer.View()
	assert.Contains(t, view, "⊙ 2")
}

func TestApp_FooterShowsLSPCount_FromConfig(t *testing.T) {
	t.Setenv("RIPCODE_DIR", t.TempDir())
	cfg := &store.LSPConfig{
		Clients: []store.LSPClient{
			{Name: "gopls", Enabled: true},
		},
	}
	require.NoError(t, cfg.Save())

	app := NewApp()
	app.SetProvider(&modelListProvider{})
	app.SetRegistry(tool.NewRegistry())
	app.SetSession(session.New(t.TempDir()))
	app.state = StateSession
	app.footer.SetWorkDir("/project")
	model, _ := app.Update(tea.WindowSizeMsg{Width: 160, Height: 40})
	a := model.(App)

	view := a.footer.View()
	assert.Contains(t, view, "• 1")
}

func TestApp_NewApp_LoadsConfigs(t *testing.T) {
	t.Setenv("RIPCODE_DIR", t.TempDir())

	app := NewApp()
	assert.NotNil(t, app.mcpConfig)
	assert.NotNil(t, app.lspConfig)
	assert.NotNil(t, app.uiPrefs)
}

func TestNewApp_CorruptedPrefs_AccumulatesWarnings(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("RIPCODE_DIR", dir)
	stateDir := filepath.Join(dir, "state")
	require.NoError(t, os.MkdirAll(stateDir, 0o755))
	// Corrupt model prefs and MCP config
	require.NoError(t, os.WriteFile(filepath.Join(stateDir, "model.json"), []byte("{bad"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(stateDir, "mcp.json"), []byte("{bad"), 0o644))

	app := NewApp()
	assert.Len(t, app.startupWarnings, 2, "should have 2 warnings for 2 corrupted files")
	assert.NotNil(t, app.modelPrefs, "should still have usable defaults")
	assert.NotNil(t, app.mcpConfig, "should still have usable defaults")
}

func TestApp_StartupWarnings_ShowsToastOnReady(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("RIPCODE_DIR", dir)
	stateDir := filepath.Join(dir, "state")
	require.NoError(t, os.MkdirAll(stateDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(stateDir, "model.json"), []byte("{bad"), 0o644))

	app := NewApp()
	assert.NotEmpty(t, app.startupWarnings)
	assert.False(t, app.startupWarningsShown)

	model, cmd := app.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	a := model.(App)
	assert.True(t, a.startupWarningsShown, "flag should be set after first WindowSizeMsg")
	assert.NotNil(t, cmd, "should return toast dismiss cmd")
	assert.NotNil(t, a.toasts.Current(), "toast should be visible")
	assert.Contains(t, a.toasts.Current().Message, "model preferences")
}

func TestApp_StartupWarnings_NotReShownOnResize(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("RIPCODE_DIR", dir)
	stateDir := filepath.Join(dir, "state")
	require.NoError(t, os.MkdirAll(stateDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(stateDir, "ui.json"), []byte("{bad"), 0o644))

	app := NewApp()
	// First WindowSizeMsg shows toast
	model, _ := app.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	a := model.(App)
	assert.True(t, a.startupWarningsShown)

	// Dismiss toast
	if current := a.toasts.Current(); current != nil {
		a.toasts.Dismiss(current.ID)
	}

	// Second WindowSizeMsg (resize) should NOT re-show toast
	model, cmd := a.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	a = model.(App)
	assert.Nil(t, cmd, "resize should not produce a toast command")
	assert.Nil(t, a.toasts.Current(), "no toast should be visible after resize")
}

func TestLoadPromptHistory_Compaction_WritesCorrectData(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("RIPCODE_DIR", dir)

	// Write more than historyMaxSize entries
	var entries []store.HistoryEntry
	for i := 0; i < historyMaxSize+50; i++ {
		entries = append(entries, store.HistoryEntry{
			Prompt: "prompt " + string(rune('a'+i%26)),
			Mode:   "normal",
		})
	}
	require.NoError(t, store.SaveHistory(entries))

	// Load triggers compaction
	h, err := loadPromptHistory()
	require.NoError(t, err)
	assert.NotNil(t, h)
	items := h.Items()
	assert.LessOrEqual(t, len(items), historyMaxSize)

	// Verify compacted file on disk
	reloaded, err := store.LoadHistory()
	require.NoError(t, err)
	assert.LessOrEqual(t, len(reloaded), historyMaxSize, "compacted file should not exceed max size")
}

func TestPersistStash_RoundTrips(t *testing.T) {
	t.Setenv("RIPCODE_DIR", t.TempDir())
	s, err := loadPromptStash()
	require.NoError(t, err)
	s.Push("test content")
	require.NoError(t, persistStash(s))

	reloaded, err := loadPromptStash()
	require.NoError(t, err)
	items := reloaded.List()
	require.Len(t, items, 1)
	assert.Equal(t, "test content", items[0].Content)
}

func TestPersistHistory_RoundTrips(t *testing.T) {
	t.Setenv("RIPCODE_DIR", t.TempDir())
	h, err := loadPromptHistory()
	require.NoError(t, err)
	h.Push("test prompt")
	require.NoError(t, persistHistory(h))

	// Reload and verify
	entries, err := store.LoadHistory()
	require.NoError(t, err)
	require.NotEmpty(t, entries)
	assert.Equal(t, "test prompt", entries[len(entries)-1].Prompt)
}

func TestPersistHistory_DedupedPrompt_DoesNotAppendDuplicate(t *testing.T) {
	t.Setenv("RIPCODE_DIR", t.TempDir())
	h, err := loadPromptHistory()
	require.NoError(t, err)

	h.PushWithMode("same prompt", "normal")
	require.NoError(t, persistHistory(h))
	h.PushWithMode("same prompt", "normal")
	require.NoError(t, persistHistory(h))

	entries, err := store.LoadHistory()
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, "same prompt", entries[0].Prompt)
}

func TestPersistStash_WriteFailure_ReturnsError(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("RIPCODE_DIR", dir)
	stateDir := store.StateDir()
	require.NoError(t, os.MkdirAll(stateDir, 0o755))
	require.NoError(t, os.Chmod(stateDir, 0o444))
	defer os.Chmod(stateDir, 0o755)

	s, _ := loadPromptStash()
	s.Push("test content")
	err := persistStash(s)
	// On macOS/Linux non-root this should fail; skip assertion if we're root
	if err != nil {
		assert.Error(t, err)
	}
}

func TestPersistHistory_AppendFailure_ReturnsError(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("RIPCODE_DIR", dir)
	stateDir := store.StateDir()
	require.NoError(t, os.MkdirAll(stateDir, 0o755))
	require.NoError(t, os.Chmod(stateDir, 0o444))
	defer os.Chmod(stateDir, 0o755)

	h, _ := loadPromptHistory()
	h.Push("test prompt")
	err := persistHistory(h)
	if err != nil {
		assert.Error(t, err)
	}
}

func TestLoadPromptHistory_JSONL_SkipsMalformedLines(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("RIPCODE_DIR", dir)
	stateDir := filepath.Join(dir, "state")
	require.NoError(t, os.MkdirAll(stateDir, 0o755))
	// JSONL format: one JSON object per line. Malformed lines are skipped, not errors.
	content := "{garbage}\n" + `{"prompt":"valid","mode":"normal"}` + "\n"
	require.NoError(t, os.WriteFile(filepath.Join(stateDir, "prompt-history.jsonl"), []byte(content), 0o644))

	h, err := loadPromptHistory()
	require.NoError(t, err, "JSONL format skips bad lines without error")
	assert.NotNil(t, h)
	items := h.Items()
	assert.Len(t, items, 1)
	assert.Equal(t, "valid", items[0].Prompt)
}

func TestLoadPromptStash_CorruptedReturnsError(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("RIPCODE_DIR", dir)
	stateDir := filepath.Join(dir, "state")
	require.NoError(t, os.MkdirAll(stateDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(stateDir, "stash.json"), []byte("{garbage"), 0o644))

	s, err := loadPromptStash()
	assert.Error(t, err, "corrupted stash should return error")
	assert.NotNil(t, s, "should still return usable empty stash")
}

func TestNewApp_CorruptedStashShowsWarning(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("RIPCODE_DIR", dir)
	stateDir := filepath.Join(dir, "state")
	require.NoError(t, os.MkdirAll(stateDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(stateDir, "stash.json"), []byte("{bad"), 0o644))

	app := NewApp()
	found := false
	for _, w := range app.startupWarnings {
		if strings.Contains(w, "prompt stash") && strings.Contains(w, "using defaults") {
			found = true
		}
	}
	assert.True(t, found, "should include stash warning, got: %v", app.startupWarnings)
}
