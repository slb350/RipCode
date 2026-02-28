package tool

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// EditTool performs precise string replacements in files.
type EditTool struct{}

func NewEditTool() *EditTool { return &EditTool{} }

func (e *EditTool) ID() string { return "edit" }

func (e *EditTool) Description() string {
	return "Replace a unique string in a file with new content. The old_string must match exactly one location."
}

func (e *EditTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"file_path": map[string]any{
				"type":        "string",
				"description": "Absolute path to the file to edit",
			},
			"old_string": map[string]any{
				"type":        "string",
				"description": "The exact string to find and replace (must be unique in the file)",
			},
			"new_string": map[string]any{
				"type":        "string",
				"description": "The replacement string",
			},
		},
		"required": []string{"file_path", "old_string", "new_string"},
	}
}

type editArgs struct {
	FilePath  string `json:"file_path"`
	OldString string `json:"old_string"`
	NewString string `json:"new_string"`
}

func (e *EditTool) Execute(ctx Context, argsJSON string) Result {
	var args editArgs
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return Result{Error: fmt.Errorf("parse args: %w", err)}
	}

	info, err := os.Stat(args.FilePath)
	if err != nil {
		return Result{Error: fmt.Errorf("stat file: %w", err)}
	}
	perm := info.Mode().Perm()

	data, err := os.ReadFile(args.FilePath)
	if err != nil {
		return Result{Error: fmt.Errorf("read file: %w", err)}
	}

	content := string(data)

	// Try exact match first
	count := strings.Count(content, args.OldString)

	if count == 0 {
		// Whitespace-flexible fallback: normalize whitespace and retry
		newContent, ok := whitespaceFlexibleReplace(content, args.OldString, args.NewString)
		if !ok {
			return Result{Error: fmt.Errorf("no match found for old_string in %s", args.FilePath)}
		}
		if err := os.WriteFile(args.FilePath, []byte(newContent), perm); err != nil {
			return Result{Error: fmt.Errorf("write file: %w", err)}
		}
		return Result{
			Output: fmt.Sprintf("Edited %s (whitespace-flexible match)", args.FilePath),
			Title:  args.FilePath,
		}
	}

	if count > 1 {
		return Result{Error: fmt.Errorf("old_string has %d matches in %s — must be unique. Provide more context", count, args.FilePath)}
	}

	newContent := strings.Replace(content, args.OldString, args.NewString, 1)

	if err := os.WriteFile(args.FilePath, []byte(newContent), perm); err != nil {
		return Result{Error: fmt.Errorf("write file: %w", err)}
	}

	return Result{
		Output: fmt.Sprintf("Edited %s", args.FilePath),
		Title:  args.FilePath,
	}
}

// whitespaceFlexibleReplace normalizes leading whitespace (tabs ↔ spaces)
// to find a match when exact matching fails. Returns the modified content
// and whether a unique match was found.
func whitespaceFlexibleReplace(content, oldStr, newStr string) (string, bool) {
	normContent := normalizeWhitespace(content)
	normOld := normalizeWhitespace(oldStr)

	count := strings.Count(normContent, normOld)
	if count != 1 {
		return "", false
	}

	// Find the position in normalized content
	normIdx := strings.Index(normContent, normOld)

	// Map normalized position back to original content.
	// Walk both strings in parallel to find corresponding positions.
	origStart := mapNormPos(content, normContent, normIdx)
	origEnd := mapNormPos(content, normContent, normIdx+len(normOld))

	result := content[:origStart] + newStr + content[origEnd:]
	return result, true
}

// normalizeWhitespace replaces leading tabs with 4 spaces and collapses
// multiple spaces in leading whitespace to single units.
func normalizeWhitespace(s string) string {
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		// Replace leading whitespace: tabs → 4 spaces
		trimmed := strings.TrimLeft(line, " \t")
		leading := line[:len(line)-len(trimmed)]
		normalized := strings.ReplaceAll(leading, "\t", "    ")
		lines[i] = normalized + trimmed
	}
	return strings.Join(lines, "\n")
}

// mapNormPos maps a position in normalized text back to the original text.
func mapNormPos(orig, norm string, normPos int) int {
	oi, ni := 0, 0
	for ni < normPos && oi < len(orig) {
		if orig[oi] == '\t' {
			// Tab expands to 4 spaces in normalized
			ni += 4
		} else {
			ni++
		}
		oi++
	}
	return oi
}
