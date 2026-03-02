package components

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/pmezard/go-difflib/difflib"

	"github.com/stephenbrandon/ripcode/internal/store"
	"github.com/stephenbrandon/ripcode/internal/tui/styles"
)

// maxDiffLines caps unified diff output to prevent terminal overflow.
const maxDiffLines = 30

// maxDiffLineWidth caps individual diff lines to prevent horizontal overflow.
const maxDiffLineWidth = 200

// ComputeDiff generates a unified diff between before and after strings.
// Returns nil if content is identical (empty diff).
// Individual lines are capped at maxDiffLineWidth chars. Total output capped
// at maxDiffLines lines.
func ComputeDiff(before, after string, contextLines int) []string {
	if before == after {
		return nil
	}

	diff := difflib.UnifiedDiff{
		A:        difflib.SplitLines(before),
		B:        difflib.SplitLines(after),
		FromFile: "before",
		ToFile:   "after",
		Context:  contextLines,
	}

	text, err := difflib.GetUnifiedDiffString(diff)
	if err != nil {
		store.LogError("diff: GetUnifiedDiffString failed", err)
		return nil
	}
	if text == "" {
		return nil
	}

	rawLines := strings.Split(strings.TrimRight(text, "\n"), "\n")

	// Skip lines before the first @@ hunk header (typically --- and +++ file headers)
	start := 0
	for i, line := range rawLines {
		if strings.HasPrefix(line, "@@") {
			start = i
			break
		}
	}
	rawLines = rawLines[start:]

	// Truncate individual lines and cap total count.
	lines := make([]string, 0, min(len(rawLines), maxDiffLines))
	for _, line := range rawLines {
		if len(lines) >= maxDiffLines {
			remaining := len(rawLines) - maxDiffLines
			lines = append(lines, fmt.Sprintf("... (%d more lines)", remaining))
			break
		}
		if lipgloss.Width(line) > maxDiffLineWidth {
			line = ansi.Truncate(line, maxDiffLineWidth, "...")
		}
		lines = append(lines, line)
	}

	return lines
}

// RenderDiffLine applies color to a single diff line based on its prefix.
func RenderDiffLine(line string, t *styles.Theme) string {
	if len(line) == 0 {
		return line
	}
	switch line[0] {
	case '+':
		return t.SuccessStyle.Render(line)
	case '-':
		return t.ErrorStyle.Render(line)
	case '@':
		return t.SecondaryStyle.Render(line)
	default:
		return t.TextMutedStyle.Render(line)
	}
}
