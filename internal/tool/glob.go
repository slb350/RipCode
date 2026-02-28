package tool

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
)

// skipDirs contains directory names to skip during glob traversal.
var skipDirs = map[string]bool{
	".git":         true,
	"node_modules": true,
	".next":        true,
	"__pycache__":  true,
	".venv":        true,
}

// GlobTool finds files matching a pattern.
type GlobTool struct{}

func NewGlobTool() *GlobTool { return &GlobTool{} }

func (g *GlobTool) ID() string { return "glob" }

func (g *GlobTool) Description() string {
	return "Find files matching a glob pattern. Supports ** for recursive matching."
}

func (g *GlobTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"pattern": map[string]any{
				"type":        "string",
				"description": "Glob pattern to match (e.g., '**/*.go', '*.txt')",
			},
			"path": map[string]any{
				"type":        "string",
				"description": "Root directory to search from (default: working directory)",
			},
		},
		"required": []string{"pattern"},
	}
}

type globArgs struct {
	Pattern string `json:"pattern"`
	Path    string `json:"path,omitempty"`
}

func (g *GlobTool) Execute(ctx Context, argsJSON string) Result {
	var args globArgs
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return Result{Error: fmt.Errorf("parse args: %w", err)}
	}

	root := args.Path
	if root == "" {
		root = ctx.WorkDir
	}

	var matches []string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil // skip errors
		}

		if d.IsDir() {
			if skipDirs[d.Name()] {
				return filepath.SkipDir
			}
			return nil
		}

		rel, err := filepath.Rel(root, path)
		if err != nil {
			return nil
		}

		matched, err := doublestar.PathMatch(args.Pattern, rel)
		if err != nil {
			return fmt.Errorf("invalid pattern: %w", err)
		}
		if matched {
			matches = append(matches, rel)
		}
		return nil
	})

	if err != nil {
		return Result{Error: fmt.Errorf("glob: %w", err)}
	}

	if len(matches) == 0 {
		return Result{
			Output: fmt.Sprintf("no matches for pattern %q in %s", args.Pattern, root),
			Title:  args.Pattern,
		}
	}

	sort.Strings(matches)

	var sb strings.Builder
	for _, m := range matches {
		sb.WriteString(m)
		sb.WriteByte('\n')
	}
	sb.WriteString(fmt.Sprintf("\n%d files matched", len(matches)))

	return Result{
		Output: sb.String(),
		Title:  args.Pattern,
	}
}
