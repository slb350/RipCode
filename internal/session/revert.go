package session

import "fmt"

// revertPoint stores messages that were removed during a revert.
type revertPoint struct {
	messages   []MessageRecord
	promptText string
}

// Revert removes the last user message and everything after it.
// Returns the user's prompt text for input restoration.
func (s *Session) Revert() (string, error) {
	// Find the last user message
	lastUser := -1
	for i := len(s.Messages) - 1; i >= 0; i-- {
		if s.Messages[i].Message.Role == "user" {
			lastUser = i
			break
		}
	}
	if lastUser < 0 {
		return "", fmt.Errorf("no user message to revert")
	}

	// Snapshot removed messages
	removed := make([]MessageRecord, len(s.Messages)-lastUser)
	copy(removed, s.Messages[lastUser:])

	prompt := s.Messages[lastUser].Message.Content

	// Push to redo stack
	s.redoStack = append(s.redoStack, revertPoint{
		messages:   removed,
		promptText: prompt,
	})

	// Truncate
	s.Messages = s.Messages[:lastUser]

	return prompt, nil
}

// Unrevert restores the most recently reverted messages.
func (s *Session) Unrevert() error {
	if len(s.redoStack) == 0 {
		return fmt.Errorf("nothing to redo")
	}

	last := s.redoStack[len(s.redoStack)-1]
	s.redoStack = s.redoStack[:len(s.redoStack)-1]

	s.Messages = append(s.Messages, last.messages...)
	return nil
}

// CanUndo returns true if there are user messages that can be reverted.
func (s *Session) CanUndo() bool {
	for _, rec := range s.Messages {
		if rec.Message.Role == "user" {
			return true
		}
	}
	return false
}

// CanRedo returns true if there are reverted messages that can be restored.
func (s *Session) CanRedo() bool {
	return len(s.redoStack) > 0
}
