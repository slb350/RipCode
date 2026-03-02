package store

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFrecency_Record_IncrementsCount(t *testing.T) {
	f := &FileFrecency{Entries: make(map[string]FrecencyEntry)}
	f.Record("main.go")
	assert.Equal(t, 1, f.Entries["main.go"].Count)
	f.Record("main.go")
	assert.Equal(t, 2, f.Entries["main.go"].Count)
}

func TestFrecency_Record_UpdatesLastUsed(t *testing.T) {
	f := &FileFrecency{Entries: make(map[string]FrecencyEntry)}
	before := time.Now()
	f.Record("main.go")
	assert.False(t, f.Entries["main.go"].LastUsed.Before(before))
}

func TestFrecency_Record_NilEntriesMap(t *testing.T) {
	f := &FileFrecency{}
	f.Record("file.go")
	assert.Equal(t, 1, f.Entries["file.go"].Count)
}

func TestFrecency_Score_NewlyRecordedFile(t *testing.T) {
	f := &FileFrecency{Entries: make(map[string]FrecencyEntry)}
	f.Record("main.go")
	f.Record("main.go")
	f.Record("main.go")
	// Just recorded: recencyWeight ≈ 1.0, score ≈ 3.0
	score := f.Score("main.go")
	assert.InDelta(t, 3.0, score, 0.1)
}

func TestFrecency_Score_OldFile(t *testing.T) {
	f := &FileFrecency{Entries: map[string]FrecencyEntry{
		"old.go": {Count: 5, LastUsed: time.Now().Add(-14 * 24 * time.Hour)},
	}}
	// 14 days old: recencyWeight = 1/(1+14/7) = 1/3 ≈ 0.33; score ≈ 5 * 0.33 = 1.67
	score := f.Score("old.go")
	assert.InDelta(t, 1.67, score, 0.1)
}

func TestFrecency_Score_UnknownFile(t *testing.T) {
	f := &FileFrecency{Entries: make(map[string]FrecencyEntry)}
	assert.Equal(t, 0.0, f.Score("unknown.go"))
}

func TestFrecency_Rank_SortsByScoreDescending(t *testing.T) {
	now := time.Now()
	f := &FileFrecency{Entries: map[string]FrecencyEntry{
		"a.go": {Count: 1, LastUsed: now},
		"b.go": {Count: 5, LastUsed: now},
		"c.go": {Count: 3, LastUsed: now},
	}}
	ranked := f.Rank([]string{"a.go", "b.go", "c.go"})
	assert.Equal(t, []string{"b.go", "c.go", "a.go"}, ranked)
}

func TestFrecency_Rank_UnscoredKeepOriginalOrder(t *testing.T) {
	now := time.Now()
	f := &FileFrecency{Entries: map[string]FrecencyEntry{
		"b.go": {Count: 2, LastUsed: now},
	}}
	ranked := f.Rank([]string{"x.go", "b.go", "y.go", "z.go"})
	assert.Equal(t, []string{"b.go", "x.go", "y.go", "z.go"}, ranked)
}

func TestFrecency_Rank_EmptyInput(t *testing.T) {
	f := &FileFrecency{Entries: make(map[string]FrecencyEntry)}
	assert.Nil(t, f.Rank(nil))
	assert.Nil(t, f.Rank([]string{}))
}

func TestFrecency_Prune_RemovesOldEntries(t *testing.T) {
	f := &FileFrecency{Entries: map[string]FrecencyEntry{
		"old.go":    {Count: 1, LastUsed: time.Now().Add(-60 * 24 * time.Hour)},
		"recent.go": {Count: 1, LastUsed: time.Now()},
	}}
	pruned := f.Prune(30 * 24 * time.Hour)
	assert.Equal(t, 1, pruned)
	assert.NotContains(t, f.Entries, "old.go")
	assert.Contains(t, f.Entries, "recent.go")
}

func TestFrecency_Prune_KeepsRecentEntries(t *testing.T) {
	f := &FileFrecency{Entries: map[string]FrecencyEntry{
		"a.go": {Count: 1, LastUsed: time.Now()},
		"b.go": {Count: 1, LastUsed: time.Now().Add(-1 * 24 * time.Hour)},
	}}
	pruned := f.Prune(30 * 24 * time.Hour)
	assert.Equal(t, 0, pruned)
	assert.Len(t, f.Entries, 2)
}

func TestFrecency_SaveLoad_Roundtrip(t *testing.T) {
	t.Setenv("RIPCODE_DIR", t.TempDir())
	f := &FileFrecency{Entries: make(map[string]FrecencyEntry)}
	f.Record("main.go")
	f.Record("main.go")
	f.Record("util.go")
	require.NoError(t, f.Save())

	loaded, err := LoadFrecency()
	require.NoError(t, err)
	assert.Equal(t, 2, loaded.Entries["main.go"].Count)
	assert.Equal(t, 1, loaded.Entries["util.go"].Count)
}

func TestFrecency_Score_FutureTimestamp(t *testing.T) {
	f := &FileFrecency{Entries: map[string]FrecencyEntry{
		"future.go": {Count: 3, LastUsed: time.Now().Add(24 * time.Hour)},
	}}
	score := f.Score("future.go")
	// Future timestamps are clamped to days=0, so recencyWeight=1.0, score=3.0
	assert.InDelta(t, 3.0, score, 0.1)
}

func TestFrecency_LoadFrecency_MissingFile(t *testing.T) {
	t.Setenv("RIPCODE_DIR", t.TempDir())
	f, err := LoadFrecency()
	require.NoError(t, err)
	assert.NotNil(t, f)
	assert.NotNil(t, f.Entries)
	assert.Empty(t, f.Entries)
}
