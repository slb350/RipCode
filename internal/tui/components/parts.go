package components

import (
	"fmt"
	"strings"
)

// Valid reports whether the PartType is a recognized variant.
func (pt PartType) Valid() bool {
	switch pt {
	case PartText, PartReasoning:
		return true
	default:
		return false
	}
}

// Valid checks that the MessagePart has a recognized type.
func (p MessagePart) Valid() error {
	if !p.Type.Valid() {
		return fmt.Errorf("invalid part type: %q", p.Type)
	}
	return nil
}

// allPartsContent concatenates content from all parts.
func (e ChatEntry) allPartsContent() string {
	var b strings.Builder
	for _, p := range e.Parts {
		b.WriteString(p.Content)
	}
	return b.String()
}

// FullContent returns all parts content including reasoning.
// Falls back to Content when no parts exist.
func (e ChatEntry) FullContent() string {
	if len(e.Parts) == 0 {
		return e.Content
	}
	return e.allPartsContent()
}

// CopyableContent returns the best content for clipboard copy.
// Uses Content if non-empty; otherwise concatenates all parts content.
func (e ChatEntry) CopyableContent() string {
	if e.Content != "" {
		return e.Content
	}
	return e.allPartsContent()
}
