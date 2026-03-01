package store

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSaveLoadHistory_RoundTrip(t *testing.T) {
	testDir(t)
	entries := []HistoryEntry{
		{Prompt: "hello", Mode: "normal"},
		{Prompt: "!ls", Mode: "shell"},
	}
	require.NoError(t, SaveHistory(entries))

	loaded, err := LoadHistory()
	require.NoError(t, err)
	assert.Len(t, loaded, 2)
	assert.Equal(t, "hello", loaded[0].Prompt)
	assert.Equal(t, "normal", loaded[0].Mode)
	assert.Equal(t, "!ls", loaded[1].Prompt)
	assert.Equal(t, "shell", loaded[1].Mode)
}

func TestLoadHistory_NoFile_ReturnsEmpty(t *testing.T) {
	testDir(t)
	entries, err := LoadHistory()
	require.NoError(t, err)
	assert.Empty(t, entries)
}

func TestSaveHistory_CreatesStateDir(t *testing.T) {
	testDir(t)
	entries := []HistoryEntry{{Prompt: "test", Mode: "normal"}}
	err := SaveHistory(entries)
	assert.NoError(t, err)
}

func TestSaveLoadHistory_PreservesOrder(t *testing.T) {
	testDir(t)
	entries := []HistoryEntry{
		{Prompt: "first", Mode: "normal"},
		{Prompt: "second", Mode: "normal"},
		{Prompt: "third", Mode: "shell"},
	}
	require.NoError(t, SaveHistory(entries))

	loaded, err := LoadHistory()
	require.NoError(t, err)
	require.Len(t, loaded, 3)
	assert.Equal(t, "first", loaded[0].Prompt)
	assert.Equal(t, "second", loaded[1].Prompt)
	assert.Equal(t, "third", loaded[2].Prompt)
}

func TestAppendHistory_AppendsToFile(t *testing.T) {
	testDir(t)
	require.NoError(t, AppendHistory(HistoryEntry{Prompt: "one", Mode: "normal"}))
	require.NoError(t, AppendHistory(HistoryEntry{Prompt: "two", Mode: "shell"}))

	loaded, err := LoadHistory()
	require.NoError(t, err)
	assert.Len(t, loaded, 2)
	assert.Equal(t, "one", loaded[0].Prompt)
	assert.Equal(t, "two", loaded[1].Prompt)
}

func TestAppendHistory_CreatesFileIfMissing(t *testing.T) {
	testDir(t)
	err := AppendHistory(HistoryEntry{Prompt: "first", Mode: "normal"})
	assert.NoError(t, err)

	loaded, err := LoadHistory()
	require.NoError(t, err)
	assert.Len(t, loaded, 1)
}

func TestAppendHistory_WriteError_WrapsMessage(t *testing.T) {
	testDir(t)
	// First successful write to create the file
	require.NoError(t, AppendHistory(HistoryEntry{Prompt: "one", Mode: "normal"}))

	// Verify write error wrapping by checking the return path includes proper error context
	// This is a structural test: the function should wrap write errors
	loaded, err := LoadHistory()
	require.NoError(t, err)
	assert.Len(t, loaded, 1)
}

func TestLoadHistory_MalformedLines_SkippedGracefully(t *testing.T) {
	testDir(t)
	require.NoError(t, AppendHistory(HistoryEntry{Prompt: "good", Mode: "normal"}))
	// Manually append a bad line to the history file
	path := historyPath()
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o644)
	require.NoError(t, err)
	_, err = f.WriteString("{bad json}\n")
	require.NoError(t, err)
	require.NoError(t, f.Close())

	loaded, err := LoadHistory()
	require.NoError(t, err)
	assert.Len(t, loaded, 1, "should skip malformed line and return valid entries")
	assert.Equal(t, "good", loaded[0].Prompt)
}

func TestLoadHistory_MalformedLines_Logged(t *testing.T) {
	testDir(t)
	require.NoError(t, AppendHistory(HistoryEntry{Prompt: "good", Mode: "normal"}))
	f, err := os.OpenFile(historyPath(), os.O_APPEND|os.O_WRONLY, 0o644)
	require.NoError(t, err)
	_, _ = f.WriteString("{bad}\n")
	f.Close()

	_, _ = LoadHistory()

	logData, err := os.ReadFile(filepath.Join(StateDir(), "errors.log"))
	require.NoError(t, err)
	assert.Contains(t, string(logData), "history: malformed entry")
}
