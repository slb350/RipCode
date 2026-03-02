//go:build !windows

package fileutil

import "os"

// ReplaceFile atomically replaces dst with src.
// On Unix, os.Rename atomically overwrites the destination.
func ReplaceFile(src, dst string) error {
	return os.Rename(src, dst)
}
