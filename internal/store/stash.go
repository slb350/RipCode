package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// StashFileEntry is a single stash entry for persistence.
type StashFileEntry struct {
	ID      string `json:"id"`
	Content string `json:"content"`
}

func stashPath() string {
	return filepath.Join(StateDir(), "stash.json")
}

// LoadStash reads all stash entries from disk.
// Returns empty slice if the file does not exist.
func LoadStash() ([]StashFileEntry, error) {
	data, err := os.ReadFile(stashPath())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read stash: %w", err)
	}
	if len(data) == 0 {
		return nil, nil
	}

	var entries []StashFileEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, fmt.Errorf("unmarshal stash: %w", err)
	}
	return entries, nil
}

// SaveStash writes all stash entries to disk as JSON.
func SaveStash(entries []StashFileEntry) error {
	dir := StateDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create state dir: %w", err)
	}

	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal stash: %w", err)
	}
	return os.WriteFile(stashPath(), data, 0o644)
}
