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
	cfg, err := LoadLSPConfig()
	require.NoError(t, err)
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

	loaded, err := LoadLSPConfig()
	require.NoError(t, err)
	assert.Equal(t, cfg.Clients, loaded.Clients)
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
