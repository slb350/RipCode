package tui

import (
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/stephenbrandon/ripcode/internal/tui/components"
	"github.com/stretchr/testify/assert"
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
