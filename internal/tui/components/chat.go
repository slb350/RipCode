package components

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/stephenbrandon/ripcode/internal/tui/styles"
)

// ChatEntry represents a single rendered entry in the chat.
type ChatEntry struct {
	Role    string // "user", "assistant", "tool", "error"
	Content string
}

// Chat is a scrollable viewport displaying conversation messages.
type Chat struct {
	entries   []ChatEntry
	scrollPos int
	width     int
	height    int
	streaming string // content being streamed (not yet committed)
}

// NewChat creates a new chat component.
func NewChat() Chat {
	return Chat{}
}

// SetSize updates the chat viewport dimensions.
func (c *Chat) SetSize(width, height int) {
	c.width = width
	c.height = height
}

// AddEntry adds a completed message to the chat.
func (c *Chat) AddEntry(entry ChatEntry) {
	c.entries = append(c.entries, entry)
	c.streaming = ""
	c.scrollToBottom()
}

// StreamContent appends to the current streaming content.
func (c *Chat) StreamContent(delta string) {
	c.streaming += delta
	c.scrollToBottom()
}

// CommitStream finalizes streaming content as an assistant entry.
func (c *Chat) CommitStream() {
	if c.streaming != "" {
		c.entries = append(c.entries, ChatEntry{
			Role:    "assistant",
			Content: c.streaming,
		})
		c.streaming = ""
	}
}

// Clear removes all entries.
func (c *Chat) Clear() {
	c.entries = nil
	c.streaming = ""
	c.scrollPos = 0
}

// Update handles scroll events.
func (c *Chat) Update(msg tea.Msg) {
	switch msg := msg.(type) {
	case tea.MouseWheelMsg:
		if msg.Button == tea.MouseWheelUp {
			c.scrollPos = max(0, c.scrollPos-3)
		} else if msg.Button == tea.MouseWheelDown {
			c.scrollPos += 3
		}
	}
}

func (c Chat) View() string {
	if c.width == 0 || c.height == 0 {
		return ""
	}

	var lines []string
	for _, entry := range c.entries {
		lines = append(lines, c.renderEntry(entry)...)
		lines = append(lines, "") // blank line between entries
	}

	// Streaming content
	if c.streaming != "" {
		prefix := styles.Assistant.Render("assistant") + " "
		lines = append(lines, prefix+c.streaming)
	}

	// Apply scroll
	totalLines := len(lines)
	visibleLines := c.height

	maxScroll := max(0, totalLines-visibleLines)
	if c.scrollPos > maxScroll {
		c.scrollPos = maxScroll
	}

	start := c.scrollPos
	end := min(start+visibleLines, totalLines)

	if start >= totalLines {
		return ""
	}

	visible := lines[start:end]

	// Pad to fill height
	for len(visible) < visibleLines {
		visible = append(visible, "")
	}

	return strings.Join(visible, "\n")
}

func (c Chat) renderEntry(entry ChatEntry) []string {
	var prefix string
	switch entry.Role {
	case "user":
		prefix = styles.User.Render("you") + " "
	case "assistant":
		prefix = styles.Assistant.Render("assistant") + " "
	case "tool":
		prefix = styles.Tool.Render("tool") + " "
	case "error":
		prefix = styles.Error.Render("error") + " "
	}

	// Word wrap content to width
	content := entry.Content
	maxWidth := c.width - lipgloss.Width(prefix)
	if maxWidth < 20 {
		maxWidth = 20
	}

	wrapped := wrapText(content, maxWidth)
	lines := strings.Split(wrapped, "\n")

	result := make([]string, len(lines))
	for i, line := range lines {
		if i == 0 {
			result[i] = prefix + line
		} else {
			result[i] = strings.Repeat(" ", lipgloss.Width(prefix)) + line
		}
	}
	return result
}

func (c *Chat) scrollToBottom() {
	// Set scrollPos to show the bottom
	c.scrollPos = max(0, c.totalLines()-c.height)
}

func (c Chat) totalLines() int {
	count := 0
	for range c.entries {
		count += 2 // entry + blank line (approximate)
	}
	if c.streaming != "" {
		count++
	}
	return count
}

func wrapText(text string, width int) string {
	if width <= 0 {
		return text
	}

	var result strings.Builder
	for _, line := range strings.Split(text, "\n") {
		if lipgloss.Width(line) <= width {
			if result.Len() > 0 {
				result.WriteByte('\n')
			}
			result.WriteString(line)
			continue
		}

		words := strings.Fields(line)
		var currentLine string
		for _, word := range words {
			if currentLine == "" {
				currentLine = word
			} else if lipgloss.Width(currentLine+" "+word) <= width {
				currentLine += " " + word
			} else {
				if result.Len() > 0 {
					result.WriteByte('\n')
				}
				result.WriteString(currentLine)
				currentLine = word
			}
		}
		if currentLine != "" {
			if result.Len() > 0 {
				result.WriteByte('\n')
			}
			result.WriteString(currentLine)
		}
	}
	return result.String()
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
