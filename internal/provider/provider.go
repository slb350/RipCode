package provider

import (
	"context"
	"strings"
)

// Role is a defined type for message roles. Use the constants
// RoleSystem, RoleUser, RoleAssistant, RoleTool.
type Role string

const (
	RoleSystem    Role = "system"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleTool      Role = "tool"
)

// Message represents a single message in a conversation.
type Message struct {
	Role       Role // RoleSystem, RoleUser, RoleAssistant, or RoleTool
	Content    string
	ToolCalls  []ToolCall // present on assistant messages with tool invocations
	ToolCallID string     // present on tool result messages
}

// ToolCall represents an LLM request to invoke a tool.
type ToolCall struct {
	ID   string
	Name string
	Args string // raw JSON arguments
}

// EventType identifies the kind of streaming event.
type EventType int

const (
	EventContentDelta EventType = iota
	EventToolCall
	EventFinish
	EventError
)

// StreamEvent is a single event in a streaming LLM response.
type StreamEvent struct {
	Type     EventType
	Content  string    // text delta (EventContentDelta)
	ToolCall *ToolCall // tool invocation (EventToolCall)
	Meta     *Metadata // finish info (EventFinish)
	Error    error     // error details (EventError)
}

// Metadata carries usage and completion information.
type Metadata struct {
	InputTokens  int
	OutputTokens int
	Model        string
	FinishReason string // "stop", "tool_calls", "length"
}

// ModelPricing holds per-million-token pricing for a model.
type ModelPricing struct {
	PromptPerMillion     float64
	CompletionPerMillion float64
}

// ModelInfo represents a model entry available from a provider.
type ModelInfo struct {
	ID            string
	Name          string
	Description   string
	ContextLength int
	Pricing       *ModelPricing
}

// ProviderName extracts the provider prefix from the model ID.
// For "anthropic/claude-4" it returns "anthropic".
// If there is no slash, it returns the full ID.
func (m ModelInfo) ProviderName() string {
	if idx := strings.IndexByte(m.ID, '/'); idx >= 0 {
		return m.ID[:idx]
	}
	return m.ID
}

// IsFree returns true if the model has no pricing or zero cost.
func (m ModelInfo) IsFree() bool {
	if m.Pricing == nil {
		return true
	}
	return m.Pricing.PromptPerMillion == 0 && m.Pricing.CompletionPerMillion == 0
}

// ToolDef describes a tool for the LLM's tool-use schema.
type ToolDef struct {
	Name        string
	Description string
	Parameters  map[string]any // JSON Schema object
}

// Provider is the interface for LLM backends.
type Provider interface {
	// Chat sends messages and streams the response as events on the returned channel.
	// The channel is closed when the stream ends (either successfully or on error).
	// Callers must drain the channel to avoid goroutine leaks.
	// Implementations should respect context cancellation and close the channel promptly.
	Chat(ctx context.Context, msgs []Message, tools []ToolDef) (<-chan StreamEvent, error)
	// Name returns the provider identifier (e.g., "openrouter").
	Name() string
}

// ModelLister is implemented by providers that can enumerate available models.
type ModelLister interface {
	ListModels(ctx context.Context) ([]ModelInfo, error)
}

// ModelSetter is implemented by providers that support changing active model at runtime.
type ModelSetter interface {
	SetModel(model string)
}

// ReasoningEffortSetter is implemented by providers that support thinking budget variants.
// Effort values: "low", "medium", "high" or "" to disable.
type ReasoningEffortSetter interface {
	SetReasoningEffort(effort string)
}
