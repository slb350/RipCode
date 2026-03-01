package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad_Defaults(t *testing.T) {
	// Clear any env vars that would interfere
	t.Setenv("OPENROUTER_API_KEY", "test-key")
	t.Setenv("RIPCODE_MODEL", "")
	t.Setenv("RIPCODE_MAX_STEPS", "")

	cfg, err := Load()
	require.NoError(t, err)

	assert.Equal(t, "test-key", cfg.APIKey)
	assert.Equal(t, DefaultModel, cfg.Model)
	assert.Equal(t, DefaultMaxSteps, cfg.MaxSteps)
	assert.NotEmpty(t, cfg.WorkDir)
}

func TestLoad_EnvOverrides(t *testing.T) {
	t.Setenv("OPENROUTER_API_KEY", "env-key-123")
	t.Setenv("RIPCODE_MODEL", "google/gemini-2.5-pro")
	t.Setenv("RIPCODE_MAX_STEPS", "50")

	cfg, err := Load()
	require.NoError(t, err)

	assert.Equal(t, "env-key-123", cfg.APIKey)
	assert.Equal(t, "google/gemini-2.5-pro", cfg.Model)
	assert.Equal(t, 50, cfg.MaxSteps)
}

func TestLoad_MissingAPIKey(t *testing.T) {
	t.Setenv("OPENROUTER_API_KEY", "")

	_, err := Load()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "OPENROUTER_API_KEY")
}

func TestLoad_InvalidMaxSteps(t *testing.T) {
	t.Setenv("OPENROUTER_API_KEY", "test-key")
	t.Setenv("RIPCODE_MAX_STEPS", "not-a-number")

	_, err := Load()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "RIPCODE_MAX_STEPS")
}

func TestLoad_DotEnvFile(t *testing.T) {
	// Create a temp dir with a .env file
	dir := t.TempDir()
	envFile := filepath.Join(dir, ".env")
	err := os.WriteFile(envFile, []byte("OPENROUTER_API_KEY=dotenv-key-456\n"), 0644)
	require.NoError(t, err)

	// Clear env so .env file is the source
	t.Setenv("OPENROUTER_API_KEY", "")

	cfg, err := LoadFrom(dir)
	require.NoError(t, err)
	assert.Equal(t, "dotenv-key-456", cfg.APIKey)
}

func TestLoad_EnvOverridesDotEnv(t *testing.T) {
	// Create a .env file
	dir := t.TempDir()
	envFile := filepath.Join(dir, ".env")
	err := os.WriteFile(envFile, []byte("OPENROUTER_API_KEY=dotenv-key\n"), 0644)
	require.NoError(t, err)

	// Set env var — should take priority
	t.Setenv("OPENROUTER_API_KEY", "env-key-wins")

	cfg, err := LoadFrom(dir)
	require.NoError(t, err)
	assert.Equal(t, "env-key-wins", cfg.APIKey)
}

func TestLoad_WorkDirResolved(t *testing.T) {
	t.Setenv("OPENROUTER_API_KEY", "test-key")

	cfg, err := Load()
	require.NoError(t, err)

	// WorkDir should be an absolute path
	assert.True(t, filepath.IsAbs(cfg.WorkDir))
}

func TestLoad_MaxStepsZeroUsesDefault(t *testing.T) {
	t.Setenv("OPENROUTER_API_KEY", "test-key")
	t.Setenv("RIPCODE_MAX_STEPS", "0")

	cfg, err := Load()
	require.NoError(t, err)
	assert.Equal(t, DefaultMaxSteps, cfg.MaxSteps)
}

func TestSaveAPIKey_WritesEnvFile(t *testing.T) {
	dir := t.TempDir()
	err := SaveAPIKey(dir, "sk-new-key-123")
	require.NoError(t, err)

	// Verify the key can be read back
	loaded, _ := godotenv.Read(filepath.Join(dir, ".env"))
	assert.Equal(t, "sk-new-key-123", loaded["OPENROUTER_API_KEY"])
}

func TestSaveAPIKey_UpdatesExistingKey(t *testing.T) {
	dir := t.TempDir()
	envFile := filepath.Join(dir, ".env")
	err := os.WriteFile(envFile, []byte("OPENROUTER_API_KEY=old-key\nRIPCODE_MODEL=test\n"), 0644)
	require.NoError(t, err)

	err = SaveAPIKey(dir, "sk-new-key")
	require.NoError(t, err)

	loaded, _ := godotenv.Read(envFile)
	assert.Equal(t, "sk-new-key", loaded["OPENROUTER_API_KEY"])
	assert.Equal(t, "test", loaded["RIPCODE_MODEL"])
}

func TestSaveAPIKey_CreatesEnvFile_WhenMissing(t *testing.T) {
	dir := t.TempDir()
	err := SaveAPIKey(dir, "sk-brand-new")
	require.NoError(t, err)

	loaded, _ := godotenv.Read(filepath.Join(dir, ".env"))
	assert.Equal(t, "sk-brand-new", loaded["OPENROUTER_API_KEY"])
}

func TestLoad_MalformedEnvWarning(t *testing.T) {
	dir := t.TempDir()
	envFile := filepath.Join(dir, ".env")
	// Write malformed .env (no = separator)
	require.NoError(t, os.WriteFile(envFile, []byte("BROKEN LINE WITHOUT EQUALS\n"), 0644))

	// Need API key from env var since .env is broken
	t.Setenv("OPENROUTER_API_KEY", "test-key")

	cfg, err := LoadFrom(dir)
	require.NoError(t, err)
	assert.NotEmpty(t, cfg.Warnings, "should have warning for malformed .env")
	assert.Contains(t, cfg.Warnings[0], ".env")
}

func TestLoad_MissingEnvNoWarning(t *testing.T) {
	dir := t.TempDir()
	// No .env file at all
	t.Setenv("OPENROUTER_API_KEY", "test-key")

	cfg, err := LoadFrom(dir)
	require.NoError(t, err)
	assert.Empty(t, cfg.Warnings, "missing .env should not produce warning")
}
