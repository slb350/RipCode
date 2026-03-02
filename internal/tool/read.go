package tool

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
)

// MaxReadLines is the default maximum number of lines returned.
// 2000 lines at ~80 chars/line is ~160 KB, well within typical LLM context
// limits while covering most source files in full.
const MaxReadLines = 2000

// ReadTool reads file contents with line numbers.
type ReadTool struct{}

func NewReadTool() *ReadTool { return &ReadTool{} }

func (r *ReadTool) ID() string { return "read" }

func (r *ReadTool) Description() string {
	return "Read a file and return its contents with line numbers."
}

func (r *ReadTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"file_path": map[string]any{
				"type":        "string",
				"description": "Absolute path to the file to read",
			},
			"offset": map[string]any{
				"type":        "integer",
				"description": "Line number to start reading from (1-based)",
			},
			"limit": map[string]any{
				"type":        "integer",
				"description": "Maximum number of lines to read",
			},
		},
		"required": []string{"file_path"},
	}
}

type readArgs struct {
	FilePath string `json:"file_path"`
	Offset   int    `json:"offset,omitempty"`
	Limit    int    `json:"limit,omitempty"`
}

func (r *ReadTool) Execute(ctx Context, argsJSON string) Result {
	var args readArgs
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return Result{Error: fmt.Errorf("parse args: %w", err)}
	}

	validated, err := ValidatePath(args.FilePath, ctx.WorkDir, true)
	if err != nil {
		return Result{Error: err}
	}

	f, err := OpenNoFollow(validated, os.O_RDONLY, 0)
	if err != nil {
		return Result{Error: fmt.Errorf("open file: %w", err)}
	}
	defer f.Close()
	data, err := io.ReadAll(f)
	if err != nil {
		return Result{Error: fmt.Errorf("read file: %w", err)}
	}

	if len(data) == 0 {
		return Result{
			Output: "(empty file)",
			Title:  validated,
		}
	}

	// Binary detection: check for null bytes in first 8KB
	checkLen := len(data)
	if checkLen > 8192 {
		checkLen = 8192
	}
	if bytes.ContainsRune(data[:checkLen], 0) {
		return Result{
			Output: fmt.Sprintf("(binary file, %d bytes)", len(data)),
			Title:  validated,
		}
	}

	lines := strings.Split(string(data), "\n")
	// Remove trailing empty line from final newline
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}

	// Apply offset (1-based)
	start := 0
	if args.Offset > 0 {
		start = args.Offset - 1
	}
	if start > len(lines) {
		start = len(lines)
	}

	// Apply limit
	limit := MaxReadLines
	if args.Limit > 0 {
		limit = args.Limit
	}

	end := start + limit
	if end > len(lines) {
		end = len(lines)
	}

	var sb strings.Builder
	for i := start; i < end; i++ {
		fmt.Fprintf(&sb, "%d\t%s\n", i+1, lines[i])
	}

	output := sb.String()
	if end < len(lines) {
		output += fmt.Sprintf("\n[%d more lines not shown]", len(lines)-end)
	}

	return Result{
		Output: output,
		Title:  validated,
	}
}
