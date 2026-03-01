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

func TestInput_CursorOffsetAndSetCursorOffset(t *testing.T) {
	i := NewInput()
	i.SetValue("hello\nworld")

	i.SetCursorOffset(7) // w in world
	assert.Equal(t, 7, i.CursorOffset())
	assert.Equal(t, "hello\nworld", i.Value())
}

func TestInput_ReplaceRange(t *testing.T) {
	i := NewInput()
	i.SetValue("/mod")

	i.ReplaceRange(0, 4, "/models")
	assert.Equal(t, "/models", i.Value())
	assert.Equal(t, len("/models"), i.CursorOffset())
}

func TestInput_CursorOffset_Unicode(t *testing.T) {
	i := NewInput()
	i.SetValue("héllo")

	// "héllo" has 5 runes, cursor should be at end = 5
	assert.Equal(t, 5, i.CursorOffset())

	i.SetCursorOffset(2) // after "hé"
	assert.Equal(t, 2, i.CursorOffset())

	i.ReplaceRange(0, 2, "HE")
	assert.Equal(t, "HEllo", i.Value())
	assert.Equal(t, 2, i.CursorOffset())
}

func TestInput_ReplaceRange_Emoji(t *testing.T) {
	i := NewInput()
	i.SetValue("@📁test ")

	// "@📁test " = 7 runes: @, 📁, t, e, s, t, ' '
	// Replace [0,7) = entire string, result is just the replacement.
	i.ReplaceRange(0, 7, "@readme.md ")
	assert.Equal(t, "@readme.md ", i.Value())
}

// --- Home tests ---

func TestHome_RendersLogo(t *testing.T) {
	h := NewHome()
	h.SetSize(80, 30)
	view := h.View()
	assert.Contains(t, view, "██████╗", "home should render ASCII logo")
}

func TestHome_RendersCodeSubtitle(t *testing.T) {
	h := NewHome()
	h.SetSize(80, 30)
	view := h.View()
	assert.Contains(t, view, "code", "home should render 'code' subtitle")
}

func TestHome_RendersInput(t *testing.T) {
	h := NewHome()
	h.SetSize(80, 30)
	view := h.View()
	assert.Contains(t, view, "┃", "home should render input accent border")
	assert.Contains(t, view, "What do you want to do?", "home should render input placeholder")
}

func TestHome_RendersFooter(t *testing.T) {
	h := NewHome()
	h.SetSize(80, 30)
	h.SetWorkDir("/tmp/project")
	h.SetVersion("v0.1.0")
	view := h.View()
	assert.Contains(t, view, "/tmp/project", "home should render workdir in footer")
	assert.Contains(t, view, "v0.1.0", "home should render version in footer")
}

func TestHome_RendersTip(t *testing.T) {
	h := NewHome()
	h.SetSize(80, 30)
	view := h.View()
	assert.Contains(t, view, "●", "home should render tip bullet")
}

func TestHome_Input(t *testing.T) {
	h := NewHome()
	h.SetSize(80, 30)
	// Should have accessible Input
	assert.NotNil(t, h.Input())
}

// --- StatusBar tests ---

func TestStatusBar_View(t *testing.T) {
	sb := NewStatusBar()
	sb.SetSize(80)
	sb.SetTitle("Session sess-1234")
	sb.SetModel("test-model")
	sb.SetMode("build")
	sb.SetTokens(1500)

	view := sb.View()
	assert.Contains(t, view, "# Session sess-1234")
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

func TestStatusBar_NoHotkeys(t *testing.T) {
	sb := NewStatusBar()
	sb.SetSize(120)

	view := sb.View()
	assert.NotContains(t, view, "^C quit")
	assert.NotContains(t, view, "Esc cancel")
}

func TestStatusBar_NarrowStillShowsHeader(t *testing.T) {
	sb := NewStatusBar()
	sb.SetTitle("Session abc")
	sb.SetSize(30) // too narrow for hotkeys

	view := sb.View()
	assert.Contains(t, view, "Session abc")
}

func TestSessionFooter_Default(t *testing.T) {
	f := NewSessionFooter()
	f.SetSize(80)
	f.SetWorkDir("/tmp/project")
	f.SetConnected(true)

	view := f.View()
	assert.Contains(t, view, "/tmp/project")
	assert.Contains(t, view, "/models · /help")
}

func TestSessionFooter_Streaming(t *testing.T) {
	f := NewSessionFooter()
	f.SetSize(80)
	f.SetWorkDir("/tmp/project")
	f.SetConnected(true)
	f.SetStreaming(true)

	view := f.View()
	assert.Contains(t, view, "● Running")
	assert.Contains(t, view, "Esc interrupt")
}

func TestSessionFooter_Disconnected(t *testing.T) {
	f := NewSessionFooter()
	f.SetSize(80)
	f.SetConnected(false)

	view := f.View()
	assert.Contains(t, view, "Set OPENROUTER_API_KEY to connect")
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
