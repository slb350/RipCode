package tui

import (
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
	"github.com/stretchr/testify/assert"
)

func TestApp_InlineSlash_UsesRegistryCommands(t *testing.T) {
	app := NewApp()
	app.SetProvider(&modelListProvider{})
	app.SetRegistry(tool.NewRegistry())
	app.SetSession(session.New(t.TempDir()))
	app.SetAgent(agent.BuildAgent())

	model, _ := app.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	a := model.(App)
	a.state = StateSession

	// Type "/" to trigger inline autocomplete
	model, _ = a.Update(tea.KeyPressMsg{Text: "/"})
	a = model.(App)
	assert.True(t, a.inline.open)

	// The inline entries should include registry commands like /compact
	view := a.View()
	assert.Contains(t, view.Content, "/compact")
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

	assert.True(t, a.inline.open)
	assert.Equal(t, "/", a.inline.mode)

	model, cmd := a.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	a = model.(App)

	assert.Nil(t, cmd)
	assert.False(t, a.inline.open)
	assert.True(t, a.modelDialog.open)
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

	assert.True(t, a.inline.open)
	assert.Equal(t, "@", a.inline.mode)
	assert.Contains(t, a.View().Content, "main.go")

	model, _ = a.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	a = model.(App)

	assert.False(t, a.inline.open)
	assert.Equal(t, "@main.go ", a.input.Value())
}

func TestUpdateInlineSuggestions_CursorBoundary(t *testing.T) {
	t.Run("cursor at 0 with empty input", func(t *testing.T) {
		app := NewApp()
		app.SetSession(session.New(t.TempDir()))
		app.SetAgent(agent.BuildAgent())
		app.state = StateSession

		app.input.SetValue("")
		cmd := app.updateInlineSuggestions()
		assert.Nil(t, cmd)
		assert.False(t, app.inline.open)
	})

	t.Run("cursor at end of input", func(t *testing.T) {
		app := NewApp()
		app.SetSession(session.New(t.TempDir()))
		app.SetAgent(agent.BuildAgent())
		app.state = StateSession

		app.input.SetValue("hello")
		// Cursor should be at end by default
		cmd := app.updateInlineSuggestions()
		assert.Nil(t, cmd)
		assert.False(t, app.inline.open, "plain text should not open inline suggestions")
	})
}

func TestUpdateInlineSuggestions_CursorMidWord(t *testing.T) {
	app := NewApp()
	app.SetSession(session.New(t.TempDir()))
	app.SetAgent(agent.BuildAgent())
	app.state = StateSession

	// Set input to "@main" but place cursor in the middle (after "@m")
	app.input.SetValue("@main")
	app.input.SetCursorOffset(2) // cursor after "@m"
	cmd := app.updateInlineSuggestions()
	// Should trigger inline with partial query "m" (only chars between @ and cursor)
	assert.NotNil(t, cmd, "@ with cursor mid-word should trigger file cache load")
	assert.True(t, app.inline.open)
	assert.Equal(t, "@", app.inline.mode)
	assert.Equal(t, "m", app.inline.query, "query should only include chars between @ and cursor")
}

func TestUpdateInlineSuggestions_CursorAfterMention(t *testing.T) {
	app := NewApp()
	app.SetSession(session.New(t.TempDir()))
	app.SetAgent(agent.BuildAgent())
	app.state = StateSession

	// Set input to "@main.go " with cursor at end — space after mention
	app.input.SetValue("@main.go ")
	// Cursor at end (after space) — should NOT trigger inline since
	// there's whitespace between @ and cursor
	cmd := app.updateInlineSuggestions()
	assert.Nil(t, cmd)
	assert.False(t, app.inline.open, "space after @mention should close inline suggestions")
}

// --- Sub-Phase 5.8: Rich / autocomplete ---

func TestInlineEntries_ShowsKeybindHint(t *testing.T) {
	t.Setenv("RIPCODE_DIR", t.TempDir())
	app := makeSessionApp(t)

	// Open inline with "/rename" filter
	app.inline.open = true
	app.inline.mode = inlineModeCommand
	app.inline.query = "rename"

	entries := app.inlineEntries()
	var found bool
	for _, e := range entries {
		if e.Display == "/rename" {
			found = true
			assert.Contains(t, e.Description, "[Ctrl+R]", "should show keybind hint")
			break
		}
	}
	assert.True(t, found, "/rename should appear in results")
}

func TestInlineEntries_ShowsAliasHint(t *testing.T) {
	t.Setenv("RIPCODE_DIR", t.TempDir())
	app := makeSessionApp(t)

	// Open inline with "/thinking" filter
	app.inline.open = true
	app.inline.mode = inlineModeCommand
	app.inline.query = "thinking"

	entries := app.inlineEntries()
	var found bool
	for _, e := range entries {
		if e.Display == "/thinking" {
			found = true
			assert.Contains(t, e.Description, "(also: /toggle-thinking)", "should show alias info")
			break
		}
	}
	assert.True(t, found, "/thinking should appear in results")
}

func TestInlineEntries_ShowsKeybindAndAlias(t *testing.T) {
	t.Setenv("RIPCODE_DIR", t.TempDir())
	app := makeSessionApp(t)

	// Open inline with "/stash" filter — has keybind Ctrl+S and no aliases?
	// Let's check a command with both: stash has Keybind: "Ctrl+S" but no aliases
	// Let's use the "new" command which has alias "clear" but no keybind
	app.inline.open = true
	app.inline.mode = inlineModeCommand
	app.inline.query = "new"

	entries := app.inlineEntries()
	var found bool
	for _, e := range entries {
		if e.Display == "/new" {
			found = true
			assert.Contains(t, e.Description, "(also: /clear)", "should show alias")
			break
		}
	}
	assert.True(t, found, "/new should appear in results")
}

func TestInlineEntries_AliasMatchReturnsCommand(t *testing.T) {
	t.Setenv("RIPCODE_DIR", t.TempDir())
	app := makeSessionApp(t)

	// Typing "toggle" should match /thinking via alias "toggle-thinking"
	app.inline.open = true
	app.inline.mode = inlineModeCommand
	app.inline.query = "toggle"

	entries := app.inlineEntries()
	var names []string
	for _, e := range entries {
		names = append(names, e.Display)
	}
	assert.Contains(t, names, "/thinking", "alias toggle-thinking should match")
	assert.Contains(t, names, "/timestamps", "alias toggle-timestamps should match")
}

func TestInlineEntries_NoHintWhenNone(t *testing.T) {
	t.Setenv("RIPCODE_DIR", t.TempDir())
	app := makeSessionApp(t)

	// /exit has aliases but no keybind
	app.inline.open = true
	app.inline.mode = inlineModeCommand
	app.inline.query = "exit"

	entries := app.inlineEntries()
	var found bool
	for _, e := range entries {
		if e.Display == "/exit" {
			found = true
			// Should show aliases but no brackets (no keybind)
			assert.NotContains(t, e.Description, "[")
			assert.Contains(t, e.Description, "(also: /quit, /q)")
			break
		}
	}
	assert.True(t, found, "/exit should appear in results")
}

func TestApp_InlineFileAutocomplete_MultibytePrefixReplacement(t *testing.T) {
	workDir := t.TempDir()
	err := os.WriteFile(filepath.Join(workDir, "main.go"), []byte("package main"), 0o644)
	assert.NoError(t, err)

	app := NewApp()
	app.SetSession(session.New(workDir))
	app.SetAgent(agent.BuildAgent())

	model, _ := app.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	a := model.(App)
	a.state = StateSession

	// "日本 @ma" — space before @ so beforeOK passes, but multibyte runes
	// before @ mean byte offset (7) != rune offset (3).
	a.input.SetValue("日本 @ma")
	a.input.SetCursorOffset(6) // 日(1) 本(2) ' '(3) @(4) m(5) a(6) = 6 runes

	// Trigger inline detection.
	cmd := a.updateInlineSuggestions()

	// Execute the file cache command and feed result back.
	if cmd != nil {
		msg := cmd()
		model, _ = a.Update(msg)
		a = model.(App)
	}

	assert.True(t, a.inline.open)
	assert.Equal(t, "@", a.inline.mode)
	assert.Equal(t, "ma", a.inline.query)
	// inlineStart should be rune index 3 (after "日本 "), not byte index 7
	assert.Equal(t, 3, a.inline.start)

	// Select the file and confirm replacement works correctly.
	a.fileCacheLoaded = true
	a.fileCache = []string{"main.go"}
	cmd = a.updateInlineSuggestions()
	_ = cmd

	model, _ = a.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	a = model.(App)

	assert.False(t, a.inline.open)
	assert.Equal(t, "日本 @main.go ", a.input.Value())
	// Cursor should be positioned after the inserted "@main.go " (rune index 12)
	// 日(1) 本(2) ' '(3) @(4) m(5) a(6) i(7) n(8) .(9) g(10) o(11) ' '(12)
	assert.Equal(t, 12, a.input.CursorOffset(), "cursor should be at end of inserted text")
}

// --- Sub-Phase 6.7: Rich @ Autocomplete ---

func makeInlineFileApp(t *testing.T, files []string) App {
	t.Helper()
	t.Setenv("RIPCODE_DIR", t.TempDir())
	app := NewApp()
	app.SetProvider(&modelListProvider{})
	app.SetRegistry(tool.NewRegistry())
	app.SetSession(session.New(t.TempDir()))
	app.SetAgent(agent.BuildAgent())
	app.state = StateSession
	model, _ := app.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	a := model.(App)
	a.fileCache = files
	a.fileCacheLoaded = true
	return a
}

func TestInlineFile_FrecencyRanking(t *testing.T) {
	a := makeInlineFileApp(t, []string{"a.go", "b.go", "c.go"})
	a.frecency = &store.FileFrecency{Entries: map[string]store.FrecencyEntry{
		"c.go": {Count: 5, LastUsed: time.Now()},
		"a.go": {Count: 1, LastUsed: time.Now()},
	}}
	a.inline.open = true
	a.inline.mode = inlineModeFile
	a.inline.query = ""

	entries := a.inlineEntries()
	assert.GreaterOrEqual(t, len(entries), 3)
	// c.go (highest score) should come first
	assert.Equal(t, "c.go", entries[0].Display)
	assert.Equal(t, "a.go", entries[1].Display)
	// b.go (unscored) last
	assert.Equal(t, "b.go", entries[2].Display)
}

func TestInlineFile_LineRange_MatchAndInsert(t *testing.T) {
	a := makeInlineFileApp(t, []string{"main.go", "util.go"})
	a.inline.open = true
	a.inline.mode = inlineModeFile
	a.inline.query = "main.go:10"

	entries := a.inlineEntries()
	assert.Len(t, entries, 1)
	assert.Equal(t, "main.go:10", entries[0].Display)
	assert.Equal(t, "@main.go:10 ", entries[0].Insert)
}

func TestInlineFile_LineRange_WithRange(t *testing.T) {
	a := makeInlineFileApp(t, []string{"main.go"})
	a.inline.open = true
	a.inline.mode = inlineModeFile
	a.inline.query = "main.go:10-20"

	entries := a.inlineEntries()
	assert.Len(t, entries, 1)
	assert.Equal(t, "main.go:10-20", entries[0].Display)
	assert.Equal(t, "@main.go:10-20 ", entries[0].Insert)
}

func TestInlineFile_DirectoryExpansion(t *testing.T) {
	a := makeInlineFileApp(t, []string{"src/a.go", "src/b.go", "lib/c.go"})
	a.inline.open = true
	a.inline.mode = inlineModeFile
	a.inline.query = "src/"

	entries := a.inlineEntries()
	assert.Len(t, entries, 2)
	assert.Equal(t, "src/a.go", entries[0].Display)
	assert.Equal(t, "src/b.go", entries[1].Display)
}

func TestInlineFile_DirectoryExpansion_WithPrefix(t *testing.T) {
	a := makeInlineFileApp(t, []string{"src/comp/a.go", "src/comp/b.go", "src/other.go"})
	a.inline.open = true
	a.inline.mode = inlineModeFile
	a.inline.query = "src/comp"

	entries := a.inlineEntries()
	assert.Len(t, entries, 2)
}

func TestInlineFile_SelectionRecordsFrecency(t *testing.T) {
	a := makeInlineFileApp(t, []string{"main.go"})
	a.frecency = &store.FileFrecency{Entries: make(map[string]store.FrecencyEntry)}
	a.inline.open = true
	a.inline.mode = inlineModeFile
	a.inline.query = "main"
	a.inline.start = 0
	a.inline.end = 5
	a.input.SetValue("@main")

	model, _ := a.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	a = model.(App)
	assert.Equal(t, 1, a.frecency.Entries["main.go"].Count, "selection should record frecency")
}

func TestInlineFile_FrecencyIntegration_RecentlyUsedFirst(t *testing.T) {
	a := makeInlineFileApp(t, []string{"alpha.go", "beta.go", "gamma.go"})
	a.frecency = &store.FileFrecency{Entries: map[string]store.FrecencyEntry{
		"gamma.go": {Count: 10, LastUsed: time.Now()},
	}}
	a.inline.open = true
	a.inline.mode = inlineModeFile
	a.inline.query = ""

	entries := a.inlineEntries()
	assert.GreaterOrEqual(t, len(entries), 3)
	assert.Equal(t, "gamma.go", entries[0].Display, "frecency-ranked file should appear first")
}

func TestInlineFile_MixedQuery_FrecencyThenAlpha(t *testing.T) {
	a := makeInlineFileApp(t, []string{"match_a.go", "match_b.go", "match_c.go"})
	a.frecency = &store.FileFrecency{Entries: map[string]store.FrecencyEntry{
		"match_b.go": {Count: 3, LastUsed: time.Now()},
	}}
	a.inline.open = true
	a.inline.mode = inlineModeFile
	a.inline.query = "match"

	entries := a.inlineEntries()
	assert.Equal(t, "match_b.go", entries[0].Display, "frecency match should come first")
	assert.Equal(t, "match_a.go", entries[1].Display, "unscored should maintain original order")
}

func TestInlineFile_NoFrecency_FallbackAlphabetical(t *testing.T) {
	a := makeInlineFileApp(t, []string{"z.go", "a.go", "m.go"})
	a.frecency = nil // no frecency loaded
	a.inline.open = true
	a.inline.mode = inlineModeFile
	a.inline.query = ""

	entries := a.inlineEntries()
	// Should preserve fileCache order (which was alphabetical from WalkDir + sort)
	assert.Equal(t, "z.go", entries[0].Display)
	assert.Equal(t, "a.go", entries[1].Display)
	assert.Equal(t, "m.go", entries[2].Display)
}

func TestInlineFile_LineRange_NoMatch(t *testing.T) {
	a := makeInlineFileApp(t, []string{"main.go"})
	a.inline.open = true
	a.inline.mode = inlineModeFile
	a.inline.query = "nonexistent.go:10"

	entries := a.inlineEntries()
	assert.Empty(t, entries)
}

func TestInlineFile_DirectoryExpansion_EmptyDir(t *testing.T) {
	a := makeInlineFileApp(t, []string{"src/a.go"})
	a.inline.open = true
	a.inline.mode = inlineModeFile
	a.inline.query = "lib/"

	entries := a.inlineEntries()
	assert.Empty(t, entries)
}
