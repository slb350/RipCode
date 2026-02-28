package session

import (
	"testing"

	"github.com/stephenbrandon/ripcode/internal/provider"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	s := New("/tmp/test")

	assert.NotEmpty(t, s.ID)
	assert.Equal(t, "/tmp/test", s.WorkDir)
	assert.NotZero(t, s.CreatedAt)
	assert.Empty(t, s.Messages)
}

func TestAddUser(t *testing.T) {
	s := New("/tmp")
	s.AddUser("hello")

	require.Len(t, s.Messages, 1)
	assert.Equal(t, "user", s.Messages[0].Role)
	assert.Equal(t, "hello", s.Messages[0].Content)
}

func TestAddAssistant(t *testing.T) {
	s := New("/tmp")
	s.AddAssistant("I'll help", nil)

	require.Len(t, s.Messages, 1)
	assert.Equal(t, "assistant", s.Messages[0].Role)
	assert.Equal(t, "I'll help", s.Messages[0].Content)
}

func TestAddAssistantWithToolCalls(t *testing.T) {
	s := New("/tmp")
	calls := []provider.ToolCall{
		{ID: "call_1", Name: "bash", Args: `{"command":"ls"}`},
	}
	s.AddAssistant("Let me check.", calls)

	require.Len(t, s.Messages, 1)
	assert.Equal(t, "assistant", s.Messages[0].Role)
	assert.Len(t, s.Messages[0].ToolCalls, 1)
	assert.Equal(t, "bash", s.Messages[0].ToolCalls[0].Name)
}

func TestAddToolResult(t *testing.T) {
	s := New("/tmp")
	s.AddToolResult("call_1", "file1.go\nfile2.go")

	require.Len(t, s.Messages, 1)
	assert.Equal(t, "tool", s.Messages[0].Role)
	assert.Equal(t, "call_1", s.Messages[0].ToolCallID)
	assert.Equal(t, "file1.go\nfile2.go", s.Messages[0].Content)
}

func TestHistory_IncludesAllMessages(t *testing.T) {
	s := New("/tmp")
	s.AddUser("list files")
	s.AddAssistant("Let me check.", []provider.ToolCall{
		{ID: "call_1", Name: "bash", Args: `{"command":"ls"}`},
	})
	s.AddToolResult("call_1", "a.go\nb.go")
	s.AddAssistant("Here are your files: a.go and b.go", nil)

	history := s.History()
	require.Len(t, history, 4)
	assert.Equal(t, "user", history[0].Role)
	assert.Equal(t, "assistant", history[1].Role)
	assert.Equal(t, "tool", history[2].Role)
	assert.Equal(t, "assistant", history[3].Role)
}

func TestAddTokens(t *testing.T) {
	s := New("/tmp")
	s.AddTokens(100, 50)
	s.AddTokens(200, 75)

	assert.Equal(t, 300, s.Tokens.Input)
	assert.Equal(t, 125, s.Tokens.Output)
}

func TestHistory_ReturnsCopy(t *testing.T) {
	s := New("/tmp")
	s.AddUser("hello")

	h1 := s.History()
	h1[0].Content = "modified"

	h2 := s.History()
	assert.Equal(t, "hello", h2[0].Content, "History should return a copy, not a reference")
}

func TestSetSystemPrompt(t *testing.T) {
	s := New("/tmp")
	s.SetSystemPrompt("You are a helpful assistant.")
	s.AddUser("hello")

	history := s.History()
	require.Len(t, history, 2)
	assert.Equal(t, "system", history[0].Role)
	assert.Equal(t, "You are a helpful assistant.", history[0].Content)
	assert.Equal(t, "user", history[1].Role)
}
