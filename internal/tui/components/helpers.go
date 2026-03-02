package components

import "strings"

// truncateStr shortens s to at most n characters, adding "..." if truncated.
func truncateStr(s string, n int) string {
	if n < 4 {
		return s
	}
	if len(s) <= n {
		return s
	}
	return s[:n-3] + "..."
}

// firstLine returns the content up to the first newline, or the full string.
func firstLine(s string) string {
	if idx := strings.IndexByte(s, '\n'); idx >= 0 {
		return s[:idx]
	}
	return s
}
