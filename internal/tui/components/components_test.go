package components

import (
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
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

// --- EditHistory tests ---

func TestEditHistory_Push(t *testing.T) {
	h := NewEditHistory(10)
	h.Push(EditState{Value: "a", CursorX: 1})
	assert.False(t, h.CanUndo(), "single state has nothing to undo to")

	h.Push(EditState{Value: "ab", CursorX: 2})
	assert.True(t, h.CanUndo(), "two states means we can undo")
}

func TestEditHistory_Undo_RestoresPreviousState(t *testing.T) {
	h := NewEditHistory(10)
	h.Push(EditState{Value: "a", CursorX: 1})
	h.Push(EditState{Value: "ab", CursorX: 2})
	s := h.Undo()
	assert.NotNil(t, s)
	assert.Equal(t, "a", s.Value)
	assert.Equal(t, 1, s.CursorX)
}

func TestEditHistory_Redo_RestoresNextState(t *testing.T) {
	h := NewEditHistory(10)
	h.Push(EditState{Value: "a", CursorX: 1})
	h.Push(EditState{Value: "ab", CursorX: 2})
	h.Undo()
	s := h.Redo()
	assert.NotNil(t, s)
	assert.Equal(t, "ab", s.Value)
	assert.Equal(t, 2, s.CursorX)
}

func TestEditHistory_Undo_AtStart_ReturnsNil(t *testing.T) {
	h := NewEditHistory(10)
	assert.Nil(t, h.Undo())

	h.Push(EditState{Value: "a"})
	h.Undo()
	assert.Nil(t, h.Undo())
}

func TestEditHistory_Redo_AtEnd_ReturnsNil(t *testing.T) {
	h := NewEditHistory(10)
	assert.Nil(t, h.Redo())

	h.Push(EditState{Value: "a"})
	assert.Nil(t, h.Redo())
}

func TestEditHistory_Push_ClearsFutureOnNewChange(t *testing.T) {
	h := NewEditHistory(10)
	h.Push(EditState{Value: "a"})
	h.Push(EditState{Value: "ab"})
	h.Push(EditState{Value: "abc"})
	h.Undo() // back to "ab"
	h.Undo() // back to "a"

	h.Push(EditState{Value: "ax"})
	assert.Nil(t, h.Redo(), "future should be cleared after push")
	assert.False(t, h.CanRedo())
}

func TestEditHistory_MaxSize_TruncatesOldest(t *testing.T) {
	h := NewEditHistory(3)
	h.Push(EditState{Value: "a"})
	h.Push(EditState{Value: "b"})
	h.Push(EditState{Value: "c"})
	h.Push(EditState{Value: "d"})

	// Should only hold 3 states: b, c, d
	s := h.Undo()
	assert.NotNil(t, s)
	assert.Equal(t, "c", s.Value)
	s = h.Undo()
	assert.NotNil(t, s)
	assert.Equal(t, "b", s.Value)
	assert.Nil(t, h.Undo(), "oldest should have been truncated")
}

func TestEditHistory_MaxSize_PreservesNewest(t *testing.T) {
	h := NewEditHistory(2)
	h.Push(EditState{Value: "a"})
	h.Push(EditState{Value: "b"})
	h.Push(EditState{Value: "c"})

	// Current is "c", can undo to "b", but "a" is gone
	s := h.Undo()
	assert.NotNil(t, s)
	assert.Equal(t, "b", s.Value)
	assert.Nil(t, h.Undo())
}

func TestEditHistory_Clear_ResetsStack(t *testing.T) {
	h := NewEditHistory(10)
	h.Push(EditState{Value: "a"})
	h.Push(EditState{Value: "b"})
	h.Clear()
	assert.False(t, h.CanUndo())
	assert.False(t, h.CanRedo())
	assert.Nil(t, h.Undo())
}

func TestInput_CtrlMinus_Undo(t *testing.T) {
	i := NewInput()
	// Type some text
	i.value = []string{"hello"}
	i.cursorX = 5
	i.pushUndo() // snapshot "hello"
	i.value = []string{"hello world"}
	i.cursorX = 11

	// ctrl+- should restore to "hello"
	i.Update(tea.KeyPressMsg{Code: '-', Mod: tea.ModCtrl})
	assert.Equal(t, "hello", i.Value())
	assert.Equal(t, 5, i.cursorX)
}

func TestInput_CtrlDot_Redo(t *testing.T) {
	i := NewInput()
	i.value = []string{"hello"}
	i.cursorX = 5
	i.pushUndo()
	i.value = []string{"hello world"}
	i.cursorX = 11
	i.pushUndo()

	i.Update(tea.KeyPressMsg{Code: '-', Mod: tea.ModCtrl}) // undo to "hello"
	i.Update(tea.KeyPressMsg{Code: '.', Mod: tea.ModCtrl}) // redo to "hello world"
	assert.Equal(t, "hello world", i.Value())
	assert.Equal(t, 11, i.cursorX)
}

func TestInput_Undo_RestoresTextAndCursor(t *testing.T) {
	i := NewInput()
	i.value = []string{"line1\nline2"}
	i.cursorX = 3
	i.cursorY = 1
	i.pushUndo()

	i.value = []string{"changed"}
	i.cursorX = 7
	i.cursorY = 0
	i.pushUndo()

	i.Update(tea.KeyPressMsg{Code: '-', Mod: tea.ModCtrl})
	assert.Equal(t, "line1\nline2", i.Value())
	assert.Equal(t, 3, i.cursorX)
	assert.Equal(t, 1, i.cursorY)
}

func TestInput_Redo_RestoresTextAndCursor(t *testing.T) {
	i := NewInput()
	i.value = []string{"first"}
	i.cursorX = 5
	i.cursorY = 0
	i.pushUndo()

	i.value = []string{"second"}
	i.cursorX = 6
	i.cursorY = 0
	i.pushUndo()

	i.Update(tea.KeyPressMsg{Code: '-', Mod: tea.ModCtrl})
	i.Update(tea.KeyPressMsg{Code: '.', Mod: tea.ModCtrl})
	assert.Equal(t, "second", i.Value())
	assert.Equal(t, 6, i.cursorX)
}

func TestInput_Typing_PushesToHistory(t *testing.T) {
	i := NewInput()
	// First keystroke after creation should push to history
	i.Update(tea.KeyPressMsg{Text: "h"})
	i.Update(tea.KeyPressMsg{Text: "i"})
	// After typing "hi", undo should restore to before typing started
	i.Update(tea.KeyPressMsg{Code: '-', Mod: tea.ModCtrl})
	assert.Equal(t, "", i.Value(), "undo should restore to before typing started")
}

func TestInput_Undo_EmptyHistory_NoChange(t *testing.T) {
	i := NewInput()
	i.value = []string{"something"}
	i.cursorX = 9
	i.Update(tea.KeyPressMsg{Code: '-', Mod: tea.ModCtrl})
	assert.Equal(t, "something", i.Value(), "undo with empty history should not change value")
}

// --- Advanced Input Keybinding tests ---

func TestInput_CtrlA_MoveToLineStart(t *testing.T) {
	i := NewInput()
	i.value = []string{"hello world"}
	i.cursorX = 7
	i.Update(tea.KeyPressMsg{Code: 'a', Mod: tea.ModCtrl})
	assert.Equal(t, 0, i.cursorX)
}

func TestInput_CtrlE_MoveToLineEnd(t *testing.T) {
	i := NewInput()
	i.value = []string{"hello world"}
	i.cursorX = 3
	i.Update(tea.KeyPressMsg{Code: 'e', Mod: tea.ModCtrl})
	assert.Equal(t, 11, i.cursorX)
}

func TestInput_Home_MoveToLineStart(t *testing.T) {
	i := NewInput()
	i.value = []string{"hello"}
	i.cursorX = 3
	i.Update(tea.KeyPressMsg{Code: tea.KeyHome})
	assert.Equal(t, 0, i.cursorX)
}

func TestInput_End_MoveToLineEnd(t *testing.T) {
	i := NewInput()
	i.value = []string{"hello"}
	i.cursorX = 1
	i.Update(tea.KeyPressMsg{Code: tea.KeyEnd})
	assert.Equal(t, 5, i.cursorX)
}

func TestInput_CtrlLeft_MoveWordLeft(t *testing.T) {
	i := NewInput()
	i.value = []string{"hello world"}
	i.cursorX = 11
	i.Update(tea.KeyPressMsg{Code: tea.KeyLeft, Mod: tea.ModCtrl})
	assert.Equal(t, 6, i.cursorX) // start of "world"
}

func TestInput_CtrlRight_MoveWordRight(t *testing.T) {
	i := NewInput()
	i.value = []string{"hello world"}
	i.cursorX = 0
	i.Update(tea.KeyPressMsg{Code: tea.KeyRight, Mod: tea.ModCtrl})
	assert.Equal(t, 5, i.cursorX) // end of "hello"
}

func TestInput_AltB_MoveWordLeft(t *testing.T) {
	i := NewInput()
	i.value = []string{"hello world"}
	i.cursorX = 11
	i.Update(tea.KeyPressMsg{Code: 'b', Mod: tea.ModAlt})
	assert.Equal(t, 6, i.cursorX)
}

func TestInput_AltF_MoveWordRight(t *testing.T) {
	i := NewInput()
	i.value = []string{"hello world"}
	i.cursorX = 0
	i.Update(tea.KeyPressMsg{Code: 'f', Mod: tea.ModAlt})
	assert.Equal(t, 5, i.cursorX)
}

func TestInput_CtrlU_DeleteToLineStart(t *testing.T) {
	i := NewInput()
	i.value = []string{"hello world"}
	i.cursorX = 5
	i.Update(tea.KeyPressMsg{Code: 'u', Mod: tea.ModCtrl})
	assert.Equal(t, " world", i.Value())
	assert.Equal(t, 0, i.cursorX)
}

func TestInput_CtrlK_DeleteToLineEnd(t *testing.T) {
	i := NewInput()
	i.value = []string{"hello world"}
	i.cursorX = 5
	i.Update(tea.KeyPressMsg{Code: 'k', Mod: tea.ModCtrl})
	assert.Equal(t, "hello", i.Value())
	assert.Equal(t, 5, i.cursorX)
}

func TestInput_CtrlW_DeleteWordLeft(t *testing.T) {
	i := NewInput()
	i.value = []string{"hello world"}
	i.cursorX = 11
	i.Update(tea.KeyPressMsg{Code: 'w', Mod: tea.ModCtrl})
	assert.Equal(t, "hello ", i.Value())
	assert.Equal(t, 6, i.cursorX)
}

func TestInput_AltD_DeleteWordRight(t *testing.T) {
	i := NewInput()
	i.value = []string{"hello world"}
	i.cursorX = 6
	i.Update(tea.KeyPressMsg{Code: 'd', Mod: tea.ModAlt})
	assert.Equal(t, "hello ", i.Value())
	assert.Equal(t, 6, i.cursorX)
}

func TestInput_CtrlD_DeleteCharRight(t *testing.T) {
	i := NewInput()
	i.value = []string{"hello"}
	i.cursorX = 2
	i.Update(tea.KeyPressMsg{Code: 'd', Mod: tea.ModCtrl})
	assert.Equal(t, "helo", i.Value())
	assert.Equal(t, 2, i.cursorX)
}

func TestInput_CtrlD_AtEndOfLine_DoesNothing(t *testing.T) {
	i := NewInput()
	i.value = []string{"hello"}
	i.cursorX = 5
	i.Update(tea.KeyPressMsg{Code: 'd', Mod: tea.ModCtrl})
	assert.Equal(t, "hello", i.Value())
	assert.Equal(t, 5, i.cursorX)
}

func TestInput_CtrlLeft_AtInputStart_StaysAtStart(t *testing.T) {
	i := NewInput()
	i.value = []string{"hello"}
	i.cursorX = 0
	i.Update(tea.KeyPressMsg{Code: tea.KeyLeft, Mod: tea.ModCtrl})
	assert.Equal(t, 0, i.cursorX)
}

func TestInput_CtrlRight_AtInputEnd_StaysAtEnd(t *testing.T) {
	i := NewInput()
	i.value = []string{"hello"}
	i.cursorX = 5
	i.Update(tea.KeyPressMsg{Code: tea.KeyRight, Mod: tea.ModCtrl})
	assert.Equal(t, 5, i.cursorX)
}

func TestInput_CtrlW_OnEmptyInput_DoesNothing(t *testing.T) {
	i := NewInput()
	i.Update(tea.KeyPressMsg{Code: 'w', Mod: tea.ModCtrl})
	assert.Equal(t, "", i.Value())
	assert.Equal(t, 0, i.cursorX)
}

func TestInput_CtrlA_OnEmptyLine_StaysAtZero(t *testing.T) {
	i := NewInput()
	i.Update(tea.KeyPressMsg{Code: 'a', Mod: tea.ModCtrl})
	assert.Equal(t, 0, i.cursorX)
}

func TestInput_WordMotion_SkipsPunctuation(t *testing.T) {
	i := NewInput()
	i.value = []string{"hello.world foo"}
	i.cursorX = 0

	// ctrl+right should skip to end of "hello" (stops at boundary)
	i.Update(tea.KeyPressMsg{Code: tea.KeyRight, Mod: tea.ModCtrl})
	assert.Equal(t, 5, i.cursorX) // end of "hello"

	// Again — skip past punctuation to end of "world"
	i.Update(tea.KeyPressMsg{Code: tea.KeyRight, Mod: tea.ModCtrl})
	assert.Equal(t, 11, i.cursorX) // end of "world"
}

// --- PromptHistory tests ---

func TestPromptHistory_Push(t *testing.T) {
	h := NewPromptHistory(10)
	h.Push("first")
	h.Push("second")
	p, ok := h.Previous()
	assert.True(t, ok)
	assert.Equal(t, "second", p)
}

func TestPromptHistory_Previous_ReturnsPreviousPrompt(t *testing.T) {
	h := NewPromptHistory(10)
	h.Push("alpha")
	h.Push("beta")
	p, _ := h.Previous()
	assert.Equal(t, "beta", p)
	p, _ = h.Previous()
	assert.Equal(t, "alpha", p)
}

func TestPromptHistory_Next_ReturnsNextPrompt(t *testing.T) {
	h := NewPromptHistory(10)
	h.Push("alpha")
	h.Push("beta")
	h.Previous() // beta
	h.Previous() // alpha
	p, ok := h.Next()
	assert.True(t, ok)
	assert.Equal(t, "beta", p)
}

func TestPromptHistory_Previous_AtOldest_ReturnsOldest(t *testing.T) {
	h := NewPromptHistory(10)
	h.Push("only")
	h.Previous()
	p, ok := h.Previous()
	assert.False(t, ok)
	assert.Equal(t, "", p)
}

func TestPromptHistory_Next_AtNewest_ReturnsDraft(t *testing.T) {
	h := NewPromptHistory(10)
	h.Push("old")
	h.SaveDraft("current typing")
	h.Previous() // old
	p, ok := h.Next()
	assert.True(t, ok)
	assert.Equal(t, "current typing", p)
}

func TestPromptHistory_SaveDraft_RestoresDraft(t *testing.T) {
	h := NewPromptHistory(10)
	h.Push("first")
	h.Push("second")
	h.SaveDraft("my draft")
	h.Previous()      // second
	h.Previous()      // first
	h.Next()          // second
	p, ok := h.Next() // draft
	assert.True(t, ok)
	assert.Equal(t, "my draft", p)
}

func TestPromptHistory_Empty_PreviousReturnsEmpty(t *testing.T) {
	h := NewPromptHistory(10)
	p, ok := h.Previous()
	assert.False(t, ok)
	assert.Equal(t, "", p)
}

func TestPromptHistory_ResetPosition_AfterPush(t *testing.T) {
	h := NewPromptHistory(10)
	h.Push("first")
	h.Previous() // navigate to "first"
	h.Push("second")
	// After push, position resets to newest
	assert.True(t, h.AtNewest())
}

func TestPromptHistory_MaxSize_EvictsOldest(t *testing.T) {
	h := NewPromptHistory(2)
	h.Push("a")
	h.Push("b")
	h.Push("c")
	// Should only hold "b" and "c"
	p, _ := h.Previous()
	assert.Equal(t, "c", p)
	p, _ = h.Previous()
	assert.Equal(t, "b", p)
	_, ok := h.Previous()
	assert.False(t, ok, "oldest should have been evicted")
}

func TestPromptHistory_Push_DuplicateConsecutive_Deduplicates(t *testing.T) {
	h := NewPromptHistory(10)
	h.Push("same")
	h.Push("same")
	h.Push("same")
	p, _ := h.Previous()
	assert.Equal(t, "same", p)
	_, ok := h.Previous()
	assert.False(t, ok, "consecutive duplicates should be deduplicated")
}

func TestPromptHistory_ModifyRecalled_DiscardOnNavigate(t *testing.T) {
	h := NewPromptHistory(10)
	h.Push("first")
	h.Push("second")
	h.Previous() // "second"
	// User modifies the recalled prompt (not saved to history)
	// Navigate further back
	p, ok := h.Previous()
	assert.True(t, ok)
	assert.Equal(t, "first", p, "modification should be discarded")
}

// --- Toast tests ---

func TestToast_New_InfoVariant(t *testing.T) {
	tm := NewToastManager()
	tm.Show("hello", ToastInfo, 3*time.Second)
	assert.NotNil(t, tm.Current())
	assert.Equal(t, ToastInfo, tm.Current().Variant)
}

func TestToast_View_ShowsMessage(t *testing.T) {
	tm := NewToastManager()
	tm.SetWidth(80)
	tm.Show("Operation complete", ToastInfo, 3*time.Second)
	view := tm.View()
	assert.Contains(t, view, "Operation complete")
}

func TestToast_View_InfoStyle(t *testing.T) {
	tm := NewToastManager()
	tm.SetWidth(80)
	tm.Show("info msg", ToastInfo, 3*time.Second)
	view := tm.View()
	assert.NotEmpty(t, view)
	assert.Contains(t, view, "info msg")
}

func TestToast_View_SuccessStyle(t *testing.T) {
	tm := NewToastManager()
	tm.SetWidth(80)
	tm.Show("success msg", ToastSuccess, 3*time.Second)
	view := tm.View()
	assert.Contains(t, view, "success msg")
}

func TestToast_View_WarningStyle(t *testing.T) {
	tm := NewToastManager()
	tm.SetWidth(80)
	tm.Show("warning msg", ToastWarning, 3*time.Second)
	view := tm.View()
	assert.Contains(t, view, "warning msg")
}

func TestToast_View_ErrorStyle(t *testing.T) {
	tm := NewToastManager()
	tm.SetWidth(80)
	tm.Show("error msg", ToastError, 3*time.Second)
	view := tm.View()
	assert.Contains(t, view, "error msg")
}

func TestToast_Expired_ReturnsTrueAfterDuration(t *testing.T) {
	toast := Toast{
		Duration: 1 * time.Second,
		Created:  time.Now().Add(-2 * time.Second),
	}
	assert.True(t, toast.Expired())
}

func TestToast_Expired_ReturnsFalseBeforeDuration(t *testing.T) {
	toast := Toast{
		Duration: 5 * time.Second,
		Created:  time.Now(),
	}
	assert.False(t, toast.Expired())
}

func TestToastManager_Show_AddsToast(t *testing.T) {
	tm := NewToastManager()
	id := tm.Show("test", ToastInfo, 3*time.Second)
	assert.Greater(t, id, int64(0))
	assert.NotNil(t, tm.Current())
	assert.Equal(t, "test", tm.Current().Message)
}

func TestToastManager_Show_ReplacesExisting(t *testing.T) {
	tm := NewToastManager()
	id1 := tm.Show("first", ToastInfo, 3*time.Second)
	id2 := tm.Show("second", ToastWarning, 3*time.Second)
	assert.NotEqual(t, id1, id2)
	assert.Equal(t, "second", tm.Current().Message)
}

func TestToastManager_Dismiss_MatchingID(t *testing.T) {
	tm := NewToastManager()
	id := tm.Show("test", ToastInfo, 3*time.Second)
	tm.Dismiss(id)
	assert.Nil(t, tm.Current())
}

func TestToastManager_Dismiss_MismatchedID(t *testing.T) {
	tm := NewToastManager()
	tm.Show("first", ToastInfo, 3*time.Second)
	id2 := tm.Show("second", ToastWarning, 3*time.Second)
	// Dismiss with first ID (stale) should not dismiss second
	tm.Dismiss(id2 - 1)
	assert.NotNil(t, tm.Current(), "mismatched ID should not dismiss")
	assert.Equal(t, "second", tm.Current().Message)
}

func TestToastManager_View_Empty_ReturnsEmpty(t *testing.T) {
	tm := NewToastManager()
	tm.SetWidth(80)
	assert.Equal(t, "", tm.View())
}

// --- Shell Mode Badge tests ---

func TestInput_ShellModeBadge_ShowsShellLabel(t *testing.T) {
	i := NewInput()
	i.SetSize(80, 6)
	i.SetShellMode(true)
	view := i.View()
	assert.Contains(t, view, "Shell")
}

func TestInput_ShellModeBadge_ReturnsToAgentOnExit(t *testing.T) {
	i := NewInput()
	i.SetSize(80, 6)
	i.SetMode("build")
	i.SetShellMode(true)
	i.SetShellMode(false)
	view := i.View()
	assert.Contains(t, view, "Build")
	assert.NotContains(t, view, "Shell")
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
