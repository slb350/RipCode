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

// validRole reports whether r is a known ChatEntry role.
func validRole(r string) bool {
	switch r {
	case RoleUser, RoleAssistant, RoleTool, RoleError, RoleSystem, RoleComplete:
		return true
	default:
		return false
	}
}

// Valid checks that the ChatEntry has a recognized role, valid parts, and
// consistent Content/Parts fields.
func (e ChatEntry) Valid() error {
	if !validRole(e.Role) {
		return fmt.Errorf("invalid role: %q", e.Role)
	}
	if e.Role == RoleTool && e.ToolID == "" {
		return fmt.Errorf("tool entry requires ToolID")
	}
	if e.Role == RoleComplete && e.Meta == nil {
		return fmt.Errorf("complete entry requires Meta")
	}
	for i, p := range e.Parts {
		if err := p.Valid(); err != nil {
			return fmt.Errorf("part[%d]: %w", i, err)
		}
	}
	if len(e.Parts) > 0 && e.Content != "" {
		expected := plainTextFromParts(e.Parts)
		if e.Content != expected {
			return fmt.Errorf("Content mismatch: got %q, want %q (from text parts)", e.Content, expected)
		}
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
// Uses Content if non-empty; otherwise concatenates all parts (including reasoning).
func (e ChatEntry) CopyableContent() string {
	if e.Content != "" {
		return e.Content
	}
	return e.allPartsContent()
}
