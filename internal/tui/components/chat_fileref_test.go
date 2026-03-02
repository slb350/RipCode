package components

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseFileRefs_NoRefs(t *testing.T) {
	refs := ParseFileRefs("just some text without refs")
	assert.Nil(t, refs)
}

func TestParseFileRefs_SingleFile(t *testing.T) {
	refs := ParseFileRefs("check @file.go please")
	assert.Equal(t, []string{"file.go"}, refs)
}

func TestParseFileRefs_WithPath(t *testing.T) {
	refs := ParseFileRefs("look at @src/main.go")
	assert.Equal(t, []string{"src/main.go"}, refs)
}

func TestParseFileRefs_WithLineRange(t *testing.T) {
	refs := ParseFileRefs("see @file.go:10-20")
	assert.Equal(t, []string{"file.go"}, refs)
}

func TestParseFileRefs_SingleLine(t *testing.T) {
	refs := ParseFileRefs("see @file.go:42")
	assert.Equal(t, []string{"file.go"}, refs)
}

func TestParseFileRefs_MultipleRefs(t *testing.T) {
	refs := ParseFileRefs("compare @a.go and @b.go")
	assert.Equal(t, []string{"a.go", "b.go"}, refs)
}

func TestParseFileRefs_Deduped(t *testing.T) {
	refs := ParseFileRefs("check @file.go then @file.go again")
	assert.Equal(t, []string{"file.go"}, refs)
}

func TestParseFileRefs_Email_NotParsed(t *testing.T) {
	refs := ParseFileRefs("send to user@example.com")
	assert.Nil(t, refs)
}

func TestParseFileRefs_DoubleAt_NotParsed(t *testing.T) {
	refs := ParseFileRefs("@@doubleAt")
	assert.Nil(t, refs)
}

func TestParseFileRefs_StartOfLine(t *testing.T) {
	refs := ParseFileRefs("@file.go is important")
	assert.Equal(t, []string{"file.go"}, refs)
}

func TestParseFileRefs_AfterSpace(t *testing.T) {
	refs := ParseFileRefs("check @file.go")
	assert.Equal(t, []string{"file.go"}, refs)
}

func TestParseFileRefs_Relative(t *testing.T) {
	refs := ParseFileRefs("see @./relative.go")
	assert.Equal(t, []string{"./relative.go"}, refs)
}

func TestFileBadge_BaseNameDisplayed(t *testing.T) {
	c := NewChat()
	c.SetSize(80, 20)
	c.AddEntry(ChatEntry{Role: RoleUser, Content: "check @src/internal/main.go"})
	view := c.View()
	assert.Contains(t, view, "📎 main.go")
}

func TestFileBadge_MaxThreeBadges(t *testing.T) {
	c := NewChat()
	c.SetSize(80, 20)
	c.AddEntry(ChatEntry{Role: RoleUser, Content: "check @a.go @b.go @c.go"})
	view := c.View()
	assert.Contains(t, view, "📎 a.go")
	assert.Contains(t, view, "📎 b.go")
	assert.Contains(t, view, "📎 c.go")
	assert.NotContains(t, view, "+")
}

func TestFileBadge_OverflowIndicator(t *testing.T) {
	c := NewChat()
	c.SetSize(80, 20)
	c.AddEntry(ChatEntry{Role: RoleUser, Content: "check @a.go @b.go @c.go @d.go @e.go"})
	view := c.View()
	assert.Contains(t, view, "📎 a.go")
	assert.Contains(t, view, "📎 b.go")
	assert.Contains(t, view, "📎 c.go")
	assert.Contains(t, view, "+2 more")
}

func TestFileBadge_SingleRef_NoOverflow(t *testing.T) {
	c := NewChat()
	c.SetSize(80, 20)
	c.AddEntry(ChatEntry{Role: RoleUser, Content: "see @main.go"})
	view := c.View()
	assert.Contains(t, view, "📎 main.go")
	assert.NotContains(t, view, fmt.Sprintf("+%d more", 0))
}

func TestFileBadge_NoRefs_NoBadgeLine(t *testing.T) {
	c := NewChat()
	c.SetSize(80, 20)
	c.AddEntry(ChatEntry{Role: RoleUser, Content: "no file refs here"})
	view := c.View()
	assert.NotContains(t, view, "📎")
}

func TestParseFileRefs_AfterNewline(t *testing.T) {
	refs := ParseFileRefs("first line\n@second.go")
	assert.Equal(t, []string{"second.go"}, refs)
}

func TestParseFileRefs_DeepPath(t *testing.T) {
	refs := ParseFileRefs("check @internal/tui/components/chat.go")
	assert.Equal(t, []string{"internal/tui/components/chat.go"}, refs)
}

func TestFileBadge_ShowsBasenameNotFullPath(t *testing.T) {
	c := NewChat()
	c.SetSize(120, 20)
	c.AddEntry(ChatEntry{Role: RoleUser, Content: "see @internal/tui/components/chat.go"})
	view := c.View()
	assert.Contains(t, view, "📎 chat.go")
	// Full path should NOT be in the badge
	lines := strings.Split(view, "\n")
	for _, line := range lines {
		if strings.Contains(line, "📎") {
			assert.NotContains(t, line, "internal/tui/components/chat.go")
		}
	}
}
