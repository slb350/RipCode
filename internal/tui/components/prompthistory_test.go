package components

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

func TestPromptHistory_PushWithMode_TracksMode(t *testing.T) {
	h := NewPromptHistory(10)
	h.PushWithMode("hello", "normal")
	h.PushWithMode("!ls", "shell")
	items := h.Items()
	require.Len(t, items, 2)
	assert.Equal(t, "normal", items[0].Mode)
	assert.Equal(t, "shell", items[1].Mode)
}

func TestPromptHistory_PushWithMode_NavigatesLikeRegularPush(t *testing.T) {
	h := NewPromptHistory(10)
	h.PushWithMode("alpha", "normal")
	h.PushWithMode("beta", "shell")
	p, ok := h.Previous()
	assert.True(t, ok)
	assert.Equal(t, "beta", p)
	p, ok = h.Previous()
	assert.True(t, ok)
	assert.Equal(t, "alpha", p)
}

func TestPromptHistory_Push_DefaultsToNormalMode(t *testing.T) {
	h := NewPromptHistory(10)
	h.Push("test")
	items := h.Items()
	require.Len(t, items, 1)
	assert.Equal(t, "normal", items[0].Mode)
}

func TestPromptHistory_LoadItems_RestoresState(t *testing.T) {
	h := NewPromptHistory(10)
	h.LoadItems([]HistoryItem{
		{Prompt: "one", Mode: "normal"},
		{Prompt: "two", Mode: "shell"},
	})
	p, ok := h.Previous()
	assert.True(t, ok)
	assert.Equal(t, "two", p)
	p, ok = h.Previous()
	assert.True(t, ok)
	assert.Equal(t, "one", p)
}

func TestPromptHistory_LoadItems_RespectsMaxSize(t *testing.T) {
	h := NewPromptHistory(2)
	h.LoadItems([]HistoryItem{
		{Prompt: "a", Mode: "normal"},
		{Prompt: "b", Mode: "normal"},
		{Prompt: "c", Mode: "normal"},
	})
	items := h.Items()
	assert.Len(t, items, 2)
	assert.Equal(t, "b", items[0].Prompt)
	assert.Equal(t, "c", items[1].Prompt)
}
