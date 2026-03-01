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
	cfg, warns, err := LoadMCPConfig()
	require.NoError(t, err)
	assert.Empty(t, warns)
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

	loaded, warns, err := LoadMCPConfig()
	require.NoError(t, err)
	assert.Empty(t, warns)
	assert.Equal(t, cfg.Servers, loaded.Servers)
}

func TestLoadMCPConfig_CorruptedJSON_ReturnsDefaultsAndError(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("RIPCODE_DIR", dir)
	stateDir := filepath.Join(dir, "state")
	require.NoError(t, os.MkdirAll(stateDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(stateDir, "mcp.json"), []byte("{bad json"), 0o644))

	cfg, warns, err := LoadMCPConfig()
	assert.Error(t, err, "corrupted JSON should return an error")
	assert.Empty(t, warns)
	assert.NotNil(t, cfg, "should return usable defaults even on error")
	assert.Empty(t, cfg.Servers)
}

func TestLoadMCPConfig_ReadError_ReturnsDefaultsAndError(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("RIPCODE_DIR", dir)
	stateDir := filepath.Join(dir, "state")
	require.NoError(t, os.MkdirAll(stateDir, 0o755))
	filePath := filepath.Join(stateDir, "mcp.json")
	require.NoError(t, os.WriteFile(filePath, []byte(`{"servers":[]}`), 0o644))
	require.NoError(t, os.Chmod(filePath, 0o000))
	t.Cleanup(func() { os.Chmod(filePath, 0o644) })

	cfg, warns, err := LoadMCPConfig()
	assert.Error(t, err, "unreadable file should return an error")
	assert.Empty(t, warns)
	assert.NotNil(t, cfg, "should return usable defaults even on read error")
	assert.Empty(t, cfg.Servers)
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
	newState, found := cfg.ToggleEnabled("srv")
	assert.True(t, found)
	assert.False(t, newState)
	assert.False(t, cfg.Servers[0].Enabled)

	// Toggle back on
	newState, found = cfg.ToggleEnabled("srv")
	assert.True(t, found)
	assert.True(t, newState)
	assert.True(t, cfg.Servers[0].Enabled)
}

func TestMCPConfig_ToggleEnabled_NotFound(t *testing.T) {
	cfg := &MCPConfig{}
	newState, found := cfg.ToggleEnabled("nonexistent")
	assert.False(t, found, "should report not found")
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
	s, ok := cfg.ByName("srv2")
	require.True(t, ok)
	assert.Equal(t, "http://test", s.URL)
}

func TestMCPConfig_ByName_NotFound(t *testing.T) {
	cfg := &MCPConfig{
		Servers: []MCPServer{{Name: "srv1", Command: "cmd1"}},
	}
	_, ok := cfg.ByName("missing")
	assert.False(t, ok)
}

func TestMCPConfig_ByName_ReturnsCopy(t *testing.T) {
	cfg := &MCPConfig{
		Servers: []MCPServer{{Name: "srv1", Command: "cmd1", Enabled: true}},
	}
	s, ok := cfg.ByName("srv1")
	require.True(t, ok)
	s.Enabled = false // mutate the copy
	assert.True(t, cfg.Servers[0].Enabled, "original should be unmodified")
}

func TestLoadMCPConfig_InvalidServers_LogsWarning(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("RIPCODE_DIR", dir)
	cfg := &MCPConfig{
		Servers: []MCPServer{
			{Name: "good", Command: "echo", Enabled: true},
			{Name: "", Command: "bad"},                      // invalid: empty name
			{Name: "both", Command: "cmd", URL: "http://x"}, // invalid: both set
		},
	}
	require.NoError(t, cfg.Save())

	loaded, warns, err := LoadMCPConfig()
	require.NoError(t, err, "invalid entries should not prevent loading")
	assert.Len(t, loaded.Servers, 3, "all entries still loaded")
	assert.Len(t, warns, 2, "should return warnings for invalid entries")
	assert.Contains(t, warns[0], "name is required")
	assert.Contains(t, warns[1], "both")

	logData, readErr := os.ReadFile(filepath.Join(dir, "state", "errors.log"))
	require.NoError(t, readErr)
	logStr := string(logData)
	assert.Contains(t, logStr, "MCP config: invalid server")
	assert.Contains(t, logStr, "name is required")
	assert.Contains(t, logStr, "both")
}

func TestMCPServer_Valid_WithCommand(t *testing.T) {
	s := MCPServer{Name: "test", Command: "echo hello"}
	assert.NoError(t, s.Valid())
}

func TestMCPServer_Valid_WithURL(t *testing.T) {
	s := MCPServer{Name: "test", URL: "http://localhost:8080"}
	assert.NoError(t, s.Valid())
}

func TestMCPServer_Valid_EmptyName(t *testing.T) {
	s := MCPServer{Command: "echo"}
	err := s.Valid()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "name")
}

func TestMCPServer_Valid_NeitherCommandNorURL(t *testing.T) {
	s := MCPServer{Name: "test"}
	err := s.Valid()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "command or url")
}

func TestMCPServer_Valid_BothCommandAndURL(t *testing.T) {
	s := MCPServer{Name: "test", Command: "echo", URL: "http://localhost"}
	err := s.Valid()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "both")
}
