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
	Messages     []provider.Message
	WorkDir      string
	CreatedAt    time.Time
	Tokens       TokenCount
	systemPrompt string
}

// TokenCount tracks cumulative token usage.
type TokenCount struct {
	Input  int
	Output int
}

// New creates a new session with a random ID.
func New(workDir string) *Session {
	return &Session{
		ID:        generateID(),
		WorkDir:   workDir,
		CreatedAt: time.Now(),
	}
}

// SetSystemPrompt sets the system message prepended to history.
func (s *Session) SetSystemPrompt(prompt string) {
	s.systemPrompt = prompt
}

// AddUser appends a user message.
func (s *Session) AddUser(content string) {
	s.Messages = append(s.Messages, provider.Message{
		Role:    "user",
		Content: content,
	})
}

// AddAssistant appends an assistant message with optional tool calls.
func (s *Session) AddAssistant(content string, toolCalls []provider.ToolCall) {
	s.Messages = append(s.Messages, provider.Message{
		Role:      "assistant",
		Content:   content,
		ToolCalls: toolCalls,
	})
}

// AddToolResult appends a tool result message.
func (s *Session) AddToolResult(callID, content string) {
	s.Messages = append(s.Messages, provider.Message{
		Role:       "tool",
		Content:    content,
		ToolCallID: callID,
	})
}

// History returns a copy of the full message history for sending to a
// provider, including the system prompt if set.
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

	copy(msgs[offset:], s.Messages)
	return msgs
}

// Reset clears messages, tokens, and generates a new session ID,
// preserving the working directory and system prompt.
func (s *Session) Reset() {
	s.ID = generateID()
	s.Messages = nil
	s.Tokens = TokenCount{}
	s.CreatedAt = time.Now()
}

// AddTokens accumulates token usage.
func (s *Session) AddTokens(input, output int) {
	s.Tokens.Input += input
	s.Tokens.Output += output
}

func generateID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return fmt.Sprintf("sess-%x", b)
}
