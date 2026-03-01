package tool

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGrep_ImplementsTool(t *testing.T) {
	var _ Tool = &GrepTool{}
}

func TestGrep_SimpleMatch(t *testing.T) {
	g := NewGrepTool()
	ctx := newTestCtx(t)
	require.NoError(t, os.WriteFile(filepath.Join(ctx.WorkDir, "test.go"), []byte("func main() {\n\tfmt.Println(\"hello\")\n}\n"), 0644))

	result := g.Execute(ctx, `{"pattern":"Println","path":"`+ctx.WorkDir+`"}`)
	require.NoError(t, result.Error)
	assert.Contains(t, result.Output, "test.go")
	assert.Contains(t, result.Output, "Println")
}

func TestGrep_RegexMatch(t *testing.T) {
	g := NewGrepTool()
	ctx := newTestCtx(t)
	require.NoError(t, os.WriteFile(filepath.Join(ctx.WorkDir, "test.go"), []byte("func hello() {}\nfunc world() {}\n"), 0644))

	result := g.Execute(ctx, `{"pattern":"func \\w+\\(","path":"`+ctx.WorkDir+`"}`)
	require.NoError(t, result.Error)
	assert.Contains(t, result.Output, "func hello(")
	assert.Contains(t, result.Output, "func world(")
}

func TestGrep_NoMatch(t *testing.T) {
	g := NewGrepTool()
	ctx := newTestCtx(t)
	require.NoError(t, os.WriteFile(filepath.Join(ctx.WorkDir, "test.go"), []byte("hello\n"), 0644))

	result := g.Execute(ctx, `{"pattern":"zzzzz","path":"`+ctx.WorkDir+`"}`)
	require.NoError(t, result.Error)
	assert.Contains(t, result.Output, "no matches")
}

func TestGrep_IncludeFilter(t *testing.T) {
	g := NewGrepTool()
	ctx := newTestCtx(t)
	require.NoError(t, os.WriteFile(filepath.Join(ctx.WorkDir, "a.go"), []byte("hello\n"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(ctx.WorkDir, "b.txt"), []byte("hello\n"), 0644))

	result := g.Execute(ctx, `{"pattern":"hello","path":"`+ctx.WorkDir+`","include":"*.go"}`)
	require.NoError(t, result.Error)
	assert.Contains(t, result.Output, "a.go")
	assert.NotContains(t, result.Output, "b.txt")
}

func TestGrep_MultipleFiles(t *testing.T) {
	g := NewGrepTool()
	ctx := newTestCtx(t)
	require.NoError(t, os.WriteFile(filepath.Join(ctx.WorkDir, "a.go"), []byte("TODO: fix\n"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(ctx.WorkDir, "b.go"), []byte("TODO: add\n"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(ctx.WorkDir, "c.go"), []byte("done\n"), 0644))

	result := g.Execute(ctx, `{"pattern":"TODO","path":"`+ctx.WorkDir+`"}`)
	require.NoError(t, result.Error)
	assert.Contains(t, result.Output, "a.go")
	assert.Contains(t, result.Output, "b.go")
	assert.NotContains(t, result.Output, "c.go")
}

func TestGrep_InvalidJSON(t *testing.T) {
	g := NewGrepTool()
	ctx := newTestCtx(t)

	result := g.Execute(ctx, `{bad}`)
	assert.Error(t, result.Error)
}

func TestGrep_SkipsGitDir(t *testing.T) {
	g := NewGrepTool()
	ctx := newTestCtx(t)
	gitDir := filepath.Join(ctx.WorkDir, ".git")
	require.NoError(t, os.MkdirAll(gitDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(gitDir, "config"), []byte("TODO\n"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(ctx.WorkDir, "main.go"), []byte("TODO\n"), 0644))

	result := g.Execute(ctx, `{"pattern":"TODO","path":"`+ctx.WorkDir+`"}`)
	require.NoError(t, result.Error)
	assert.Contains(t, result.Output, "main.go")
	assert.NotContains(t, result.Output, ".git")
}

func TestGrep_SkipsBinaryFiles(t *testing.T) {
	g := NewGrepTool()
	ctx := newTestCtx(t)
	require.NoError(t, os.WriteFile(filepath.Join(ctx.WorkDir, "binary.bin"), []byte("TODO\x00binary"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(ctx.WorkDir, "text.go"), []byte("TODO\n"), 0644))

	result := g.Execute(ctx, `{"pattern":"TODO","path":"`+ctx.WorkDir+`"}`)
	require.NoError(t, result.Error)
	assert.Contains(t, result.Output, "text.go")
	assert.NotContains(t, result.Output, "binary.bin")
}

func TestGrep_Parameters(t *testing.T) {
	g := NewGrepTool()
	params := g.Parameters()

	props := params["properties"].(map[string]any)
	assert.Contains(t, props, "pattern")
	assert.Contains(t, props, "path")
	assert.Contains(t, props, "include")
}

func TestGrep_PathTraversalBlocked(t *testing.T) {
	g := NewGrepTool()
	ctx := newTestCtx(t)
	outside := t.TempDir()

	result := g.Execute(ctx, `{"pattern":"TODO","path":"`+outside+`"}`)
	assert.Error(t, result.Error)
}

func TestGrep_SkipErrors_ReportsCount(t *testing.T) {
	g := NewGrepTool()
	ctx := newTestCtx(t)
	require.NoError(t, os.WriteFile(filepath.Join(ctx.WorkDir, "ok.go"), []byte("TODO fix\n"), 0644))
	badDir := filepath.Join(ctx.WorkDir, "noperm")
	require.NoError(t, os.MkdirAll(badDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(badDir, "hidden.go"), []byte("TODO hidden\n"), 0644))
	require.NoError(t, os.Chmod(badDir, 0000))
	t.Cleanup(func() { os.Chmod(badDir, 0755) })

	result := g.Execute(ctx, `{"pattern":"TODO","path":"`+ctx.WorkDir+`"}`)
	require.NoError(t, result.Error)
	assert.Contains(t, result.Output, "ok.go")
	assert.Contains(t, result.Output, "skipped due to errors")
}
