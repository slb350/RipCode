package components

import (
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/stephenbrandon/ripcode/internal/tui/styles"
)

// Chat entry role constants.
const (
	RoleUser      = "user"
	RoleAssistant = "assistant"
	RoleTool      = "tool"
	RoleError     = "error"
	RoleSystem    = "system"
	RoleComplete  = "complete"
)

// Tool status constants.
const (
	StatusPending = "pending"
	StatusSuccess = "success"
	StatusError   = "error"
)

// CompleteMeta holds metadata for a completion info bar entry.
type CompleteMeta struct {
	Mode     string
	Model    string
	Duration time.Duration
}

// ChatEntry represents a single rendered entry in the chat.
type ChatEntry struct {
	Role       string // RoleUser, RoleAssistant, RoleTool, RoleError, RoleSystem, RoleComplete
	Content    string
	ToolID     string        // tool call ID for matching updates
	ToolName   string        // tool name (bash, read, write, etc.)
	ToolStatus string        // StatusPending, StatusSuccess, StatusError
	Meta       *CompleteMeta // for RoleComplete entries
}

// Chat is a scrollable viewport displaying conversation messages.
type Chat struct {
	entries   []ChatEntry
	scrollPos int
	width     int
	height    int
	streaming string
	mode      string
	theme     *styles.Theme
}

// NewChat creates a new chat component.
func NewChat() Chat {
	return Chat{
		mode:  "build",
		theme: styles.DefaultTheme,
	}
}

// SetSize updates the chat viewport dimensions.
func (c *Chat) SetSize(width, height int) {
	c.width = width
	c.height = height
}

// SetMode sets the current agent mode for accent colors.
func (c *Chat) SetMode(mode string) { c.mode = mode }

// SetTheme sets the theme.
func (c *Chat) SetTheme(t *styles.Theme) { c.theme = t }

// AddEntry adds a completed message to the chat.
func (c *Chat) AddEntry(entry ChatEntry) {
	c.entries = append(c.entries, entry)
	c.streaming = ""
	c.scrollToBottom()
}

// UpdateLastTool updates the last tool entry matching the given ID.
func (c *Chat) UpdateLastTool(id string, entry ChatEntry) {
	for i := len(c.entries) - 1; i >= 0; i-- {
		if c.entries[i].ToolID == id {
			c.entries[i] = entry
			return
		}
	}
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
			Role:    RoleAssistant,
			Content: c.streaming,
		})
		c.streaming = ""
	}
}

// PageUp scrolls up by one page height.
func (c *Chat) PageUp() { c.scrollPos = max(0, c.scrollPos-c.height) }

// PageDown scrolls down by one page height.
func (c *Chat) PageDown() { c.scrollPos += c.height }

// HalfPageUp scrolls up by half a page.
func (c *Chat) HalfPageUp() { c.scrollPos = max(0, c.scrollPos-c.height/2) }

// HalfPageDown scrolls down by half a page.
func (c *Chat) HalfPageDown() { c.scrollPos += c.height / 2 }

// LineUp scrolls up by one line.
func (c *Chat) LineUp() { c.scrollPos = max(0, c.scrollPos-1) }

// LineDown scrolls down by one line.
func (c *Chat) LineDown() { c.scrollPos++ }

// ScrollToTop scrolls to the very top.
func (c *Chat) ScrollToTop() { c.scrollPos = 0 }

// ScrollToBottom scrolls to the very bottom.
func (c *Chat) ScrollToBottom() { c.scrollToBottom() }

// ScrollPos returns the current scroll position.
func (c Chat) ScrollPos() int { return c.scrollPos }

// SetScrollPos sets the scroll position directly.
func (c *Chat) SetScrollPos(pos int) { c.scrollPos = max(0, pos) }

// LineOffsetForUserMessage returns the rendered line offset for the nth user
// message (0-based), or false if the index is out of range.
func (c Chat) LineOffsetForUserMessage(idx int) (int, bool) {
	if idx < 0 {
		return 0, false
	}

	userIdx := 0
	linePos := 0
	for _, entry := range c.entries {
		if entry.Role == RoleUser {
			if userIdx == idx {
				return linePos, true
			}
			userIdx++
		}
		// Mirror View rendering: entry lines + one blank separator line.
		linePos += len(c.renderEntry(entry)) + 1
	}
	return 0, false
}

// NextUserMessage jumps scroll to the next user message entry after current view
// (uses 2-line-per-entry approximation).
func (c *Chat) NextUserMessage() {
	linePos := 0
	for _, entry := range c.entries {
		entryLines := 2 // entry + blank (approximate)
		if entry.Role == RoleUser && linePos > c.scrollPos {
			c.scrollPos = linePos
			return
		}
		linePos += entryLines
	}
}

// PrevUserMessage jumps scroll to the previous user message entry before current view
// (uses 2-line-per-entry approximation).
func (c *Chat) PrevUserMessage() {
	linePos := 0
	lastUser := -1
	for _, entry := range c.entries {
		if entry.Role == RoleUser && linePos < c.scrollPos {
			lastUser = linePos
		}
		linePos += 2 // entry + blank (approximate)
	}
	if lastUser >= 0 {
		c.scrollPos = lastUser
	}
}

// Entries returns the current chat entries.
func (c Chat) Entries() []ChatEntry {
	out := make([]ChatEntry, len(c.entries))
	copy(out, c.entries)
	return out
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
		lines = append(lines, "   "+c.streaming)
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
	t := c.theme
	if t == nil {
		t = styles.DefaultTheme
	}

	switch entry.Role {
	case RoleUser:
		return c.renderUserEntry(entry, t)
	case RoleAssistant:
		return c.renderAssistantEntry(entry, t)
	case RoleTool:
		return c.renderToolEntry(entry, t)
	case RoleError:
		return c.renderErrorEntry(entry, t)
	case RoleSystem:
		return c.renderSystemEntry(entry, t)
	case RoleComplete:
		return c.renderCompleteEntry(entry, t)
	default:
		return []string{entry.Content}
	}
}

// renderUserEntry renders user messages with left accent border.
func (c Chat) renderUserEntry(entry ChatEntry, t *styles.Theme) []string {
	modeColor := t.ModeColor(c.mode)
	accentStyle := lipgloss.NewStyle().Foreground(modeColor)

	maxWidth := c.width - 4 // ┃ + space + padding
	if maxWidth < 20 {
		maxWidth = 20
	}

	wrapped := wrapText(entry.Content, maxWidth)
	contentLines := strings.Split(wrapped, "\n")

	result := make([]string, 0, len(contentLines)+1)
	for _, line := range contentLines {
		result = append(result, accentStyle.Render("┃")+" "+line)
	}
	result = append(result, accentStyle.Render("╹"))
	return result
}

// renderAssistantEntry renders assistant messages with 3-space indent.
func (c Chat) renderAssistantEntry(entry ChatEntry, _ *styles.Theme) []string {
	maxWidth := c.width - 3
	if maxWidth < 20 {
		maxWidth = 20
	}

	wrapped := wrapText(entry.Content, maxWidth)
	lines := strings.Split(wrapped, "\n")
	result := make([]string, len(lines))
	for i, line := range lines {
		result[i] = "   " + line
	}
	return result
}

// toolIcon returns the icon for a tool name.
func toolIcon(name string) string {
	switch name {
	case "bash":
		return "$"
	case "read":
		return "→"
	case "write", "edit":
		return "←"
	case "glob", "grep", "ls":
		return "⌕"
	case "todo":
		return "☐"
	default:
		return "·"
	}
}

// statusIcon returns the status indicator.
func statusIcon(status string) string {
	switch status {
	case StatusSuccess:
		return "✓"
	case StatusError:
		return "✗"
	default:
		return "~"
	}
}

// renderToolEntry renders inline tool calls with icons.
func (c Chat) renderToolEntry(entry ChatEntry, t *styles.Theme) []string {
	icon := toolIcon(entry.ToolName)
	status := statusIcon(entry.ToolStatus)

	var statusStyle lipgloss.Style
	switch entry.ToolStatus {
	case StatusSuccess:
		statusStyle = t.SuccessStyle
	case StatusError:
		statusStyle = t.ErrorStyle
	default:
		statusStyle = t.TextMutedStyle
	}

	content := truncateStr(firstLine(entry.Content), 60)
	line := "   " + statusStyle.Render(status) + " " + icon + " " +
		t.TextMutedStyle.Render(entry.ToolName) + " · " +
		t.TextMutedStyle.Render(content)

	return []string{line}
}

// renderErrorEntry renders error messages.
func (c Chat) renderErrorEntry(entry ChatEntry, t *styles.Theme) []string {
	return []string{t.ErrorStyle.Render("error") + " " + entry.Content}
}

// renderSystemEntry renders system messages with muted ~ prefix.
func (c Chat) renderSystemEntry(entry ChatEntry, t *styles.Theme) []string {
	prefix := t.TextMutedStyle.Render("~") + " "

	maxWidth := c.width - lipgloss.Width(prefix)
	if maxWidth < 20 {
		maxWidth = 20
	}

	wrapped := wrapText(entry.Content, maxWidth)
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

// renderCompleteEntry renders the completion info bar.
func (c Chat) renderCompleteEntry(entry ChatEntry, t *styles.Theme) []string {
	if entry.Meta == nil {
		return nil
	}

	modeColor := t.ModeColor(entry.Meta.Mode)
	accentStyle := lipgloss.NewStyle().Foreground(modeColor)

	modeLabel := strings.ToUpper(entry.Meta.Mode[:1]) + entry.Meta.Mode[1:]
	dur := fmt.Sprintf("%.1fs", entry.Meta.Duration.Seconds())

	line := "   " + accentStyle.Render("▣") + " " +
		t.TextMutedStyle.Render(modeLabel+" · "+entry.Meta.Model+" · "+dur)

	return []string{line}
}

func (c *Chat) scrollToBottom() {
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
