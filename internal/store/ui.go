package store

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
)

// UIPrefs holds UI preferences like collapsed sidebar sections.
type UIPrefs struct {
	CollapsedSections       map[string]bool `json:"collapsed_sections"`
	GettingStartedDismissed bool            `json:"getting_started_dismissed"`
}

const uiPrefsFile = "ui.json"

// LoadUIPrefs reads UI preferences from disk.
// Returns defaults if the file does not exist.
func LoadUIPrefs() (*UIPrefs, error) {
	path := filepath.Join(StateDir(), uiPrefsFile)
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return &UIPrefs{}, nil
		}
		return nil, err
	}
	var p UIPrefs
	if err := json.Unmarshal(data, &p); err != nil {
		return &UIPrefs{}, nil
	}
	return &p, nil
}

// Save writes UI preferences to disk.
func (p *UIPrefs) Save() error {
	dir := StateDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, uiPrefsFile), data, 0o644)
}

// IsCollapsed returns whether a sidebar section is collapsed.
func (p *UIPrefs) IsCollapsed(section string) bool {
	if p.CollapsedSections == nil {
		return false
	}
	return p.CollapsedSections[section]
}

// ToggleCollapsed toggles a section's collapsed state.
// Returns the new collapsed state.
func (p *UIPrefs) ToggleCollapsed(section string) bool {
	if p.CollapsedSections == nil {
		p.CollapsedSections = make(map[string]bool)
	}
	p.CollapsedSections[section] = !p.CollapsedSections[section]
	return p.CollapsedSections[section]
}

// DismissGettingStarted marks the getting started card as dismissed.
func (p *UIPrefs) DismissGettingStarted() {
	p.GettingStartedDismissed = true
}
