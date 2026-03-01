package tool

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

// WriteTool writes content to a file.
type WriteTool struct{}

func NewWriteTool() *WriteTool { return &WriteTool{} }

func (w *WriteTool) ID() string { return "write" }

func (w *WriteTool) Description() string {
	return "Write content to a file, creating parent directories as needed."
}

func (w *WriteTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"file_path": map[string]any{
				"type":        "string",
				"description": "Absolute path to the file to write",
			},
			"content": map[string]any{
				"type":        "string",
				"description": "The content to write to the file",
			},
		},
		"required": []string{"file_path", "content"},
	}
}

type writeArgs struct {
	FilePath string `json:"file_path"`
	Content  string `json:"content"`
}

func (w *WriteTool) Execute(ctx Context, argsJSON string) Result {
	var args writeArgs
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return Result{Error: fmt.Errorf("parse args: %w", err)}
	}

	if args.FilePath == "" {
		return Result{Error: fmt.Errorf("file_path is required")}
	}

	validated, err := ValidatePath(args.FilePath, ctx.WorkDir, false)
	if err != nil {
		return Result{Error: err}
	}

	// Create parent directories
	dir := filepath.Dir(validated)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return Result{Error: fmt.Errorf("create directories: %w", err)}
	}

	// O_NOFOLLOW rejects symlinks atomically, eliminating the TOCTOU race
	// that existed with the previous Lstat-then-WriteFile approach.
	f, err := os.OpenFile(validated, os.O_WRONLY|os.O_CREATE|os.O_TRUNC|syscall.O_NOFOLLOW, 0644)
	if err != nil {
		if errors.Is(err, syscall.ELOOP) {
			return Result{Error: fmt.Errorf("refusing to write through symlink: %s", validated)}
		}
		return Result{Error: fmt.Errorf("open file: %w", err)}
	}
	if _, err := f.Write([]byte(args.Content)); err != nil {
		f.Close()
		return Result{Error: fmt.Errorf("write file: %w", err)}
	}
	if err := f.Close(); err != nil {
		return Result{Error: fmt.Errorf("close file: %w", err)}
	}

	return Result{
		Output: fmt.Sprintf("Wrote %d bytes to %s", len(args.Content), validated),
		Title:  validated,
	}
}
