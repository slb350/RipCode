package store

import "fmt"

// LSPClient represents a configured LSP client.
type LSPClient struct {
	Name    string `json:"name"`
	Root    string `json:"root"`
	Enabled bool   `json:"enabled"`
}

// Valid returns an error if the client configuration is invalid.
func (c LSPClient) Valid() error {
	if c.Name == "" {
		return fmt.Errorf("client name is required")
	}
	if c.Root == "" {
		return fmt.Errorf("client %q requires a root path", c.Name)
	}
	return nil
}

// LSPConfig holds LSP client configurations.
type LSPConfig struct {
	Clients []LSPClient `json:"clients"`
}

const lspConfigFile = "lsp.json"

// LoadLSPConfig reads LSP configuration from disk.
// Returns empty config if the file does not exist.
// Returns warnings for any invalid client entries found in the file.
func LoadLSPConfig() (*LSPConfig, []string, error) {
	cfg, err := loadState[LSPConfig](lspConfigFile, "LSP config")
	if err != nil {
		return cfg, nil, err
	}
	var warnings []string
	for _, c := range cfg.Clients {
		if verr := c.Valid(); verr != nil {
			LogError("LSP config: invalid client", verr)
			warnings = append(warnings, fmt.Sprintf("LSP client %q: %v", c.Name, verr))
		}
	}
	return cfg, warnings, nil
}

// Save writes LSP configuration to disk.
// Returns an error if any client entry is invalid.
func (c *LSPConfig) Save() error {
	for _, cl := range c.Clients {
		if err := cl.Valid(); err != nil {
			return fmt.Errorf("invalid client %q: %w", cl.Name, err)
		}
	}
	return saveState(lspConfigFile, c)
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
