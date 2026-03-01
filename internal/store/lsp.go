package store

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
)

// LSPClient represents a configured LSP client.
type LSPClient struct {
	Name    string `json:"name"`
	Root    string `json:"root"`
	Enabled bool   `json:"enabled"`
}

// LSPConfig holds LSP client configurations.
type LSPConfig struct {
	Clients []LSPClient `json:"clients"`
}

const lspConfigFile = "lsp.json"

// LoadLSPConfig reads LSP configuration from disk.
// Returns empty config if the file does not exist.
func LoadLSPConfig() (*LSPConfig, error) {
	path := filepath.Join(StateDir(), lspConfigFile)
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return &LSPConfig{}, nil
		}
		return nil, err
	}
	var c LSPConfig
	if err := json.Unmarshal(data, &c); err != nil {
		return &LSPConfig{}, nil
	}
	return &c, nil
}

// Save writes LSP configuration to disk.
func (c *LSPConfig) Save() error {
	dir := StateDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, lspConfigFile), data, 0o644)
}

// CountEnabled returns the number of enabled clients.
func (c *LSPConfig) CountEnabled() int {
	n := 0
	for _, cl := range c.Clients {
		if cl.Enabled {
			n++
		}
	}
	return n
}
