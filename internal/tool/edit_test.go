package tool

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEdit_ImplementsTool(t *testing.T) {
	var _ Tool = &EditTool{}
}

func TestEdit_SimpleReplace(t *testing.T) {
	e := NewEditTool()
	ctx := newTestCtx(t)
	path := filepath.Join(ctx.WorkDir, "test.go")
	require.NoError(t, os.WriteFile(path, []byte("func hello() {\n\treturn\n}\n"), 0644))

	result := e.Execute(ctx, `{"file_path":"`+path+`","old_string":"return","new_string":"return \"world\""}`)
	require.NoError(t, result.Error)

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Contains(t, string(data), `return "world"`)
	assert.NotContains(t, string(data), "\treturn\n")
}

func TestEdit_NonUniqueMatch(t *testing.T) {
	e := NewEditTool()
	ctx := newTestCtx(t)
	path := filepath.Join(ctx.WorkDir, "test.go")
	require.NoError(t, os.WriteFile(path, []byte("foo\nfoo\nbar\n"), 0644))

	result := e.Execute(ctx, `{"file_path":"`+path+`","old_string":"foo","new_string":"baz"}`)
	assert.Error(t, result.Error)
	assert.Contains(t, result.Error.Error(), "2 matches")
}

func TestEdit_EmptyOldString(t *testing.T) {
	e := NewEditTool()
	ctx := newTestCtx(t)
	path := filepath.Join(ctx.WorkDir, "test.go")
	require.NoError(t, os.WriteFile(path, []byte("content"), 0644))

	result := e.Execute(ctx, `{"file_path":"`+path+`","old_string":"","new_string":"replacement"}`)
	assert.Error(t, result.Error)
	assert.Contains(t, result.Error.Error(), "old_string")
}

func TestEdit_NoMatch(t *testing.T) {
	e := NewEditTool()
	ctx := newTestCtx(t)
	path := filepath.Join(ctx.WorkDir, "test.go")
	require.NoError(t, os.WriteFile(path, []byte("hello world\n"), 0644))

	result := e.Execute(ctx, `{"file_path":"`+path+`","old_string":"nonexistent","new_string":"replaced"}`)
	assert.Error(t, result.Error)
	assert.Contains(t, result.Error.Error(), "no match")
}

func TestEdit_WhitespaceFallback(t *testing.T) {
	e := NewEditTool()
	ctx := newTestCtx(t)
	path := filepath.Join(ctx.WorkDir, "test.go")
	// File has tabs
	require.NoError(t, os.WriteFile(path, []byte("func main() {\n\tfmt.Println(\"hello\")\n}\n"), 0644))

	// Provide with spaces instead of tabs — should still match via whitespace normalization
	result := e.Execute(ctx, `{"file_path":"`+path+`","old_string":"    fmt.Println(\"hello\")","new_string":"\tfmt.Println(\"world\")"}`)
	require.NoError(t, result.Error)

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Contains(t, string(data), `fmt.Println("world")`)
}

func TestEdit_MissingFile(t *testing.T) {
	e := NewEditTool()
	ctx := newTestCtx(t)

	path := filepath.Join(ctx.WorkDir, "nonexistent.go")
	result := e.Execute(ctx, `{"file_path":"`+path+`","old_string":"a","new_string":"b"}`)
	assert.Error(t, result.Error)
}

func TestEdit_InvalidJSON(t *testing.T) {
	e := NewEditTool()
	ctx := newTestCtx(t)

	result := e.Execute(ctx, `{bad}`)
	assert.Error(t, result.Error)
}

func TestEdit_MultiLineReplace(t *testing.T) {
	e := NewEditTool()
	ctx := newTestCtx(t)
	path := filepath.Join(ctx.WorkDir, "test.go")
	content := "line1\nline2\nline3\nline4\n"
	require.NoError(t, os.WriteFile(path, []byte(content), 0644))

	result := e.Execute(ctx, `{"file_path":"`+path+`","old_string":"line2\nline3","new_string":"replaced2\nreplaced3"}`)
	require.NoError(t, result.Error)

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, "line1\nreplaced2\nreplaced3\nline4\n", string(data))
}

func TestEdit_PreservesFilePermissions(t *testing.T) {
	e := NewEditTool()
	ctx := newTestCtx(t)
	path := filepath.Join(ctx.WorkDir, "script.sh")
	require.NoError(t, os.WriteFile(path, []byte("#!/bin/bash\necho old\n"), 0755))

	result := e.Execute(ctx, `{"file_path":"`+path+`","old_string":"echo old","new_string":"echo new"}`)
	require.NoError(t, result.Error)

	info, err := os.Stat(path)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0755), info.Mode().Perm())
}

func TestEdit_Parameters(t *testing.T) {
	e := NewEditTool()
	params := e.Parameters()

	props := params["properties"].(map[string]any)
	assert.Contains(t, props, "file_path")
	assert.Contains(t, props, "old_string")
	assert.Contains(t, props, "new_string")
}

func TestEdit_PathTraversalBlocked(t *testing.T) {
	e := NewEditTool()
	ctx := newTestCtx(t)

	result := e.Execute(ctx, `{"file_path":"../../../etc/passwd","old_string":"a","new_string":"b"}`)
	assert.Error(t, result.Error)
}

func TestEdit_SymlinkEditBlocked(t *testing.T) {
	e := NewEditTool()
	ctx := newTestCtx(t)
	outside := t.TempDir()
	target := filepath.Join(outside, "secret.txt")
	require.NoError(t, os.WriteFile(target, []byte("secret data"), 0644))

	link := filepath.Join(ctx.WorkDir, "link.txt")
	require.NoError(t, os.Symlink(target, link))

	result := e.Execute(ctx, `{"file_path":"`+link+`","old_string":"secret","new_string":"public"}`)
	assert.Error(t, result.Error)
}

func TestEdit_SymlinkWithinWorkDir_Blocked(t *testing.T) {
	e := NewEditTool()
	ctx := newTestCtx(t)

	// Create a real file and a symlink to it, both within workdir
	target := filepath.Join(ctx.WorkDir, "real.txt")
	require.NoError(t, os.WriteFile(target, []byte("original content"), 0644))

	link := filepath.Join(ctx.WorkDir, "link.txt")
	require.NoError(t, os.Symlink(target, link))

	// Edit via the symlink should be blocked (O_NOFOLLOW rejects symlinks)
	result := e.Execute(ctx, `{"file_path":"`+link+`","old_string":"original","new_string":"modified"}`)
	assert.Error(t, result.Error)
	assert.Contains(t, result.Error.Error(), "symlink")

	// Original file should be unchanged
	data, err := os.ReadFile(target)
	require.NoError(t, err)
	assert.Equal(t, "original content", string(data))
}

func TestEdit_SymlinkWrite_Blocked(t *testing.T) {
	// Verify that the write path (not just the read path) rejects symlinks.
	// This catches TOCTOU issues where a symlink is swapped in after the
	// read-open succeeds but before the write-open.
	e := NewEditTool()
	ctx := newTestCtx(t)

	// Create a real file and a symlink pointing to it, both within workdir.
	target := filepath.Join(ctx.WorkDir, "target.txt")
	require.NoError(t, os.WriteFile(target, []byte("original content"), 0644))

	link := filepath.Join(ctx.WorkDir, "link.txt")
	require.NoError(t, os.Symlink(target, link))

	// The edit tool should reject the symlink on the read-open,
	// never reaching the write path.
	result := e.Execute(ctx, `{"file_path":"`+link+`","old_string":"original","new_string":"modified"}`)
	assert.Error(t, result.Error)
	assert.Contains(t, result.Error.Error(), "symlink")

	// Target file should be unchanged.
	data, err := os.ReadFile(target)
	require.NoError(t, err)
	assert.Equal(t, "original content", string(data))
}

func TestEdit_MapNormPos_TabBoundary(t *testing.T) {
	orig := "\thello"
	norm := normalizeWhitespace(orig)
	assert.Equal(t, "    hello", norm)

	// Position mid-tab: normPos=2 is inside the 4-space expansion.
	pos := mapNormPos(orig, norm, 2)
	assert.LessOrEqual(t, pos, len(orig))

	// Position at end of normalized string.
	pos = mapNormPos(orig, norm, len(norm))
	assert.Equal(t, len(orig), pos)

	// Position 0.
	pos = mapNormPos(orig, norm, 0)
	assert.Equal(t, 0, pos)
}
