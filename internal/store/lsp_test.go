package store

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLSPConfig_LoadEmpty_ReturnsEmpty(t *testing.T) {
	t.Setenv("RIPCODE_DIR", t.TempDir())
	cfg, warns, err := LoadLSPConfig()
	require.NoError(t, err)
	assert.Empty(t, warns)
	assert.Empty(t, cfg.Clients)
}

func TestLSPConfig_SaveLoad_RoundTrips(t *testing.T) {
	t.Setenv("RIPCODE_DIR", t.TempDir())
	cfg := &LSPConfig{
		Clients: []LSPClient{
			{Name: "gopls", Root: "/home/user/project", Enabled: true},
			{Name: "tsserver", Root: "/home/user/web", Enabled: false},
		},
	}
	require.NoError(t, cfg.Save())

	loaded, warns, err := LoadLSPConfig()
	require.NoError(t, err)
	assert.Empty(t, warns)
	assert.Equal(t, cfg.Clients, loaded.Clients)
}

func TestLoadLSPConfig_CorruptedJSON_ReturnsDefaultsAndError(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("RIPCODE_DIR", dir)
	stateDir := filepath.Join(dir, "state")
	require.NoError(t, os.MkdirAll(stateDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(stateDir, "lsp.json"), []byte("{bad json"), 0o644))

	cfg, warns, err := LoadLSPConfig()
	assert.Error(t, err, "corrupted JSON should return an error")
	assert.Empty(t, warns)
	assert.NotNil(t, cfg, "should return usable defaults even on error")
	assert.Empty(t, cfg.Clients)
}

func TestLoadLSPConfig_ReadError_ReturnsDefaultsAndError(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("RIPCODE_DIR", dir)
	stateDir := filepath.Join(dir, "state")
	require.NoError(t, os.MkdirAll(stateDir, 0o755))
	filePath := filepath.Join(stateDir, "lsp.json")
	require.NoError(t, os.WriteFile(filePath, []byte(`{"clients":[]}`), 0o644))
	require.NoError(t, os.Chmod(filePath, 0o000))
	t.Cleanup(func() { os.Chmod(filePath, 0o644) })

	cfg, warns, err := LoadLSPConfig()
	assert.Error(t, err, "unreadable file should return an error")
	assert.Empty(t, warns)
	assert.NotNil(t, cfg, "should return usable defaults even on read error")
	assert.Empty(t, cfg.Clients)
}

func TestLSPConfig_Save_CreatesStateDir(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("RIPCODE_DIR", dir)
	cfg := &LSPConfig{
		Clients: []LSPClient{{Name: "gopls", Root: "/tmp", Enabled: true}},
	}
	require.NoError(t, cfg.Save())

	_, err := os.Stat(filepath.Join(dir, "state", "lsp.json"))
	assert.NoError(t, err)
}

func TestLSPConfig_CountEnabled(t *testing.T) {
	cfg := &LSPConfig{
		Clients: []LSPClient{
			{Name: "a", Enabled: true},
			{Name: "b", Enabled: false},
			{Name: "c", Enabled: true},
		},
	}
	assert.Equal(t, 2, cfg.CountEnabled())
}

func TestLSPConfig_CountEnabled_Empty(t *testing.T) {
	cfg := &LSPConfig{}
	assert.Equal(t, 0, cfg.CountEnabled())
}

func TestLoadLSPConfig_InvalidClients_LogsWarning(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("RIPCODE_DIR", dir)
	// Write invalid config directly to disk (bypassing Save validation)
	// to test that Load handles corrupted files gracefully.
	raw := `{"clients":[
		{"name":"gopls","root":"/tmp","enabled":true},
		{"name":"","root":"/tmp"},
		{"name":"tsserver","root":"","enabled":true}
	]}`
	stateDir := filepath.Join(dir, "state")
	require.NoError(t, os.MkdirAll(stateDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(stateDir, "lsp.json"), []byte(raw), 0o644))

	loaded, warns, err := LoadLSPConfig()
	require.NoError(t, err, "invalid entries should not prevent loading")
	assert.Len(t, loaded.Clients, 3, "all entries still loaded")
	assert.Len(t, warns, 2, "should return warnings for invalid entries")
	assert.Contains(t, warns[0], "client name is required")
	assert.Contains(t, warns[1], "requires a root path")

	logData, readErr := os.ReadFile(filepath.Join(dir, "state", "errors.log"))
	require.NoError(t, readErr)
	logStr := string(logData)
	assert.Contains(t, logStr, "LSP config: invalid client")
	assert.Contains(t, logStr, "client name is required")
	assert.Contains(t, logStr, "requires a root path")
}

func TestLSPConfig_Save_RejectsInvalidClient(t *testing.T) {
	t.Setenv("RIPCODE_DIR", t.TempDir())
	cfg := &LSPConfig{
		Clients: []LSPClient{
			{Name: "gopls", Root: "/tmp", Enabled: true},
			{Name: "", Root: "/tmp"}, // invalid: empty name
		},
	}
	err := cfg.Save()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid client")
}

func TestLSPConfig_Save_RejectsEmptyRoot(t *testing.T) {
	t.Setenv("RIPCODE_DIR", t.TempDir())
	cfg := &LSPConfig{
		Clients: []LSPClient{
			{Name: "tsserver", Root: ""}, // invalid: empty root
		},
	}
	err := cfg.Save()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "requires a root path")
}

func TestLSPClient_Valid_EmptyName_ReturnsError(t *testing.T) {
	c := LSPClient{Name: "", Root: "/tmp"}
	err := c.Valid()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "client name is required")
}

func TestLSPClient_Valid_EmptyRoot_ReturnsError(t *testing.T) {
	c := LSPClient{Name: "gopls", Root: ""}
	err := c.Valid()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "requires a root path")
}

func TestLSPClient_Valid_ValidConfig_ReturnsNil(t *testing.T) {
	c := LSPClient{Name: "gopls", Root: "/home/user/project", Enabled: true}
	assert.NoError(t, c.Valid())
}
