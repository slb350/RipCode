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
	w := NewWriteTool()
	ctx := newTestCtx(t)
	path := filepath.Join(ctx.WorkDir, "new.txt")

	result := w.Execute(ctx, `{"file_path":"`+path+`","content":"hello world"}`)
	require.NoError(t, result.Error)
	assert.Contains(t, result.Output, "11") // bytes written

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, "hello world", string(data))
}

func TestWrite_OverwriteFile(t *testing.T) {
	w := NewWriteTool()
	ctx := newTestCtx(t)
	path := filepath.Join(ctx.WorkDir, "existing.txt")
	require.NoError(t, os.WriteFile(path, []byte("old content"), 0644))

	result := w.Execute(ctx, `{"file_path":"`+path+`","content":"new content"}`)
	require.NoError(t, result.Error)

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, "new content", string(data))
}

func TestWrite_CreatesParentDirs(t *testing.T) {
	w := NewWriteTool()
	ctx := newTestCtx(t)
	path := filepath.Join(ctx.WorkDir, "a", "b", "c", "deep.txt")

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
	w := NewWriteTool()
	ctx := newTestCtx(t)
	path := filepath.Join(ctx.WorkDir, "empty.txt")

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

func TestWrite_PathTraversalBlocked(t *testing.T) {
	w := NewWriteTool()
	ctx := newTestCtx(t)

	result := w.Execute(ctx, `{"file_path":"../../../tmp/evil.txt","content":"pwned"}`)
	assert.Error(t, result.Error)
}

func TestWrite_SymlinkOverwriteBlocked(t *testing.T) {
	w := NewWriteTool()
	ctx := newTestCtx(t)
	outside := t.TempDir()
	target := filepath.Join(outside, "secret.txt")
	require.NoError(t, os.WriteFile(target, []byte("secret"), 0644))

	link := filepath.Join(ctx.WorkDir, "link.txt")
	require.NoError(t, os.Symlink(target, link))

	result := w.Execute(ctx, `{"file_path":"`+link+`","content":"overwritten"}`)
	assert.Error(t, result.Error)

	// Verify original file is untouched
	data, err := os.ReadFile(target)
	require.NoError(t, err)
	assert.Equal(t, "secret", string(data))
}

func TestWrite_AbsoluteOutsideBlocked(t *testing.T) {
	w := NewWriteTool()
	ctx := newTestCtx(t)
	outside := t.TempDir()
	path := filepath.Join(outside, "evil.txt")

	result := w.Execute(ctx, `{"file_path":"`+path+`","content":"pwned"}`)
	assert.Error(t, result.Error)
}
