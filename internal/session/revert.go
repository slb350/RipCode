package session

import (
	"fmt"

	"github.com/stephenbrandon/ripcode/internal/provider"
)

// revertPoint stores messages that were removed during a revert.
type revertPoint struct {
	messages   []MessageRecord
	promptText string
}

// Revert removes the last user message and everything after it.
// Returns the user's prompt text for input restoration.
// Pushes removed messages onto the redo stack (cleared by AddUser).
// Returns an error if no user message exists to revert.
func (s *Session) Revert() (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	// Find the last user message
	lastUser := -1
	for i := len(s.messages) - 1; i >= 0; i-- {
		if s.messages[i].Message.Role == provider.RoleUser {
			lastUser = i
			break
		}
	}
	if lastUser < 0 {
		return "", fmt.Errorf("no user message to revert")
	}

	// Snapshot removed messages
	removed := make([]MessageRecord, len(s.messages)-lastUser)
	copy(removed, s.messages[lastUser:])

	prompt := s.messages[lastUser].Message.Content

	// Push to redo stack
	s.redoStack = append(s.redoStack, revertPoint{
		messages:   removed,
		promptText: prompt,
	})

	// Truncate
	s.messages = s.messages[:lastUser]

	return prompt, nil
}

// Unrevert restores the most recently reverted messages.
func (s *Session) Unrevert() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.redoStack) == 0 {
		return fmt.Errorf("nothing to redo")
	}

	last := s.redoStack[len(s.redoStack)-1]
	s.redoStack = s.redoStack[:len(s.redoStack)-1]

	s.messages = append(s.messages, last.messages...)
	return nil
}

// CanUndo returns true if there are user messages that can be reverted.
func (s *Session) CanUndo() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, rec := range s.messages {
		if rec.Message.Role == provider.RoleUser {
			return true
		}
	}
	return false
}

// CanRedo returns true if there are reverted messages that can be restored.
func (s *Session) CanRedo() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.redoStack) > 0
}
