package store

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUIPrefs_LoadEmpty_ReturnsDefaults(t *testing.T) {
	t.Setenv("RIPCODE_DIR", t.TempDir())
	p, err := LoadUIPrefs()
	require.NoError(t, err)
	assert.False(t, p.GettingStartedDismissed)
	assert.Empty(t, p.CollapsedSections)
}

func TestUIPrefs_SaveLoad_RoundTrips(t *testing.T) {
	t.Setenv("RIPCODE_DIR", t.TempDir())
	p := &UIPrefs{
		CollapsedSections:       map[string]bool{"mcp": true, "lsp": false},
		GettingStartedDismissed: true,
	}
	require.NoError(t, p.Save())

	loaded, err := LoadUIPrefs()
	require.NoError(t, err)
	assert.Equal(t, p.CollapsedSections, loaded.CollapsedSections)
	assert.Equal(t, p.GettingStartedDismissed, loaded.GettingStartedDismissed)
}

func TestLoadUIPrefs_CorruptedJSON_ReturnsDefaultsAndError(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("RIPCODE_DIR", dir)
	stateDir := filepath.Join(dir, "state")
	require.NoError(t, os.MkdirAll(stateDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(stateDir, "ui.json"), []byte("{bad json"), 0o644))

	p, err := LoadUIPrefs()
	assert.Error(t, err, "corrupted JSON should return an error")
	assert.NotNil(t, p, "should return usable defaults even on error")
	assert.False(t, p.GettingStartedDismissed)
}

func TestLoadUIPrefs_ReadError_ReturnsDefaultsAndError(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("RIPCODE_DIR", dir)
	stateDir := filepath.Join(dir, "state")
	require.NoError(t, os.MkdirAll(stateDir, 0o755))
	filePath := filepath.Join(stateDir, "ui.json")
	require.NoError(t, os.WriteFile(filePath, []byte(`{}`), 0o644))
	require.NoError(t, os.Chmod(filePath, 0o000))
	t.Cleanup(func() { os.Chmod(filePath, 0o644) })

	p, err := LoadUIPrefs()
	assert.Error(t, err, "unreadable file should return an error")
	assert.NotNil(t, p, "should return usable defaults even on read error")
	assert.False(t, p.GettingStartedDismissed)
}

func TestUIPrefs_Save_CreatesStateDir(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("RIPCODE_DIR", dir)
	p := &UIPrefs{}
	require.NoError(t, p.Save())

	_, err := os.Stat(filepath.Join(dir, "state", "ui.json"))
	assert.NoError(t, err)
}

func TestUIPrefs_IsCollapsed_Default(t *testing.T) {
	p := &UIPrefs{}
	assert.False(t, p.IsCollapsed("mcp"))
}

func TestUIPrefs_ToggleCollapsed(t *testing.T) {
	p := &UIPrefs{}

	// Toggle on
	newState := p.ToggleCollapsed("mcp")
	assert.True(t, newState)
	assert.True(t, p.IsCollapsed("mcp"))

	// Toggle off
	newState = p.ToggleCollapsed("mcp")
	assert.False(t, newState)
	assert.False(t, p.IsCollapsed("mcp"))
}

func TestUIPrefs_ToggleCollapsed_InitializesMap(t *testing.T) {
	p := &UIPrefs{}
	assert.Nil(t, p.CollapsedSections)
	p.ToggleCollapsed("context")
	assert.NotNil(t, p.CollapsedSections)
}

func TestUIPrefs_DismissGettingStarted(t *testing.T) {
	p := &UIPrefs{}
	assert.False(t, p.GettingStartedDismissed)
	p.DismissGettingStarted()
	assert.True(t, p.GettingStartedDismissed)
}
