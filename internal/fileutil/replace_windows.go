//go:build windows

package fileutil

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"syscall"
)

// ReplaceFile atomically replaces dst with src.
// On Windows, os.Rename cannot overwrite an existing destination,
// so we remove the target first and retry.
func ReplaceFile(src, dst string) error {
	if err := os.Rename(src, dst); err == nil {
		return nil
	} else {
		if !isRenameDestExistsErr(err) {
			return err
		}
		// Windows rename cannot replace an existing destination path.
		info, statErr := os.Lstat(dst)
		if statErr != nil {
			return err
		}
		if info.IsDir() {
			return err
		}
		if rmErr := os.Remove(dst); rmErr != nil {
			return fmt.Errorf("remove existing destination: %w (original rename error: %v)", rmErr, err)
		}
		if err2 := os.Rename(src, dst); err2 != nil {
			return fmt.Errorf("rename after remove: %w (original rename error: %v)", err2, err)
		}
		return nil
	}
}

// isRenameDestExistsErr reports whether a rename failure happened because dst
// already exists (the Windows-specific case we can safely handle by remove+retry).
func isRenameDestExistsErr(err error) bool {
	if errors.Is(err, fs.ErrExist) {
		return true
	}
	var linkErr *os.LinkError
	if !errors.As(err, &linkErr) {
		return false
	}
	return errors.Is(linkErr.Err, syscall.ERROR_ALREADY_EXISTS) ||
		errors.Is(linkErr.Err, syscall.ERROR_FILE_EXISTS)
}
