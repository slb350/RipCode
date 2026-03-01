package tool

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidatePath_WithinWorkDir(t *testing.T) {
	dir := t.TempDir()

	// Absolute path inside workDir
	absPath := filepath.Join(dir, "file.txt")
	require.NoError(t, os.WriteFile(absPath, []byte("x"), 0644))

	got, err := ValidatePath(absPath, dir, true)
	require.NoError(t, err)
	assert.Equal(t, absPath, got)

	// Relative path inside workDir
	got, err = ValidatePath("file.txt", dir, true)
	require.NoError(t, err)
	assert.Equal(t, absPath, got)
}

func TestValidatePath_SubdirWithinWorkDir(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "sub")
	require.NoError(t, os.MkdirAll(sub, 0755))
	f := filepath.Join(sub, "deep.txt")
	require.NoError(t, os.WriteFile(f, []byte("x"), 0644))

	got, err := ValidatePath(f, dir, true)
	require.NoError(t, err)
	assert.Equal(t, f, got)
}

func TestValidatePath_TraversalBlocked(t *testing.T) {
	dir := t.TempDir()
	traversal := filepath.Join(dir, "..", "..", "..", "etc", "passwd")

	_, err := ValidatePath(traversal, dir, false)
	assert.Error(t, err)

	var pathErr *PathError
	assert.ErrorAs(t, err, &pathErr)
}

func TestValidatePath_AbsoluteOutsideBlocked(t *testing.T) {
	dir := t.TempDir()
	outside := t.TempDir() // a different temp dir

	outsideFile := filepath.Join(outside, "secret.txt")
	require.NoError(t, os.WriteFile(outsideFile, []byte("secret"), 0644))

	_, err := ValidatePath(outsideFile, dir, true)
	assert.Error(t, err)

	var pathErr *PathError
	assert.ErrorAs(t, err, &pathErr)
}

func TestValidatePath_SymlinkInsideAllowed(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "real.txt")
	link := filepath.Join(dir, "link.txt")

	require.NoError(t, os.WriteFile(target, []byte("data"), 0644))
	require.NoError(t, os.Symlink(target, link))

	got, err := ValidatePath(link, dir, true)
	require.NoError(t, err)
	assert.Equal(t, link, got)
}

func TestValidatePath_SymlinkOutsideBlocked(t *testing.T) {
	dir := t.TempDir()
	outside := t.TempDir()
	target := filepath.Join(outside, "secret.txt")
	link := filepath.Join(dir, "escape.txt")

	require.NoError(t, os.WriteFile(target, []byte("secret"), 0644))
	require.NoError(t, os.Symlink(target, link))

	_, err := ValidatePath(link, dir, true)
	assert.Error(t, err)

	var pathErr *PathError
	assert.ErrorAs(t, err, &pathErr)
	assert.Contains(t, pathErr.Reason, "symlink target outside")
}

func TestValidatePath_NewFileInBoundary(t *testing.T) {
	dir := t.TempDir()
	newFile := filepath.Join(dir, "newfile.txt")

	got, err := ValidatePath(newFile, dir, false)
	require.NoError(t, err)
	assert.Equal(t, newFile, got)
}

func TestValidatePath_NewFileOutsideBoundary(t *testing.T) {
	dir := t.TempDir()
	outside := t.TempDir()
	newFile := filepath.Join(outside, "newfile.txt")

	_, err := ValidatePath(newFile, dir, false)
	assert.Error(t, err)
}

func TestValidatePath_EmptyWorkDir(t *testing.T) {
	_, err := ValidatePath("/some/file", "", false)
	assert.Error(t, err)

	var pathErr *PathError
	assert.ErrorAs(t, err, &pathErr)
	assert.Contains(t, pathErr.Reason, "empty work directory")
}

func TestValidatePath_MustExistMissing(t *testing.T) {
	dir := t.TempDir()
	missing := filepath.Join(dir, "nope.txt")

	_, err := ValidatePath(missing, dir, true)
	assert.Error(t, err)

	var pathErr *PathError
	assert.ErrorAs(t, err, &pathErr)
	assert.Contains(t, pathErr.Reason, "does not exist")
}

func TestValidatePath_NewFileWithNewParent(t *testing.T) {
	dir := t.TempDir()
	newFile := filepath.Join(dir, "a", "b", "c.txt")

	got, err := ValidatePath(newFile, dir, false)
	require.NoError(t, err)
	assert.Equal(t, newFile, got)
}

func TestValidatePath_WorkspaceRootAllowed(t *testing.T) {
	dir := t.TempDir()

	// The workspace root itself should be a valid path
	got, err := ValidatePath(dir, dir, true)
	require.NoError(t, err)
	assert.Equal(t, dir, got)
}

func TestValidatePath_DotPathAllowed(t *testing.T) {
	dir := t.TempDir()

	// "." resolves to workDir — should be allowed
	got, err := ValidatePath(".", dir, true)
	require.NoError(t, err)
	assert.Equal(t, dir, got)
}

func TestValidatePath_DotSlashPathAllowed(t *testing.T) {
	dir := t.TempDir()

	// "./" resolves to workDir — should be allowed
	got, err := ValidatePath("./", dir, true)
	require.NoError(t, err)
	assert.Equal(t, dir, got)
}
