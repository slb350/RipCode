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
// Returns warnings for any invalid server entries found in the file.
func LoadMCPConfig() (*MCPConfig, []string, error) {
	cfg, err := loadState[MCPConfig](mcpConfigFile, "MCP config")
	if err != nil {
		return cfg, nil, err
	}
	var warnings []string
	for _, s := range cfg.Servers {
		if verr := s.Valid(); verr != nil {
			LogError("MCP config: invalid server", verr)
			warnings = append(warnings, fmt.Sprintf("MCP server %q: %v", s.Name, verr))
		}
	}
	return cfg, warnings, nil
}

// Save writes MCP configuration to disk.
// Returns an error if any server entry is invalid.
func (c *MCPConfig) Save() error {
	for _, s := range c.Servers {
		if err := s.Valid(); err != nil {
			return fmt.Errorf("invalid server %q: %w", s.Name, err)
		}
	}
	return saveState(mcpConfigFile, c)
}

// ToggleEnabled toggles a server's enabled state by name.
// Returns the new enabled state and whether the server was found.
func (c *MCPConfig) ToggleEnabled(name string) (enabled, found bool) {
	for i := range c.Servers {
		if c.Servers[i].Name == name {
			c.Servers[i].Enabled = !c.Servers[i].Enabled
			return c.Servers[i].Enabled, true
		}
	}
	return false, false
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

// ByName returns a copy of the server with the given name.
// Returns the server and true if found, or a zero value and false if not found.
func (c *MCPConfig) ByName(name string) (MCPServer, bool) {
	for _, s := range c.Servers {
		if s.Name == name {
			return s, true
		}
	}
	return MCPServer{}, false
}
