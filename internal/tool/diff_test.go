package tool

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDiffInfo_TypeCreation(t *testing.T) {
	d := &DiffInfo{Path: "/tmp/test.go", Before: "old", After: "new"}
	assert.Equal(t, "/tmp/test.go", d.Path)
	assert.Equal(t, "old", d.Before)
	assert.Equal(t, "new", d.After)
	assert.False(t, d.Binary)
}

func TestEdit_Result_IncludesDiffInfo(t *testing.T) {
	e := NewEditTool()
	ctx := newTestCtx(t)
	path := filepath.Join(ctx.WorkDir, "test.go")
	require.NoError(t, os.WriteFile(path, []byte("func hello() {\n\treturn\n}\n"), 0644))

	result := e.Execute(ctx, `{"file_path":"`+path+`","old_string":"return","new_string":"return \"world\""}`)
	require.NoError(t, result.Error)
	require.NotNil(t, result.Diff)
	assert.Equal(t, path, result.Diff.Path)
	assert.Contains(t, result.Diff.Before, "return")
	assert.Contains(t, result.Diff.After, `return "world"`)
	assert.False(t, result.Diff.Binary)
}

func TestEdit_WhitespaceFlexible_IncludesDiffInfo(t *testing.T) {
	e := NewEditTool()
	ctx := newTestCtx(t)
	path := filepath.Join(ctx.WorkDir, "test.go")
	require.NoError(t, os.WriteFile(path, []byte("func hello() {\n\treturn\n}\n"), 0644))

	result := e.Execute(ctx, `{"file_path":"`+path+`","old_string":"    return","new_string":"    return \"world\""}`)
	require.NoError(t, result.Error)
	require.NotNil(t, result.Diff)
	assert.Contains(t, result.Diff.Before, "return")
	assert.Contains(t, result.Diff.After, `return "world"`)
}

func TestWrite_NewFile_EmptyBefore(t *testing.T) {
	w := NewWriteTool()
	ctx := newTestCtx(t)
	path := filepath.Join(ctx.WorkDir, "new.txt")

	result := w.Execute(ctx, `{"file_path":"`+path+`","content":"hello world"}`)
	require.NoError(t, result.Error)
	require.NotNil(t, result.Diff)
	assert.Equal(t, "", result.Diff.Before, "new file should have empty Before")
	assert.Equal(t, "hello world", result.Diff.After)
}

func TestWrite_Overwrite_CapturesBefore(t *testing.T) {
	w := NewWriteTool()
	ctx := newTestCtx(t)
	path := filepath.Join(ctx.WorkDir, "existing.txt")
	require.NoError(t, os.WriteFile(path, []byte("original"), 0644))

	result := w.Execute(ctx, `{"file_path":"`+path+`","content":"updated"}`)
	require.NoError(t, result.Error)
	require.NotNil(t, result.Diff)
	assert.Equal(t, "original", result.Diff.Before)
	assert.Equal(t, "updated", result.Diff.After)
}

func TestWrite_SymlinkTarget_RejectsViaOpenNoFollow(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlinks require elevated privileges on Windows")
	}
	ctx := newTestCtx(t)
	target := filepath.Join(ctx.WorkDir, "real.txt")
	link := filepath.Join(ctx.WorkDir, "link.txt")
	require.NoError(t, os.WriteFile(target, []byte("secret"), 0644))
	require.NoError(t, os.Symlink(target, link))

	w := NewWriteTool()
	result := w.Execute(ctx, `{"file_path":"`+link+`","content":"hacked"}`)
	// The write should fail because of symlink rejection in writeAtomic
	require.Error(t, result.Error)
	assert.Contains(t, result.Error.Error(), "symlink")
}

func TestDiffInfo_NilForNonEditWriteTools(t *testing.T) {
	// Read tool returns no DiffInfo
	r := NewReadTool()
	ctx := newTestCtx(t)
	path := filepath.Join(ctx.WorkDir, "test.txt")
	require.NoError(t, os.WriteFile(path, []byte("content"), 0644))
	result := r.Execute(ctx, `{"file_path":"`+path+`"}`)
	require.NoError(t, result.Error)
	assert.Nil(t, result.Diff)
}

func TestIsBinaryContent_WithNullBytes(t *testing.T) {
	data := []byte("hello\x00world")
	assert.True(t, isBinaryContent(data))
}

func TestIsBinaryContent_TextOnly(t *testing.T) {
	data := []byte("hello world\nthis is text\n")
	assert.False(t, isBinaryContent(data))
}

func TestIsBinaryContent_NullByteAfter8KB(t *testing.T) {
	data := make([]byte, 9000)
	for i := range data {
		data[i] = 'a'
	}
	data[8193] = 0 // Past 8KB boundary
	assert.False(t, isBinaryContent(data))
}

func TestIsBinaryContent_EmptyData(t *testing.T) {
	assert.False(t, isBinaryContent([]byte{}))
}

func TestEdit_BinaryFile_DiffInfoBinaryTrue(t *testing.T) {
	e := NewEditTool()
	ctx := newTestCtx(t)
	path := filepath.Join(ctx.WorkDir, "binary.bin")
	// Binary content with null byte
	content := "hello\x00world"
	require.NoError(t, os.WriteFile(path, []byte(content), 0644))

	result := e.Execute(ctx, `{"file_path":"`+path+`","old_string":"hello","new_string":"goodbye"}`)
	require.NoError(t, result.Error)
	require.NotNil(t, result.Diff)
	assert.True(t, result.Diff.Binary)
}

func TestCapDiffContent_UnderLimit(t *testing.T) {
	s := "short content"
	assert.Equal(t, s, capDiffContent(s))
}

func TestCapDiffContent_OverLimit(t *testing.T) {
	s := strings.Repeat("a", maxDiffContentSize+100)
	capped := capDiffContent(s)
	assert.Equal(t, maxDiffContentSize+len("\n[truncated]"), len(capped))
	assert.True(t, strings.HasSuffix(capped, "\n[truncated]"))
}

func TestEdit_EmptyAfter(t *testing.T) {
	e := NewEditTool()
	ctx := newTestCtx(t)
	path := filepath.Join(ctx.WorkDir, "test.go")
	require.NoError(t, os.WriteFile(path, []byte("remove me"), 0644))

	result := e.Execute(ctx, `{"file_path":"`+path+`","old_string":"remove me","new_string":""}`)
	require.NoError(t, result.Error)
	require.NotNil(t, result.Diff)
	assert.Equal(t, "remove me", result.Diff.Before)
	assert.Equal(t, "", result.Diff.After)
}

func TestWrite_ConsecutiveWrites_SecondSeesFirst(t *testing.T) {
	w := NewWriteTool()
	ctx := newTestCtx(t)
	path := filepath.Join(ctx.WorkDir, "file.txt")

	r1 := w.Execute(ctx, `{"file_path":"`+path+`","content":"first"}`)
	require.NoError(t, r1.Error)
	require.NotNil(t, r1.Diff)
	assert.Equal(t, "", r1.Diff.Before, "first write: no existing content")

	r2 := w.Execute(ctx, `{"file_path":"`+path+`","content":"second"}`)
	require.NoError(t, r2.Error)
	require.NotNil(t, r2.Diff)
	assert.Equal(t, "first", r2.Diff.Before, "second write: should see first write's content")
	assert.Equal(t, "second", r2.Diff.After)
}
