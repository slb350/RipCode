package store

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestModelPrefs_SaveAndLoad_RoundTrips(t *testing.T) {
	t.Setenv("RIPCODE_DIR", t.TempDir())
	p := &ModelPrefs{}
	ref := ModelRef{ProviderID: "anthropic", ModelID: "anthropic/claude-4"}
	p.AddRecent(ref)
	p.ToggleFavorite(ref)
	p.SetVariant("anthropic/claude-4", "high")

	require.NoError(t, p.Save())

	loaded, err := LoadModelPrefs()
	require.NoError(t, err)
	assert.Equal(t, p.Recent, loaded.Recent)
	assert.Equal(t, p.Favorite, loaded.Favorite)
	assert.Equal(t, p.Variant, loaded.Variant)
}

func TestModelPrefs_AddRecent_Prepends(t *testing.T) {
	p := &ModelPrefs{}
	a := ModelRef{ProviderID: "a", ModelID: "a/m1"}
	b := ModelRef{ProviderID: "b", ModelID: "b/m2"}
	p.AddRecent(a)
	p.AddRecent(b)
	assert.Equal(t, b, p.Recent[0])
	assert.Equal(t, a, p.Recent[1])
}

func TestModelPrefs_AddRecent_LimitsTo10(t *testing.T) {
	p := &ModelPrefs{}
	for i := 0; i < 15; i++ {
		p.AddRecent(ModelRef{ProviderID: "p", ModelID: "p/m" + string(rune('a'+i))})
	}
	assert.Len(t, p.Recent, 10)
}

func TestModelPrefs_AddRecent_DeduplicatesExisting(t *testing.T) {
	p := &ModelPrefs{}
	a := ModelRef{ProviderID: "a", ModelID: "a/m1"}
	b := ModelRef{ProviderID: "b", ModelID: "b/m2"}
	p.AddRecent(a)
	p.AddRecent(b)
	p.AddRecent(a) // move a to front
	assert.Len(t, p.Recent, 2)
	assert.Equal(t, a, p.Recent[0])
	assert.Equal(t, b, p.Recent[1])
}

func TestModelPrefs_ToggleFavorite_AddsWhenAbsent(t *testing.T) {
	p := &ModelPrefs{}
	ref := ModelRef{ProviderID: "a", ModelID: "a/m1"}
	isFav := p.ToggleFavorite(ref)
	assert.True(t, isFav)
	assert.Len(t, p.Favorite, 1)
}

func TestModelPrefs_ToggleFavorite_RemovesWhenPresent(t *testing.T) {
	p := &ModelPrefs{}
	ref := ModelRef{ProviderID: "a", ModelID: "a/m1"}
	p.ToggleFavorite(ref)          // add
	isFav := p.ToggleFavorite(ref) // remove
	assert.False(t, isFav)
	assert.Empty(t, p.Favorite)
}

func TestModelPrefs_IsFavorite(t *testing.T) {
	p := &ModelPrefs{}
	ref := ModelRef{ProviderID: "a", ModelID: "a/m1"}
	assert.False(t, p.IsFavorite(ref))
	p.ToggleFavorite(ref)
	assert.True(t, p.IsFavorite(ref))
}

func TestModelPrefs_SetVariant(t *testing.T) {
	p := &ModelPrefs{}
	p.SetVariant("anthropic/claude-4", "high")
	assert.Equal(t, "high", p.GetVariant("anthropic/claude-4"))
}

func TestModelPrefs_GetVariant_EmptyReturnsEmpty(t *testing.T) {
	p := &ModelPrefs{}
	assert.Equal(t, "", p.GetVariant("nonexistent"))
}

func TestModelPrefs_Load_MissingFile_ReturnsEmpty(t *testing.T) {
	t.Setenv("RIPCODE_DIR", t.TempDir())
	p, err := LoadModelPrefs()
	require.NoError(t, err)
	assert.Empty(t, p.Recent)
	assert.Empty(t, p.Favorite)
}

func TestModelPrefs_AddRecent_AlreadyAtPosition0_NoOp(t *testing.T) {
	p := &ModelPrefs{}
	a := ModelRef{ProviderID: "a", ModelID: "a/m1"}
	b := ModelRef{ProviderID: "b", ModelID: "b/m2"}
	p.AddRecent(b)
	p.AddRecent(a)
	// a is already at position 0, adding again should be a no-op
	p.AddRecent(a)
	assert.Len(t, p.Recent, 2)
	assert.Equal(t, a, p.Recent[0])
	assert.Equal(t, b, p.Recent[1])
}

func TestLoadModelPrefs_CorruptedJSON_ReturnsDefaultsAndError(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("RIPCODE_DIR", dir)
	stateDir := filepath.Join(dir, "state")
	require.NoError(t, os.MkdirAll(stateDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(stateDir, "model.json"), []byte("{bad json"), 0o644))

	p, err := LoadModelPrefs()
	assert.Error(t, err, "corrupted JSON should return an error")
	assert.NotNil(t, p, "should return usable defaults even on error")
	assert.Empty(t, p.Recent)
	assert.Empty(t, p.Favorite)
}

func TestLoadModelPrefs_ReadError_ReturnsDefaultsAndError(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("RIPCODE_DIR", dir)
	stateDir := filepath.Join(dir, "state")
	require.NoError(t, os.MkdirAll(stateDir, 0o755))
	// Create file then make it unreadable
	filePath := filepath.Join(stateDir, "model.json")
	require.NoError(t, os.WriteFile(filePath, []byte(`{"recent":[]}`), 0o644))
	require.NoError(t, os.Chmod(filePath, 0o000))
	t.Cleanup(func() { os.Chmod(filePath, 0o644) })

	p, err := LoadModelPrefs()
	assert.Error(t, err, "unreadable file should return an error")
	assert.NotNil(t, p, "should return usable defaults even on read error")
	assert.Empty(t, p.Recent)
}

func TestModelPrefs_Save_CreatesStateDir(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("RIPCODE_DIR", dir)
	p := &ModelPrefs{}
	p.AddRecent(ModelRef{ProviderID: "a", ModelID: "a/m1"})
	require.NoError(t, p.Save())

	_, err := os.Stat(filepath.Join(dir, "state", "model.json"))
	assert.NoError(t, err)
}
