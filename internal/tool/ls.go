package tool

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// LsTool lists directory contents.
type LsTool struct{}

func NewLsTool() *LsTool { return &LsTool{} }

func (l *LsTool) ID() string { return "ls" }

func (l *LsTool) Description() string {
	return "List directory contents with permissions and sizes."
}

func (l *LsTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path": map[string]any{
				"type":        "string",
				"description": "Directory to list (default: working directory)",
			},
			"all": map[string]any{
				"type":        "boolean",
				"description": "Include hidden files (default: false)",
			},
		},
	}
}

type lsArgs struct {
	Path string `json:"path,omitempty"`
	All  bool   `json:"all,omitempty"`
}

func (l *LsTool) Execute(ctx Context, argsJSON string) Result {
	var args lsArgs
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return Result{Error: fmt.Errorf("%s: parse args: %w", l.ID(), err)}
	}

	dir := args.Path
	if dir == "" {
		dir = ctx.WorkDir
	}

	validatedDir, err := ValidatePath(dir, ctx.WorkDir, true)
	if err != nil {
		return Result{Error: err}
	}
	dir = validatedDir

	entries, err := os.ReadDir(dir)
	if err != nil {
		return Result{Error: fmt.Errorf("read directory: %w", err)}
	}

	var sb strings.Builder
	count := 0
	skips := newSkipTracker()

	for _, entry := range entries {
		name := entry.Name()

		// Skip .git directory always
		if name == ".git" {
			continue
		}

		// Skip hidden files unless all flag is set
		if !args.All && strings.HasPrefix(name, ".") {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			skips.add(err)
			continue
		}

		perm := info.Mode().String()
		size := formatSize(info.Size())
		display := name
		if entry.IsDir() {
			display += "/"
		}

		fmt.Fprintf(&sb, "%s  %6s  %s\n", perm, size, display)
		count++
	}

	sb.WriteString(skips.note("entries"))

	if count == 0 && skips.count() == 0 {
		return Result{
			Output: fmt.Sprintf("(empty directory: %s)", dir),
			Title:  dir,
		}
	}

	return Result{
		Output: sb.String(),
		Title:  dir,
	}
}

// formatSize returns a human-readable size string.
func formatSize(bytes int64) string {
	switch {
	case bytes >= 1024*1024*1024:
		return fmt.Sprintf("%.1fG", float64(bytes)/(1024*1024*1024))
	case bytes >= 1024*1024:
		return fmt.Sprintf("%.1fM", float64(bytes)/(1024*1024))
	case bytes >= 1024:
		return fmt.Sprintf("%.1fK", float64(bytes)/1024)
	default:
		return fmt.Sprintf("%dB", bytes)
	}
}
