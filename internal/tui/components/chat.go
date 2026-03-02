package components

import (
	"fmt"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/stephenbrandon/ripcode/internal/store"
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

// PartType identifies the kind of content in a message part.
// When adding new variants, update Valid() and all switch statements
// in parts.go and chat.go.
type PartType string

const (
	PartText      PartType = "text"      // user-visible assistant response content
	PartReasoning PartType = "reasoning" // model reasoning/thinking (may be hidden)
)

// MessagePart is a single content segment within an assistant message.
type MessagePart struct {
	Type    PartType
	Content string
}

// DiffInfo holds before/after content for tool diff rendering.
// Intentionally separate from tool.DiffInfo for layer separation — TUI components
// should not import from the tool package.
type DiffInfo struct {
	Path   string
	Before string
	After  string
	Binary bool // if true, render "[Binary file changed]" instead of diff
}

// ChatEntry represents a single rendered entry in the chat.
type ChatEntry struct {
	Role       string // RoleUser, RoleAssistant, RoleTool, RoleError, RoleSystem, RoleComplete
	Content    string
	Parts      []MessagePart // non-nil for assistant entries with mixed parts; nil for simple
	CreatedAt  time.Time     // for timestamp display
	ToolID     string        // tool call ID for matching updates
	ToolName   string        // tool name (bash, read, write, etc.)
	ToolStatus string        // StatusPending, StatusSuccess, StatusError
	Meta       *CompleteMeta // for RoleComplete entries
	Diff       *DiffInfo     // optional, for write/edit tool entries (transient, not persisted)
	FileRefs   []string      // cached @file references (parsed once at creation)
}

// minContentWidth is the floor for wrapped content width, preventing
// degenerate layouts when the terminal is very narrow.
const minContentWidth = 20

// Chat is a scrollable viewport displaying conversation messages.
type Chat struct {
	entries        []ChatEntry
	scrollPos      int
	width          int
	height         int
	streaming      string        // legacy streaming content
	streamingParts []MessagePart // part-based streaming accumulator
	mode           string
	theme          *styles.Theme

	showThinking   bool // controls reasoning part visibility
	showDetails    bool // controls tool output detail level
	showTimestamps bool // controls timestamp prefix on entries
	showCodeBlocks bool // true = show code, false = conceal
}

// NewChat creates a new chat component.
func NewChat() Chat {
	return Chat{
		mode:           "build",
		theme:          styles.DefaultTheme,
		showCodeBlocks: true,
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

// SetShowThinking controls whether reasoning parts are rendered.
func (c *Chat) SetShowThinking(v bool) { c.showThinking = v }

// ShowThinking reports whether reasoning parts are rendered.
func (c Chat) ShowThinking() bool { return c.showThinking }

// SetShowDetails controls whether tool output details are shown.
func (c *Chat) SetShowDetails(v bool) { c.showDetails = v }

// ShowDetails reports whether tool output details are shown.
func (c Chat) ShowDetails() bool { return c.showDetails }

// SetShowTimestamps controls whether timestamp prefixes are shown.
func (c *Chat) SetShowTimestamps(v bool) { c.showTimestamps = v }

// ShowTimestamps reports whether timestamp prefixes are shown.
func (c Chat) ShowTimestamps() bool { return c.showTimestamps }

// SetShowCodeBlocks controls whether fenced code blocks are visible.
func (c *Chat) SetShowCodeBlocks(v bool) { c.showCodeBlocks = v }

// ShowCodeBlocks reports whether fenced code blocks are visible.
func (c Chat) ShowCodeBlocks() bool { return c.showCodeBlocks }

// AddEntry adds a completed message to the chat.
func (c *Chat) AddEntry(entry ChatEntry) {
	if entry.CreatedAt.IsZero() {
		entry.CreatedAt = time.Now()
	}
	// Parse @file refs once at creation for user entries (avoids regex on every render).
	if entry.Role == RoleUser && entry.FileRefs == nil {
		entry.FileRefs = ParseFileRefs(entry.Content)
	}
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

// StreamContent appends to the current streaming content (legacy path).
func (c *Chat) StreamContent(delta string) {
	c.streaming += delta
	c.scrollToBottom()
}

// StreamPart appends a delta to the streaming parts accumulator.
// Consecutive deltas of the same type are merged into one part; a type change
// starts a new part. Invalid PartTypes are logged but still accumulated
// (non-blocking) to preserve content visibility during streaming.
func (c *Chat) StreamPart(typ PartType, delta string) {
	if delta == "" {
		return
	}
	if !typ.Valid() {
		store.LogErrorf("chat: StreamPart called with invalid type %q", typ)
	}
	if n := len(c.streamingParts); n > 0 && c.streamingParts[n-1].Type == typ {
		c.streamingParts[n-1].Content += delta
	} else {
		c.streamingParts = append(c.streamingParts, MessagePart{Type: typ, Content: delta})
	}
	c.scrollToBottom()
}

// CommitStream finalizes streaming content as an assistant entry.
// Two streaming paths exist for backward compatibility:
//   - Legacy: StreamContent() accumulates raw deltas (pre-Phase 5 path)
//   - Parts: StreamPart() accumulates typed content segments (text/reasoning)
//
// When using the parts path with a single text-only part, the entry uses only
// the Content field (not Parts) for backward compatibility with code that
// expects plain Content-based entries.
//
// If both paths are populated (shouldn't happen in normal use), parts take
// precedence and legacy streaming is cleared with a warning log.
func (c *Chat) CommitStream() {
	now := time.Now()

	// Part-based streaming path — takes precedence over legacy
	if len(c.streamingParts) > 0 {
		if c.streaming != "" {
			store.LogError("chat: CommitStream called with both streamingParts and legacy streaming populated; legacy content discarded", nil)
		}
		// Single text part — fall back to Content field for backward compat
		if len(c.streamingParts) == 1 && c.streamingParts[0].Type == PartText {
			c.entries = append(c.entries, ChatEntry{
				Role:      RoleAssistant,
				Content:   c.streamingParts[0].Content,
				CreatedAt: now,
			})
		} else {
			parts := make([]MessagePart, len(c.streamingParts))
			copy(parts, c.streamingParts)
			c.entries = append(c.entries, ChatEntry{
				Role:      RoleAssistant,
				Content:   plainTextFromParts(parts),
				Parts:     parts,
				CreatedAt: now,
			})
		}
		c.streamingParts = nil
		c.streaming = ""
		return
	}

	// Legacy streaming path
	if c.streaming != "" {
		c.entries = append(c.entries, ChatEntry{
			Role:      RoleAssistant,
			Content:   c.streaming,
			CreatedAt: now,
		})
		c.streaming = ""
	}
}

// plainTextFromParts extracts and concatenates user-visible text segments.
func plainTextFromParts(parts []MessagePart) string {
	var b strings.Builder
	for _, p := range parts {
		if p.Type == PartText {
			b.WriteString(p.Content)
		}
	}
	return b.String()
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

	offsets := c.userMessageOffsets()
	if idx >= len(offsets) {
		return 0, false
	}
	return offsets[idx], true
}

// userMessageOffsets returns rendered line offsets for each user message.
// Offsets mirror View rendering: each entry contributes renderEntry lines plus
// one blank separator line.
func (c Chat) userMessageOffsets() []int {
	offsets := make([]int, 0, len(c.entries)/2+1)
	linePos := 0
	for _, entry := range c.entries {
		if entry.Role == RoleUser {
			offsets = append(offsets, linePos)
		}
		linePos += len(c.renderEntry(entry)) + 1
	}
	return offsets
}

// NextUserMessage jumps scroll to the next rendered user-message offset.
func (c *Chat) NextUserMessage() {
	linePos := 0
	for _, entry := range c.entries {
		if entry.Role == RoleUser && linePos > c.scrollPos {
			c.scrollPos = linePos
			return
		}
		linePos += len(c.renderEntry(entry)) + 1
	}
}

// PrevUserMessage jumps scroll to the previous rendered user-message offset.
func (c *Chat) PrevUserMessage() {
	lastUser := -1
	linePos := 0
	for _, entry := range c.entries {
		if entry.Role == RoleUser {
			if linePos >= c.scrollPos {
				break
			}
			lastUser = linePos
		}
		linePos += len(c.renderEntry(entry)) + 1
	}
	if lastUser >= 0 {
		c.scrollPos = lastUser
	}
}

// EntryAtLine maps a rendered line position to a chat entry index.
// Returns the entry index and true if found, or (0, false) if the line
// is out of range or there are no entries.
func (c Chat) EntryAtLine(linePos int) (int, bool) {
	if linePos < 0 || len(c.entries) == 0 {
		return 0, false
	}
	pos := 0
	for i, entry := range c.entries {
		lines := c.renderEntry(entry)
		entryLines := len(lines) + 1 // +1 for blank separator
		if linePos < pos+entryLines {
			return i, true
		}
		pos += entryLines
	}
	return 0, false
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
	c.streamingParts = nil
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

	// Streaming content (part-based or legacy)
	if len(c.streamingParts) > 0 {
		lines = append(lines, c.renderAssistantParts(c.streamingParts, c.effectiveTheme())...)
	} else if c.streaming != "" {
		lines = append(lines, "   "+c.streaming)
	}

	// Apply scroll
	totalLines := len(lines)
	visibleLines := c.height

	maxScroll := max(0, totalLines-visibleLines)
	scrollPos := c.scrollPos
	if scrollPos > maxScroll {
		scrollPos = maxScroll
	}

	start := scrollPos
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

// effectiveTheme returns the chat's theme, falling back to the default.
func (c Chat) effectiveTheme() *styles.Theme {
	if c.theme != nil {
		return c.theme
	}
	return styles.DefaultTheme
}

func (c Chat) renderEntry(entry ChatEntry) []string {
	t := c.effectiveTheme()

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
		store.LogErrorf("chat: unknown entry role %q rendered as plain text", entry.Role)
		return []string{entry.Content}
	}
}

// renderUserEntry renders user messages with left accent border.
func (c Chat) renderUserEntry(entry ChatEntry, t *styles.Theme) []string {
	modeColor := t.ModeColor(c.mode)
	accentStyle := lipgloss.NewStyle().Foreground(modeColor)

	maxWidth := c.width - 4 // ┃ + space + padding
	if maxWidth < minContentWidth {
		maxWidth = minContentWidth
	}

	tsPrefix := c.timestampPrefix(entry, t)
	firstLineWidth := maxWidth - lipgloss.Width(tsPrefix)
	if firstLineWidth < 1 {
		firstLineWidth = 1
	}

	wrapped := wrapTextWithFirstLineWidth(entry.Content, firstLineWidth, maxWidth)
	contentLines := strings.Split(wrapped, "\n")

	result := make([]string, 0, len(contentLines)+2)
	for i, line := range contentLines {
		rendered := accentStyle.Render("┃") + " " + line
		if i == 0 {
			rendered = tsPrefix + rendered
		}
		result = append(result, rendered)
	}

	// File attachment badges (pre-parsed at entry creation)
	if len(entry.FileRefs) > 0 {
		badgeLine := accentStyle.Render("┃") + " "
		shown := min(len(entry.FileRefs), maxFileBadges)
		for _, ref := range entry.FileRefs[:shown] {
			badgeLine += t.TextMutedStyle.Render("📎 "+filepath.Base(ref)) + "  "
		}
		if len(entry.FileRefs) > maxFileBadges {
			badgeLine += t.TextMutedStyle.Render(fmt.Sprintf("+%d more", len(entry.FileRefs)-maxFileBadges))
		}
		result = append(result, badgeLine)
	}

	result = append(result, accentStyle.Render("╹"))
	return result
}

// contentWidth returns the maximum content width for assistant messages.
func (c Chat) contentWidth() int {
	maxWidth := c.width - 3 // 3-space indent
	if maxWidth < minContentWidth {
		return minContentWidth
	}
	return maxWidth
}

// renderAssistantEntry renders assistant messages with 3-space indent.
func (c Chat) renderAssistantEntry(entry ChatEntry, t *styles.Theme) []string {
	if len(entry.Parts) > 0 {
		result := c.renderAssistantParts(entry.Parts, t)
		if len(result) > 0 {
			result[0] = c.prependTimestamp(result[0], entry, t)
		}
		return result
	}

	maxWidth := c.contentWidth()
	content := entry.Content
	if !c.showCodeBlocks {
		content = concealCodeBlocks(content)
	}

	wrapped := wrapText(content, maxWidth)
	lines := strings.Split(wrapped, "\n")
	result := make([]string, len(lines))
	for i, line := range lines {
		result[i] = "   " + line
	}
	if len(result) > 0 {
		result[0] = c.prependTimestamp(result[0], entry, t)
	}
	return result
}

// renderAssistantParts renders a parts-based assistant message.
// Unknown PartTypes are logged as errors but rendered as text to preserve
// content visibility — graceful degradation over data loss.
func (c Chat) renderAssistantParts(parts []MessagePart, t *styles.Theme) []string {
	maxWidth := c.contentWidth()

	var result []string
	inConcealedBlock := false
	var textBuf strings.Builder
	flushText := func() {
		if textBuf.Len() == 0 {
			return
		}
		wrapped := wrapText(textBuf.String(), maxWidth)
		for _, line := range strings.Split(wrapped, "\n") {
			result = append(result, "   "+line)
		}
		textBuf.Reset()
	}

	for _, p := range parts {
		switch p.Type {
		case PartReasoning:
			if !c.showThinking {
				continue
			}
			flushText()
			reasoningStyle := t.TextMutedStyle.Italic(true)
			wrapped := wrapText(p.Content, maxWidth)
			for _, line := range strings.Split(wrapped, "\n") {
				result = append(result, "   "+reasoningStyle.Render(line))
			}
		default: // PartText or unknown (unknown types are logged and rendered as text)
			if p.Type != PartText {
				store.LogErrorf("chat: unknown part type %q rendered as text", p.Type)
			}
			text := p.Content
			if !c.showCodeBlocks {
				text, inConcealedBlock = concealCodeBlocksWithState(text, inConcealedBlock)
			}
			textBuf.WriteString(text)
		}
	}
	flushText()
	if len(result) == 0 {
		// All parts were hidden reasoning — show muted indicator
		hasReasoning := slices.ContainsFunc(parts, func(p MessagePart) bool {
			return p.Type == PartReasoning
		})
		if hasReasoning && !c.showThinking {
			result = append(result, "   "+t.TextMutedStyle.Render("[thinking]"))
		} else {
			result = append(result, "   ")
		}
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

	lines := []string{line}

	if c.showDetails && entry.ToolStatus != StatusPending {
		if entry.Diff != nil && entry.ToolStatus == StatusSuccess {
			if entry.Diff.Binary {
				lines = append(lines, "      "+t.TextMutedStyle.Render("[Binary file changed]"))
			} else if diffLines := ComputeDiff(entry.Diff.Before, entry.Diff.After, 3); len(diffLines) > 0 {
				for _, dl := range diffLines {
					lines = append(lines, "      "+RenderDiffLine(dl, t))
				}
			}
			// Empty diff (Before == After): skip diff block entirely
		} else if entry.Content != "" {
			maxW := c.width - 8
			if maxW < minContentWidth {
				maxW = minContentWidth
			}
			for _, dl := range strings.Split(entry.Content, "\n") {
				if lipgloss.Width(dl) > maxW {
					dl = ansi.Truncate(dl, maxW, "")
				}
				lines = append(lines, "      "+t.TextMutedStyle.Render(dl))
			}
		}
	}

	return lines
}

// renderErrorEntry renders error messages.
func (c Chat) renderErrorEntry(entry ChatEntry, t *styles.Theme) []string {
	return []string{t.ErrorStyle.Render("error") + " " + entry.Content}
}

// renderSystemEntry renders system messages with muted ~ prefix.
func (c Chat) renderSystemEntry(entry ChatEntry, t *styles.Theme) []string {
	prefix := t.TextMutedStyle.Render("~") + " "

	maxWidth := c.width - lipgloss.Width(prefix)
	if maxWidth < minContentWidth {
		maxWidth = minContentWidth
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

// concealCodeBlocks replaces fenced code blocks (``` ... ```) with a placeholder.
// Inline `code` is not affected. Unclosed blocks are concealed to end of string.
func concealCodeBlocks(text string) string {
	out, _ := concealCodeBlocksWithState(text, false)
	return out
}

// concealCodeBlocksWithState replaces fenced code blocks while preserving
// whether concealment is currently inside an unclosed fence. This enables
// processing text across multiple message parts where a code block may
// start in one part and end in another.
//
// State machine:
//   - inBlock=false: output lines until ``` opens a block
//   - inBlock=true: skip lines until ``` closes the block
//   - One-line blocks (```code```) are detected and handled
//   - Unclosed blocks at end of text remain in inBlock=true state
func concealCodeBlocksWithState(text string, inBlock bool) (string, bool) {
	lines := strings.Split(text, "\n")
	var result []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !inBlock && strings.HasPrefix(trimmed, "```") {
			inBlock = true
			result = append(result, "[code block hidden]")
			// Check if the same line closes it (e.g. ```code``` on one line)
			// Only if there's a second ``` after the opening one
			rest := trimmed[3:]
			if idx := strings.Index(rest, "```"); idx >= 0 {
				inBlock = false
			}
			continue
		}
		if inBlock {
			if strings.HasPrefix(trimmed, "```") {
				inBlock = false
			}
			continue
		}
		result = append(result, line)
	}
	return strings.Join(result, "\n"), inBlock
}

// prependTimestamp prepends a 12-hour timestamp prefix to a line if timestamps are enabled
// and the entry has a non-zero CreatedAt.
func (c Chat) prependTimestamp(line string, entry ChatEntry, t *styles.Theme) string {
	return c.timestampPrefix(entry, t) + line
}

func (c Chat) timestampPrefix(entry ChatEntry, t *styles.Theme) string {
	if !c.showTimestamps || entry.CreatedAt.IsZero() {
		return ""
	}
	return t.TextMutedStyle.Render(entry.CreatedAt.Format("3:04 PM") + "  ")
}

// fileRefPattern matches @file references: requires start-of-string or whitespace before @,
// captures the path (no spaces, @ or colons), with optional :N or :N-M line range suffix.
// Rejects emails (user@host) because (?:^|\s) requires whitespace or start-of-string before @.
var fileRefPattern = regexp.MustCompile(`(?:^|\s)@([^\s@:]+)(?::\d+(?:-\d+)?)?`)

// ParseFileRefs extracts @file references from text.
// Returns unique file paths found. Handles @path/to/file and @path/to/file:10-20 syntax.
func ParseFileRefs(text string) []string {
	matches := fileRefPattern.FindAllStringSubmatch(text, -1)
	if len(matches) == 0 {
		return nil
	}
	seen := make(map[string]bool, len(matches))
	var refs []string
	for _, m := range matches {
		path := m[1]
		if !seen[path] {
			seen[path] = true
			refs = append(refs, path)
		}
	}
	return refs
}

// maxFileBadges is the maximum number of file badges shown before overflow.
const maxFileBadges = 3

func (c *Chat) scrollToBottom() {
	c.scrollPos = max(0, c.totalLines()-c.height)
}

func (c Chat) totalLines() int {
	count := 0
	for _, entry := range c.entries {
		count += len(c.renderEntry(entry)) + 1
	}
	if len(c.streamingParts) > 0 {
		count += len(c.renderAssistantParts(c.streamingParts, c.effectiveTheme()))
	} else if c.streaming != "" {
		count++
	}
	return count
}

func wrapText(text string, width int) string {
	return wrapTextWithFirstLineWidth(text, width, width)
}

// wrapTextWithFirstLineWidth wraps text with a custom width for the first
// rendered line, and a separate width for all subsequent lines.
func wrapTextWithFirstLineWidth(text string, firstWidth, width int) string {
	if width <= 0 {
		return text
	}
	if firstWidth < 1 {
		firstWidth = 1
	}
	if firstWidth > width {
		firstWidth = width
	}

	var result strings.Builder
	firstRendered := true
	appendLine := func(line string) {
		if result.Len() > 0 {
			result.WriteByte('\n')
		}
		result.WriteString(line)
		firstRendered = false
	}
	currentLimit := func() int {
		if firstRendered {
			return firstWidth
		}
		return width
	}

	for _, line := range strings.Split(text, "\n") {
		limit := currentLimit()
		if lipgloss.Width(line) <= limit {
			appendLine(line)
			continue
		}

		words := strings.Fields(line)
		var currentLine string
		for _, word := range words {
			if currentLine == "" {
				currentLine = word
			} else if lipgloss.Width(currentLine+" "+word) <= limit {
				currentLine += " " + word
			} else {
				appendLine(currentLine)
				limit = currentLimit()
				currentLine = word
			}
		}
		if currentLine != "" {
			appendLine(currentLine)
		}
	}
	return result.String()
}
