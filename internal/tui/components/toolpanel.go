package components

import (
	"fmt"
	"strings"

	"github.com/stephenbrandon/ripcode/internal/agent"
	"github.com/stephenbrandon/ripcode/internal/tui/styles"
)

// ToolPanel displays tool execution feedback.
type ToolPanel struct {
	width    int
	events   []agent.ToolEvent
	maxShown int
}

// NewToolPanel creates a new tool panel.
func NewToolPanel() ToolPanel {
	return ToolPanel{maxShown: 5}
}

// SetSize updates the panel width.
func (tp *ToolPanel) SetSize(width int) {
	tp.width = width
}

// AddEvent records a tool event.
func (tp *ToolPanel) AddEvent(event agent.ToolEvent) {
	tp.events = append(tp.events, event)
	// Keep only recent events
	if len(tp.events) > tp.maxShown*2 {
		tp.events = tp.events[len(tp.events)-tp.maxShown:]
	}
}

// Clear removes all events.
func (tp *ToolPanel) Clear() {
	tp.events = nil
}

// View renders the most recent tool events.
func (tp ToolPanel) View() string {
	if len(tp.events) == 0 {
		return ""
	}

	var sb strings.Builder
	start := max(0, len(tp.events)-tp.maxShown)
	for i := start; i < len(tp.events); i++ {
		event := tp.events[i]
		if event.Error != "" {
			sb.WriteString(styles.Error.Render(fmt.Sprintf("  ✗ %s: %s", event.Name, truncateStr(event.Error, 60))))
		} else if event.Output != "" {
			sb.WriteString(styles.ToolCmd.Render(fmt.Sprintf("  ✓ %s", event.Name)))
			sb.WriteString(styles.Muted.Render(fmt.Sprintf(" (%s)", truncateStr(firstLine(event.Output), 50))))
		} else {
			sb.WriteString(styles.Tool.Render(fmt.Sprintf("  ⋯ %s", event.Name)))
		}
		sb.WriteByte('\n')
	}

	return sb.String()
}

func truncateStr(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-3] + "..."
}

func firstLine(s string) string {
	if idx := strings.IndexByte(s, '\n'); idx >= 0 {
		return s[:idx]
	}
	return s
}
