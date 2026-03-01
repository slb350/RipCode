package components

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestPromptStash_Push_AddsEntry(t *testing.T) {
	s := NewPromptStash()
	s.Push("hello world")
	assert.Len(t, s.List(), 1)
}

func TestPromptStash_Pop_ReturnsLatest(t *testing.T) {
	s := NewPromptStash()
	s.Push("first")
	s.Push("second")
	entry, ok := s.Pop()
	assert.True(t, ok)
	assert.Equal(t, "second", entry.Content)
	assert.Len(t, s.List(), 1) // second popped, first remains
}

func TestPromptStash_Pop_Empty_ReturnsFalse(t *testing.T) {
	s := NewPromptStash()
	_, ok := s.Pop()
	assert.False(t, ok)
}

func TestPromptStash_List_ReturnsAll(t *testing.T) {
	s := NewPromptStash()
	s.Push("one")
	s.Push("two")
	s.Push("three")
	assert.Len(t, s.List(), 3)
}

func TestPromptStash_Delete_RemovesEntry(t *testing.T) {
	s := NewPromptStash()
	id := s.Push("hello")
	assert.True(t, s.Delete(id))
	assert.Empty(t, s.List())
}

func TestPromptStash_Delete_Unknown_ReturnsFalse(t *testing.T) {
	s := NewPromptStash()
	assert.False(t, s.Delete("nonexistent"))
}

func TestPromptStash_Push_SetsTimestamp(t *testing.T) {
	s := NewPromptStash()
	before := time.Now()
	s.Push("test")
	entries := s.List()
	assert.False(t, entries[0].CreatedAt.Before(before))
}
