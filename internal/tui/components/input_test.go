package components

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
)

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

// --- Input undo/redo integration tests ---

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

// --- Advanced Input keybinding tests ---

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
