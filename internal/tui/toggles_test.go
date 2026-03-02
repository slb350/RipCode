package tui

import (
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/stephenbrandon/ripcode/internal/tui/components"
	"github.com/stretchr/testify/assert"
)

// --- Sub-Phase 5.4: Thinking visibility toggle ---

func TestThinkingToggle_PropagatestoChat(t *testing.T) {
	t.Setenv("RIPCODE_DIR", t.TempDir())
	app := makeSessionApp(t)

	cmd := app.cmdRegistry.Get("thinking")
	assert.NotNil(t, cmd)

	// Initially off
	assert.False(t, app.chat.ShowThinking())

	// Toggle on
	cmd.Handler(&app)
	assert.True(t, app.chat.ShowThinking())

	// Toggle off
	cmd.Handler(&app)
	assert.False(t, app.chat.ShowThinking())
}

func TestThinkingToggle_ReasoningRendersWhenEnabled(t *testing.T) {
	t.Setenv("RIPCODE_DIR", t.TempDir())
	app := makeSessionApp(t)
	app.chat.Clear()
	app.chat.SetSize(80, 20)

	// Add entry with reasoning parts
	app.chat.AddEntry(components.ChatEntry{
		Role: components.RoleAssistant,
		Parts: []components.MessagePart{
			{Type: components.PartReasoning, Content: "deep thought"},
			{Type: components.PartText, Content: "answer"},
		},
	})

	// Thinking off — reasoning hidden
	view := app.chat.View()
	assert.NotContains(t, view, "deep thought")
	assert.Contains(t, view, "answer")

	// Toggle on
	app.chat.SetShowThinking(true)
	view = app.chat.View()
	assert.Contains(t, view, "deep thought")
	assert.Contains(t, view, "answer")
}

func TestThinkingToggle_ReasoningRedacted(t *testing.T) {
	t.Setenv("RIPCODE_DIR", t.TempDir())
	app := makeSessionApp(t)
	app.chat.Clear()
	app.chat.SetSize(80, 20)
	app.chat.SetShowThinking(true)

	app.chat.AddEntry(components.ChatEntry{
		Role: components.RoleAssistant,
		Parts: []components.MessagePart{
			{Type: components.PartReasoning, Content: "[REDACTED]"},
			{Type: components.PartText, Content: "visible"},
		},
	})

	view := app.chat.View()
	assert.Contains(t, view, "[REDACTED]")
	assert.Contains(t, view, "visible")
}

// --- Sub-Phase 5.5: Tool details toggle ---

func TestDetailsToggle_PropagatesToChat(t *testing.T) {
	t.Setenv("RIPCODE_DIR", t.TempDir())
	app := makeSessionApp(t)

	cmd := app.cmdRegistry.Get("details")
	assert.NotNil(t, cmd)

	assert.False(t, app.chat.ShowDetails())

	cmd.Handler(&app)
	assert.True(t, app.chat.ShowDetails())

	cmd.Handler(&app)
	assert.False(t, app.chat.ShowDetails())
}

func TestDetailsToggle_ToolEntryExpandsWhenEnabled(t *testing.T) {
	c := components.NewChat()
	c.SetSize(80, 20)
	c.SetShowDetails(true)

	c.AddEntry(components.ChatEntry{
		Role:       components.RoleTool,
		Content:    "line1\nline2\nline3",
		ToolName:   "bash",
		ToolStatus: components.StatusSuccess,
		ToolID:     "t1",
	})

	view := c.View()
	assert.Contains(t, view, "line1")
	assert.Contains(t, view, "line2")
	assert.Contains(t, view, "line3")
}

func TestDetailsToggle_ToolEntryCollapsedWhenDisabled(t *testing.T) {
	c := components.NewChat()
	c.SetSize(80, 20)
	c.SetShowDetails(false) // default

	c.AddEntry(components.ChatEntry{
		Role:       components.RoleTool,
		Content:    "line1\nline2\nline3",
		ToolName:   "bash",
		ToolStatus: components.StatusSuccess,
		ToolID:     "t1",
	})

	view := c.View()
	// Should show summary line only, not the full output
	assert.Contains(t, view, "bash")
	assert.NotContains(t, view, "line2", "details should be hidden when showDetails is false")
}

func TestDetailsToggle_PendingToolAlwaysSingleLine(t *testing.T) {
	c := components.NewChat()
	c.SetSize(80, 20)
	c.SetShowDetails(true)

	c.AddEntry(components.ChatEntry{
		Role:       components.RoleTool,
		Content:    "waiting\nmore output",
		ToolName:   "bash",
		ToolStatus: components.StatusPending,
		ToolID:     "t1",
	})

	view := c.View()
	assert.NotContains(t, view, "more output", "pending tool should not expand")
}

func TestDetailsToggle_ErrorToolShowsErrorRegardless(t *testing.T) {
	c := components.NewChat()
	c.SetSize(80, 20)
	c.SetShowDetails(false) // details off

	c.AddEntry(components.ChatEntry{
		Role:       components.RoleTool,
		Content:    "error message",
		ToolName:   "write",
		ToolStatus: components.StatusError,
		ToolID:     "t1",
	})

	view := c.View()
	// Error tool should show the error indicator
	assert.Contains(t, view, "✗")
}

// --- Sub-Phase 5.6: Timestamps toggle ---

func TestTimestampsToggle_PropagatesToChat(t *testing.T) {
	t.Setenv("RIPCODE_DIR", t.TempDir())
	app := makeSessionApp(t)

	cmd := app.cmdRegistry.Get("timestamps")
	assert.NotNil(t, cmd)

	assert.False(t, app.chat.ShowTimestamps())

	cmd.Handler(&app)
	assert.True(t, app.chat.ShowTimestamps())

	cmd.Handler(&app)
	assert.False(t, app.chat.ShowTimestamps())
}

func TestTimestamps_UserMessage_ShowsTimestampWhenEnabled(t *testing.T) {
	c := components.NewChat()
	c.SetSize(80, 20)
	c.SetShowTimestamps(true)

	ts := time.Date(2024, 3, 15, 14, 30, 0, 0, time.Local)
	c.AddEntry(components.ChatEntry{
		Role:      components.RoleUser,
		Content:   "hello",
		CreatedAt: ts,
	})

	view := c.View()
	assert.Contains(t, view, "2:30 PM")
	assert.Contains(t, view, "hello")
}

func TestTimestamps_Hidden_WhenFlagDisabled(t *testing.T) {
	c := components.NewChat()
	c.SetSize(80, 20)
	c.SetShowTimestamps(false)

	ts := time.Date(2024, 3, 15, 14, 30, 0, 0, time.Local)
	c.AddEntry(components.ChatEntry{
		Role:      components.RoleUser,
		Content:   "hello",
		CreatedAt: ts,
	})

	view := c.View()
	assert.NotContains(t, view, "2:30 PM")
	assert.Contains(t, view, "hello")
}

func TestTimestamps_Hidden_WhenCreatedAtZero(t *testing.T) {
	c := components.NewChat()
	c.SetSize(80, 20)
	c.SetShowTimestamps(true)

	c.AddEntry(components.ChatEntry{
		Role:    components.RoleUser,
		Content: "no time",
	})

	view := c.View()
	// Should not crash and should render without timestamp
	assert.Contains(t, view, "no time")
}

func TestTimestamps_AssistantMessage_ShowsTimestamp(t *testing.T) {
	c := components.NewChat()
	c.SetSize(80, 20)
	c.SetShowTimestamps(true)

	ts := time.Date(2024, 3, 15, 9, 5, 0, 0, time.Local)
	c.AddEntry(components.ChatEntry{
		Role:      components.RoleAssistant,
		Content:   "response",
		CreatedAt: ts,
	})

	view := c.View()
	assert.Contains(t, view, "9:05 AM")
	assert.Contains(t, view, "response")
}

func TestTimestamps_AssistantParts_ShowsTimestamp(t *testing.T) {
	c := components.NewChat()
	c.SetSize(80, 20)
	c.SetShowTimestamps(true)
	c.SetShowThinking(true)

	ts := time.Date(2024, 6, 10, 15, 45, 0, 0, time.Local)
	c.AddEntry(components.ChatEntry{
		Role:      components.RoleAssistant,
		CreatedAt: ts,
		Parts: []components.MessagePart{
			{Type: components.PartReasoning, Content: "thinking"},
			{Type: components.PartText, Content: "answer"},
		},
	})

	view := c.View()
	assert.Contains(t, view, "3:45 PM")
	assert.Contains(t, view, "answer")
}

// --- Sub-Phase 5.7: Code block concealment ---

func TestConcealCommand_Registered(t *testing.T) {
	t.Setenv("RIPCODE_DIR", t.TempDir())
	app := makeSessionApp(t)

	cmd := app.cmdRegistry.Get("conceal")
	assert.NotNil(t, cmd)
	assert.True(t, cmd.Hidden, "conceal should be a hidden command")
}

func TestConcealCommand_TogglesShowCodeBlocks(t *testing.T) {
	t.Setenv("RIPCODE_DIR", t.TempDir())
	app := makeSessionApp(t)

	assert.True(t, app.chat.ShowCodeBlocks(), "default should be true")

	cmd := app.cmdRegistry.Get("conceal")
	cmd.Handler(&app)
	assert.False(t, app.chat.ShowCodeBlocks())

	cmd.Handler(&app)
	assert.True(t, app.chat.ShowCodeBlocks())
}

func TestLeaderH_DispatchesToConceal(t *testing.T) {
	t.Setenv("RIPCODE_DIR", t.TempDir())
	app := makeSessionApp(t)
	app.leaderPending = true

	assert.True(t, app.chat.ShowCodeBlocks())

	model, _ := app.Update(tea.KeyPressMsg{Code: 'h'})
	a := model.(App)

	assert.False(t, a.leaderPending, "leader should be consumed")
	assert.False(t, a.chat.ShowCodeBlocks(), "should have toggled code blocks off")
}

func TestLeaderH_NoConcealCommand_NoOp(t *testing.T) {
	t.Setenv("RIPCODE_DIR", t.TempDir())
	app := makeSessionApp(t)

	// Remove the conceal command from registry to simulate it not existing
	originalCodeBlocks := app.chat.ShowCodeBlocks()
	app.cmdRegistry = NewCommandRegistry() // empty registry
	app.leaderPending = true

	model, cmd := app.Update(tea.KeyPressMsg{Code: 'h'})
	a := model.(App)

	assert.False(t, a.leaderPending, "leader should be consumed")
	assert.Nil(t, cmd, "should produce no command")
	assert.Equal(t, originalCodeBlocks, a.chat.ShowCodeBlocks(), "code block visibility should be unchanged")
}

func TestTimestamps_Format12Hour(t *testing.T) {
	c := components.NewChat()
	c.SetSize(80, 20)
	c.SetShowTimestamps(true)

	// Test AM time
	am := time.Date(2024, 1, 1, 0, 0, 0, 0, time.Local)
	c.AddEntry(components.ChatEntry{
		Role:      components.RoleUser,
		Content:   "midnight",
		CreatedAt: am,
	})

	view := c.View()
	assert.Contains(t, view, "12:00 AM")
}
