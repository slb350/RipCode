package tool

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/stephenbrandon/ripcode/internal/provider"
)

// skipDirs contains directory names to skip during file traversal.
var skipDirs = map[string]bool{
	".git":         true,
	"node_modules": true,
	".next":        true,
	"__pycache__":  true,
	".venv":        true,
}

const maxTrackedPaths = 10

// skipTracker collects skip counts categorized by error reason.
type skipTracker struct {
	reasons map[string]int
	paths   []string
}

func newSkipTracker() *skipTracker {
	return &skipTracker{reasons: make(map[string]int)}
}

func (s *skipTracker) add(err error) {
	reason := classifyError(err)
	s.reasons[reason]++
}

func (s *skipTracker) addPath(path string, err error) {
	s.add(err)
	if len(s.paths) < maxTrackedPaths {
		s.paths = append(s.paths, path)
	}
}

func (s *skipTracker) count() int {
	n := 0
	for _, c := range s.reasons {
		n += c
	}
	return n
}

// note returns a formatted skip message, or "" if nothing was skipped.
// Format: "\n(3 paths skipped: 2 permission denied, 1 not a directory)"
func (s *skipTracker) note(noun string) string {
	if len(s.reasons) == 0 {
		return ""
	}

	total := s.count()

	// Sort reasons for deterministic output.
	type entry struct {
		reason string
		count  int
	}
	entries := make([]entry, 0, len(s.reasons))
	for r, c := range s.reasons {
		entries = append(entries, entry{r, c})
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].count != entries[j].count {
			return entries[i].count > entries[j].count // highest count first
		}
		return entries[i].reason < entries[j].reason
	})

	parts := make([]string, len(entries))
	for i, e := range entries {
		parts[i] = fmt.Sprintf("%d %s", e.count, e.reason)
	}

	msg := fmt.Sprintf("\n(%d %s skipped: %s)", total, noun, strings.Join(parts, ", "))
	if len(s.paths) > 0 {
		msg += " [" + strings.Join(s.paths, ", ")
		if total > len(s.paths) {
			msg += fmt.Sprintf(", ... and %d more", total-len(s.paths))
		}
		msg += "]"
	}
	return msg
}

// classifyError maps an error to a human-readable reason category.
// Returns "unknown" for nil errors (defensive; callers should not pass nil).
func classifyError(err error) string {
	switch {
	case err == nil:
		return "unknown"
	case errors.Is(err, os.ErrPermission):
		return "permission denied"
	case errors.Is(err, os.ErrNotExist):
		return "not found"
	default:
		return "error"
	}
}

// Context carries execution context for tool invocations.
type Context struct {
	SessionID string
	WorkDir   string
	Abort     context.Context
	OnStatus  func(title string) // optional callback for TUI status
}

// Result holds the output of a tool execution.
type Result struct {
	Output string
	Title  string
	Error  error
}

// Tool defines the interface that all tools must implement.
type Tool interface {
	ID() string
	Description() string
	Parameters() map[string]any // JSON Schema
	Execute(ctx Context, argsJSON string) Result
}

// Registry manages available tools.
type Registry struct {
	tools map[string]Tool
	order []string // insertion order
}

// NewRegistry creates an empty tool registry.
func NewRegistry() *Registry {
	return &Registry{
		tools: make(map[string]Tool),
	}
}

// Register adds a tool to the registry.
func (r *Registry) Register(t Tool) {
	id := t.ID()
	if _, exists := r.tools[id]; !exists {
		r.order = append(r.order, id)
	}
	r.tools[id] = t
}

// Get retrieves a tool by ID.
func (r *Registry) Get(id string) (Tool, bool) {
	t, ok := r.tools[id]
	return t, ok
}

// List returns all registered tools in insertion order.
func (r *Registry) List() []Tool {
	result := make([]Tool, 0, len(r.order))
	for _, id := range r.order {
		result = append(result, r.tools[id])
	}
	return result
}

// Definitions returns provider.ToolDef for all registered tools,
// suitable for passing to a provider.Provider.Chat call.
func (r *Registry) Definitions() []provider.ToolDef {
	defs := make([]provider.ToolDef, 0, len(r.order))
	for _, id := range r.order {
		t := r.tools[id]
		defs = append(defs, provider.ToolDef{
			Name:        t.ID(),
			Description: t.Description(),
			Parameters:  t.Parameters(),
		})
	}
	return defs
}
