package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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

func TestDir_EmptyHome_FallsBackToRelative(t *testing.T) {
	t.Setenv("RIPCODE_DIR", "")
	t.Setenv("HOME", "")
	d := Dir()
	assert.Equal(t, ".ripcode", d, "should fall back to relative .ripcode when HOME is empty")
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
	require.Len(t, loaded.Records(), 2)
	assert.Equal(t, provider.RoleUser, loaded.Records()[0].Message.Role)
	assert.Equal(t, "hello", loaded.Records()[0].Message.Content)
	assert.Equal(t, provider.RoleAssistant, loaded.Records()[1].Message.Role)
	assert.Equal(t, "world", loaded.Records()[1].Message.Content)
}

func TestLoad_PreservesMetadata(t *testing.T) {
	testDir(t)
	sess := makeTestSession(t)
	require.NoError(t, Save(sess))

	loaded, err := Load(sess.ID)
	require.NoError(t, err)
	require.NotNil(t, loaded.Records()[1].Meta)
	assert.Equal(t, "gpt-4", loaded.Records()[1].Meta.Model)
	assert.Equal(t, 100, loaded.Records()[1].Meta.InputTokens)
	assert.Equal(t, 50, loaded.Records()[1].Meta.OutputTokens)
	assert.Equal(t, "stop", loaded.Records()[1].Meta.FinishReason)
	assert.Equal(t, 2*time.Second, loaded.Records()[1].Meta.Duration)
}

func TestLoad_PreservesMessageIDs(t *testing.T) {
	testDir(t)
	sess := makeTestSession(t)
	require.NoError(t, Save(sess))

	loaded, err := Load(sess.ID)
	require.NoError(t, err)
	assert.Equal(t, sess.Records()[0].ID, loaded.Records()[0].ID)
	assert.Equal(t, sess.Records()[1].ID, loaded.Records()[1].ID)
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

	summaries, corrupted, err := List()
	require.NoError(t, err)
	assert.Len(t, summaries, 2)
	assert.Empty(t, corrupted)
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

	summaries, _, err := List()
	require.NoError(t, err)
	require.Len(t, summaries, 2)
	assert.Equal(t, "newer", summaries[0].Title)
	assert.Equal(t, "older", summaries[1].Title)
}

func TestList_EmptyDir_ReturnsEmpty(t *testing.T) {
	testDir(t)
	summaries, corrupted, err := List()
	require.NoError(t, err)
	assert.Empty(t, summaries)
	assert.Empty(t, corrupted)
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

	summaries, _, err := List()
	require.NoError(t, err)
	require.Len(t, summaries, 1)
	assert.Equal(t, "test session", summaries[0].Title)
}

func TestSessionSummary_HasDates(t *testing.T) {
	testDir(t)
	sess := makeTestSession(t)
	require.NoError(t, Save(sess))

	summaries, _, err := List()
	require.NoError(t, err)
	require.Len(t, summaries, 1)
	assert.False(t, summaries[0].CreatedAt.IsZero())
	assert.False(t, summaries[0].UpdatedAt.IsZero())
}

func TestSessionSummary_HasMessageCount(t *testing.T) {
	testDir(t)
	sess := makeTestSession(t)
	require.NoError(t, Save(sess))

	summaries, _, err := List()
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

func TestList_CorruptedJSONFile_SkipsAndContinues(t *testing.T) {
	testDir(t)
	// Save a valid session
	s1 := makeTestSession(t)
	require.NoError(t, Save(s1))

	// Write a corrupted JSON file alongside it
	dir := SessionsDir()
	corruptPath := filepath.Join(dir, "corrupted-session.json")
	require.NoError(t, os.WriteFile(corruptPath, []byte("{not valid json!!!"), 0o644))

	summaries, corrupted, err := List()
	require.NoError(t, err, "List should not return error for corrupted files")
	assert.Len(t, summaries, 1, "should return only the valid session")
	assert.Equal(t, s1.ID, summaries[0].ID)
	assert.Equal(t, []string{"corrupted-session.json"}, corrupted)
}

func TestList_CorruptedReturnsMultipleFilenames(t *testing.T) {
	testDir(t)
	dir := SessionsDir()
	require.NoError(t, os.MkdirAll(dir, 0o755))

	require.NoError(t, os.WriteFile(filepath.Join(dir, "bad1.json"), []byte("{bad"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "bad2.json"), []byte("{bad"), 0o644))

	summaries, corrupted, err := List()
	require.NoError(t, err)
	assert.Empty(t, summaries)
	assert.Len(t, corrupted, 2)
	assert.Contains(t, corrupted, "bad1.json")
	assert.Contains(t, corrupted, "bad2.json")
}

func TestList_CorruptedFile_Logged(t *testing.T) {
	testDir(t)
	dir := SessionsDir()
	require.NoError(t, os.MkdirAll(dir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "bad.json"), []byte("{invalid"), 0o644))

	_, corrupted, err := List()
	require.NoError(t, err)
	assert.Contains(t, corrupted, "bad.json")

	logData, err := os.ReadFile(filepath.Join(StateDir(), "errors.log"))
	require.NoError(t, err)
	assert.Contains(t, string(logData), "sessions: corrupted file: bad.json")
}

func TestSave_WritesMessageCountToJSON(t *testing.T) {
	testDir(t)
	sess := makeTestSession(t) // 2 messages (user + assistant)
	require.NoError(t, Save(sess))

	// Read the raw JSON and verify messageCount is present and correct
	path := filepath.Join(SessionsDir(), sess.ID+".json")
	data, err := os.ReadFile(path)
	require.NoError(t, err)

	var raw map[string]any
	require.NoError(t, json.Unmarshal(data, &raw))
	mc, ok := raw["messageCount"]
	require.True(t, ok, "JSON should contain messageCount field")
	assert.Equal(t, float64(2), mc)
}

func TestList_LegacyFileWithoutMessageCount(t *testing.T) {
	testDir(t)
	dir := SessionsDir()
	require.NoError(t, os.MkdirAll(dir, 0o755))

	// Write a v1 file without the messageCount field (legacy format)
	legacy := `{
		"version": 1,
		"id": "legacy-001",
		"title": "old session",
		"workDir": "/tmp",
		"createdAt": "2025-01-01T00:00:00Z",
		"updatedAt": "2025-01-01T00:00:00Z",
		"tokens": {"input": 0, "output": 0},
		"messages": [
			{"id": "m1", "role": "user", "content": "hi", "createdAt": "2025-01-01T00:00:00Z"},
			{"id": "m2", "role": "assistant", "content": "hello", "createdAt": "2025-01-01T00:00:00Z"},
			{"id": "m3", "role": "user", "content": "bye", "createdAt": "2025-01-01T00:00:00Z"}
		]
	}`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "legacy-001.json"), []byte(legacy), 0o644))

	summaries, corrupted, err := List()
	require.NoError(t, err)
	assert.Empty(t, corrupted)
	require.Len(t, summaries, 1)
	assert.Equal(t, "old session", summaries[0].Title)
	// Legacy file has no messageCount field — List should fall back to
	// counting the messages array.
	assert.Equal(t, 3, summaries[0].MessageCount)
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
	require.Len(t, loaded.Records(), 3)
	require.Len(t, loaded.Records()[1].Message.ToolCalls, 1)
	assert.Equal(t, "call_1", loaded.Records()[1].Message.ToolCalls[0].ID)
	assert.Equal(t, "bash", loaded.Records()[1].Message.ToolCalls[0].Name)
	assert.Equal(t, provider.RoleTool, loaded.Records()[2].Message.Role)
	assert.Equal(t, "call_1", loaded.Records()[2].Message.ToolCallID)
}

func TestAtomicWrite_Success(t *testing.T) {
	dir := testDir(t)
	path := filepath.Join(dir, "test.json")
	require.NoError(t, atomicWrite(path, []byte(`{"ok":true}`), 0o644))

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, `{"ok":true}`, string(data))

	// Temp file should be cleaned up.
	_, err = os.Stat(path + ".tmp")
	assert.True(t, os.IsNotExist(err))
}

func TestAtomicWrite_PreservesOriginalOnFailure(t *testing.T) {
	dir := testDir(t)
	path := filepath.Join(dir, "data.json")
	require.NoError(t, os.WriteFile(path, []byte("original"), 0o644))

	// Make the directory read-only so temp file write fails.
	require.NoError(t, os.Chmod(dir, 0o555))
	t.Cleanup(func() { os.Chmod(dir, 0o755) })

	err := atomicWrite(path, []byte("replacement"), 0o644)
	assert.Error(t, err)

	// Restore permissions to verify original is intact.
	os.Chmod(dir, 0o755)
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, "original", string(data))
}

func TestLoad_PathTraversalBlocked(t *testing.T) {
	testDir(t)
	for _, id := range []string{"../../../etc/passwd", "", "foo/bar", "a/../b"} {
		_, err := Load(id)
		assert.Error(t, err, "Load(%q) should error", id)
		if id != "" {
			assert.Contains(t, err.Error(), "invalid session ID", "Load(%q)", id)
		}
	}
}

func TestDelete_PathTraversalBlocked(t *testing.T) {
	testDir(t)
	for _, id := range []string{"../../../etc/passwd", "", "foo/bar"} {
		err := Delete(id)
		assert.Error(t, err, "Delete(%q) should error", id)
		if id != "" {
			assert.Contains(t, err.Error(), "invalid session ID", "Delete(%q)", id)
		}
	}
}

func TestSave_PathTraversalBlocked(t *testing.T) {
	testDir(t)
	s := session.New("/tmp")
	s.ID = "../../../etc/evil"
	err := Save(s)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid session ID")
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
	assert.WithinDuration(t, sess.Records()[0].CreatedAt, loaded.Records()[0].CreatedAt, time.Second)
}

func TestAtomicWrite_ConcurrentWrites_NoCorruption(t *testing.T) {
	dir := testDir(t)

	// Use separate files per goroutine to verify atomicWrite is safe for
	// concurrent use across different paths. The shared .tmp suffix means
	// same-path concurrent writes can race — this tests the more realistic
	// scenario of concurrent session saves to different files.
	const n = 20
	errs := make(chan error, n)
	for i := 0; i < n; i++ {
		go func(i int) {
			path := filepath.Join(dir, fmt.Sprintf("file-%d.json", i))
			data := []byte(fmt.Sprintf(`{"writer":%d}`, i))
			errs <- atomicWrite(path, data, 0o644)
		}(i)
	}

	for i := 0; i < n; i++ {
		assert.NoError(t, <-errs, "goroutine %d", i)
	}

	// Every file should contain valid JSON with the correct writer value.
	for i := 0; i < n; i++ {
		path := filepath.Join(dir, fmt.Sprintf("file-%d.json", i))
		data, err := os.ReadFile(path)
		require.NoError(t, err, "file %d should exist", i)
		var result map[string]any
		require.NoError(t, json.Unmarshal(data, &result), "file %d should contain valid JSON, got: %s", i, string(data))
		assert.Equal(t, float64(i), result["writer"], "file %d should have correct writer", i)

		// No temp files should remain.
		_, err = os.Stat(path + ".tmp")
		assert.True(t, os.IsNotExist(err), "temp file for %d should be cleaned up", i)
	}
}

func TestLoad_InvalidRecords_ReturnsSessionWithError(t *testing.T) {
	dir := testDir(t)
	// Create a valid session first
	sess := makeTestSession(t)
	require.NoError(t, Save(sess))

	// Corrupt one message record's role in the JSON file
	path := filepath.Join(SessionsDir(), sess.ID+".json")
	data, err := os.ReadFile(path)
	require.NoError(t, err)

	var f map[string]any
	require.NoError(t, json.Unmarshal(data, &f))
	msgs := f["messages"].([]any)
	// Set an invalid role on the first message
	msgs[0].(map[string]any)["role"] = "bogus"
	data, err = json.MarshalIndent(f, "", "  ")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, data, 0o644))

	loaded, err := Load(sess.ID)
	assert.Error(t, err, "should return error for skipped records")
	assert.Contains(t, err.Error(), "1 invalid record(s) skipped")
	assert.NotNil(t, loaded)
	// Only the valid assistant message should remain
	assert.Equal(t, 1, loaded.Len())

	// Verify error was also logged to disk
	logData, readErr := os.ReadFile(filepath.Join(dir, "state", "errors.log"))
	require.NoError(t, readErr)
	assert.Contains(t, string(logData), "invalid record")
}

func TestAtomicWrite_ConcurrentSamePath_NoCorruption(t *testing.T) {
	dir := testDir(t)
	path := filepath.Join(dir, "shared.json")

	// Each goroutine uses a unique temp file (CreateTemp), so all should succeed.
	// The key invariant: the final file must contain valid, complete JSON
	// from exactly one writer — never a partial or corrupted write.
	const n = 20
	errs := make(chan error, n)
	for i := 0; i < n; i++ {
		go func(i int) {
			data := []byte(fmt.Sprintf(`{"writer":%d}`, i))
			errs <- atomicWrite(path, data, 0o644)
		}(i)
	}

	for i := 0; i < n; i++ {
		assert.NoError(t, <-errs, "all writers should succeed with unique temp files")
	}

	// File should contain valid JSON from one of the writers.
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	var result map[string]any
	require.NoError(t, json.Unmarshal(data, &result), "file should contain valid JSON, got: %s", string(data))
	writer := int(result["writer"].(float64))
	assert.GreaterOrEqual(t, writer, 0)
	assert.Less(t, writer, n)

	// No leftover temp files should remain.
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	for _, e := range entries {
		assert.False(t, strings.Contains(e.Name(), ".tmp."),
			"leftover temp file found: %s", e.Name())
	}
}

func TestLoad_IDMismatch_ReturnsError(t *testing.T) {
	dir := testDir(t)
	sess := makeTestSession(t)
	require.NoError(t, Save(sess))

	// Tamper with the internal ID in the file.
	path := filepath.Join(SessionsDir(), sess.ID+".json")
	data, err := os.ReadFile(path)
	require.NoError(t, err)

	var f map[string]any
	require.NoError(t, json.Unmarshal(data, &f))
	f["id"] = "different-valid-id"
	data, err = json.MarshalIndent(f, "", "  ")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, data, 0o644))

	_, err = Load(sess.ID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "internal ID mismatch")
	assert.Contains(t, err.Error(), "different-valid-id")

	// Even a valid-looking but mismatched ID should be rejected.
	f["id"] = "another-valid"
	data, _ = json.MarshalIndent(f, "", "  ")
	_ = os.WriteFile(filepath.Join(dir, "sessions", sess.ID+".json"), data, 0o644)
	_, err = Load(sess.ID)
	assert.Error(t, err, "mismatched IDs should always error")
}
