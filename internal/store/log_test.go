package store

import (
	"fmt"
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
