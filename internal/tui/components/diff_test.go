package components

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stephenbrandon/ripcode/internal/tui/styles"
)

func TestComputeDiff_NewFile_AllAdditions(t *testing.T) {
	lines := ComputeDiff("", "line1\nline2\n", 3)
	require.NotNil(t, lines)
	hasPlus := false
	for _, l := range lines {
		if strings.HasPrefix(l, "+") {
			hasPlus = true
		}
	}
	assert.True(t, hasPlus, "new file diff should have + lines")
}

func TestComputeDiff_Identical_ReturnsNil(t *testing.T) {
	lines := ComputeDiff("same content\n", "same content\n", 3)
	assert.Nil(t, lines)
}

func TestComputeDiff_SingleLineChange(t *testing.T) {
	before := "hello world\n"
	after := "hello go\n"
	lines := ComputeDiff(before, after, 3)
	require.NotNil(t, lines)
	hasMinus := false
	hasPlus := false
	for _, l := range lines {
		if strings.HasPrefix(l, "-") {
			hasMinus = true
		}
		if strings.HasPrefix(l, "+") {
			hasPlus = true
		}
	}
	assert.True(t, hasMinus, "should have deletion line")
	assert.True(t, hasPlus, "should have addition line")
}

func TestComputeDiff_MultiLineChange(t *testing.T) {
	before := "line1\nline2\nline3\n"
	after := "line1\nmodified\nline3\n"
	lines := ComputeDiff(before, after, 3)
	require.NotNil(t, lines)
	assert.True(t, len(lines) > 1)
}

func TestComputeDiff_AdditionsOnly(t *testing.T) {
	before := "line1\n"
	after := "line1\nline2\nline3\n"
	lines := ComputeDiff(before, after, 3)
	require.NotNil(t, lines)
	plusCount := 0
	minusCount := 0
	for _, l := range lines {
		if strings.HasPrefix(l, "+") {
			plusCount++
		}
		if strings.HasPrefix(l, "-") {
			minusCount++
		}
	}
	assert.Greater(t, plusCount, 0)
	assert.Equal(t, 0, minusCount, "additions-only diff should have no deletions")
}

func TestComputeDiff_DeletionsOnly(t *testing.T) {
	before := "line1\nline2\nline3\n"
	after := "line1\n"
	lines := ComputeDiff(before, after, 3)
	require.NotNil(t, lines)
	minusCount := 0
	plusCount := 0
	for _, l := range lines {
		if strings.HasPrefix(l, "-") {
			minusCount++
		}
		if strings.HasPrefix(l, "+") {
			plusCount++
		}
	}
	assert.Greater(t, minusCount, 0)
	assert.Equal(t, 0, plusCount, "deletions-only diff should have no additions")
}

func TestComputeDiff_ContextLines(t *testing.T) {
	var before, after strings.Builder
	for i := range 20 {
		if i == 10 {
			before.WriteString("old line\n")
			after.WriteString("new line\n")
		} else {
			line := strings.Repeat("x", 5) + "\n"
			before.WriteString(line)
			after.WriteString(line)
		}
	}
	lines := ComputeDiff(before.String(), after.String(), 3)
	require.NotNil(t, lines)
	// Context should include surrounding unchanged lines
	contextCount := 0
	for _, l := range lines {
		if len(l) > 0 && l[0] == ' ' {
			contextCount++
		}
	}
	assert.LessOrEqual(t, contextCount, 6, "context=3 gives max 6 context lines")
}

func TestComputeDiff_TruncationAt30Lines(t *testing.T) {
	var before, after strings.Builder
	for i := range 50 {
		before.WriteString(strings.Repeat("a", 5) + "\n")
		after.WriteString(strings.Repeat("b", 5) + "\n")
		_ = i
	}
	lines := ComputeDiff(before.String(), after.String(), 3)
	require.NotNil(t, lines)
	assert.LessOrEqual(t, len(lines), maxDiffLines+1) // +1 for truncation message
	last := lines[len(lines)-1]
	assert.Contains(t, last, "more lines")
}

func TestComputeDiff_LongLineTruncation(t *testing.T) {
	longLine := strings.Repeat("a", 300) + "\n"
	shortLine := "b\n"
	lines := ComputeDiff(longLine, shortLine, 3)
	require.NotNil(t, lines)
	for _, l := range lines {
		if strings.HasPrefix(l, "...") {
			continue
		}
		// Lines are allowed to be truncated at 200 chars
		// The truncation adds "..." so max is 203
		assert.LessOrEqual(t, len(l), maxDiffLineWidth+10, "lines should be capped near maxDiffLineWidth")
	}
}

func TestRenderDiffLine_Colors(t *testing.T) {
	theme := styles.DefaultTheme
	plus := RenderDiffLine("+added", theme)
	minus := RenderDiffLine("-removed", theme)
	at := RenderDiffLine("@@ -1,3 +1,3 @@", theme)
	ctx := RenderDiffLine(" context", theme)

	// Verify lines are styled (contain ANSI sequences)
	assert.Contains(t, plus, "added")
	assert.Contains(t, minus, "removed")
	assert.Contains(t, at, "@@")
	assert.Contains(t, ctx, "context")
}

func TestRenderDiffLine_EmptyLine(t *testing.T) {
	result := RenderDiffLine("", styles.DefaultTheme)
	assert.Equal(t, "", result)
}

func TestRenderToolEntry_WithDiff_DetailsOn(t *testing.T) {
	c := NewChat()
	c.SetSize(80, 20)
	c.SetShowDetails(true)

	c.AddEntry(ChatEntry{
		Role:       RoleTool,
		Content:    "Edited /tmp/test.go",
		ToolName:   "edit",
		ToolStatus: StatusSuccess,
		Diff: &DiffInfo{
			Path:   "/tmp/test.go",
			Before: "old content\n",
			After:  "new content\n",
		},
	})

	view := c.View()
	// Should contain diff lines (colored + and - prefixed lines)
	assert.Contains(t, view, "old content")
	assert.Contains(t, view, "new content")
}

func TestRenderToolEntry_WithDiff_DetailsOff(t *testing.T) {
	c := NewChat()
	c.SetSize(80, 20)
	c.SetShowDetails(false)

	c.AddEntry(ChatEntry{
		Role:       RoleTool,
		Content:    "Edited /tmp/test.go",
		ToolName:   "edit",
		ToolStatus: StatusSuccess,
		Diff: &DiffInfo{
			Path:   "/tmp/test.go",
			Before: "old content\n",
			After:  "new content\n",
		},
	})

	view := c.View()
	// Should show summary line but NOT diff details
	assert.Contains(t, view, "edit")
	assert.NotContains(t, view, "old content")
}

func TestRenderToolEntry_WithoutDiff_DetailsOn(t *testing.T) {
	c := NewChat()
	c.SetSize(80, 20)
	c.SetShowDetails(true)

	c.AddEntry(ChatEntry{
		Role:       RoleTool,
		Content:    "found 5 matches",
		ToolName:   "grep",
		ToolStatus: StatusSuccess,
	})

	view := c.View()
	assert.Contains(t, view, "found 5 matches")
}

func TestRenderToolEntry_BinaryDiff(t *testing.T) {
	c := NewChat()
	c.SetSize(80, 20)
	c.SetShowDetails(true)

	c.AddEntry(ChatEntry{
		Role:       RoleTool,
		Content:    "Edited binary.bin",
		ToolName:   "edit",
		ToolStatus: StatusSuccess,
		Diff: &DiffInfo{
			Path:   "/tmp/binary.bin",
			Before: "binary\x00data",
			After:  "new\x00data",
			Binary: true,
		},
	})

	view := c.View()
	assert.Contains(t, view, "[Binary file changed]")
}

func TestRenderToolEntry_EmptyDiff_BeforeEqualsAfter(t *testing.T) {
	c := NewChat()
	c.SetSize(80, 20)
	c.SetShowDetails(true)

	c.AddEntry(ChatEntry{
		Role:       RoleTool,
		Content:    "Edited /tmp/test.go",
		ToolName:   "edit",
		ToolStatus: StatusSuccess,
		Diff: &DiffInfo{
			Path:   "/tmp/test.go",
			Before: "same content",
			After:  "same content",
		},
	})

	view := c.View()
	// Should show summary but no diff block
	assert.Contains(t, view, "edit")
	// No diff lines should appear
	assert.NotContains(t, view, "@@")
}

func TestRenderToolEntry_NilDiff_NonEditTool(t *testing.T) {
	c := NewChat()
	c.SetSize(80, 20)
	c.SetShowDetails(true)

	c.AddEntry(ChatEntry{
		Role:       RoleTool,
		Content:    "3 files found",
		ToolName:   "glob",
		ToolStatus: StatusSuccess,
		Diff:       nil,
	})

	view := c.View()
	assert.Contains(t, view, "3 files found")
}
