package tui

import (
	"os"
	"path/filepath"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/stephenbrandon/ripcode/internal/agent"
	"github.com/stephenbrandon/ripcode/internal/provider"
	"github.com/stephenbrandon/ripcode/internal/session"
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
}
