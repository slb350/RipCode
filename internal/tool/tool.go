package tool

import (
	"context"

	"github.com/stephenbrandon/ripcode/internal/provider"
)

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
