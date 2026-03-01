package components

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

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
