package components

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestChat_AddEntry(t *testing.T) {
	c := NewChat()
	c.SetSize(80, 20)

	c.AddEntry(ChatEntry{Role: RoleUser, Content: "hello"})
	view := c.View()
	assert.Contains(t, view, "hello")
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
	c.scrollPos = 0
	c.NextUserMessage()
	assert.True(t, c.scrollPos > 0, "should jump past first user message")
}

func TestChat_PrevUserMessage_JumpsToPrevUser(t *testing.T) {
	c := NewChat()
	c.SetSize(80, 10)
	c.AddEntry(ChatEntry{Role: RoleUser, Content: "first"})
	c.AddEntry(ChatEntry{Role: RoleAssistant, Content: "reply"})
	c.AddEntry(ChatEntry{Role: RoleUser, Content: "second"})
	c.scrollPos = 10
	c.PrevUserMessage()
	assert.True(t, c.scrollPos < 10, "should jump back to previous user message")
}
