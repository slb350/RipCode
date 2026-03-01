package tool

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLs_ImplementsTool(t *testing.T) {
	var _ Tool = &LsTool{}
}

func TestLs_ListDir(t *testing.T) {
	l := NewLsTool()
	ctx := newTestCtx(t)
	require.NoError(t, os.WriteFile(filepath.Join(ctx.WorkDir, "file.go"), []byte("go"), 0644))
	require.NoError(t, os.MkdirAll(filepath.Join(ctx.WorkDir, "subdir"), 0755))

	result := l.Execute(ctx, `{"path":"`+ctx.WorkDir+`"}`)
	require.NoError(t, result.Error)
	assert.Contains(t, result.Output, "file.go")
	assert.Contains(t, result.Output, "subdir/")
}

func TestLs_HiddenFiles(t *testing.T) {
	l := NewLsTool()
	ctx := newTestCtx(t)
	require.NoError(t, os.WriteFile(filepath.Join(ctx.WorkDir, ".hidden"), []byte("x"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(ctx.WorkDir, "visible"), []byte("x"), 0644))

	// Without all flag — hidden files should be excluded
	result := l.Execute(ctx, `{"path":"`+ctx.WorkDir+`"}`)
	require.NoError(t, result.Error)
	assert.Contains(t, result.Output, "visible")
	assert.NotContains(t, result.Output, ".hidden")

	// With all flag — hidden files should be included
	result = l.Execute(ctx, `{"path":"`+ctx.WorkDir+`","all":true}`)
	require.NoError(t, result.Error)
	assert.Contains(t, result.Output, ".hidden")
	assert.Contains(t, result.Output, "visible")
}

func TestLs_SkipsGitDir(t *testing.T) {
	l := NewLsTool()
	ctx := newTestCtx(t)
	require.NoError(t, os.MkdirAll(filepath.Join(ctx.WorkDir, ".git"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(ctx.WorkDir, "main.go"), []byte("go"), 0644))

	result := l.Execute(ctx, `{"path":"`+ctx.WorkDir+`","all":true}`)
	require.NoError(t, result.Error)
	assert.NotContains(t, result.Output, ".git")
	assert.Contains(t, result.Output, "main.go")
}

func TestLs_EmptyDir(t *testing.T) {
	l := NewLsTool()
	ctx := newTestCtx(t)

	result := l.Execute(ctx, `{"path":"`+ctx.WorkDir+`"}`)
	require.NoError(t, result.Error)
	assert.Contains(t, result.Output, "empty")
}

func TestLs_DefaultPath(t *testing.T) {
	l := NewLsTool()
	ctx := newTestCtx(t)

	result := l.Execute(ctx, `{}`)
	require.NoError(t, result.Error)
}

func TestLs_InvalidJSON(t *testing.T) {
	l := NewLsTool()
	ctx := newTestCtx(t)

	result := l.Execute(ctx, `{bad}`)
	assert.Error(t, result.Error)
}

func TestLs_Parameters(t *testing.T) {
	l := NewLsTool()
	params := l.Parameters()

	props := params["properties"].(map[string]any)
	assert.Contains(t, props, "path")
	assert.Contains(t, props, "all")
}

func TestLs_PathTraversalBlocked(t *testing.T) {
	l := NewLsTool()
	ctx := newTestCtx(t)
	outside := t.TempDir()

	result := l.Execute(ctx, `{"path":"`+outside+`"}`)
	assert.Error(t, result.Error)
}
