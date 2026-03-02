package tool

import (
	"encoding/json"
	"fmt"
	"io"
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

func readExistingDiffSnapshot(r io.Reader) (string, bool, error) {
	data, err := io.ReadAll(io.LimitReader(r, maxDiffContentSize+1))
	if err != nil {
		return "", false, err
	}
	return capDiffContent(string(data)), isBinaryContent(data), nil
}

func (w *WriteTool) Execute(ctx Context, argsJSON string) Result {
	var args writeArgs
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return Result{Error: fmt.Errorf("%s: parse args: %w", w.ID(), err)}
	}

	if args.FilePath == "" {
		return Result{Error: fmt.Errorf("file_path is required")}
	}

	validated, err := ValidatePath(args.FilePath, ctx.WorkDir, false)
	if err != nil {
		return Result{Error: err}
	}

	perm := os.FileMode(0o644)
	if info, err := os.Lstat(validated); err == nil {
		perm = info.Mode().Perm()
	} else if !os.IsNotExist(err) {
		return Result{Error: fmt.Errorf("stat file: %w", err)}
	}

	// Capture existing content via OpenNoFollow (symlink-safe) for diff rendering.
	var before string
	var beforeBinary bool
	if f, err := OpenNoFollow(validated, os.O_RDONLY, 0); err == nil {
		if snap, binary, err := readExistingDiffSnapshot(f); err == nil {
			before = snap
			beforeBinary = binary
		}
		f.Close()
	}

	// Create parent directories
	dir := filepath.Dir(validated)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return Result{Error: fmt.Errorf("create directories: %w", err)}
	}

	if err := writeAtomic(validated, []byte(args.Content), perm); err != nil {
		return Result{Error: fmt.Errorf("write file: %w", err)}
	}

	return Result{
		Output: fmt.Sprintf("Wrote %d bytes to %s", len(args.Content), validated),
		Title:  validated,
		Diff: &DiffInfo{
			Path:   validated,
			Before: before,
			After:  capDiffContent(args.Content),
			Binary: beforeBinary || isBinaryContent([]byte(args.Content)),
		},
	}
}
