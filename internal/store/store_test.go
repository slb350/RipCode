package store

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stephenbrandon/ripcode/internal/provider"
	"github.com/stephenbrandon/ripcode/internal/session"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("RIPCODE_DIR", dir)
	return dir
}

func makeTestSession(t *testing.T) *session.Session {
	t.Helper()
	sess := session.New("/tmp/project")
	sess.Title = "test session"
	sess.AddUser("hello")
	sess.AddAssistant("world", nil, &session.AssistantMeta{
		Model:        "gpt-4",
		InputTokens:  100,
		OutputTokens: 50,
		FinishReason: "stop",
		Duration:     2 * time.Second,
	})
	sess.AddTokens(100, 50)
	return sess
}

func TestDir_ReturnsRipcodeDir(t *testing.T) {
	dir := testDir(t)
	assert.Equal(t, dir, Dir())
}

func TestSessionsDir_ReturnsSessionsSubdir(t *testing.T) {
	dir := testDir(t)
	assert.Equal(t, filepath.Join(dir, "sessions"), SessionsDir())
}

func TestSave_CreatesFile(t *testing.T) {
	testDir(t)
	sess := makeTestSession(t)
	err := Save(sess)
	require.NoError(t, err)

	path := filepath.Join(SessionsDir(), sess.ID+".json")
	_, err = os.Stat(path)
	assert.NoError(t, err)
}

func TestSave_RoundTrips(t *testing.T) {
	testDir(t)
	sess := makeTestSession(t)
	err := Save(sess)
	require.NoError(t, err)

	loaded, err := Load(sess.ID)
	require.NoError(t, err)
	assert.Equal(t, sess.ID, loaded.ID)
	assert.Equal(t, sess.Title, loaded.Title)
	assert.Equal(t, sess.WorkDir, loaded.WorkDir)
}

func TestLoad_ReturnsSession(t *testing.T) {
	testDir(t)
	sess := makeTestSession(t)
	require.NoError(t, Save(sess))

	loaded, err := Load(sess.ID)
	require.NoError(t, err)
	assert.NotNil(t, loaded)
}

func TestLoad_PreservesMessages(t *testing.T) {
	testDir(t)
	sess := makeTestSession(t)
	require.NoError(t, Save(sess))

	loaded, err := Load(sess.ID)
	require.NoError(t, err)
	require.Len(t, loaded.Messages, 2)
	assert.Equal(t, "user", loaded.Messages[0].Message.Role)
	assert.Equal(t, "hello", loaded.Messages[0].Message.Content)
	assert.Equal(t, "assistant", loaded.Messages[1].Message.Role)
	assert.Equal(t, "world", loaded.Messages[1].Message.Content)
}

func TestLoad_PreservesMetadata(t *testing.T) {
	testDir(t)
	sess := makeTestSession(t)
	require.NoError(t, Save(sess))

	loaded, err := Load(sess.ID)
	require.NoError(t, err)
	require.NotNil(t, loaded.Messages[1].Meta)
	assert.Equal(t, "gpt-4", loaded.Messages[1].Meta.Model)
	assert.Equal(t, 100, loaded.Messages[1].Meta.InputTokens)
	assert.Equal(t, 50, loaded.Messages[1].Meta.OutputTokens)
	assert.Equal(t, "stop", loaded.Messages[1].Meta.FinishReason)
	assert.Equal(t, 2*time.Second, loaded.Messages[1].Meta.Duration)
}

func TestLoad_PreservesMessageIDs(t *testing.T) {
	testDir(t)
	sess := makeTestSession(t)
	require.NoError(t, Save(sess))

	loaded, err := Load(sess.ID)
	require.NoError(t, err)
	assert.Equal(t, sess.Messages[0].ID, loaded.Messages[0].ID)
	assert.Equal(t, sess.Messages[1].ID, loaded.Messages[1].ID)
}

func TestLoad_NonExistent_ReturnsError(t *testing.T) {
	testDir(t)
	_, err := Load("nonexistent")
	assert.Error(t, err)
}

func TestList_ReturnsAllSessions(t *testing.T) {
	testDir(t)
	s1 := makeTestSession(t)
	s2 := session.New("/tmp/other")
	s2.Title = "other session"
	s2.AddUser("test")
	require.NoError(t, Save(s1))
	require.NoError(t, Save(s2))

	summaries, err := List()
	require.NoError(t, err)
	assert.Len(t, summaries, 2)
}

func TestList_SortsByUpdatedAtDesc(t *testing.T) {
	testDir(t)
	s1 := session.New("/tmp/a")
	s1.Title = "older"
	s1.AddUser("first")
	require.NoError(t, Save(s1))

	// Ensure s2 has a later UpdatedAt
	time.Sleep(10 * time.Millisecond)
	s2 := session.New("/tmp/b")
	s2.Title = "newer"
	s2.AddUser("second")
	require.NoError(t, Save(s2))

	summaries, err := List()
	require.NoError(t, err)
	require.Len(t, summaries, 2)
	assert.Equal(t, "newer", summaries[0].Title)
	assert.Equal(t, "older", summaries[1].Title)
}

func TestList_EmptyDir_ReturnsEmpty(t *testing.T) {
	testDir(t)
	summaries, err := List()
	require.NoError(t, err)
	assert.Empty(t, summaries)
}

func TestDelete_RemovesFile(t *testing.T) {
	testDir(t)
	sess := makeTestSession(t)
	require.NoError(t, Save(sess))

	err := Delete(sess.ID)
	require.NoError(t, err)

	_, err = Load(sess.ID)
	assert.Error(t, err)
}

func TestDelete_NonExistent_ReturnsError(t *testing.T) {
	testDir(t)
	err := Delete("nonexistent")
	assert.Error(t, err)
}

func TestSessionSummary_HasTitle(t *testing.T) {
	testDir(t)
	sess := makeTestSession(t)
	require.NoError(t, Save(sess))

	summaries, err := List()
	require.NoError(t, err)
	require.Len(t, summaries, 1)
	assert.Equal(t, "test session", summaries[0].Title)
}

func TestSessionSummary_HasDates(t *testing.T) {
	testDir(t)
	sess := makeTestSession(t)
	require.NoError(t, Save(sess))

	summaries, err := List()
	require.NoError(t, err)
	require.Len(t, summaries, 1)
	assert.False(t, summaries[0].CreatedAt.IsZero())
	assert.False(t, summaries[0].UpdatedAt.IsZero())
}

func TestSessionSummary_HasMessageCount(t *testing.T) {
	testDir(t)
	sess := makeTestSession(t)
	require.NoError(t, Save(sess))

	summaries, err := List()
	require.NoError(t, err)
	require.Len(t, summaries, 1)
	assert.Equal(t, 2, summaries[0].MessageCount)
}

func TestSaveLoad_PreservesTokens(t *testing.T) {
	testDir(t)
	sess := makeTestSession(t)
	require.NoError(t, Save(sess))

	loaded, err := Load(sess.ID)
	require.NoError(t, err)
	assert.Equal(t, 100, loaded.Tokens.Input)
	assert.Equal(t, 50, loaded.Tokens.Output)
}

func TestSaveLoad_PreservesTitle(t *testing.T) {
	testDir(t)
	sess := makeTestSession(t)
	sess.Title = "my project"
	require.NoError(t, Save(sess))

	loaded, err := Load(sess.ID)
	require.NoError(t, err)
	assert.Equal(t, "my project", loaded.Title)
}

func TestSaveLoad_PreservesToolCalls(t *testing.T) {
	testDir(t)
	sess := session.New("/tmp/tools")
	sess.AddUser("do something")
	sess.AddAssistant("sure", []provider.ToolCall{
		{ID: "call_1", Name: "bash", Args: `{"cmd":"ls"}`},
	}, nil)
	sess.AddToolResult("call_1", "file1\nfile2")
	require.NoError(t, Save(sess))

	loaded, err := Load(sess.ID)
	require.NoError(t, err)
	require.Len(t, loaded.Messages, 3)
	require.Len(t, loaded.Messages[1].Message.ToolCalls, 1)
	assert.Equal(t, "call_1", loaded.Messages[1].Message.ToolCalls[0].ID)
	assert.Equal(t, "bash", loaded.Messages[1].Message.ToolCalls[0].Name)
	assert.Equal(t, "tool", loaded.Messages[2].Message.Role)
	assert.Equal(t, "call_1", loaded.Messages[2].Message.ToolCallID)
}

func TestSaveLoad_PreservesTimestamps(t *testing.T) {
	testDir(t)
	sess := makeTestSession(t)
	require.NoError(t, Save(sess))

	loaded, err := Load(sess.ID)
	require.NoError(t, err)
	// Timestamps should be preserved within second precision (JSON marshaling truncation)
	assert.WithinDuration(t, sess.CreatedAt, loaded.CreatedAt, time.Second)
	assert.WithinDuration(t, sess.UpdatedAt, loaded.UpdatedAt, time.Second)
	assert.WithinDuration(t, sess.Messages[0].CreatedAt, loaded.Messages[0].CreatedAt, time.Second)
}
