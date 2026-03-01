package tool

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGlob_ImplementsTool(t *testing.T) {
	var _ Tool = &GlobTool{}
}

func TestGlob_FindFiles(t *testing.T) {
	g := NewGlobTool()
	ctx := newTestCtx(t)
	require.NoError(t, os.WriteFile(filepath.Join(ctx.WorkDir, "a.go"), []byte("go"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(ctx.WorkDir, "b.go"), []byte("go"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(ctx.WorkDir, "c.txt"), []byte("txt"), 0644))

	result := g.Execute(ctx, `{"pattern":"*.go","path":"`+ctx.WorkDir+`"}`)
	require.NoError(t, result.Error)
	assert.Contains(t, result.Output, "a.go")
	assert.Contains(t, result.Output, "b.go")
	assert.NotContains(t, result.Output, "c.txt")
}

func TestGlob_RecursivePattern(t *testing.T) {
	g := NewGlobTool()
	ctx := newTestCtx(t)
	sub := filepath.Join(ctx.WorkDir, "sub")
	require.NoError(t, os.MkdirAll(sub, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(ctx.WorkDir, "top.go"), []byte("go"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(sub, "deep.go"), []byte("go"), 0644))

	result := g.Execute(ctx, `{"pattern":"**/*.go","path":"`+ctx.WorkDir+`"}`)
	require.NoError(t, result.Error)
	assert.Contains(t, result.Output, "top.go")
	assert.Contains(t, result.Output, "deep.go")
}

func TestGlob_NoMatches(t *testing.T) {
	g := NewGlobTool()
	ctx := newTestCtx(t)

	result := g.Execute(ctx, `{"pattern":"*.xyz","path":"`+ctx.WorkDir+`"}`)
	require.NoError(t, result.Error)
	assert.Contains(t, result.Output, "no matches")
}

func TestGlob_DefaultPath(t *testing.T) {
	g := NewGlobTool()
	ctx := newTestCtx(t)
	// When no path specified, uses ctx.WorkDir
	result := g.Execute(ctx, `{"pattern":"*"}`)
	require.NoError(t, result.Error)
}

func TestGlob_InvalidJSON(t *testing.T) {
	g := NewGlobTool()
	ctx := newTestCtx(t)

	result := g.Execute(ctx, `{bad}`)
	assert.Error(t, result.Error)
}

func TestGlob_SkipsGitDir(t *testing.T) {
	g := NewGlobTool()
	ctx := newTestCtx(t)
	gitDir := filepath.Join(ctx.WorkDir, ".git", "objects")
	require.NoError(t, os.MkdirAll(gitDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(gitDir, "pack.idx"), []byte("data"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(ctx.WorkDir, "main.go"), []byte("go"), 0644))

	result := g.Execute(ctx, `{"pattern":"**/*","path":"`+ctx.WorkDir+`"}`)
	require.NoError(t, result.Error)
	assert.Contains(t, result.Output, "main.go")
	assert.NotContains(t, result.Output, "pack.idx")
}

func TestGlob_Parameters(t *testing.T) {
	g := NewGlobTool()
	params := g.Parameters()

	props := params["properties"].(map[string]any)
	assert.Contains(t, props, "pattern")
	assert.Contains(t, props, "path")
}

func TestGlob_PathTraversalBlocked(t *testing.T) {
	g := NewGlobTool()
	ctx := newTestCtx(t)
	outside := t.TempDir()

	result := g.Execute(ctx, `{"pattern":"*","path":"`+outside+`"}`)
	assert.Error(t, result.Error)
}
