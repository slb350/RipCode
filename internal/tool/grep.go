package tool

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
)

// maxGrepFiles limits search results to prevent overwhelming the LLM context.
// 100 files typically provides sufficient context without exceeding token budgets.
const maxGrepFiles = 100

// GrepTool searches file contents with regex patterns.
type GrepTool struct{}

func NewGrepTool() *GrepTool { return &GrepTool{} }

func (g *GrepTool) ID() string { return "grep" }

func (g *GrepTool) Description() string {
	return "Search file contents for a regex pattern. Returns matching lines with file paths and line numbers."
}

func (g *GrepTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"pattern": map[string]any{
				"type":        "string",
				"description": "Regex pattern to search for",
			},
			"path": map[string]any{
				"type":        "string",
				"description": "Directory or file to search in (default: working directory)",
			},
			"include": map[string]any{
				"type":        "string",
				"description": "Glob pattern to filter files (e.g., '*.go', '*.txt')",
			},
		},
		"required": []string{"pattern"},
	}
}

type grepArgs struct {
	Pattern string `json:"pattern"`
	Path    string `json:"path,omitempty"`
	Include string `json:"include,omitempty"`
}

func (g *GrepTool) Execute(ctx Context, argsJSON string) Result {
	var args grepArgs
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return Result{Error: fmt.Errorf("%s: parse args: %w", g.ID(), err)}
	}

	re, err := regexp.Compile(args.Pattern)
	if err != nil {
		return Result{Error: fmt.Errorf("invalid regex: %w", err)}
	}

	root := args.Path
	if root == "" {
		root = ctx.WorkDir
	}

	validatedRoot, err := ValidatePath(root, ctx.WorkDir, true)
	if err != nil {
		return Result{Error: err}
	}
	root = validatedRoot

	var sb strings.Builder
	fileCount := 0
	skips := newSkipTracker()

	walkErr := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			skips.addPath(path, err)
			return nil
		}

		if d.IsDir() {
			if skipDirs[d.Name()] {
				return filepath.SkipDir
			}
			return nil
		}

		if fileCount >= maxGrepFiles {
			return filepath.SkipAll
		}

		// Apply include filter
		if args.Include != "" {
			matched, matchErr := doublestar.PathMatch(args.Include, d.Name())
			if matchErr != nil {
				return fmt.Errorf("invalid include pattern %q: %w", args.Include, matchErr)
			}
			if !matched {
				return nil
			}
		}

		data, err := os.ReadFile(path)
		if err != nil {
			skips.add(err)
			return nil
		}

		// Skip binary files
		if isBinaryContent(data) {
			return nil
		}

		rel, err := filepath.Rel(root, path)
		if err != nil {
			skips.add(err)
			return nil
		}
		lines := strings.Split(string(data), "\n")

		matched := false
		for i, line := range lines {
			if re.MatchString(line) {
				if !matched {
					matched = true
					fileCount++
				}
				fmt.Fprintf(&sb, "%s:%d: %s\n", rel, i+1, line)
			}
		}
		return nil
	})

	if walkErr != nil {
		return Result{Error: fmt.Errorf("grep: %w", walkErr)}
	}

	if sb.Len() == 0 {
		return Result{
			Output: fmt.Sprintf("no matches for pattern %q in %s", args.Pattern, root) + skips.note("paths"),
			Title:  args.Pattern,
		}
	}

	if fileCount >= maxGrepFiles {
		sb.WriteString(fmt.Sprintf("\n[results limited to %d files]", maxGrepFiles))
	}
	sb.WriteString(skips.note("paths"))

	return Result{
		Output: sb.String(),
		Title:  args.Pattern,
	}
}
