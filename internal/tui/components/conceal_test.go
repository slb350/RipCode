package components

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConcealCodeBlocks_NoCodeBlocks(t *testing.T) {
	input := "This is normal text without code."
	assert.Equal(t, input, concealCodeBlocks(input))
}

func TestConcealCodeBlocks_SingleFencedBlock(t *testing.T) {
	input := "Before\n```go\nfunc main() {}\n```\nAfter"
	result := concealCodeBlocks(input)
	assert.Contains(t, result, "Before")
	assert.Contains(t, result, "After")
	assert.Contains(t, result, "[code block hidden]")
	assert.NotContains(t, result, "func main")
}

func TestConcealCodeBlocks_MultipleFencedBlocks(t *testing.T) {
	input := "Start\n```\nblock1\n```\nMiddle\n```python\nblock2\n```\nEnd"
	result := concealCodeBlocks(input)
	assert.Contains(t, result, "Start")
	assert.Contains(t, result, "Middle")
	assert.Contains(t, result, "End")
	assert.NotContains(t, result, "block1")
	assert.NotContains(t, result, "block2")
}

func TestConcealCodeBlocks_UnclosedBlock(t *testing.T) {
	input := "Before\n```\nunclosed code\nmore code"
	result := concealCodeBlocks(input)
	assert.Contains(t, result, "Before")
	assert.Contains(t, result, "[code block hidden]")
	assert.NotContains(t, result, "unclosed code")
}

func TestConcealCodeBlocks_InlineCode_NotConcealed(t *testing.T) {
	input := "Use `inline code` here."
	assert.Equal(t, input, concealCodeBlocks(input))
}

func TestConcealCodeBlocks_EmptyFencedBlock(t *testing.T) {
	input := "Before\n```\n```\nAfter"
	result := concealCodeBlocks(input)
	assert.Contains(t, result, "Before")
	assert.Contains(t, result, "After")
	assert.Contains(t, result, "[code block hidden]")
}

func TestChat_RenderWithConcealedCode(t *testing.T) {
	c := NewChat()
	c.SetSize(80, 20)
	c.SetShowCodeBlocks(false)

	c.AddEntry(ChatEntry{
		Role:    RoleAssistant,
		Content: "Here is code:\n```go\nfunc main() {}\n```\nDone.",
	})

	view := c.View()
	assert.Contains(t, view, "Here is code:")
	assert.Contains(t, view, "[code block hidden]")
	assert.NotContains(t, view, "func main")
	assert.Contains(t, view, "Done.")
}

func TestChat_RenderWithVisibleCode(t *testing.T) {
	c := NewChat()
	c.SetSize(80, 20)
	c.SetShowCodeBlocks(true) // default

	c.AddEntry(ChatEntry{
		Role:    RoleAssistant,
		Content: "Here is code:\n```go\nfunc main() {}\n```\nDone.",
	})

	view := c.View()
	assert.Contains(t, view, "func main")
	assert.NotContains(t, view, "[code block hidden]")
}

func TestChat_ConcealAffectsPartsText(t *testing.T) {
	c := NewChat()
	c.SetSize(80, 20)
	c.SetShowCodeBlocks(false)
	c.SetShowThinking(true)

	c.AddEntry(ChatEntry{
		Role: RoleAssistant,
		Parts: []MessagePart{
			{Type: PartText, Content: "Text\n```\ncode\n```\nMore"},
		},
	})

	view := c.View()
	assert.Contains(t, view, "Text")
	assert.Contains(t, view, "[code block hidden]")
	assert.NotContains(t, view, "code\n")
}

func TestConcealCodeBlocks_NestedBackticks(t *testing.T) {
	// Content with ``` inside code that's already inside a fenced block
	input := "Before\n```\nsome `inline` code\n```\nAfter"
	result := concealCodeBlocks(input)
	assert.Contains(t, result, "Before")
	assert.Contains(t, result, "After")
	assert.Contains(t, result, "[code block hidden]")
	assert.NotContains(t, result, "some `inline` code")
}

func TestConcealCodeBlocks_FourBackticksFence(t *testing.T) {
	// ```` is treated the same as ``` — any ``` prefix starts/closes a fence.
	// The inner ``` closes the ```` block, then func main is visible,
	// then the second ``` starts a new block that ```` closes.
	input := "Text\n````\ninner\n```\nfunc main() {}\n```\nouter\n````\nEnd"
	result := concealCodeBlocks(input)
	assert.Contains(t, result, "Text")
	assert.Contains(t, result, "[code block hidden]")
	// The ``` inside closes the first block, so "func main" becomes visible
	assert.Contains(t, result, "func main")
	assert.Contains(t, result, "End")
}

func TestChat_ConcealAcrossInterleavedParts(t *testing.T) {
	c := NewChat()
	c.SetSize(100, 20)
	c.SetShowCodeBlocks(false)
	c.SetShowThinking(true)

	c.AddEntry(ChatEntry{
		Role: RoleAssistant,
		Parts: []MessagePart{
			{Type: PartText, Content: "```go\nfmt"},
			{Type: PartReasoning, Content: "thinking"},
			{Type: PartText, Content: ".Println(1)\n```\nAfter"},
		},
	})

	view := c.View()
	assert.Contains(t, view, "[code block hidden]")
	assert.Equal(t, 1, strings.Count(view, "[code block hidden]"))
	assert.NotContains(t, view, "Println(1)")
	assert.Contains(t, view, "thinking")
	assert.Contains(t, view, "After")
}

func TestChat_StreamingParts_ConcealCodeBlock(t *testing.T) {
	c := NewChat()
	c.SetSize(80, 20)
	c.SetShowCodeBlocks(false)

	c.StreamPart(PartText, "Before\n```go\nfunc main() {}\n```\nAfter")

	view := c.View()
	assert.Contains(t, view, "Before")
	assert.Contains(t, view, "[code block hidden]")
	assert.NotContains(t, view, "func main")
	assert.Contains(t, view, "After")
}

func TestChat_StreamingParts_ConcealUnclosedBlock(t *testing.T) {
	c := NewChat()
	c.SetSize(80, 20)
	c.SetShowCodeBlocks(false)

	// Code block opened but not closed during streaming
	c.StreamPart(PartText, "Before\n```go\nfunc main")

	view := c.View()
	assert.Contains(t, view, "Before")
	assert.Contains(t, view, "[code block hidden]")
	assert.NotContains(t, view, "func main")
}

func TestConcealCodeBlocks_FenceAtEndOfText(t *testing.T) {
	input := "Before\n```"
	result := concealCodeBlocks(input)
	assert.Contains(t, result, "Before")
	assert.Contains(t, result, "[code block hidden]")
}
