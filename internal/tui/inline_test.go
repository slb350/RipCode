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
	assert.True(t, a.inlineOpen)

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

	assert.True(t, a.inlineOpen)
	assert.Equal(t, "@", a.inlineMode)
	assert.Equal(t, "ma", a.inlineQuery)
	// inlineStart should be rune index 3 (after "日本 "), not byte index 7
	assert.Equal(t, 3, a.inlineStart)

	// Select the file and confirm replacement works correctly.
	a.fileCacheLoaded = true
	a.fileCache = []string{"main.go"}
	cmd = a.updateInlineSuggestions()
	_ = cmd

	model, _ = a.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	a = model.(App)

	assert.False(t, a.inlineOpen)
	assert.Equal(t, "日本 @main.go ", a.input.Value())
}
