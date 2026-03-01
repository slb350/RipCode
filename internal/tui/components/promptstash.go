package components

import (
	"crypto/rand"
	"fmt"
	"time"
)

// StashEntry is a saved prompt draft.
type StashEntry struct {
	ID        string
	Content   string
	CreatedAt time.Time
}

// PromptStash stores prompt drafts.
type PromptStash struct {
	entries []StashEntry
}

// NewPromptStash creates a new empty stash.
func NewPromptStash() *PromptStash {
	return &PromptStash{}
}

// Push saves content to the stash and returns the entry ID.
func (s *PromptStash) Push(content string) string {
	id := generateStashID()
	s.entries = append(s.entries, StashEntry{
		ID:        id,
		Content:   content,
		CreatedAt: time.Now(),
	})
	return id
}

// Pop removes and returns the most recent entry.
func (s *PromptStash) Pop() (StashEntry, bool) {
	if len(s.entries) == 0 {
		return StashEntry{}, false
	}
	last := s.entries[len(s.entries)-1]
	s.entries = s.entries[:len(s.entries)-1]
	return last, true
}

// List returns all entries in push order.
func (s *PromptStash) List() []StashEntry {
	out := make([]StashEntry, len(s.entries))
	copy(out, s.entries)
	return out
}

// Delete removes an entry by ID. Returns false if not found.
func (s *PromptStash) Delete(id string) bool {
	for i, e := range s.entries {
		if e.ID == id {
			s.entries = append(s.entries[:i], s.entries[i+1:]...)
			return true
		}
	}
	return false
}

func generateStashID() string {
	b := make([]byte, 4)
	rand.Read(b)
	return fmt.Sprintf("stash-%x", b)
}
