package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// ModelRef identifies a model by its provider and full model ID.
type ModelRef struct {
	ProviderID string `json:"providerID"`
	ModelID    string `json:"modelID"`
}

// ModelPrefs holds user preferences for models: recents, favorites, variants.
type ModelPrefs struct {
	Recent   []ModelRef        `json:"recent"`
	Favorite []ModelRef        `json:"favorite"`
	Variant  map[string]string `json:"variant"`
}

const modelPrefsFile = "model.json"
const maxRecent = 10

// LoadModelPrefs reads model preferences from disk.
// Returns empty prefs if the file does not exist.
func LoadModelPrefs() (*ModelPrefs, error) {
	path := filepath.Join(StateDir(), modelPrefsFile)
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return &ModelPrefs{}, nil
		}
		return nil, err
	}
	var p ModelPrefs
	if err := json.Unmarshal(data, &p); err != nil {
		return &ModelPrefs{}, fmt.Errorf("parse model preferences: %w", err)
	}
	return &p, nil
}

// Save writes model preferences to disk.
func (p *ModelPrefs) Save() error {
	dir := StateDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, modelPrefsFile), data, 0o644)
}

// AddRecent prepends a model to the recents list, deduplicating and limiting to 10.
func (p *ModelPrefs) AddRecent(ref ModelRef) {
	if len(p.Recent) > 0 && p.Recent[0] == ref {
		return
	}
	filtered := make([]ModelRef, 0, len(p.Recent))
	for _, r := range p.Recent {
		if r != ref {
			filtered = append(filtered, r)
		}
	}
	p.Recent = append([]ModelRef{ref}, filtered...)
	if len(p.Recent) > maxRecent {
		p.Recent = p.Recent[:maxRecent]
	}
}

// ToggleFavorite adds or removes a model from favorites.
// Returns true if the model is a favorite after the toggle.
func (p *ModelPrefs) ToggleFavorite(ref ModelRef) bool {
	for i, f := range p.Favorite {
		if f == ref {
			p.Favorite = append(p.Favorite[:i], p.Favorite[i+1:]...)
			return false
		}
	}
	p.Favorite = append(p.Favorite, ref)
	return true
}

// IsFavorite returns whether the given model is a favorite.
func (p *ModelPrefs) IsFavorite(ref ModelRef) bool {
	for _, f := range p.Favorite {
		if f == ref {
			return true
		}
	}
	return false
}

// SetVariant sets the variant for a model ID.
func (p *ModelPrefs) SetVariant(modelID, variant string) {
	if p.Variant == nil {
		p.Variant = make(map[string]string)
	}
	if variant == "" {
		delete(p.Variant, modelID)
	} else {
		p.Variant[modelID] = variant
	}
}

// GetVariant returns the variant for a model ID, or empty string if unset.
func (p *ModelPrefs) GetVariant(modelID string) string {
	if p.Variant == nil {
		return ""
	}
	return p.Variant[modelID]
}
