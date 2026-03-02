package store

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSaveLoadStash_RoundTrip(t *testing.T) {
	testDir(t)
	entries := []StashFileEntry{
		{ID: "s1", Content: "draft one"},
		{ID: "s2", Content: "draft two"},
	}
	require.NoError(t, SaveStash(entries))

	loaded, err := LoadStash()
	require.NoError(t, err)
	require.Len(t, loaded, 2)
	assert.Equal(t, "s1", loaded[0].ID)
	assert.Equal(t, "draft one", loaded[0].Content)
	assert.Equal(t, "s2", loaded[1].ID)
	assert.Equal(t, "draft two", loaded[1].Content)
}

func TestLoadStash_NoFile_ReturnsEmpty(t *testing.T) {
	testDir(t)
	entries, err := LoadStash()
	require.NoError(t, err)
	assert.Empty(t, entries)
}

func TestSaveStash_CreatesStateDir(t *testing.T) {
	testDir(t)
	err := SaveStash([]StashFileEntry{{ID: "x", Content: "y"}})
	assert.NoError(t, err)
}

func TestSaveLoadStash_EmptyList(t *testing.T) {
	testDir(t)
	require.NoError(t, SaveStash(nil))
	loaded, err := LoadStash()
	require.NoError(t, err)
	assert.Empty(t, loaded)
}
