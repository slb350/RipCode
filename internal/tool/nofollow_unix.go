//go:build !windows

package tool

import (
	"errors"
	"fmt"
	"os"
	"syscall"
)

// OpenNoFollow opens a file with the given flags, rejecting symlinks atomically.
// On Unix, O_NOFOLLOW prevents the kernel from following symlinks at the target path.
func OpenNoFollow(path string, flags int, perm os.FileMode) (*os.File, error) {
	f, err := os.OpenFile(path, flags|syscall.O_NOFOLLOW, perm)
	if err != nil {
		if errors.Is(err, syscall.ELOOP) {
			return nil, fmt.Errorf("refusing to follow symlink: %s", path)
		}
		return nil, err
	}
	return f, nil
}
