package store

import (
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
