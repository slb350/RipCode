package session

import (
	"crypto/rand"
	"fmt"
	"sync"
	"time"

	"github.com/stephenbrandon/ripcode/internal/provider"
)

// Session manages a conversation's message history and metadata.
type Session struct {
	ID           string
	Title        string
	ParentID     string
	messages     []MessageRecord
	WorkDir      string
	CreatedAt    time.Time
	UpdatedAt    time.Time
	Tokens       TokenCount
	systemPrompt string
	redoStack    []revertPoint
	mu           sync.RWMutex
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
	s.mu.Lock()
	defer s.mu.Unlock()
	s.systemPrompt = prompt
}

// AddUser appends a user message and returns the record.
// Adding a new user message clears the redo stack.
func (s *Session) AddUser(content string) *MessageRecord {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.redoStack = nil // new message invalidates redo history
	rec := MessageRecord{
		ID: generateMessageID(),
		Message: provider.Message{
			Role:    provider.RoleUser,
			Content: content,
		},
		CreatedAt: time.Now(),
	}
	s.messages = append(s.messages, rec)
	s.UpdatedAt = time.Now()
	return &s.messages[len(s.messages)-1]
}

// AddAssistant appends an assistant message with optional tool calls and metadata.
func (s *Session) AddAssistant(content string, toolCalls []provider.ToolCall, meta *AssistantMeta) *MessageRecord {
	s.mu.Lock()
	defer s.mu.Unlock()
	rec := MessageRecord{
		ID: generateMessageID(),
		Message: provider.Message{
			Role:      provider.RoleAssistant,
			Content:   content,
			ToolCalls: toolCalls,
		},
		CreatedAt: time.Now(),
		Meta:      meta,
	}
	s.messages = append(s.messages, rec)
	s.UpdatedAt = time.Now()
	return &s.messages[len(s.messages)-1]
}

// AddToolResult appends a tool result message.
func (s *Session) AddToolResult(callID, content string) *MessageRecord {
	s.mu.Lock()
	defer s.mu.Unlock()
	rec := MessageRecord{
		ID: generateMessageID(),
		Message: provider.Message{
			Role:       provider.RoleTool,
			Content:    content,
			ToolCallID: callID,
		},
		CreatedAt: time.Now(),
	}
	s.messages = append(s.messages, rec)
	s.UpdatedAt = time.Now()
	return &s.messages[len(s.messages)-1]
}

// History returns a copy of the full message history for sending to a
// provider, including the system prompt if set. Extracts provider.Message
// from each MessageRecord.
func (s *Session) History() []provider.Message {
	s.mu.RLock()
	defer s.mu.RUnlock()
	offset := 0
	if s.systemPrompt != "" {
		offset = 1
	}

	msgs := make([]provider.Message, offset+len(s.messages))

	if s.systemPrompt != "" {
		msgs[0] = provider.Message{
			Role:    provider.RoleSystem,
			Content: s.systemPrompt,
		}
	}

	for i, rec := range s.messages {
		msgs[offset+i] = rec.Message
	}
	return msgs
}

// Records returns all message records.
func (s *Session) Records() []MessageRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]MessageRecord, len(s.messages))
	copy(out, s.messages)
	return out
}

// RecordByID finds a message record by ID, or returns nil if not found.
func (s *Session) RecordByID(id string) *MessageRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for i := range s.messages {
		if s.messages[i].ID == id {
			return &s.messages[i]
		}
	}
	return nil
}

// MessageCount returns the count of messages matching the given role.
// If role is empty, returns the total count.
func (s *Session) MessageCount(role provider.Role) int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if role == "" {
		return len(s.messages)
	}
	count := 0
	for _, rec := range s.messages {
		if rec.Message.Role == role {
			count++
		}
	}
	return count
}

// Reset clears messages, tokens, and generates a new session ID,
// preserving the working directory and system prompt.
func (s *Session) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ID = generateID()
	s.Title = ""
	s.messages = nil
	s.redoStack = nil
	s.Tokens = TokenCount{}
	now := time.Now()
	s.CreatedAt = now
	s.UpdatedAt = now
}

// Fork creates a new session containing messages up to and including the
// given index. The new session gets a fresh ID, links back via ParentID,
// and preserves the system prompt. Message records are deep-copied with
// new IDs so the fork is fully independent.
func (s *Session) Fork(upToIndex int) (*Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if len(s.messages) == 0 {
		return nil, fmt.Errorf("cannot fork empty session")
	}
	if upToIndex < 0 || upToIndex >= len(s.messages) {
		return nil, fmt.Errorf("fork index %d out of range [0, %d)", upToIndex, len(s.messages))
	}

	now := time.Now()
	forked := &Session{
		ID:        generateID(),
		ParentID:  s.ID,
		WorkDir:   s.WorkDir,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if s.systemPrompt != "" {
		forked.SetSystemPrompt(s.systemPrompt)
	}

	src := s.messages[:upToIndex+1]
	forked.messages = make([]MessageRecord, len(src))
	for i, rec := range src {
		forked.messages[i] = MessageRecord{
			ID:        generateMessageID(),
			Message:   rec.Message,
			CreatedAt: rec.CreatedAt,
		}
		// Deep-copy tool calls slice
		if len(rec.Message.ToolCalls) > 0 {
			tc := make([]provider.ToolCall, len(rec.Message.ToolCalls))
			copy(tc, rec.Message.ToolCalls)
			forked.messages[i].Message.ToolCalls = tc
		}
		// Deep-copy meta
		if rec.Meta != nil {
			meta := *rec.Meta
			forked.messages[i].Meta = &meta
		}
	}

	return forked, nil
}

// Len returns the number of messages in the session.
func (s *Session) Len() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.messages)
}

// ClearMessages removes all messages and clears the redo stack.
func (s *Session) ClearMessages() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.messages = nil
	s.redoStack = nil
}

// LoadRecord adds a pre-built message record during session loading from disk.
func (s *Session) LoadRecord(rec MessageRecord) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.messages = append(s.messages, rec)
}

// AddTokens accumulates token usage.
func (s *Session) AddTokens(input, output int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Tokens.Input += input
	s.Tokens.Output += output
}

// generatePrefixedID generates a random ID with the given prefix. Panics if
// crypto/rand fails, which indicates a catastrophic system issue (no entropy source).
func generatePrefixedID(prefix string, byteLen int) string {
	b := make([]byte, byteLen)
	if _, err := rand.Read(b); err != nil {
		panic(fmt.Sprintf("crypto/rand.Read failed: %v", err))
	}
	return fmt.Sprintf("%s-%x", prefix, b)
}

func generateID() string {
	return generatePrefixedID("sess", 8)
}
