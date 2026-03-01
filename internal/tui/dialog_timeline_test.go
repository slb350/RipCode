package tui

import (
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/stephenbrandon/ripcode/internal/tui/components"
	"github.com/stretchr/testify/assert"
)

func TestApp_TimelineDialog_OpensWithSlashTimeline(t *testing.T) {
	a := makeSessionAppWithHistory(t)
	model, _ := a.Update(components.InputSubmitMsg{Value: "/timeline"})
	a = model.(App)
	assert.True(t, a.timelineDialog.open)
}

func TestApp_TimelineDialog_ShowsUserMessages(t *testing.T) {
	a := makeSessionAppWithHistory(t)
	model, _ := a.Update(components.InputSubmitMsg{Value: "/timeline"})
	a = model.(App)
	assert.True(t, a.timelineDialog.open)
	// Should have entries for user messages
	assert.NotEmpty(t, a.timelineEntries())
}

func TestApp_TimelineDialog_EscCloses(t *testing.T) {
	a := makeSessionAppWithHistory(t)
	model, _ := a.Update(components.InputSubmitMsg{Value: "/timeline"})
	a = model.(App)
	assert.True(t, a.timelineDialog.open)
	model, _ = a.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	a = model.(App)
	assert.False(t, a.timelineDialog.open)
}

func TestApp_TimelineDialog_ArrowNavigates(t *testing.T) {
	a := makeSessionAppWithHistory(t)
	model, _ := a.Update(components.InputSubmitMsg{Value: "/timeline"})
	a = model.(App)
	model, _ = a.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	a = model.(App)
	assert.Equal(t, 1, a.timelineDialog.selected)
}
