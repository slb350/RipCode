package tool

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
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

	// Create parent directories
	dir := filepath.Dir(args.FilePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return Result{Error: fmt.Errorf("create directories: %w", err)}
	}

	if err := os.WriteFile(args.FilePath, []byte(args.Content), 0644); err != nil {
		return Result{Error: fmt.Errorf("write file: %w", err)}
	}

	return Result{
		Output: fmt.Sprintf("Wrote %d bytes to %s", len(args.Content), args.FilePath),
		Title:  args.FilePath,
	}
}
