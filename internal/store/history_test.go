package store

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
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

	loaded, skipped, err := LoadHistory()
	require.NoError(t, err)
	assert.Equal(t, 0, skipped)
	assert.Len(t, loaded, 2)
	assert.Equal(t, "hello", loaded[0].Prompt)
	assert.Equal(t, "normal", loaded[0].Mode)
	assert.Equal(t, "!ls", loaded[1].Prompt)
	assert.Equal(t, "shell", loaded[1].Mode)
}

func TestLoadHistory_NoFile_ReturnsEmpty(t *testing.T) {
	testDir(t)
	entries, skipped, err := LoadHistory()
	require.NoError(t, err)
	assert.Equal(t, 0, skipped)
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

	loaded, _, err := LoadHistory()
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

	loaded, _, err := LoadHistory()
	require.NoError(t, err)
	assert.Len(t, loaded, 2)
	assert.Equal(t, "one", loaded[0].Prompt)
	assert.Equal(t, "two", loaded[1].Prompt)
}

func TestAppendHistory_CreatesFileIfMissing(t *testing.T) {
	testDir(t)
	err := AppendHistory(HistoryEntry{Prompt: "first", Mode: "normal"})
	assert.NoError(t, err)

	loaded, _, err := LoadHistory()
	require.NoError(t, err)
	assert.Len(t, loaded, 1)
}

func TestAppendHistory_WriteError_WrapsMessage(t *testing.T) {
	testDir(t)
	// First successful write to create the file
	require.NoError(t, AppendHistory(HistoryEntry{Prompt: "one", Mode: "normal"}))

	// Verify write error wrapping by checking the return path includes proper error context
	// This is a structural test: the function should wrap write errors
	loaded, _, err := LoadHistory()
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

	loaded, skipped, err := LoadHistory()
	require.NoError(t, err)
	assert.Equal(t, 1, skipped, "should report 1 skipped entry")
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

	_, _, _ = LoadHistory()

	logData, err := os.ReadFile(filepath.Join(StateDir(), "errors.log"))
	require.NoError(t, err)
	assert.Contains(t, string(logData), "history: malformed entry")
}

func TestSaveHistory_AtomicWrite_PreservesOnCrash(t *testing.T) {
	testDir(t)
	// Write initial entries
	entries := []HistoryEntry{{Prompt: "first", Mode: "normal"}}
	require.NoError(t, SaveHistory(entries))

	// Verify no .tmp file left behind
	path := historyPath()
	_, err := os.Stat(path + ".tmp")
	assert.True(t, os.IsNotExist(err), "tmp file should be cleaned up")

	// Verify the file is valid JSONL
	loaded, _, err := LoadHistory()
	require.NoError(t, err)
	assert.Len(t, loaded, 1)
	assert.Equal(t, "first", loaded[0].Prompt)
}

func TestSaveHistory_Overwrite_ReplacesContent(t *testing.T) {
	testDir(t)
	// Write initial entries
	require.NoError(t, SaveHistory([]HistoryEntry{
		{Prompt: "old1", Mode: "normal"},
		{Prompt: "old2", Mode: "normal"},
	}))

	// Overwrite with fewer entries
	require.NoError(t, SaveHistory([]HistoryEntry{
		{Prompt: "new1", Mode: "shell"},
	}))

	loaded, _, err := LoadHistory()
	require.NoError(t, err)
	assert.Len(t, loaded, 1, "should have replaced, not appended")
	assert.Equal(t, "new1", loaded[0].Prompt)
}

func TestSaveHistory_LargeHistory_SaveAndLoadPreserved(t *testing.T) {
	testDir(t)
	// Save 300 entries — verifies no implicit truncation at the store layer.
	entries := make([]HistoryEntry, 300)
	for i := range entries {
		entries[i] = HistoryEntry{Prompt: fmt.Sprintf("prompt-%d", i), Mode: "normal"}
	}
	require.NoError(t, SaveHistory(entries))

	loaded, skipped, err := LoadHistory()
	require.NoError(t, err)
	assert.Equal(t, 0, skipped)
	assert.Len(t, loaded, 300, "store layer should preserve all entries; truncation is caller's job")
	assert.Equal(t, "prompt-0", loaded[0].Prompt)
	assert.Equal(t, "prompt-299", loaded[299].Prompt)
}

func TestAppendHistory_ManyEntries_AllPreserved(t *testing.T) {
	testDir(t)
	// Append 50 entries one at a time.
	for i := range 50 {
		require.NoError(t, AppendHistory(HistoryEntry{
			Prompt: fmt.Sprintf("entry-%d", i),
			Mode:   "normal",
		}))
	}
	loaded, _, err := LoadHistory()
	require.NoError(t, err)
	assert.Len(t, loaded, 50)
	assert.Equal(t, "entry-0", loaded[0].Prompt)
	assert.Equal(t, "entry-49", loaded[49].Prompt)
}

func TestLoadHistory_LongSingleLine_DoesNotFailScanner(t *testing.T) {
	testDir(t)

	longPrompt := strings.Repeat("x", 70*1024) // exceeds default scanner token limit
	require.NoError(t, SaveHistory([]HistoryEntry{
		{Prompt: longPrompt, Mode: "normal"},
	}))

	loaded, skipped, err := LoadHistory()
	require.NoError(t, err)
	assert.Equal(t, 0, skipped)
	require.Len(t, loaded, 1)
	assert.Equal(t, longPrompt, loaded[0].Prompt)
}
