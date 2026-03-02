//go:build windows

package fileutil

import (
	"os"
	"syscall"
	"testing"
)

func TestIsRenameDestExistsErr(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "already exists",
			err: &os.LinkError{
				Op:  "rename",
				Old: "a",
				New: "b",
				Err: syscall.ERROR_ALREADY_EXISTS,
			},
			want: true,
		},
		{
			name: "file exists",
			err: &os.LinkError{
				Op:  "rename",
				Old: "a",
				New: "b",
				Err: syscall.ERROR_FILE_EXISTS,
			},
			want: true,
		},
		{
			name: "access denied",
			err: &os.LinkError{
				Op:  "rename",
				Old: "a",
				New: "b",
				Err: syscall.ERROR_ACCESS_DENIED,
			},
			want: false,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := isRenameDestExistsErr(tc.err); got != tc.want {
				t.Fatalf("isRenameDestExistsErr() = %v, want %v", got, tc.want)
			}
		})
	}
}
