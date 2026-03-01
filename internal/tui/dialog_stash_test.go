package tui

import (
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/stephenbrandon/ripcode/internal/tui/components"
	"github.com/stretchr/testify/assert"
)

func TestApp_StashListDialog_OpensWithSlashStashList(t *testing.T) {
	app := makeSessionApp(t)
	app.stash.Push("one")
	app.stash.Push("two")
	model, _ := app.Update(components.InputSubmitMsg{Value: "/stash-list"})
	a := model.(App)
	assert.True(t, a.stashDialogOpen)
}

func TestApp_StashListDialog_EscCloses(t *testing.T) {
	app := makeSessionApp(t)
	app.stash.Push("one")
	model, _ := app.Update(components.InputSubmitMsg{Value: "/stash-list"})
	a := model.(App)
	model, _ = a.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	a = model.(App)
	assert.False(t, a.stashDialogOpen)
}

func TestApp_StashListDialog_EnterRestores(t *testing.T) {
	app := makeSessionApp(t)
	app.stash.Push("first draft")
	app.stash.Push("second draft")
	model, _ := app.Update(components.InputSubmitMsg{Value: "/stash-list"})
	a := model.(App)
	// Select first entry (default select=0, which is newest = "second draft")
	model, _ = a.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	a = model.(App)
	assert.False(t, a.stashDialogOpen)
	// Should restore the selected entry to input
	assert.Contains(t, a.input.Value(), "draft")
}
