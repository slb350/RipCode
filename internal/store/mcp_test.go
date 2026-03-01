package store

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMCPConfig_LoadEmpty_ReturnsEmpty(t *testing.T) {
	t.Setenv("RIPCODE_DIR", t.TempDir())
	cfg, err := LoadMCPConfig()
	require.NoError(t, err)
	assert.Empty(t, cfg.Servers)
}

func TestMCPConfig_SaveLoad_RoundTrips(t *testing.T) {
	t.Setenv("RIPCODE_DIR", t.TempDir())
	cfg := &MCPConfig{
		Servers: []MCPServer{
			{Name: "github", Command: "gh mcp", Enabled: true},
			{Name: "api", URL: "http://localhost:8080", Enabled: false},
		},
	}
	require.NoError(t, cfg.Save())

	loaded, err := LoadMCPConfig()
	require.NoError(t, err)
	assert.Equal(t, cfg.Servers, loaded.Servers)
}

func TestMCPConfig_Save_CreatesStateDir(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("RIPCODE_DIR", dir)
	cfg := &MCPConfig{
		Servers: []MCPServer{{Name: "test", Command: "echo", Enabled: true}},
	}
	require.NoError(t, cfg.Save())

	_, err := os.Stat(filepath.Join(dir, "state", "mcp.json"))
	assert.NoError(t, err)
}

func TestMCPConfig_ToggleEnabled(t *testing.T) {
	cfg := &MCPConfig{
		Servers: []MCPServer{
			{Name: "srv", Command: "cmd", Enabled: true},
		},
	}

	// Toggle off
	newState := cfg.ToggleEnabled("srv")
	assert.False(t, newState)
	assert.False(t, cfg.Servers[0].Enabled)

	// Toggle back on
	newState = cfg.ToggleEnabled("srv")
	assert.True(t, newState)
	assert.True(t, cfg.Servers[0].Enabled)
}

func TestMCPConfig_ToggleEnabled_NotFound(t *testing.T) {
	cfg := &MCPConfig{}
	newState := cfg.ToggleEnabled("nonexistent")
	assert.False(t, newState)
}

func TestMCPConfig_CountEnabled(t *testing.T) {
	cfg := &MCPConfig{
		Servers: []MCPServer{
			{Name: "a", Enabled: true},
			{Name: "b", Enabled: false},
			{Name: "c", Enabled: true},
		},
	}
	assert.Equal(t, 2, cfg.CountEnabled())
}

func TestMCPConfig_CountEnabled_Empty(t *testing.T) {
	cfg := &MCPConfig{}
	assert.Equal(t, 0, cfg.CountEnabled())
}

func TestMCPConfig_ByName_Found(t *testing.T) {
	cfg := &MCPConfig{
		Servers: []MCPServer{
			{Name: "srv1", Command: "cmd1", Enabled: true},
			{Name: "srv2", URL: "http://test", Enabled: false},
		},
	}
	s := cfg.ByName("srv2")
	require.NotNil(t, s)
	assert.Equal(t, "http://test", s.URL)
}

func TestMCPConfig_ByName_NotFound(t *testing.T) {
	cfg := &MCPConfig{
		Servers: []MCPServer{{Name: "srv1", Command: "cmd1"}},
	}
	assert.Nil(t, cfg.ByName("missing"))
}
