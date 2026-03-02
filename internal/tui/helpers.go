package tui

import (
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
)

// inlineEntry represents one item in the inline autocomplete popup.
type inlineEntry struct {
	Display     string
	Insert      string
	Description string
	Execute     bool
}

// pickerItem is a generic display item for renderPickerList.
type pickerItem struct {
	Label       string
	Description string
}

// renderPickerList renders a windowed selection list with "> " marker.
// header is the title line, items are the display items, selected is the
// highlighted index, and maxRows is the visible window size.
// Returns a "no matches" message if items is empty.
func renderPickerList(header string, items []pickerItem, selected, maxRows int) string {
	if len(items) == 0 {
		return header + "\n  no matches"
	}

	selected = clamp(selected, 0, len(items)-1)
	start := 0
	if selected >= maxRows {
		start = selected - maxRows + 1
	}
	end := min(len(items), start+maxRows)

	var sb strings.Builder
	sb.WriteString(header)
	for i := start; i < end; i++ {
		prefix := "  "
		if i == selected {
			prefix = "> "
		}
		sb.WriteString("\n")
		sb.WriteString(prefix)
		sb.WriteString(items[i].Label)
		if items[i].Description != "" {
			sb.WriteString("  ")
			sb.WriteString(items[i].Description)
		}
	}
	if len(items) > end {
		sb.WriteString(fmt.Sprintf("\n  ... %d more", len(items)-end))
	}
	return sb.String()
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// backspaceRune removes the last rune from s, returning the shorter string.
// Returns empty string if s is already empty.
func backspaceRune(s string) string {
	if s == "" {
		return ""
	}
	runes := []rune(s)
	return string(runes[:len(runes)-1])
}

func repeatStr(ch string, n int) string {
	if n <= 0 {
		return ""
	}
	return strings.Repeat(ch, n)
}

func containsWhitespace(s string) bool {
	return strings.ContainsAny(s, " \t\n\r")
}

func shortSessionTitle(id string) string {
	trim := strings.TrimSpace(id)
	if trim == "" {
		return "Session"
	}
	if len(trim) > 14 {
		return "Session " + trim[:14]
	}
	return "Session " + trim
}

func displayModelName(model string) string {
	parts := strings.Split(model, "/")
	if len(parts) > 1 {
		return parts[len(parts)-1]
	}
	return model
}

func formatNumber(n int) string {
	if n < 1000 {
		return fmt.Sprintf("%d", n)
	}
	if n < 1_000_000 {
		thousands := n / 1000
		remainder := n % 1000
		return fmt.Sprintf("%d,%03d", thousands, remainder)
	}
	millions := n / 1_000_000
	remainder := (n % 1_000_000) / 1000
	return fmt.Sprintf("%d,%03d,%03d", millions, remainder, n%1000)
}

// toastDismissCmd returns a tea.Cmd that dismisses a toast after 3 seconds.
// Use this in value-receiver dialog handlers that cannot call *App.ShowToast.
func toastDismissCmd(id int64) tea.Cmd {
	return func() tea.Msg {
		time.Sleep(3 * time.Second)
		return ToastDismissMsg{ID: id}
	}
}

func enabledIcon(enabled bool) string {
	if enabled {
		return "●"
	}
	return "○"
}
