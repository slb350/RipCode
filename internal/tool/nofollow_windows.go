//go:build windows

package tool

import (
	"fmt"
	"os"
)

// OpenNoFollow opens a file with the given flags, rejecting symlinks.
// Windows lacks O_NOFOLLOW in Go's syscall package, so this uses a
// best-effort Lstat check (TOCTOU race is possible but unlikely).
func OpenNoFollow(path string, flags int, perm os.FileMode) (*os.File, error) {
	if info, err := os.Lstat(path); err == nil {
		if info.Mode()&os.ModeSymlink != 0 {
			return nil, fmt.Errorf("refusing to follow symlink: %s", path)
		}
	} else if !os.IsNotExist(err) {
		return nil, err
	}

	return os.OpenFile(path, flags, perm)
}
