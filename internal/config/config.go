package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/joho/godotenv"
)

const (
	DefaultModel    = "z-ai/glm-5"
	DefaultMaxSteps = 100
)

// Config holds all runtime configuration for ripcode.
type Config struct {
	APIKey   string
	Model    string
	WorkDir  string
	MaxSteps int
	Warnings []string
}

// Load reads configuration from environment variables and the current
// directory's .env file. Env vars take priority over .env values.
func Load() (*Config, error) {
	wd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("resolve working directory: %w", err)
	}
	return LoadFrom(wd)
}

// LoadFrom reads configuration using dir as the working directory and
// the location to search for a .env file.
func LoadFrom(dir string) (*Config, error) {
	// Read .env if present — used as fallback when env vars are unset
	var warnings []string
	envPath := filepath.Join(dir, ".env")
	dotenv, err := godotenv.Read(envPath)
	if err != nil && !os.IsNotExist(err) {
		warnings = append(warnings, fmt.Sprintf(".env file malformed: %v", err))
		dotenv = make(map[string]string)
	}
	if dotenv == nil {
		dotenv = make(map[string]string)
	}

	apiKey := envLookup("OPENROUTER_API_KEY", dotenv)
	if apiKey == "" {
		return nil, fmt.Errorf("OPENROUTER_API_KEY is required (set via env or .env)")
	}

	model := envLookup("RIPCODE_MODEL", dotenv)
	if model == "" {
		model = DefaultModel
	}

	maxSteps := DefaultMaxSteps
	if raw := envLookup("RIPCODE_MAX_STEPS", dotenv); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil {
			return nil, fmt.Errorf("RIPCODE_MAX_STEPS must be a number: %w", err)
		}
		if n > 0 {
			maxSteps = n
		}
	}

	absDir, err := filepath.Abs(dir)
	if err != nil {
		return nil, fmt.Errorf("resolve working directory: %w", err)
	}

	return &Config{
		APIKey:   apiKey,
		Model:    model,
		WorkDir:  absDir,
		MaxSteps: maxSteps,
		Warnings: warnings,
	}, nil
}

// envLookup returns the value of key from the environment, falling back
// to the dotenv map if the env var is empty or unset.
func envLookup(key string, dotenv map[string]string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return dotenv[key]
}

// SaveAPIKey writes or updates the OPENROUTER_API_KEY in the .env file at dir.
// Creates the file if it doesn't exist. Preserves other entries.
func SaveAPIKey(dir, apiKey string) error {
	envPath := filepath.Join(dir, ".env")

	existing, _ := godotenv.Read(envPath)
	if existing == nil {
		existing = make(map[string]string)
	}
	existing["OPENROUTER_API_KEY"] = apiKey

	return godotenv.Write(existing, envPath)
}
