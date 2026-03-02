package tui

import (
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/stephenbrandon/ripcode/internal/agent"
	"github.com/stephenbrandon/ripcode/internal/provider"
	"github.com/stephenbrandon/ripcode/internal/session"
	"github.com/stephenbrandon/ripcode/internal/tool"
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

// --- Timeline badge tests ---

func makeTimelineApp(t *testing.T) App {
	t.Helper()
	t.Setenv("RIPCODE_DIR", t.TempDir())
	app := NewApp()
	app.SetProvider(&modelListProvider{})
	app.SetRegistry(tool.NewRegistry())
	sess := session.New(t.TempDir())
	app.SetSession(sess)
	app.SetAgent(agent.BuildAgent())
	app.state = StateSession
	model, _ := app.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	return model.(App)
}

func TestTimelineEntries_UserOnly_NoBadges(t *testing.T) {
	a := makeTimelineApp(t)
	a.session.AddUser("question one")
	a.session.AddUser("question two")

	entries := a.timelineEntries()
	require.Len(t, entries, 2)
	assert.Equal(t, 0, entries[0].Tokens)
	assert.Equal(t, 0, entries[0].Tools)
	assert.Equal(t, time.Duration(0), entries[0].Duration)
	assert.Equal(t, timelineStatusOK, entries[0].Status)
}

func TestTimelineEntries_WithAssistant_TokensDuration(t *testing.T) {
	a := makeTimelineApp(t)
	a.session.AddUser("test prompt")
	a.session.AddAssistant("response", nil, &session.AssistantMeta{
		InputTokens:  500,
		OutputTokens: 300,
		Duration:     2500 * time.Millisecond,
		FinishReason: "stop",
	})

	entries := a.timelineEntries()
	require.Len(t, entries, 1)
	assert.Equal(t, 800, entries[0].Tokens)
	assert.Equal(t, 2500*time.Millisecond, entries[0].Duration)
	assert.Equal(t, timelineStatusOK, entries[0].Status)
}

func TestTimelineEntries_WithToolCalls_ToolCount(t *testing.T) {
	a := makeTimelineApp(t)
	a.session.AddUser("run some tools")
	a.session.AddAssistant("I'll help", []provider.ToolCall{
		{ID: "t1", Name: "bash", Args: "{}"},
		{ID: "t2", Name: "read", Args: "{}"},
	}, &session.AssistantMeta{
		InputTokens:  100,
		OutputTokens: 50,
	})
	a.session.AddToolResult("t1", "output1")
	a.session.AddToolResult("t2", "output2")

	entries := a.timelineEntries()
	require.Len(t, entries, 1)
	assert.Equal(t, 2, entries[0].Tools)
}

func TestTimelineEntries_MultipleExchanges_Independent(t *testing.T) {
	a := makeTimelineApp(t)
	a.session.AddUser("first")
	a.session.AddAssistant("first response", nil, &session.AssistantMeta{
		InputTokens:  100,
		OutputTokens: 50,
	})
	a.session.AddUser("second")
	a.session.AddAssistant("second response", nil, &session.AssistantMeta{
		InputTokens:  200,
		OutputTokens: 100,
	})

	entries := a.timelineEntries()
	require.Len(t, entries, 2)
	assert.Equal(t, 150, entries[0].Tokens)
	assert.Equal(t, 300, entries[1].Tokens)
}

func TestTimelineEntries_SumsTokensAcrossAssistantSteps(t *testing.T) {
	a := makeTimelineApp(t)
	a.session.AddUser("multi-step")
	a.session.AddAssistant("step 1", nil, &session.AssistantMeta{
		InputTokens:  100,
		OutputTokens: 50,
	})
	a.session.AddToolResult("call-1", "ok")
	a.session.AddAssistant("step 2", nil, &session.AssistantMeta{
		InputTokens:  60,
		OutputTokens: 40,
	})

	entries := a.timelineEntries()
	require.Len(t, entries, 1)
	assert.Equal(t, 250, entries[0].Tokens, "timeline token badge should total all assistant steps in the exchange")
}

func TestTimelineEntries_Interrupted(t *testing.T) {
	a := makeTimelineApp(t)
	a.session.AddUser("long request")
	a.session.AddAssistant("partial...", nil, &session.AssistantMeta{
		FinishReason: "length",
	})

	entries := a.timelineEntries()
	require.Len(t, entries, 1)
	assert.Equal(t, timelineStatusInterrupted, entries[0].Status)
}

func TestTimelineEntries_EmptySession(t *testing.T) {
	a := makeTimelineApp(t)
	entries := a.timelineEntries()
	assert.Nil(t, entries)
}

func TestTimelineBadgeRendering_TokensFormatted(t *testing.T) {
	a := makeTimelineApp(t)
	a.session.AddUser("test")
	a.session.AddAssistant("response", nil, &session.AssistantMeta{
		InputTokens:  800,
		OutputTokens: 400,
		Duration:     3 * time.Second,
	})
	a.timelineDialog.open = true

	rendered := a.renderTimelineDialog()
	assert.Contains(t, rendered, "⊙")
	assert.Contains(t, rendered, "3.0s")
}

func TestTimelineBadgeRendering_ToolCount(t *testing.T) {
	a := makeTimelineApp(t)
	a.session.AddUser("tools test")
	a.session.AddAssistant("using tools", []provider.ToolCall{
		{ID: "t1", Name: "bash", Args: "{}"},
	}, &session.AssistantMeta{InputTokens: 100, OutputTokens: 50})
	a.session.AddToolResult("t1", "ok")
	a.timelineDialog.open = true

	rendered := a.renderTimelineDialog()
	assert.Contains(t, rendered, "⚡ 1 tool")
	assert.NotContains(t, rendered, "⚡ 1 tools")
}

func TestTimelineBadgeRendering_ToolCountPlural(t *testing.T) {
	a := makeTimelineApp(t)
	a.session.AddUser("tools test")
	a.session.AddAssistant("using tools", []provider.ToolCall{
		{ID: "t1", Name: "bash", Args: "{}"},
		{ID: "t2", Name: "read", Args: "{}"},
	}, &session.AssistantMeta{InputTokens: 100, OutputTokens: 50})
	a.session.AddToolResult("t1", "ok")
	a.session.AddToolResult("t2", "ok")
	a.timelineDialog.open = true

	rendered := a.renderTimelineDialog()
	assert.Contains(t, rendered, "⚡ 2 tools")
}

func TestTimelineBadgeRendering_Interrupted(t *testing.T) {
	a := makeTimelineApp(t)
	a.session.AddUser("interrupted test")
	a.session.AddAssistant("partial", nil, &session.AssistantMeta{
		FinishReason: "length",
	})
	a.timelineDialog.open = true

	rendered := a.renderTimelineDialog()
	assert.Contains(t, rendered, "⚠ interrupted")
}

func TestTimelineBadgeRendering_NoBadgesWhenAllZero(t *testing.T) {
	a := makeTimelineApp(t)
	a.session.AddUser("just a question")
	a.timelineDialog.open = true

	rendered := a.renderTimelineDialog()
	assert.NotContains(t, rendered, "⊙")
	assert.NotContains(t, rendered, "⚡")
}

func TestTimelineBadgeRendering_SelectionHighlightPreserved(t *testing.T) {
	a := makeTimelineApp(t)
	a.session.AddUser("first")
	a.session.AddAssistant("r1", nil, &session.AssistantMeta{InputTokens: 100, OutputTokens: 50})
	a.session.AddUser("second")
	a.session.AddAssistant("r2", nil, &session.AssistantMeta{InputTokens: 200, OutputTokens: 100})
	a.timelineDialog.open = true
	a.timelineDialog.selected = 1

	rendered := a.renderTimelineDialog()
	lines := strings.Split(rendered, "\n")
	// Find the selected entry (starts with "> ")
	found := false
	for _, line := range lines {
		if strings.HasPrefix(line, "> ") {
			found = true
			assert.Contains(t, line, "second")
		}
	}
	assert.True(t, found, "selected entry should have > marker")
}
