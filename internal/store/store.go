package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// Dir returns the ripcode data directory.
// Uses RIPCODE_DIR env var if set, otherwise ~/.ripcode.
// Falls back to ".ripcode" in the current directory if HOME cannot be resolved,
// and logs a warning via stderr since data may be written to an unexpected location.
func Dir() string {
	if dir := os.Getenv("RIPCODE_DIR"); dir != "" {
		return dir
	}
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		fmt.Fprintf(os.Stderr, "[ripcode] warning: cannot resolve home directory, using .ripcode in current directory\n")
		return ".ripcode"
	}
	return filepath.Join(home, ".ripcode")
}

// SessionsDir returns the sessions subdirectory.
func SessionsDir() string {
	return filepath.Join(Dir(), "sessions")
}

// StateDir returns the state subdirectory for persistent UI state.
func StateDir() string {
	return filepath.Join(Dir(), "state")
}

// loadState reads a JSON state file and unmarshals it into T.
// Returns a zero-value T if the file does not exist.
// Returns a zero-value T plus an error for read or parse failures.
func loadState[T any](filename, desc string) (*T, error) {
	path := filepath.Join(StateDir(), filename)
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return new(T), nil
		}
		return new(T), fmt.Errorf("read %s: %w", desc, err)
	}
	var v T
	if err := json.Unmarshal(data, &v); err != nil {
		return new(T), fmt.Errorf("parse %s: %w", desc, err)
	}
	return &v, nil
}

// atomicWrite writes data to path atomically via write-to-temp-then-rename.
// On POSIX, os.Rename is atomic, so the target is either intact or fully replaced.
func atomicWrite(path string, data []byte, perm os.FileMode) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, perm); err != nil {
		return fmt.Errorf("write temp file: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("rename temp file: %w", err)
	}
	return nil
}

// saveState marshals v as indented JSON and writes it to the state directory.
func saveState(filename string, v any) error {
	dir := StateDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return atomicWrite(filepath.Join(dir, filename), data, 0o644)
}
