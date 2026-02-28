package tool

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWrite_ImplementsTool(t *testing.T) {
	var _ Tool = &WriteTool{}
}

func TestWrite_CreateNewFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "new.txt")

	w := NewWriteTool()
	ctx := newTestCtx(t)

	result := w.Execute(ctx, `{"file_path":"`+path+`","content":"hello world"}`)
	require.NoError(t, result.Error)
	assert.Contains(t, result.Output, "11") // bytes written

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, "hello world", string(data))
}

func TestWrite_OverwriteFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "existing.txt")
	require.NoError(t, os.WriteFile(path, []byte("old content"), 0644))

	w := NewWriteTool()
	ctx := newTestCtx(t)

	result := w.Execute(ctx, `{"file_path":"`+path+`","content":"new content"}`)
	require.NoError(t, result.Error)

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, "new content", string(data))
}

func TestWrite_CreatesParentDirs(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "a", "b", "c", "deep.txt")

	w := NewWriteTool()
	ctx := newTestCtx(t)

	result := w.Execute(ctx, `{"file_path":"`+path+`","content":"deep file"}`)
	require.NoError(t, result.Error)

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, "deep file", string(data))
}

func TestWrite_InvalidJSON(t *testing.T) {
	w := NewWriteTool()
	ctx := newTestCtx(t)

	result := w.Execute(ctx, `{bad}`)
	assert.Error(t, result.Error)
}

func TestWrite_MissingFilePath(t *testing.T) {
	w := NewWriteTool()
	ctx := newTestCtx(t)

	result := w.Execute(ctx, `{"content":"data"}`)
	assert.Error(t, result.Error)
}

func TestWrite_EmptyContent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.txt")

	w := NewWriteTool()
	ctx := newTestCtx(t)

	result := w.Execute(ctx, `{"file_path":"`+path+`","content":""}`)
	require.NoError(t, result.Error)

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, "", string(data))
}

func TestWrite_Parameters(t *testing.T) {
	w := NewWriteTool()
	params := w.Parameters()

	props := params["properties"].(map[string]any)
	assert.Contains(t, props, "file_path")
	assert.Contains(t, props, "content")

	required := params["required"].([]string)
	assert.Contains(t, required, "file_path")
	assert.Contains(t, required, "content")
}
