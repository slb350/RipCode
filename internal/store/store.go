package store

import (
	"os"
	"path/filepath"
)

// Dir returns the ripcode data directory.
// Uses RIPCODE_DIR env var if set, otherwise ~/.ripcode.
// Falls back to ".ripcode" in the current directory if HOME cannot be resolved.
func Dir() string {
	if dir := os.Getenv("RIPCODE_DIR"); dir != "" {
		return dir
	}
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
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
