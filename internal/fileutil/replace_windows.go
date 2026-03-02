//go:build windows

package fileutil

import (
	"fmt"
	"os"
)

// ReplaceFile atomically replaces dst with src.
// On Windows, os.Rename cannot overwrite an existing destination,
// so we remove the target first and retry.
func ReplaceFile(src, dst string) error {
	if err := os.Rename(src, dst); err == nil {
		return nil
	} else {
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
