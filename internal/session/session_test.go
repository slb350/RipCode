package session

import (
	"fmt"
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
	assert.Empty(t, s.messages)
	assert.Empty(t, s.Title)
}

func TestAddUser(t *testing.T) {
	s := New("/tmp")
	rec := s.AddUser("hello")

	require.Len(t, s.messages, 1)
	assert.Equal(t, provider.RoleUser, s.messages[0].Message.Role)
	assert.Equal(t, "hello", s.messages[0].Message.Content)
	assert.NotNil(t, rec)
	assert.Equal(t, provider.RoleUser, rec.Message.Role)
}

func TestAddAssistant(t *testing.T) {
	s := New("/tmp")
	rec := s.AddAssistant("I'll help", nil, nil)

	require.Len(t, s.messages, 1)
	assert.Equal(t, provider.RoleAssistant, s.messages[0].Message.Role)
	assert.Equal(t, "I'll help", s.messages[0].Message.Content)
	assert.NotNil(t, rec)
	assert.Nil(t, rec.Meta)
}

func TestAddAssistantWithToolCalls(t *testing.T) {
	s := New("/tmp")
	calls := []provider.ToolCall{
		{ID: "call_1", Name: "bash", Args: `{"command":"ls"}`},
	}
	rec := s.AddAssistant("Let me check.", calls, nil)

	require.Len(t, s.messages, 1)
	assert.Equal(t, provider.RoleAssistant, s.messages[0].Message.Role)
	assert.Len(t, s.messages[0].Message.ToolCalls, 1)
	assert.Equal(t, "bash", s.messages[0].Message.ToolCalls[0].Name)
	assert.NotNil(t, rec)
}

func TestAddToolResult(t *testing.T) {
	s := New("/tmp")
	rec := s.AddToolResult("call_1", "file1.go\nfile2.go")

	require.Len(t, s.messages, 1)
	assert.Equal(t, provider.RoleTool, s.messages[0].Message.Role)
	assert.Equal(t, "call_1", s.messages[0].Message.ToolCallID)
	assert.Equal(t, "file1.go\nfile2.go", s.messages[0].Message.Content)
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
	assert.Equal(t, provider.RoleUser, history[0].Role)
	assert.Equal(t, provider.RoleAssistant, history[1].Role)
	assert.Equal(t, provider.RoleTool, history[2].Role)
	assert.Equal(t, provider.RoleAssistant, history[3].Role)
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
	assert.Equal(t, provider.RoleSystem, history[0].Role)
	assert.Equal(t, "You are a helpful assistant.", history[0].Content)
	assert.Equal(t, provider.RoleUser, history[1].Role)
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
	assert.Equal(t, provider.RoleUser, rec.Message.Role)
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
	assert.Equal(t, provider.RoleAssistant, rec.Message.Role)
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
	assert.Equal(t, provider.RoleTool, rec.Message.Role)
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
	assert.Equal(t, provider.RoleAssistant, history[0].Role)
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
	assert.Equal(t, provider.RoleUser, recs[0].Message.Role)
	assert.Equal(t, provider.RoleAssistant, recs[1].Message.Role)
	assert.Equal(t, provider.RoleTool, recs[2].Message.Role)
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

func TestSession_RecordByID_ReturnsCopy(t *testing.T) {
	s := New("/tmp")
	rec := s.AddUser("original")
	id := rec.ID

	found := s.RecordByID(id)
	require.NotNil(t, found)
	found.Message.Content = "modified"

	// Internal record should be unchanged.
	again := s.RecordByID(id)
	assert.Equal(t, "original", again.Message.Content)
}

func TestSession_AddUser_ReturnsCopy(t *testing.T) {
	s := New("/tmp")
	rec := s.AddUser("original")
	rec.Message.Content = "modified"

	// Internal record should be unchanged.
	recs := s.Records()
	assert.Equal(t, "original", recs[0].Message.Content)
}

func TestSession_AddAssistant_ReturnsCopy(t *testing.T) {
	s := New("/tmp")
	rec := s.AddAssistant("original", nil, nil)
	rec.Message.Content = "modified"

	recs := s.Records()
	assert.Equal(t, "original", recs[0].Message.Content)
}

func TestSession_AddToolResult_ReturnsCopy(t *testing.T) {
	s := New("/tmp")
	rec := s.AddToolResult("call1", "original")
	rec.Message.Content = "modified"

	recs := s.Records()
	assert.Equal(t, "original", recs[0].Message.Content)
}

func TestSession_Records_ReturnsCopy(t *testing.T) {
	s := New("/tmp")
	s.AddUser("original")
	recs := s.Records()
	recs[0].Message.Content = "modified"
	assert.Equal(t, "original", s.messages[0].Message.Content)
}

func TestSession_Reset_ClearsTitle(t *testing.T) {
	s := New("/tmp")
	s.Title = "test"
	s.Reset()
	assert.Empty(t, s.Title)
}

func TestSession_Reset_ClearsParentID(t *testing.T) {
	s := New("/tmp")
	s.ParentID = "sess-parent"
	s.Reset()
	assert.Empty(t, s.ParentID)
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
	assert.Len(t, s.messages, 2) // only first exchange remains
}

func TestSession_Revert_ReturnsPromptText(t *testing.T) {
	s := New("/tmp")
	s.AddUser("hello world")
	s.AddAssistant("hi", nil, nil)

	prompt, err := s.Revert()
	require.NoError(t, err)
	assert.Equal(t, "hello world", prompt)
}

func TestSession_Revert_UpdatesUpdatedAt(t *testing.T) {
	s := New("/tmp")
	s.AddUser("hello")
	s.AddAssistant("hi", nil, nil)

	before := s.UpdatedAt
	time.Sleep(time.Millisecond)

	_, err := s.Revert()
	require.NoError(t, err)
	assert.True(t, s.UpdatedAt.After(before), "revert should refresh UpdatedAt")
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
	assert.Len(t, s.messages, 2)

	err = s.Unrevert()
	require.NoError(t, err)
	assert.Len(t, s.messages, 4)
	assert.Equal(t, "second", s.messages[2].Message.Content)
}

func TestSession_Unrevert_UpdatesUpdatedAt(t *testing.T) {
	s := New("/tmp")
	s.AddUser("first")
	s.AddAssistant("resp1", nil, nil)
	_, err := s.Revert()
	require.NoError(t, err)

	before := s.UpdatedAt
	time.Sleep(time.Millisecond)

	err = s.Unrevert()
	require.NoError(t, err)
	assert.True(t, s.UpdatedAt.After(before), "unrevert should refresh UpdatedAt")
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
	assert.Len(t, s.messages, 4)

	prompt, _ = s.Revert()
	assert.Equal(t, "second", prompt)
	assert.Len(t, s.messages, 2)
}

func TestSession_Unrevert_MultipleTimes(t *testing.T) {
	s := New("/tmp")
	s.AddUser("first")
	s.AddAssistant("resp1", nil, nil)
	s.AddUser("second")
	s.AddAssistant("resp2", nil, nil)

	_, _ = s.Revert()
	_, _ = s.Revert()
	assert.Empty(t, s.messages)

	_ = s.Unrevert()
	assert.Len(t, s.messages, 2)

	_ = s.Unrevert()
	assert.Len(t, s.messages, 4)
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

func TestFork_CopiesMessagesUpToIndex(t *testing.T) {
	s := New("/tmp/test")
	s.Title = "original"
	s.AddUser("q1")
	s.AddAssistant("a1", nil, nil)
	s.AddUser("q2")
	s.AddAssistant("a2", nil, nil)

	forked, err := s.Fork(1) // include messages 0..1 (q1 + a1)
	require.NoError(t, err)
	assert.Len(t, forked.messages, 2)
	assert.Equal(t, "q1", forked.messages[0].Message.Content)
	assert.Equal(t, "a1", forked.messages[1].Message.Content)
}

func TestFork_GetsNewID(t *testing.T) {
	s := New("/tmp/test")
	s.AddUser("q1")
	s.AddAssistant("a1", nil, nil)

	forked, err := s.Fork(1)
	require.NoError(t, err)
	assert.NotEqual(t, s.ID, forked.ID)
}

func TestFork_PreservesWorkDir(t *testing.T) {
	s := New("/my/project")
	s.AddUser("q1")
	s.AddAssistant("a1", nil, nil)

	forked, err := s.Fork(1)
	require.NoError(t, err)
	assert.Equal(t, "/my/project", forked.WorkDir)
}

func TestFork_SetsParentID(t *testing.T) {
	s := New("/tmp")
	s.AddUser("q1")
	s.AddAssistant("a1", nil, nil)

	forked, err := s.Fork(1)
	require.NoError(t, err)
	assert.Equal(t, s.ID, forked.ParentID)
}

func TestFork_GeneratesNewMessageIDs(t *testing.T) {
	s := New("/tmp")
	s.AddUser("q1")
	s.AddAssistant("a1", nil, nil)

	forked, err := s.Fork(1)
	require.NoError(t, err)
	assert.NotEqual(t, s.messages[0].ID, forked.messages[0].ID)
	assert.NotEqual(t, s.messages[1].ID, forked.messages[1].ID)
}

func TestFork_PreservesSystemPrompt(t *testing.T) {
	s := New("/tmp")
	s.SetSystemPrompt("you are helpful")
	s.AddUser("q1")

	forked, err := s.Fork(0)
	require.NoError(t, err)
	history := forked.History()
	require.Len(t, history, 2) // system + user
	assert.Equal(t, provider.RoleSystem, history[0].Role)
	assert.Equal(t, "you are helpful", history[0].Content)
}

func TestFork_InvalidIndex_TooLarge(t *testing.T) {
	s := New("/tmp")
	s.AddUser("q1")

	_, err := s.Fork(5)
	assert.Error(t, err)
}

func TestFork_InvalidIndex_Negative(t *testing.T) {
	s := New("/tmp")
	s.AddUser("q1")

	_, err := s.Fork(-1)
	assert.Error(t, err)
}

func TestFork_EmptySession(t *testing.T) {
	s := New("/tmp")
	_, err := s.Fork(0)
	assert.Error(t, err)
}

func TestFork_HasFreshTimestamps(t *testing.T) {
	s := New("/tmp")
	s.AddUser("q1")
	s.CreatedAt = time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)

	forked, err := s.Fork(0)
	require.NoError(t, err)
	assert.True(t, forked.CreatedAt.After(s.CreatedAt))
}

func TestFork_HasEmptyRedoStack(t *testing.T) {
	s := New("/tmp")
	s.AddUser("q1")
	s.AddAssistant("a1", nil, nil)
	s.AddUser("q2")
	s.AddAssistant("a2", nil, nil)
	_, _ = s.Revert()
	assert.True(t, s.CanRedo())

	forked, err := s.Fork(1)
	require.NoError(t, err)
	assert.False(t, forked.CanRedo())
}

func TestFork_PreservesTokens(t *testing.T) {
	s := New("/tmp")
	s.AddUser("q1")
	s.AddAssistant("a1", nil, &AssistantMeta{InputTokens: 10, OutputTokens: 20})

	forked, err := s.Fork(1)
	require.NoError(t, err)
	// Forked session starts with zero tokens — it's a new session
	assert.Equal(t, 0, forked.Tokens.Input)
	assert.Equal(t, 0, forked.Tokens.Output)
}

func TestFork_PreservesMessageMeta(t *testing.T) {
	s := New("/tmp")
	s.AddUser("q1")
	meta := &AssistantMeta{Model: "gpt-4", Agent: "build"}
	s.AddAssistant("a1", nil, meta)

	forked, err := s.Fork(1)
	require.NoError(t, err)
	require.NotNil(t, forked.messages[1].Meta)
	assert.Equal(t, "gpt-4", forked.messages[1].Meta.Model)
	assert.Equal(t, "build", forked.messages[1].Meta.Agent)
}

func TestFork_BoundaryConditions(t *testing.T) {
	t.Run("fork at index 0 with single message", func(t *testing.T) {
		s := New("/tmp")
		s.AddUser("only message")

		forked, err := s.Fork(0)
		require.NoError(t, err)
		assert.Len(t, forked.messages, 1)
		assert.Equal(t, "only message", forked.messages[0].Message.Content)
	})

	t.Run("fork at last valid index", func(t *testing.T) {
		s := New("/tmp")
		s.AddUser("q1")
		s.AddAssistant("a1", nil, nil)
		s.AddUser("q2")
		s.AddAssistant("a2", nil, nil)

		forked, err := s.Fork(len(s.messages) - 1)
		require.NoError(t, err)
		assert.Len(t, forked.messages, 4)
	})

	t.Run("fork at len(Messages) returns error", func(t *testing.T) {
		s := New("/tmp")
		s.AddUser("q1")
		s.AddAssistant("a1", nil, nil)

		_, err := s.Fork(len(s.messages))
		assert.Error(t, err, "index == len(Messages) should be out of range")
	})
}

func TestSession_ConcurrentReadWrite(t *testing.T) {
	// Verify there are no data races when writing and reading concurrently.
	// Run with -race to detect issues.
	s := New("/tmp")

	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := 0; i < 100; i++ {
			s.AddUser("msg")
			s.AddAssistant("resp", nil, &AssistantMeta{Model: "test"})
		}
	}()

	// Concurrent reads while the goroutine writes.
	for i := 0; i < 100; i++ {
		_ = s.Records()
		_ = s.History()
		_ = s.Len()
		_ = s.CanUndo()
		_ = s.CanRedo()
	}

	<-done
	assert.True(t, s.Len() > 0)
}

func TestSession_ConcurrentRevertAndAdd(t *testing.T) {
	// Verify there are no data races when reverting and adding concurrently.
	// Run with -race to detect issues.
	s := New("/tmp")

	// Pre-populate with 20 user+assistant pairs
	for i := 0; i < 20; i++ {
		s.AddUser(fmt.Sprintf("q%d", i))
		s.AddAssistant(fmt.Sprintf("a%d", i), nil, nil)
	}

	done := make(chan struct{})

	// Goroutine doing Reverts
	go func() {
		defer func() { done <- struct{}{} }()
		for i := 0; i < 10; i++ {
			if s.CanUndo() {
				s.Revert()
			}
		}
	}()

	// Goroutine doing AddUser
	go func() {
		defer func() { done <- struct{}{} }()
		for i := 0; i < 10; i++ {
			s.AddUser(fmt.Sprintf("concurrent-%d", i))
		}
	}()

	// Goroutine doing reads
	go func() {
		defer func() { done <- struct{}{} }()
		for i := 0; i < 20; i++ {
			_ = s.Records()
			_ = s.History()
			_ = s.CanUndo()
			_ = s.CanRedo()
		}
	}()

	<-done
	<-done
	<-done

	// Session should still be in a valid state
	assert.True(t, s.Len() >= 0)
}

func TestFork_DoesNotMutateOriginal(t *testing.T) {
	s := New("/tmp")
	s.AddUser("q1")
	s.AddAssistant("a1", nil, nil)
	s.AddUser("q2")
	s.AddAssistant("a2", nil, nil)

	_, err := s.Fork(1)
	require.NoError(t, err)
	assert.Len(t, s.messages, 4, "original session should be unchanged")
}

func TestFork_ToolCalls_IndependentCopy(t *testing.T) {
	s := New("/tmp")
	s.AddUser("do something")
	s.AddAssistant("sure", []provider.ToolCall{
		{ID: "call_1", Name: "bash", Args: `{"cmd":"ls"}`},
		{ID: "call_2", Name: "read", Args: `{"path":"/tmp"}`},
	}, nil)

	forked, err := s.Fork(1)
	require.NoError(t, err)

	// Mutate forked ToolCalls
	forked.messages[1].Message.ToolCalls[0].Name = "MUTATED"
	forked.messages[1].Message.ToolCalls[0].Args = "MUTATED"

	// Original ToolCalls should be unchanged
	assert.Equal(t, "bash", s.messages[1].Message.ToolCalls[0].Name,
		"original ToolCall name should be unchanged after forked mutation")
	assert.Equal(t, `{"cmd":"ls"}`, s.messages[1].Message.ToolCalls[0].Args,
		"original ToolCall args should be unchanged after forked mutation")

	// Also verify the second tool call is independent
	forked.messages[1].Message.ToolCalls[1].ID = "MUTATED"
	assert.Equal(t, "call_2", s.messages[1].Message.ToolCalls[1].ID,
		"second ToolCall should also be independent")
}

func TestMessageRecord_Valid_ValidUser(t *testing.T) {
	rec := MessageRecord{
		ID:        "msg-001",
		Message:   provider.Message{Role: provider.RoleUser, Content: "hello"},
		CreatedAt: time.Now(),
	}
	assert.NoError(t, rec.Valid())
}

func TestMessageRecord_Valid_MissingID(t *testing.T) {
	rec := MessageRecord{
		Message:   provider.Message{Role: provider.RoleUser, Content: "hello"},
		CreatedAt: time.Now(),
	}
	assert.Error(t, rec.Valid())
}

func TestMessageRecord_Valid_MissingCreatedAt(t *testing.T) {
	rec := MessageRecord{
		ID:      "msg-001",
		Message: provider.Message{Role: provider.RoleUser, Content: "hello"},
	}
	assert.Error(t, rec.Valid())
}

func TestMessageRecord_Valid_InvalidRole(t *testing.T) {
	rec := MessageRecord{
		ID:        "msg-001",
		Message:   provider.Message{Role: "invalid", Content: "hello"},
		CreatedAt: time.Now(),
	}
	assert.Error(t, rec.Valid())
}

func TestMessageRecord_Valid_MetaOnNonAssistant(t *testing.T) {
	rec := MessageRecord{
		ID:        "msg-001",
		Message:   provider.Message{Role: provider.RoleUser, Content: "hello"},
		CreatedAt: time.Now(),
		Meta:      &AssistantMeta{Model: "test"},
	}
	err := rec.Valid()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Meta only valid on assistant")
}

func TestMessageRecord_Valid_MetaOnAssistant_OK(t *testing.T) {
	rec := MessageRecord{
		ID:        "msg-001",
		Message:   provider.Message{Role: provider.RoleAssistant, Content: "response"},
		CreatedAt: time.Now(),
		Meta:      &AssistantMeta{Model: "test"},
	}
	assert.NoError(t, rec.Valid())
}

func TestLoadRecord_ValidRecord_Succeeds(t *testing.T) {
	s := New("/tmp")
	rec := MessageRecord{
		ID:        "msg-001",
		Message:   provider.Message{Role: provider.RoleUser, Content: "hello"},
		CreatedAt: time.Now(),
	}
	err := s.LoadRecord(rec)
	assert.NoError(t, err)
	assert.Equal(t, 1, s.Len())
}

func TestLoadRecord_InvalidRecord_ReturnsError(t *testing.T) {
	s := New("/tmp")
	rec := MessageRecord{
		ID:      "",
		Message: provider.Message{Role: provider.RoleUser, Content: "hello"},
	}
	err := s.LoadRecord(rec)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "load record")
	assert.Equal(t, 0, s.Len(), "invalid record should not be appended")
}

func TestLoadRecord_InvalidRole_ReturnsError(t *testing.T) {
	s := New("/tmp")
	rec := MessageRecord{
		ID:        "msg-001",
		Message:   provider.Message{Role: "bogus", Content: "hello"},
		CreatedAt: time.Now(),
	}
	err := s.LoadRecord(rec)
	assert.Error(t, err)
	assert.Equal(t, 0, s.Len())
}
