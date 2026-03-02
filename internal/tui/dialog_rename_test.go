package tui

import (
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/stephenbrandon/ripcode/internal/store"
	"github.com/stephenbrandon/ripcode/internal/tui/components"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApp_RenameDialog_OpensWithSlashRename(t *testing.T) {
	app := makeSessionApp(t)
	model, _ := app.Update(components.InputSubmitMsg{Value: "/rename"})
	a := model.(App)
	assert.True(t, a.renameDialog.open)
}

func TestApp_RenameDialog_OpensWithCtrlR(t *testing.T) {
	app := makeSessionApp(t)
	model, _ := app.Update(tea.KeyPressMsg{Mod: tea.ModCtrl, Code: 'r'})
	a := model.(App)
	assert.True(t, a.renameDialog.open)
}

func TestApp_RenameDialog_PrefillsCurrentTitle(t *testing.T) {
	app := makeSessionApp(t)
	app.session.Title = "existing title"
	model, _ := app.Update(components.InputSubmitMsg{Value: "/rename"})
	a := model.(App)
	assert.True(t, a.renameDialog.open)
	assert.Equal(t, "existing title", a.renameDialog.value)
}

func TestApp_RenameDialog_TypingAppendsToValue(t *testing.T) {
	app := makeSessionApp(t)
	model, _ := app.Update(components.InputSubmitMsg{Value: "/rename"})
	a := model.(App)
	model, _ = a.Update(tea.KeyPressMsg{Text: "h"})
	a = model.(App)
	model, _ = a.Update(tea.KeyPressMsg{Text: "i"})
	a = model.(App)
	assert.Equal(t, "hi", a.renameDialog.value)
}

func TestApp_RenameDialog_BackspaceDeletesChar(t *testing.T) {
	app := makeSessionApp(t)
	app.session.Title = "abc"
	model, _ := app.Update(components.InputSubmitMsg{Value: "/rename"})
	a := model.(App)
	model, _ = a.Update(tea.KeyPressMsg{Code: tea.KeyBackspace})
	a = model.(App)
	assert.Equal(t, "ab", a.renameDialog.value)
}

func TestApp_RenameDialog_EnterAppliesAndCloses(t *testing.T) {
	app := makeSessionApp(t)
	model, _ := app.Update(components.InputSubmitMsg{Value: "/rename"})
	a := model.(App)
	model, _ = a.Update(tea.KeyPressMsg{Text: "n"})
	a = model.(App)
	model, _ = a.Update(tea.KeyPressMsg{Text: "e"})
	a = model.(App)
	model, _ = a.Update(tea.KeyPressMsg{Text: "w"})
	a = model.(App)
	model, _ = a.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	a = model.(App)
	assert.False(t, a.renameDialog.open)
	assert.Equal(t, "new", a.session.Title)
}

func TestApp_RenameDialog_EscCancelsWithoutApplying(t *testing.T) {
	app := makeSessionApp(t)
	app.session.Title = "old"
	model, _ := app.Update(components.InputSubmitMsg{Value: "/rename"})
	a := model.(App)
	model, _ = a.Update(tea.KeyPressMsg{Text: "x"})
	a = model.(App)
	model, _ = a.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	a = model.(App)
	assert.False(t, a.renameDialog.open)
	assert.Equal(t, "old", a.session.Title)
}

func TestApp_RenameDialog_EmptyTitle_ShowsWarning(t *testing.T) {
	app := makeSessionApp(t)
	model, _ := app.Update(components.InputSubmitMsg{Value: "/rename"})
	a := model.(App)
	// Just press Enter with empty value
	model, _ = a.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	a = model.(App)
	// Dialog should remain open
	assert.True(t, a.renameDialog.open)
}

func TestApp_RenameDialog_UpdatesSessionTitle(t *testing.T) {
	app := makeSessionApp(t)
	app.session.Title = ""
	model, _ := app.Update(components.InputSubmitMsg{Value: "/rename"})
	a := model.(App)
	model, _ = a.Update(tea.KeyPressMsg{Text: "m"})
	a = model.(App)
	model, _ = a.Update(tea.KeyPressMsg{Text: "y"})
	a = model.(App)
	model, _ = a.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	a = model.(App)
	assert.Equal(t, "my", a.session.Title)
}

func TestApp_RenameDialog_ShowsSuccessToast(t *testing.T) {
	app := makeSessionApp(t)
	model, _ := app.Update(components.InputSubmitMsg{Value: "/rename"})
	a := model.(App)
	model, _ = a.Update(tea.KeyPressMsg{Text: "x"})
	a = model.(App)
	model, cmd := a.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	a = model.(App)
	assert.False(t, a.renameDialog.open)
	assert.NotNil(t, cmd) // dismiss cmd is returned
	// Toast should be visible immediately after Enter
	toast := a.toasts.Current()
	assert.NotNil(t, toast)
	assert.Contains(t, toast.Message, "Renamed")
}

func TestApp_RenameDialog_PersistsToDisk(t *testing.T) {
	app := makeSessionApp(t)
	sessID := app.session.ID
	model, _ := app.Update(components.InputSubmitMsg{Value: "/rename"})
	a := model.(App)
	model, _ = a.Update(tea.KeyPressMsg{Text: "m"})
	a = model.(App)
	model, _ = a.Update(tea.KeyPressMsg{Text: "y"})
	a = model.(App)
	model, _ = a.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	_ = model.(App)

	// Load from disk and verify title was persisted
	loaded, err := store.Load(sessID)
	require.NoError(t, err, "session should be persisted to disk on rename")
	assert.Equal(t, "my", loaded.Title, "renamed title should be persisted")
}
