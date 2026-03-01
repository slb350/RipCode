package session

import (
	"crypto/rand"
	"fmt"
	"time"

	"github.com/stephenbrandon/ripcode/internal/provider"
)

// Session manages a conversation's message history and metadata.
type Session struct {
	ID           string
	Title        string
	Messages     []MessageRecord
	WorkDir      string
	CreatedAt    time.Time
	UpdatedAt    time.Time
	Tokens       TokenCount
	systemPrompt string
	redoStack    []revertPoint
}

// TokenCount tracks cumulative token usage.
type TokenCount struct {
	Input  int
	Output int
}

// New creates a new session with a random ID.
func New(workDir string) *Session {
	now := time.Now()
	return &Session{
		ID:        generateID(),
		WorkDir:   workDir,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// SetSystemPrompt sets the system message prepended to history.
func (s *Session) SetSystemPrompt(prompt string) {
	s.systemPrompt = prompt
}

// AddUser appends a user message and returns the record.
// Adding a new user message clears the redo stack.
func (s *Session) AddUser(content string) *MessageRecord {
	s.redoStack = nil // new message invalidates redo history
	rec := MessageRecord{
		ID: generateMessageID(),
		Message: provider.Message{
			Role:    "user",
			Content: content,
		},
		CreatedAt: time.Now(),
	}
	s.Messages = append(s.Messages, rec)
	s.UpdatedAt = time.Now()
	return &s.Messages[len(s.Messages)-1]
}

// AddAssistant appends an assistant message with optional tool calls and metadata.
func (s *Session) AddAssistant(content string, toolCalls []provider.ToolCall, meta *AssistantMeta) *MessageRecord {
	rec := MessageRecord{
		ID: generateMessageID(),
		Message: provider.Message{
			Role:      "assistant",
			Content:   content,
			ToolCalls: toolCalls,
		},
		CreatedAt: time.Now(),
		Meta:      meta,
	}
	s.Messages = append(s.Messages, rec)
	s.UpdatedAt = time.Now()
	return &s.Messages[len(s.Messages)-1]
}

// AddToolResult appends a tool result message.
func (s *Session) AddToolResult(callID, content string) *MessageRecord {
	rec := MessageRecord{
		ID: generateMessageID(),
		Message: provider.Message{
			Role:       "tool",
			Content:    content,
			ToolCallID: callID,
		},
		CreatedAt: time.Now(),
	}
	s.Messages = append(s.Messages, rec)
	s.UpdatedAt = time.Now()
	return &s.Messages[len(s.Messages)-1]
}

// History returns a copy of the full message history for sending to a
// provider, including the system prompt if set. Extracts provider.Message
// from each MessageRecord.
func (s *Session) History() []provider.Message {
	offset := 0
	if s.systemPrompt != "" {
		offset = 1
	}

	msgs := make([]provider.Message, offset+len(s.Messages))

	if s.systemPrompt != "" {
		msgs[0] = provider.Message{
			Role:    "system",
			Content: s.systemPrompt,
		}
	}

	for i, rec := range s.Messages {
		msgs[offset+i] = rec.Message
	}
	return msgs
}

// Records returns all message records.
func (s *Session) Records() []MessageRecord {
	out := make([]MessageRecord, len(s.Messages))
	copy(out, s.Messages)
	return out
}

// RecordByID finds a message record by ID, or returns nil if not found.
func (s *Session) RecordByID(id string) *MessageRecord {
	for i := range s.Messages {
		if s.Messages[i].ID == id {
			return &s.Messages[i]
		}
	}
	return nil
}

// MessageCount returns the count of messages matching the given role.
// If role is empty, returns the total count.
func (s *Session) MessageCount(role string) int {
	if role == "" {
		return len(s.Messages)
	}
	count := 0
	for _, rec := range s.Messages {
		if rec.Message.Role == role {
			count++
		}
	}
	return count
}

// Reset clears messages, tokens, and generates a new session ID,
// preserving the working directory and system prompt.
func (s *Session) Reset() {
	s.ID = generateID()
	s.Title = ""
	s.Messages = nil
	s.redoStack = nil
	s.Tokens = TokenCount{}
	now := time.Now()
	s.CreatedAt = now
	s.UpdatedAt = now
}

// AddTokens accumulates token usage.
func (s *Session) AddTokens(input, output int) {
	s.Tokens.Input += input
	s.Tokens.Output += output
}

func generatePrefixedID(prefix string, byteLen int) string {
	b := make([]byte, byteLen)
	rand.Read(b)
	return fmt.Sprintf("%s-%x", prefix, b)
}

func generateID() string {
	return generatePrefixedID("sess", 8)
}
