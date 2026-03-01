package tool

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRead_ImplementsTool(t *testing.T) {
	var _ Tool = &ReadTool{}
}

func TestRead_SimpleFile(t *testing.T) {
	r := NewReadTool()
	ctx := newTestCtx(t)
	path := filepath.Join(ctx.WorkDir, "test.txt")
	require.NoError(t, os.WriteFile(path, []byte("line one\nline two\nline three\n"), 0644))

	result := r.Execute(ctx, `{"file_path":"`+path+`"}`)
	require.NoError(t, result.Error)
	assert.Contains(t, result.Output, "1\tline one")
	assert.Contains(t, result.Output, "2\tline two")
	assert.Contains(t, result.Output, "3\tline three")
}

func TestRead_WithOffset(t *testing.T) {
	r := NewReadTool()
	ctx := newTestCtx(t)
	path := filepath.Join(ctx.WorkDir, "test.txt")
	content := "line1\nline2\nline3\nline4\nline5\n"
	require.NoError(t, os.WriteFile(path, []byte(content), 0644))

	result := r.Execute(ctx, `{"file_path":"`+path+`","offset":3}`)
	require.NoError(t, result.Error)
	assert.NotContains(t, result.Output, "line1")
	assert.NotContains(t, result.Output, "line2")
	assert.Contains(t, result.Output, "3\tline3")
}

func TestRead_WithLimit(t *testing.T) {
	r := NewReadTool()
	ctx := newTestCtx(t)
	path := filepath.Join(ctx.WorkDir, "test.txt")
	content := "line1\nline2\nline3\nline4\nline5\n"
	require.NoError(t, os.WriteFile(path, []byte(content), 0644))

	result := r.Execute(ctx, `{"file_path":"`+path+`","limit":2}`)
	require.NoError(t, result.Error)
	assert.Contains(t, result.Output, "1\tline1")
	assert.Contains(t, result.Output, "2\tline2")
	assert.NotContains(t, result.Output, "line3")
}

func TestRead_MissingFile(t *testing.T) {
	r := NewReadTool()
	ctx := newTestCtx(t)

	path := filepath.Join(ctx.WorkDir, "nonexistent.txt")
	result := r.Execute(ctx, `{"file_path":"`+path+`"}`)
	assert.Error(t, result.Error)
}

func TestRead_BinaryDetection(t *testing.T) {
	r := NewReadTool()
	ctx := newTestCtx(t)
	path := filepath.Join(ctx.WorkDir, "binary.bin")
	// Write bytes with null chars — indicates binary
	data := []byte{0x89, 0x50, 0x4E, 0x47, 0x00, 0x00, 0x00}
	require.NoError(t, os.WriteFile(path, data, 0644))

	result := r.Execute(ctx, `{"file_path":"`+path+`"}`)
	require.NoError(t, result.Error)
	assert.Contains(t, result.Output, "binary")
}

func TestRead_MaxLines(t *testing.T) {
	r := NewReadTool()
	ctx := newTestCtx(t)
	path := filepath.Join(ctx.WorkDir, "big.txt")

	var sb strings.Builder
	for i := 0; i < 3000; i++ {
		sb.WriteString("line\n")
	}
	require.NoError(t, os.WriteFile(path, []byte(sb.String()), 0644))

	result := r.Execute(ctx, `{"file_path":"`+path+`"}`)
	require.NoError(t, result.Error)
	// Should be capped at MaxReadLines
	lines := strings.Split(strings.TrimRight(result.Output, "\n"), "\n")
	assert.LessOrEqual(t, len(lines), MaxReadLines+5) // allow a few extra for notices
}

func TestRead_InvalidJSON(t *testing.T) {
	r := NewReadTool()
	ctx := newTestCtx(t)

	result := r.Execute(ctx, `{bad}`)
	assert.Error(t, result.Error)
}

func TestRead_EmptyFile(t *testing.T) {
	r := NewReadTool()
	ctx := newTestCtx(t)
	path := filepath.Join(ctx.WorkDir, "empty.txt")
	require.NoError(t, os.WriteFile(path, []byte{}, 0644))

	result := r.Execute(ctx, `{"file_path":"`+path+`"}`)
	require.NoError(t, result.Error)
	assert.Contains(t, result.Output, "empty")
}

func TestRead_Parameters(t *testing.T) {
	r := NewReadTool()
	params := r.Parameters()

	props := params["properties"].(map[string]any)
	assert.Contains(t, props, "file_path")
	assert.Contains(t, props, "offset")
	assert.Contains(t, props, "limit")
}

func TestRead_PathTraversalBlocked(t *testing.T) {
	r := NewReadTool()
	ctx := newTestCtx(t)

	result := r.Execute(ctx, `{"file_path":"../../../etc/passwd"}`)
	assert.Error(t, result.Error)
}

func TestRead_SymlinkOutsideBlocked(t *testing.T) {
	r := NewReadTool()
	ctx := newTestCtx(t)

	link := filepath.Join(ctx.WorkDir, "outside_link")
	require.NoError(t, os.Symlink("/etc/hosts", link))

	result := r.Execute(ctx, `{"file_path":"`+link+`"}`)
	assert.Error(t, result.Error)
	assert.Contains(t, result.Error.Error(), "outside")
}
