package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/stephenbrandon/ripcode/internal/session"
	"github.com/stephenbrandon/ripcode/internal/tui/components"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestApp_TimelineDialog_ShowsTimestamps(t *testing.T) {
	a := makeSessionAppWithHistory(t)
	model, _ := a.Update(components.InputSubmitMsg{Value: "/timeline"})
	a = model.(App)
	assert.True(t, a.timelineDialog.open)

	rendered := a.renderTimelineDialog()
	// Timeline should render timestamps in HH:MM format.
	assert.Contains(t, rendered, ":", "timeline should contain timestamp with colon separator")
	// Entries should include both the time and the message content.
	entries := a.timelineEntries()
	require.NotEmpty(t, entries)
	timeStr := entries[0].Time.Format("15:04")
	assert.Contains(t, rendered, timeStr, "timeline should show message timestamp")
}

func TestApp_TimelineDialog_JumpUsesRenderedLineOffset(t *testing.T) {
	a := makeSessionApp(t)
	sess := session.New(t.TempDir())
	sess.AddUser("first question")
	sess.AddAssistant(strings.Repeat("long assistant content ", 20), nil, nil)
	sess.AddUser("second question target")
	sess.AddAssistant("second answer", nil, nil)
	a.SetSession(sess)
	a.rebuildChatFromSession()
	a.chat.SetSize(50, 200)

	lines := strings.Split(a.chat.View(), "\n")
	expected := -1
	for i, line := range lines {
		if strings.Contains(line, "second question target") {
			expected = i
			break
		}
	}
	require.GreaterOrEqual(t, expected, 0, "expected to find target user message in rendered chat")

	a.scrollToUserMessage(1)
	assert.Equal(t, expected, a.chat.ScrollPos())
}
