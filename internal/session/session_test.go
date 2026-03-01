package session

import (
	"testing"
	"time"

	"github.com/stephenbrandon/ripcode/internal/provider"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	s := New("/tmp/test")

	assert.NotEmpty(t, s.ID)
	assert.Equal(t, "/tmp/test", s.WorkDir)
	assert.NotZero(t, s.CreatedAt)
	assert.NotZero(t, s.UpdatedAt)
	assert.Empty(t, s.Messages)
	assert.Empty(t, s.Title)
}

func TestAddUser(t *testing.T) {
	s := New("/tmp")
	rec := s.AddUser("hello")

	require.Len(t, s.Messages, 1)
	assert.Equal(t, "user", s.Messages[0].Message.Role)
	assert.Equal(t, "hello", s.Messages[0].Message.Content)
	assert.NotNil(t, rec)
	assert.Equal(t, "user", rec.Message.Role)
}

func TestAddAssistant(t *testing.T) {
	s := New("/tmp")
	rec := s.AddAssistant("I'll help", nil, nil)

	require.Len(t, s.Messages, 1)
	assert.Equal(t, "assistant", s.Messages[0].Message.Role)
	assert.Equal(t, "I'll help", s.Messages[0].Message.Content)
	assert.NotNil(t, rec)
	assert.Nil(t, rec.Meta)
}

func TestAddAssistantWithToolCalls(t *testing.T) {
	s := New("/tmp")
	calls := []provider.ToolCall{
		{ID: "call_1", Name: "bash", Args: `{"command":"ls"}`},
	}
	rec := s.AddAssistant("Let me check.", calls, nil)

	require.Len(t, s.Messages, 1)
	assert.Equal(t, "assistant", s.Messages[0].Message.Role)
	assert.Len(t, s.Messages[0].Message.ToolCalls, 1)
	assert.Equal(t, "bash", s.Messages[0].Message.ToolCalls[0].Name)
	assert.NotNil(t, rec)
}

func TestAddToolResult(t *testing.T) {
	s := New("/tmp")
	rec := s.AddToolResult("call_1", "file1.go\nfile2.go")

	require.Len(t, s.Messages, 1)
	assert.Equal(t, "tool", s.Messages[0].Message.Role)
	assert.Equal(t, "call_1", s.Messages[0].Message.ToolCallID)
	assert.Equal(t, "file1.go\nfile2.go", s.Messages[0].Message.Content)
	assert.NotNil(t, rec)
}

func TestHistory_IncludesAllMessages(t *testing.T) {
	s := New("/tmp")
	s.AddUser("list files")
	s.AddAssistant("Let me check.", []provider.ToolCall{
		{ID: "call_1", Name: "bash", Args: `{"command":"ls"}`},
	}, nil)
	s.AddToolResult("call_1", "a.go\nb.go")
	s.AddAssistant("Here are your files: a.go and b.go", nil, nil)

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

// --- New Phase 2 tests ---

func TestMessageRecord_HasID(t *testing.T) {
	s := New("/tmp")
	rec := s.AddUser("test")
	assert.NotEmpty(t, rec.ID)
	assert.Contains(t, rec.ID, "msg-")
}

func TestMessageRecord_HasCreatedAt(t *testing.T) {
	before := time.Now()
	s := New("/tmp")
	rec := s.AddUser("test")
	assert.False(t, rec.CreatedAt.Before(before))
}

func TestMessageRecord_IDsAreUnique(t *testing.T) {
	s := New("/tmp")
	r1 := s.AddUser("a")
	r2 := s.AddUser("b")
	r3 := s.AddAssistant("c", nil, nil)
	assert.NotEqual(t, r1.ID, r2.ID)
	assert.NotEqual(t, r2.ID, r3.ID)
	assert.NotEqual(t, r1.ID, r3.ID)
}

func TestSession_AddUser_ReturnsMessageRecord(t *testing.T) {
	s := New("/tmp")
	rec := s.AddUser("hello")
	assert.Equal(t, "user", rec.Message.Role)
	assert.Equal(t, "hello", rec.Message.Content)
}

func TestSession_AddUser_SetsIDAndTimestamp(t *testing.T) {
	s := New("/tmp")
	rec := s.AddUser("test")
	assert.NotEmpty(t, rec.ID)
	assert.NotZero(t, rec.CreatedAt)
}

func TestSession_AddAssistant_ReturnsMessageRecord(t *testing.T) {
	s := New("/tmp")
	rec := s.AddAssistant("response", nil, nil)
	assert.Equal(t, "assistant", rec.Message.Role)
	assert.Equal(t, "response", rec.Message.Content)
}

func TestSession_AddAssistant_AcceptsMeta(t *testing.T) {
	s := New("/tmp")
	meta := &AssistantMeta{
		Model:        "gpt-4o",
		InputTokens:  100,
		OutputTokens: 50,
		FinishReason: "stop",
	}
	rec := s.AddAssistant("response", nil, meta)
	require.NotNil(t, rec.Meta)
	assert.Equal(t, "gpt-4o", rec.Meta.Model)
	assert.Equal(t, 100, rec.Meta.InputTokens)
	assert.Equal(t, 50, rec.Meta.OutputTokens)
	assert.Equal(t, "stop", rec.Meta.FinishReason)
}

func TestSession_AddAssistant_NilMetaOK(t *testing.T) {
	s := New("/tmp")
	rec := s.AddAssistant("response", nil, nil)
	assert.Nil(t, rec.Meta)
}

func TestSession_AddToolResult_ReturnsMessageRecord(t *testing.T) {
	s := New("/tmp")
	rec := s.AddToolResult("call_1", "output")
	assert.Equal(t, "tool", rec.Message.Role)
	assert.Equal(t, "call_1", rec.Message.ToolCallID)
}

func TestSession_History_ReturnsProviderMessages(t *testing.T) {
	s := New("/tmp")
	s.AddUser("hello")
	s.AddAssistant("world", nil, &AssistantMeta{Model: "test"})

	history := s.History()
	require.Len(t, history, 2)
	assert.Equal(t, "hello", history[0].Content)
	assert.Equal(t, "world", history[1].Content)
}

func TestSession_History_ExcludesMetadata(t *testing.T) {
	s := New("/tmp")
	s.AddAssistant("response", nil, &AssistantMeta{Model: "test", InputTokens: 500})

	history := s.History()
	// provider.Message has no Meta field — it's pure wire format
	require.Len(t, history, 1)
	assert.Equal(t, "assistant", history[0].Role)
	assert.Equal(t, "response", history[0].Content)
}

func TestSession_Title_DefaultsToEmpty(t *testing.T) {
	s := New("/tmp")
	assert.Empty(t, s.Title)
}

func TestSession_Title_Settable(t *testing.T) {
	s := New("/tmp")
	s.Title = "My Session"
	assert.Equal(t, "My Session", s.Title)
}

func TestSession_UpdatedAt_SetOnAddMessage(t *testing.T) {
	s := New("/tmp")
	initial := s.UpdatedAt

	time.Sleep(time.Millisecond)
	s.AddUser("test")
	assert.False(t, s.UpdatedAt.Before(initial))
}

func TestSession_Records_ReturnsAllMessageRecords(t *testing.T) {
	s := New("/tmp")
	s.AddUser("a")
	s.AddAssistant("b", nil, nil)
	s.AddToolResult("c1", "c")

	recs := s.Records()
	require.Len(t, recs, 3)
	assert.Equal(t, "user", recs[0].Message.Role)
	assert.Equal(t, "assistant", recs[1].Message.Role)
	assert.Equal(t, "tool", recs[2].Message.Role)
}

func TestSession_RecordByID_FindsByID(t *testing.T) {
	s := New("/tmp")
	rec := s.AddUser("find me")
	found := s.RecordByID(rec.ID)
	require.NotNil(t, found)
	assert.Equal(t, "find me", found.Message.Content)
}

func TestSession_RecordByID_ReturnsNilForUnknown(t *testing.T) {
	s := New("/tmp")
	s.AddUser("hello")
	assert.Nil(t, s.RecordByID("nonexistent"))
}

func TestSession_MessageCount_ByRole(t *testing.T) {
	s := New("/tmp")
	s.AddUser("a")
	s.AddUser("b")
	s.AddAssistant("c", nil, nil)
	s.AddToolResult("t1", "d")

	assert.Equal(t, 4, s.MessageCount(""))
	assert.Equal(t, 2, s.MessageCount("user"))
	assert.Equal(t, 1, s.MessageCount("assistant"))
	assert.Equal(t, 1, s.MessageCount("tool"))
	assert.Equal(t, 0, s.MessageCount("system"))
}

func TestSession_Records_ReturnsCopy(t *testing.T) {
	s := New("/tmp")
	s.AddUser("original")
	recs := s.Records()
	recs[0].Message.Content = "modified"
	assert.Equal(t, "original", s.Messages[0].Message.Content)
}

func TestSession_Reset_ClearsTitle(t *testing.T) {
	s := New("/tmp")
	s.Title = "test"
	s.Reset()
	assert.Empty(t, s.Title)
}

// --- Revert/Unrevert tests ---

func TestSession_Revert_TruncatesFromUserMessage(t *testing.T) {
	s := New("/tmp")
	s.AddUser("first")
	s.AddAssistant("resp1", nil, nil)
	s.AddUser("second")
	s.AddAssistant("resp2", nil, nil)

	prompt, err := s.Revert()
	require.NoError(t, err)
	assert.Equal(t, "second", prompt)
	assert.Len(t, s.Messages, 2) // only first exchange remains
}

func TestSession_Revert_ReturnsPromptText(t *testing.T) {
	s := New("/tmp")
	s.AddUser("hello world")
	s.AddAssistant("hi", nil, nil)

	prompt, err := s.Revert()
	require.NoError(t, err)
	assert.Equal(t, "hello world", prompt)
}

func TestSession_Revert_EmptySession_ReturnsError(t *testing.T) {
	s := New("/tmp")
	_, err := s.Revert()
	assert.Error(t, err)
}

func TestSession_CanUndo_TrueWhenHasMessages(t *testing.T) {
	s := New("/tmp")
	s.AddUser("hello")
	assert.True(t, s.CanUndo())
}

func TestSession_CanUndo_FalseWhenEmpty(t *testing.T) {
	s := New("/tmp")
	assert.False(t, s.CanUndo())
}

func TestSession_Unrevert_RestoresMessages(t *testing.T) {
	s := New("/tmp")
	s.AddUser("first")
	s.AddAssistant("resp1", nil, nil)
	s.AddUser("second")
	s.AddAssistant("resp2", nil, nil)

	_, err := s.Revert()
	require.NoError(t, err)
	assert.Len(t, s.Messages, 2)

	err = s.Unrevert()
	require.NoError(t, err)
	assert.Len(t, s.Messages, 4)
	assert.Equal(t, "second", s.Messages[2].Message.Content)
}

func TestSession_CanRedo_TrueAfterRevert(t *testing.T) {
	s := New("/tmp")
	s.AddUser("hello")
	s.AddAssistant("hi", nil, nil)

	_, _ = s.Revert()
	assert.True(t, s.CanRedo())
}

func TestSession_CanRedo_FalseAfterNewMessage(t *testing.T) {
	s := New("/tmp")
	s.AddUser("hello")
	s.AddAssistant("hi", nil, nil)

	_, _ = s.Revert()
	assert.True(t, s.CanRedo())

	s.AddUser("new message")
	assert.False(t, s.CanRedo())
}

func TestSession_Revert_MultipleTimes(t *testing.T) {
	s := New("/tmp")
	s.AddUser("first")
	s.AddAssistant("resp1", nil, nil)
	s.AddUser("second")
	s.AddAssistant("resp2", nil, nil)
	s.AddUser("third")
	s.AddAssistant("resp3", nil, nil)

	prompt, _ := s.Revert()
	assert.Equal(t, "third", prompt)
	assert.Len(t, s.Messages, 4)

	prompt, _ = s.Revert()
	assert.Equal(t, "second", prompt)
	assert.Len(t, s.Messages, 2)
}

func TestSession_Unrevert_MultipleTimes(t *testing.T) {
	s := New("/tmp")
	s.AddUser("first")
	s.AddAssistant("resp1", nil, nil)
	s.AddUser("second")
	s.AddAssistant("resp2", nil, nil)

	_, _ = s.Revert()
	_, _ = s.Revert()
	assert.Empty(t, s.Messages)

	_ = s.Unrevert()
	assert.Len(t, s.Messages, 2)

	_ = s.Unrevert()
	assert.Len(t, s.Messages, 4)
}

func TestSession_RevertClearsRedoOnNewMessage(t *testing.T) {
	s := New("/tmp")
	s.AddUser("hello")
	s.AddAssistant("hi", nil, nil)

	_, _ = s.Revert()
	assert.True(t, s.CanRedo())

	s.AddUser("different")
	assert.False(t, s.CanRedo())

	err := s.Unrevert()
	assert.Error(t, err, "should not be able to redo after new message")
}
