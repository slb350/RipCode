package store

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
	return loadState[LSPConfig](lspConfigFile, "LSP config")
}

// Save writes LSP configuration to disk.
func (c *LSPConfig) Save() error {
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
