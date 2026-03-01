package store

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
)

// MCPServer represents a configured MCP server.
type MCPServer struct {
	Name    string `json:"name"`
	Command string `json:"command,omitempty"`
	URL     string `json:"url,omitempty"`
	Enabled bool   `json:"enabled"`
}

// MCPConfig holds MCP server configurations.
type MCPConfig struct {
	Servers []MCPServer `json:"servers"`
}

const mcpConfigFile = "mcp.json"

// LoadMCPConfig reads MCP configuration from disk.
// Returns empty config if the file does not exist.
func LoadMCPConfig() (*MCPConfig, error) {
	path := filepath.Join(StateDir(), mcpConfigFile)
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return &MCPConfig{}, nil
		}
		return nil, err
	}
	var c MCPConfig
	if err := json.Unmarshal(data, &c); err != nil {
		return &MCPConfig{}, nil
	}
	return &c, nil
}

// Save writes MCP configuration to disk.
func (c *MCPConfig) Save() error {
	dir := StateDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, mcpConfigFile), data, 0o644)
}

// ToggleEnabled toggles a server's enabled state by name.
// Returns the new enabled state. Returns false if not found.
func (c *MCPConfig) ToggleEnabled(name string) bool {
	for i := range c.Servers {
		if c.Servers[i].Name == name {
			c.Servers[i].Enabled = !c.Servers[i].Enabled
			return c.Servers[i].Enabled
		}
	}
	return false
}

// CountEnabled returns the number of enabled servers.
func (c *MCPConfig) CountEnabled() int {
	n := 0
	for _, s := range c.Servers {
		if s.Enabled {
			n++
		}
	}
	return n
}

// ByName returns a pointer to the server with the given name, or nil.
func (c *MCPConfig) ByName(name string) *MCPServer {
	for i := range c.Servers {
		if c.Servers[i].Name == name {
			return &c.Servers[i]
		}
	}
	return nil
}
