package store

import "fmt"

// MCPServer represents a configured MCP server.
type MCPServer struct {
	Name    string `json:"name"`
	Command string `json:"command,omitempty"`
	URL     string `json:"url,omitempty"`
	Enabled bool   `json:"enabled"`
}

// Valid returns an error if the server configuration is invalid.
func (s MCPServer) Valid() error {
	if s.Name == "" {
		return fmt.Errorf("server name is required")
	}
	if s.Command == "" && s.URL == "" {
		return fmt.Errorf("server %q requires command or url", s.Name)
	}
	if s.Command != "" && s.URL != "" {
		return fmt.Errorf("server %q cannot have both command and url", s.Name)
	}
	return nil
}

// MCPConfig holds MCP server configurations.
type MCPConfig struct {
	Servers []MCPServer `json:"servers"`
}

const mcpConfigFile = "mcp.json"

// LoadMCPConfig reads MCP configuration from disk.
// Returns empty config if the file does not exist.
func LoadMCPConfig() (*MCPConfig, error) {
	return loadState[MCPConfig](mcpConfigFile, "MCP config")
}

// Save writes MCP configuration to disk.
func (c *MCPConfig) Save() error {
	return saveState(mcpConfigFile, c)
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
