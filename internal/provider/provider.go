package provider

import "context"

// Message represents a single message in a conversation.
type Message struct {
	Role       string // "system", "user", "assistant", "tool"
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

// ModelInfo represents a model entry available from a provider.
type ModelInfo struct {
	ID   string
	Name string
}

// ToolDef describes a tool for the LLM's tool-use schema.
type ToolDef struct {
	Name        string
	Description string
	Parameters  map[string]any // JSON Schema object
}

// Provider is the interface for LLM backends.
type Provider interface {
	// Chat sends messages and streams the response as events.
	Chat(ctx context.Context, msgs []Message, tools []ToolDef) (<-chan StreamEvent, error)
	// Name returns the provider identifier.
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
