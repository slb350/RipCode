package components

import (
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"charm.land/lipgloss/v2"
	"github.com/stretchr/testify/assert"
)

func TestChat_AddEntry(t *testing.T) {
	c := NewChat()
	c.SetSize(80, 20)

	c.AddEntry(ChatEntry{Role: RoleUser, Content: "hello"})
	view := c.View()
	assert.Contains(t, view, "hello")
}

func TestChat_AddEntry_SetsCreatedAtWhenUnset(t *testing.T) {
	c := NewChat()
	c.SetSize(80, 20)

	before := time.Now()
	c.AddEntry(ChatEntry{Role: RoleUser, Content: "hello"})

	entries := c.Entries()
	assert.Len(t, entries, 1)
	assert.False(t, entries[0].CreatedAt.IsZero(), "add entry should default CreatedAt")
	assert.False(t, entries[0].CreatedAt.Before(before), "CreatedAt should be set at add time")
}

func TestChat_UserMessage_HasAccentBorder(t *testing.T) {
	c := NewChat()
	c.SetSize(80, 20)

	c.AddEntry(ChatEntry{Role: RoleUser, Content: "Fix the bug"})
	view := c.View()
	assert.Contains(t, view, "┃", "user message should have left accent border")
	assert.Contains(t, view, "Fix the bug")
}

func TestChat_AssistantMessage_HasIndent(t *testing.T) {
	c := NewChat()
	c.SetSize(80, 20)

	c.AddEntry(ChatEntry{Role: RoleAssistant, Content: "Here is my analysis"})
	view := c.View()
	assert.Contains(t, view, "   Here is my analysis", "assistant message should have 3-space indent")
}

func TestChat_ToolCall_ShowsIcon(t *testing.T) {
	c := NewChat()
	c.SetSize(80, 20)

	c.AddEntry(ChatEntry{Role: RoleTool, Content: "ls -la", ToolName: "bash", ToolStatus: StatusSuccess})
	view := c.View()
	assert.Contains(t, view, "$", "bash tool should show $ icon")
	assert.Contains(t, view, "✓", "success tool should show ✓")
}

func TestChat_ToolCall_ReadIcon(t *testing.T) {
	c := NewChat()
	c.SetSize(80, 20)

	c.AddEntry(ChatEntry{Role: RoleTool, Content: "main.go", ToolName: "read", ToolStatus: StatusSuccess})
	view := c.View()
	assert.Contains(t, view, "→", "read tool should show → icon")
}

func TestChat_ToolCall_WriteIcon(t *testing.T) {
	c := NewChat()
	c.SetSize(80, 20)

	c.AddEntry(ChatEntry{Role: RoleTool, Content: "main.go", ToolName: "write", ToolStatus: StatusSuccess})
	view := c.View()
	assert.Contains(t, view, "←", "write tool should show ← icon")
}

func TestChat_ToolCall_GlobIcon(t *testing.T) {
	c := NewChat()
	c.SetSize(80, 20)

	c.AddEntry(ChatEntry{Role: RoleTool, Content: "*.go", ToolName: "glob", ToolStatus: StatusSuccess})
	view := c.View()
	assert.Contains(t, view, "⌕", "glob tool should show ⌕ icon")
}

func TestChat_ToolCall_PendingStatus(t *testing.T) {
	c := NewChat()
	c.SetSize(80, 20)

	c.AddEntry(ChatEntry{Role: RoleTool, Content: "ls", ToolName: "bash", ToolStatus: StatusPending})
	view := c.View()
	assert.Contains(t, view, "~", "pending tool should show ~")
}

func TestChat_ToolCall_ErrorStatus(t *testing.T) {
	c := NewChat()
	c.SetSize(80, 20)

	c.AddEntry(ChatEntry{Role: RoleTool, Content: "failed", ToolName: "write", ToolStatus: StatusError})
	view := c.View()
	assert.Contains(t, view, "✗", "error tool should show ✗")
}

func TestChat_CompletionBar(t *testing.T) {
	c := NewChat()
	c.SetSize(80, 20)

	c.AddEntry(ChatEntry{
		Role: RoleComplete,
		Meta: &CompleteMeta{Mode: "build", Model: "glm-5", Duration: 4700 * time.Millisecond},
	})
	view := c.View()
	assert.Contains(t, view, "▣", "completion bar should show mode icon")
	assert.Contains(t, view, "Build", "completion bar should show mode name")
	assert.Contains(t, view, "glm-5", "completion bar should show model")
	assert.Contains(t, view, "4.7s", "completion bar should show duration")
}

func TestChat_UpdateLastTool(t *testing.T) {
	c := NewChat()
	c.SetSize(80, 20)

	c.AddEntry(ChatEntry{Role: RoleTool, Content: "ls", ToolName: "bash", ToolStatus: StatusPending, ToolID: "t1"})
	c.UpdateLastTool("t1", ChatEntry{Role: RoleTool, Content: "file.go", ToolName: "bash", ToolStatus: StatusSuccess, ToolID: "t1"})

	view := c.View()
	assert.Contains(t, view, "✓")
	assert.NotContains(t, view, "~")
}

func TestChat_StreamAndCommit(t *testing.T) {
	c := NewChat()
	c.SetSize(80, 20)

	c.StreamContent("Hello ")
	c.StreamContent("world")
	view := c.View()
	assert.Contains(t, view, "Hello world")

	c.CommitStream()
	view = c.View()
	assert.Contains(t, view, "Hello world")
}

func TestChat_SystemRole(t *testing.T) {
	c := NewChat()
	c.SetSize(80, 20)

	c.AddEntry(ChatEntry{Role: RoleSystem, Content: "Welcome to ripcode."})
	view := c.View()
	assert.Contains(t, view, "Welcome to ripcode.")
	assert.Contains(t, view, "~")
}

func TestChat_Clear(t *testing.T) {
	c := NewChat()
	c.SetSize(80, 20)

	c.AddEntry(ChatEntry{Role: RoleUser, Content: "hello"})
	c.Clear()

	view := c.View()
	assert.NotContains(t, view, "hello")
}

func TestChat_Clear_ResetsStreamingParts(t *testing.T) {
	c := NewChat()
	c.SetSize(80, 20)
	c.StreamPart(PartText, "streaming content")
	c.Clear()

	assert.Empty(t, c.streamingParts, "Clear should reset streamingParts")
	assert.Empty(t, c.streaming, "Clear should reset legacy streaming")
	assert.Equal(t, 0, c.scrollPos, "Clear should reset scroll position")
}

func TestChat_RenderEntry_UnknownRole_RendersAsPlainText(t *testing.T) {
	c := NewChat()
	c.SetSize(80, 20)

	c.AddEntry(ChatEntry{Role: "unknown", Content: "mystery content"})
	view := c.View()
	assert.Contains(t, view, "mystery content")
}

func TestChat_CommitStream_BothPaths_LogsWarning(t *testing.T) {
	c := NewChat()
	c.SetSize(80, 20)

	// Populate both paths
	c.StreamContent("legacy")
	c.StreamPart(PartText, "parts")
	c.CommitStream()

	// Parts should win, legacy discarded
	entries := c.Entries()
	assert.Len(t, entries, 1)
	assert.Equal(t, "parts", entries[0].Content)
}

func TestChat_Entries_ReturnsAllEntries(t *testing.T) {
	c := NewChat()
	c.AddEntry(ChatEntry{Role: RoleUser, Content: "hello"})
	c.AddEntry(ChatEntry{Role: RoleAssistant, Content: "world"})

	entries := c.Entries()
	assert.Len(t, entries, 2)
	assert.Equal(t, RoleUser, entries[0].Role)
	assert.Equal(t, RoleAssistant, entries[1].Role)
}

func TestChat_Entries_EmptyWhenCleared(t *testing.T) {
	c := NewChat()
	c.AddEntry(ChatEntry{Role: RoleUser, Content: "hello"})
	c.Clear()
	assert.Empty(t, c.Entries())
}

func TestChat_Entries_ReturnsCopy(t *testing.T) {
	c := NewChat()
	c.AddEntry(ChatEntry{Role: RoleUser, Content: "original"})
	entries := c.Entries()
	entries[0].Content = "modified"
	assert.Equal(t, "original", c.Entries()[0].Content)
}

// --- Chat scroll tests ---

func TestChat_PageUp_MovesScrollByHeight(t *testing.T) {
	c := NewChat()
	c.SetSize(80, 10)
	// Add enough entries to have scrollable content
	for i := 0; i < 20; i++ {
		c.AddEntry(ChatEntry{Role: RoleUser, Content: "msg"})
	}
	c.scrollPos = 20
	c.PageUp()
	assert.Equal(t, 10, c.scrollPos)
}

func TestChat_PageDown_MovesScrollByHeight(t *testing.T) {
	c := NewChat()
	c.SetSize(80, 10)
	c.scrollPos = 0
	c.PageDown()
	assert.Equal(t, 10, c.scrollPos)
}

func TestChat_HalfPageUp_MovesHalfHeight(t *testing.T) {
	c := NewChat()
	c.SetSize(80, 10)
	c.scrollPos = 20
	c.HalfPageUp()
	assert.Equal(t, 15, c.scrollPos)
}

func TestChat_HalfPageDown_MovesHalfHeight(t *testing.T) {
	c := NewChat()
	c.SetSize(80, 10)
	c.scrollPos = 0
	c.HalfPageDown()
	assert.Equal(t, 5, c.scrollPos)
}

func TestChat_LineUp_MovesOneLineUp(t *testing.T) {
	c := NewChat()
	c.SetSize(80, 10)
	c.scrollPos = 5
	c.LineUp()
	assert.Equal(t, 4, c.scrollPos)
}

func TestChat_LineDown_MovesOneLineDown(t *testing.T) {
	c := NewChat()
	c.SetSize(80, 10)
	c.scrollPos = 5
	c.LineDown()
	assert.Equal(t, 6, c.scrollPos)
}

func TestChat_ScrollToTop_MovesToZero(t *testing.T) {
	c := NewChat()
	c.SetSize(80, 10)
	c.scrollPos = 50
	c.ScrollToTop()
	assert.Equal(t, 0, c.scrollPos)
}

func TestChat_ScrollToBottom_MovesToEnd(t *testing.T) {
	c := NewChat()
	c.SetSize(80, 10)
	for i := 0; i < 20; i++ {
		c.AddEntry(ChatEntry{Role: RoleUser, Content: "msg"})
	}
	c.scrollPos = 0
	c.ScrollToBottom()
	assert.True(t, c.scrollPos > 0, "scroll should move to bottom")
}

func TestChat_PageUp_ClampsAtZero(t *testing.T) {
	c := NewChat()
	c.SetSize(80, 10)
	c.scrollPos = 3
	c.PageUp()
	assert.Equal(t, 0, c.scrollPos)
}

func TestChat_PageDown_ClampsAtMax(t *testing.T) {
	c := NewChat()
	c.SetSize(80, 10)
	c.AddEntry(ChatEntry{Role: RoleUser, Content: "short"})
	c.scrollPos = 0
	c.PageDown()
	// View() will clamp scrollPos, but the method itself doesn't
	// What we verify is that it doesn't go negative
	assert.GreaterOrEqual(t, c.scrollPos, 0)
}

func TestChat_NextUserMessage_JumpsToNextUser(t *testing.T) {
	c := NewChat()
	c.SetSize(80, 10)
	c.AddEntry(ChatEntry{Role: RoleUser, Content: "first"})
	c.AddEntry(ChatEntry{Role: RoleAssistant, Content: "reply"})
	c.AddEntry(ChatEntry{Role: RoleUser, Content: "second"})
	expected, ok := c.LineOffsetForUserMessage(1)
	assert.True(t, ok)
	c.scrollPos = 0
	c.NextUserMessage()
	assert.Equal(t, expected, c.scrollPos, "should jump to the next user message offset")
}

func TestChat_PrevUserMessage_JumpsToPrevUser(t *testing.T) {
	c := NewChat()
	c.SetSize(80, 10)
	c.AddEntry(ChatEntry{Role: RoleUser, Content: "first"})
	c.AddEntry(ChatEntry{Role: RoleAssistant, Content: "reply"})
	c.AddEntry(ChatEntry{Role: RoleUser, Content: "second"})
	secondOffset, ok := c.LineOffsetForUserMessage(1)
	assert.True(t, ok)
	firstOffset, ok := c.LineOffsetForUserMessage(0)
	assert.True(t, ok)
	c.scrollPos = secondOffset
	c.PrevUserMessage()
	assert.Equal(t, firstOffset, c.scrollPos, "should jump to the previous user message offset")
}

func TestChat_NextUserMessage_UsesRenderedHeights(t *testing.T) {
	c := NewChat()
	c.SetSize(20, 10)
	c.AddEntry(ChatEntry{Role: RoleUser, Content: "first"})
	c.AddEntry(ChatEntry{
		Role:    RoleAssistant,
		Content: "this assistant message wraps over many rendered lines to verify real-offset navigation",
	})
	c.AddEntry(ChatEntry{Role: RoleUser, Content: "second"})

	expected, ok := c.LineOffsetForUserMessage(1)
	assert.True(t, ok)
	c.scrollPos = 0
	c.NextUserMessage()

	assert.Equal(t, expected, c.scrollPos)
}

func TestChat_PrevUserMessage_UsesRenderedHeights(t *testing.T) {
	c := NewChat()
	c.SetSize(20, 10)
	c.AddEntry(ChatEntry{Role: RoleUser, Content: "first"})
	c.AddEntry(ChatEntry{
		Role:    RoleAssistant,
		Content: "this assistant message wraps over many rendered lines to verify real-offset navigation",
	})
	c.AddEntry(ChatEntry{Role: RoleUser, Content: "second"})

	secondOffset, ok := c.LineOffsetForUserMessage(1)
	assert.True(t, ok)
	firstOffset, ok := c.LineOffsetForUserMessage(0)
	assert.True(t, ok)

	c.scrollPos = secondOffset
	c.PrevUserMessage()

	assert.Equal(t, firstOffset, c.scrollPos)
}

func TestChat_ConcealCodeAcrossInterleavedParts(t *testing.T) {
	c := NewChat()
	c.SetSize(100, 20)
	c.SetShowCodeBlocks(false)
	c.SetShowThinking(true)

	c.AddEntry(ChatEntry{
		Role: RoleAssistant,
		Parts: []MessagePart{
			{Type: PartText, Content: "```go\nfmt"},
			{Type: PartReasoning, Content: "thinking"},
			{Type: PartText, Content: ".Println(1)\n```\nAfter"},
		},
	})

	view := c.View()
	assert.Contains(t, view, "[code block hidden]")
	assert.Equal(t, 1, strings.Count(view, "[code block hidden]"))
	assert.NotContains(t, view, "Println(1)")
	assert.Contains(t, view, "thinking")
	assert.Contains(t, view, "After")
}

func TestChat_HiddenReasoning_PreservesAssistantTextFlow(t *testing.T) {
	c := NewChat()
	c.SetSize(100, 20)
	c.SetShowThinking(false)

	c.AddEntry(ChatEntry{
		Role: RoleAssistant,
		Parts: []MessagePart{
			{Type: PartText, Content: "The sea is "},
			{Type: PartReasoning, Content: "hidden reasoning"},
			{Type: PartText, Content: "calm tonight."},
		},
	})

	view := c.View()
	assert.Contains(t, view, "The sea is calm tonight.")
}

func TestChat_RenderUserEntry_TimestampedFirstLine_RespectsViewportWidth(t *testing.T) {
	c := NewChat()
	c.SetSize(40, 20)
	c.SetShowTimestamps(true)

	lines := c.renderUserEntry(ChatEntry{
		Role:      RoleUser,
		Content:   strings.TrimSpace(strings.Repeat("word ", 9)),
		CreatedAt: time.Date(2026, time.January, 1, 15, 4, 0, 0, time.UTC),
	}, c.effectiveTheme())

	assert.NotEmpty(t, lines)
	assert.LessOrEqual(t, lipgloss.Width(lines[0]), c.width, "timestamp prefix should be accounted for in wrapping")
}

func TestChat_RenderToolEntry_DetailsTruncation_PreservesUTF8(t *testing.T) {
	c := NewChat()
	c.SetSize(28, 20) // details width = 20
	c.SetShowDetails(true)

	lines := c.renderToolEntry(ChatEntry{
		Role:       RoleTool,
		Content:    strings.Repeat("a", 19) + "éz",
		ToolName:   "read",
		ToolStatus: StatusSuccess,
	}, c.effectiveTheme())

	assert.GreaterOrEqual(t, len(lines), 2, "details line should render")
	assert.True(t, utf8.ValidString(lines[1]), "detail truncation should not cut through UTF-8 bytes")
}
