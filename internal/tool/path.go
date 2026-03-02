package tool

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// PathError represents a path validation failure.
type PathError struct {
	Path   string
	Reason string
}

func (e *PathError) Error() string {
	return fmt.Sprintf("path rejected: %s (%s)", e.Path, e.Reason)
}

// ValidatePath checks that path resolves to a location within workDir.
// If mustExist is true, the resolved path must exist.
// For existing paths, symlinks are resolved and the real target is checked.
// Returns the cleaned absolute path on success.
func ValidatePath(path, workDir string, mustExist bool) (string, error) {
	if workDir == "" {
		return "", &PathError{Path: path, Reason: "empty work directory"}
	}

	// Resolve workDir to absolute, real path
	absWorkDir, err := filepath.Abs(workDir)
	if err != nil {
		return "", &PathError{Path: path, Reason: fmt.Sprintf("resolve workdir: %v", err)}
	}
	realWorkDir, err := filepath.EvalSymlinks(absWorkDir)
	if err != nil {
		return "", &PathError{Path: path, Reason: fmt.Sprintf("resolve workdir symlinks: %v", err)}
	}

	// Make path absolute relative to workDir
	absPath := path
	if !filepath.IsAbs(path) {
		absPath = filepath.Join(absWorkDir, path)
	}
	absPath = filepath.Clean(absPath)

	// For existing paths, resolve symlinks and check real location
	if _, err := os.Lstat(absPath); err == nil {
		realPath, err := filepath.EvalSymlinks(absPath)
		if err != nil {
			return "", &PathError{Path: path, Reason: fmt.Sprintf("resolve symlinks: %v", err)}
		}
		if !isWithin(realPath, realWorkDir) && realPath != realWorkDir {
			return "", &PathError{Path: path, Reason: "symlink target outside work directory"}
		}
		return absPath, nil
	}

	if mustExist {
		return "", &PathError{Path: path, Reason: "file does not exist"}
	}

	// For new files, resolve the parent directory
	parent := filepath.Dir(absPath)
	if _, err := os.Lstat(parent); err == nil {
		realParent, err := filepath.EvalSymlinks(parent)
		if err != nil {
			return "", &PathError{Path: path, Reason: fmt.Sprintf("resolve parent symlinks: %v", err)}
		}
		if !isWithin(realParent, realWorkDir) && realParent != realWorkDir {
			return "", &PathError{Path: path, Reason: "parent directory outside work directory"}
		}
		return absPath, nil
	}

	// Parent doesn't exist yet — walk up to find an existing ancestor
	ancestor := parent
	for ancestor != filepath.Dir(ancestor) { // stop at root
		if _, err := os.Lstat(ancestor); err == nil {
			realAncestor, err := filepath.EvalSymlinks(ancestor)
			if err != nil {
				return "", &PathError{Path: path, Reason: fmt.Sprintf("resolve ancestor symlinks: %v", err)}
			}
			if !isWithin(realAncestor, realWorkDir) && realAncestor != realWorkDir {
				return "", &PathError{Path: path, Reason: "path outside work directory"}
			}
			return absPath, nil
		}
		ancestor = filepath.Dir(ancestor)
	}

	return "", &PathError{Path: path, Reason: "path outside work directory"}
}

// isWithin reports whether path is inside dir (but not equal to dir).
func isWithin(path, dir string) bool {
	path = filepath.Clean(path)
	dir = filepath.Clean(dir)
	if path == dir {
		return false
	}
	if dir == string(filepath.Separator) {
		return strings.HasPrefix(path, dir)
	}
	return strings.HasPrefix(path, dir+string(filepath.Separator))
}
