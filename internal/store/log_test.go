package store

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLogError_WritesToFile(t *testing.T) {
	testDir(t)
	LogError("save session", fmt.Errorf("disk full"))

	path := filepath.Join(StateDir(), "errors.log")
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	content := string(data)
	assert.Contains(t, content, "save session")
	assert.Contains(t, content, "disk full")
}

func TestLogError_AppendsMultipleEntries(t *testing.T) {
	testDir(t)
	LogError("first", fmt.Errorf("err1"))
	LogError("second", fmt.Errorf("err2"))

	path := filepath.Join(StateDir(), "errors.log")
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	assert.Len(t, lines, 2)
	assert.Contains(t, lines[0], "first")
	assert.Contains(t, lines[1], "second")
}

func TestLogError_CreatesStateDir(t *testing.T) {
	testDir(t)
	// StateDir doesn't exist yet
	LogError("test", fmt.Errorf("oops"))

	_, err := os.Stat(StateDir())
	assert.NoError(t, err)
}

func TestLogError_FallsBackToStderr(t *testing.T) {
	dir := testDir(t)
	// Make the state dir unwritable so file logging fails.
	stateDir := filepath.Join(dir, "state")
	require.NoError(t, os.MkdirAll(stateDir, 0o755))
	require.NoError(t, os.Chmod(stateDir, 0o444))
	t.Cleanup(func() { os.Chmod(stateDir, 0o755) })

	// Capture stderr.
	oldStderr := os.Stderr
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stderr = w

	LogError("test message", fmt.Errorf("test error"))

	w.Close()
	os.Stderr = oldStderr
	var buf bytes.Buffer
	_, err = io.Copy(&buf, r)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "[ripcode] (log file unavailable)")
	assert.Contains(t, output, "test message")
	assert.Contains(t, output, "test error")
}
