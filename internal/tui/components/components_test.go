package components

import (
	"testing"
	"time"

	"github.com/stephenbrandon/ripcode/internal/agent"
	"github.com/stretchr/testify/assert"
)

// --- Chat tests ---

func TestChat_AddEntry(t *testing.T) {
	c := NewChat()
	c.SetSize(80, 20)

	c.AddEntry(ChatEntry{Role: "user", Content: "hello"})
	view := c.View()
	assert.Contains(t, view, "hello")
}

func TestChat_UserMessage_HasAccentBorder(t *testing.T) {
	c := NewChat()
	c.SetSize(80, 20)

	c.AddEntry(ChatEntry{Role: "user", Content: "Fix the bug"})
	view := c.View()
	assert.Contains(t, view, "┃", "user message should have left accent border")
	assert.Contains(t, view, "Fix the bug")
}

func TestChat_AssistantMessage_HasIndent(t *testing.T) {
	c := NewChat()
	c.SetSize(80, 20)

	c.AddEntry(ChatEntry{Role: "assistant", Content: "Here is my analysis"})
	view := c.View()
	assert.Contains(t, view, "   Here is my analysis", "assistant message should have 3-space indent")
}

func TestChat_ToolCall_ShowsIcon(t *testing.T) {
	c := NewChat()
	c.SetSize(80, 20)

	c.AddEntry(ChatEntry{Role: "tool", Content: "ls -la", ToolName: "bash", ToolStatus: "success"})
	view := c.View()
	assert.Contains(t, view, "$", "bash tool should show $ icon")
	assert.Contains(t, view, "✓", "success tool should show ✓")
}

func TestChat_ToolCall_ReadIcon(t *testing.T) {
	c := NewChat()
	c.SetSize(80, 20)

	c.AddEntry(ChatEntry{Role: "tool", Content: "main.go", ToolName: "read", ToolStatus: "success"})
	view := c.View()
	assert.Contains(t, view, "→", "read tool should show → icon")
}

func TestChat_ToolCall_WriteIcon(t *testing.T) {
	c := NewChat()
	c.SetSize(80, 20)

	c.AddEntry(ChatEntry{Role: "tool", Content: "main.go", ToolName: "write", ToolStatus: "success"})
	view := c.View()
	assert.Contains(t, view, "←", "write tool should show ← icon")
}

func TestChat_ToolCall_GlobIcon(t *testing.T) {
	c := NewChat()
	c.SetSize(80, 20)

	c.AddEntry(ChatEntry{Role: "tool", Content: "*.go", ToolName: "glob", ToolStatus: "success"})
	view := c.View()
	assert.Contains(t, view, "⌕", "glob tool should show ⌕ icon")
}

func TestChat_ToolCall_PendingStatus(t *testing.T) {
	c := NewChat()
	c.SetSize(80, 20)

	c.AddEntry(ChatEntry{Role: "tool", Content: "ls", ToolName: "bash", ToolStatus: "pending"})
	view := c.View()
	assert.Contains(t, view, "~", "pending tool should show ~")
}

func TestChat_ToolCall_ErrorStatus(t *testing.T) {
	c := NewChat()
	c.SetSize(80, 20)

	c.AddEntry(ChatEntry{Role: "tool", Content: "failed", ToolName: "write", ToolStatus: "error"})
	view := c.View()
	assert.Contains(t, view, "✗", "error tool should show ✗")
}

func TestChat_CompletionBar(t *testing.T) {
	c := NewChat()
	c.SetSize(80, 20)

	c.AddEntry(ChatEntry{
		Role: "complete",
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

	c.AddEntry(ChatEntry{Role: "tool", Content: "ls", ToolName: "bash", ToolStatus: "pending", ToolID: "t1"})
	c.UpdateLastTool("t1", ChatEntry{Role: "tool", Content: "file.go", ToolName: "bash", ToolStatus: "success", ToolID: "t1"})

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

	c.AddEntry(ChatEntry{Role: "system", Content: "Welcome to ripcode."})
	view := c.View()
	assert.Contains(t, view, "Welcome to ripcode.")
	assert.Contains(t, view, "~")
}

func TestChat_Clear(t *testing.T) {
	c := NewChat()
	c.SetSize(80, 20)

	c.AddEntry(ChatEntry{Role: "user", Content: "hello"})
	c.Clear()

	view := c.View()
	assert.NotContains(t, view, "hello")
}

// --- Input tests ---

func TestInput_Value(t *testing.T) {
	i := NewInput()
	assert.Equal(t, "", i.Value())
}

func TestInput_Reset(t *testing.T) {
	i := NewInput()
	i.value = []string{"some text"}
	i.cursorX = 5
	i.Reset()
	assert.Equal(t, "", i.Value())
	assert.Equal(t, 0, i.cursorX)
}

func TestInput_View_HasAccentBorder(t *testing.T) {
	i := NewInput()
	i.SetSize(80, 6)
	view := i.View()
	assert.Contains(t, view, "┃", "input should render left accent border")
	assert.Contains(t, view, "╹", "input should render bottom cap")
}

func TestInput_View_ShowsPlaceholder(t *testing.T) {
	i := NewInput()
	i.SetSize(80, 6)
	view := i.View()
	assert.Contains(t, view, "What do you want to do?", "empty input shows placeholder")
}

func TestInput_View_HidesPlaceholderWhenTyping(t *testing.T) {
	i := NewInput()
	i.SetSize(80, 6)
	i.value = []string{"hello world"}
	i.cursorX = 11
	view := i.View()
	assert.NotContains(t, view, "What do you want to do?")
	assert.Contains(t, view, "hello world")
}

func TestInput_View_ShowsBadge(t *testing.T) {
	i := NewInput()
	i.SetSize(80, 6)
	i.SetMode("build")
	i.SetModel("glm-5")
	view := i.View()
	assert.Contains(t, view, "▣")
	assert.Contains(t, view, "Build")
	assert.Contains(t, view, "glm-5")
}

func TestInput_View_ShowsHints(t *testing.T) {
	i := NewInput()
	i.SetSize(80, 6)
	view := i.View()
	assert.Contains(t, view, "Enter send")
	assert.Contains(t, view, "Shift+Enter newline")
}

func TestInput_SetMode(t *testing.T) {
	i := NewInput()
	i.SetMode("plan")
	assert.Equal(t, "plan", i.mode)
}

func TestInput_SetModel(t *testing.T) {
	i := NewInput()
	i.SetModel("test-model")
	assert.Equal(t, "test-model", i.model)
}

// --- StatusBar tests ---

func TestStatusBar_View(t *testing.T) {
	sb := NewStatusBar()
	sb.SetSize(80)
	sb.SetModel("test-model")
	sb.SetMode("build")
	sb.SetTokens(1500)

	view := sb.View()
	assert.Contains(t, view, "ripcode")
	assert.Contains(t, view, "build")
	assert.Contains(t, view, "test-model")
	assert.Contains(t, view, "1.5K")
}

func TestStatusBar_Spinning(t *testing.T) {
	sb := NewStatusBar()
	sb.SetSize(80)
	sb.SetSpinning(true)

	view := sb.View()
	assert.Contains(t, view, "●")
}

func TestStatusBar_ShowsHotkeys(t *testing.T) {
	sb := NewStatusBar()
	sb.SetSize(120)

	view := sb.View()
	assert.Contains(t, view, "^C quit")
	assert.Contains(t, view, "Esc cancel")
	assert.Contains(t, view, "^L clear")
}

func TestStatusBar_HidesHotkeys_WhenNarrow(t *testing.T) {
	sb := NewStatusBar()
	sb.SetSize(30) // too narrow for hotkeys

	view := sb.View()
	assert.Contains(t, view, "ripcode")
	assert.NotContains(t, view, "^C quit")
}

// --- ToolPanel tests ---

func TestToolPanel_AddEvent(t *testing.T) {
	tp := NewToolPanel()
	tp.SetSize(80)

	tp.AddEvent(agent.ToolEvent{
		Name:   "bash",
		Output: "file1.go\nfile2.go",
	})

	view := tp.View()
	assert.Contains(t, view, "bash")
}

func TestToolPanel_ErrorEvent(t *testing.T) {
	tp := NewToolPanel()
	tp.SetSize(80)

	tp.AddEvent(agent.ToolEvent{
		Name:  "bash",
		Error: "command not found",
	})

	view := tp.View()
	assert.Contains(t, view, "command not found")
}

func TestToolPanel_Clear(t *testing.T) {
	tp := NewToolPanel()
	tp.AddEvent(agent.ToolEvent{Name: "bash", Output: "ok"})
	tp.Clear()

	view := tp.View()
	assert.Empty(t, view)
}
