package provider

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockProvider is a test double implementing Provider.
type mockProvider struct {
	name   string
	events []StreamEvent
}

func (m *mockProvider) Chat(_ context.Context, _ []Message, _ []ToolDef) (<-chan StreamEvent, error) {
	ch := make(chan StreamEvent, len(m.events))
	for _, e := range m.events {
		ch <- e
	}
	close(ch)
	return ch, nil
}

func (m *mockProvider) Name() string { return m.name }

func TestProvider_InterfaceCompliance(t *testing.T) {
	var p Provider = &mockProvider{name: "test"}
	assert.Equal(t, "test", p.Name())
}

func TestProvider_ChatReturnsEvents(t *testing.T) {
	events := []StreamEvent{
		{Type: EventContentDelta, Content: "Hello"},
		{Type: EventFinish, Meta: &Metadata{InputTokens: 10, OutputTokens: 5}},
	}
	p := &mockProvider{name: "mock", events: events}

	ch, err := p.Chat(context.Background(), nil, nil)
	require.NoError(t, err)

	var received []StreamEvent
	for e := range ch {
		received = append(received, e)
	}
	assert.Len(t, received, 2)
	assert.Equal(t, EventContentDelta, received[0].Type)
	assert.Equal(t, "Hello", received[0].Content)
	assert.Equal(t, EventFinish, received[1].Type)
	assert.Equal(t, 10, received[1].Meta.InputTokens)
}

func TestMessage_Construction(t *testing.T) {
	msg := Message{
		Role:    "assistant",
		Content: "I'll help with that.",
		ToolCalls: []ToolCall{
			{ID: "call_1", Name: "bash", Args: `{"command":"ls"}`},
		},
	}

	assert.Equal(t, RoleAssistant, msg.Role)
	assert.Len(t, msg.ToolCalls, 1)
	assert.Equal(t, "bash", msg.ToolCalls[0].Name)
}

func TestMessage_ToolResult(t *testing.T) {
	msg := Message{
		Role:       "tool",
		Content:    "file1.go\nfile2.go\n",
		ToolCallID: "call_1",
	}

	assert.Equal(t, RoleTool, msg.Role)
	assert.Equal(t, "call_1", msg.ToolCallID)
}

func TestToolDef_Structure(t *testing.T) {
	def := ToolDef{
		Name:        "read",
		Description: "Read a file",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"file_path": map[string]any{
					"type":        "string",
					"description": "Path to read",
				},
			},
			"required": []string{"file_path"},
		},
	}

	assert.Equal(t, "read", def.Name)
	props := def.Parameters["properties"].(map[string]any)
	assert.Contains(t, props, "file_path")
}

func TestEventType_Values(t *testing.T) {
	// Verify enum ordering is stable
	assert.Equal(t, EventType(0), EventContentDelta)
	assert.Equal(t, EventType(1), EventToolCall)
	assert.Equal(t, EventType(2), EventFinish)
	assert.Equal(t, EventType(3), EventError)
}

func TestModelInfo_ProviderName_ExtractsFromID(t *testing.T) {
	m := ModelInfo{ID: "anthropic/claude-4", Name: "Claude 4"}
	assert.Equal(t, "anthropic", m.ProviderName())
}

func TestModelInfo_ProviderName_NoSlash_ReturnsID(t *testing.T) {
	m := ModelInfo{ID: "gpt-4", Name: "GPT-4"}
	assert.Equal(t, "gpt-4", m.ProviderName())
}

func TestModelInfo_IsFree_NilPricing(t *testing.T) {
	m := ModelInfo{ID: "test/model", Pricing: nil}
	assert.True(t, m.IsFree())
}

func TestModelInfo_IsFree_ZeroCost(t *testing.T) {
	m := ModelInfo{
		ID:      "test/model",
		Pricing: &ModelPricing{PromptPerMillion: 0, CompletionPerMillion: 0},
	}
	assert.True(t, m.IsFree())
}

func TestModelInfo_IsFree_NonZeroCost(t *testing.T) {
	m := ModelInfo{
		ID:      "anthropic/claude-4",
		Pricing: &ModelPricing{PromptPerMillion: 3.0, CompletionPerMillion: 15.0},
	}
	assert.False(t, m.IsFree())
}

func TestRole_Valid_KnownRoles(t *testing.T) {
	assert.True(t, RoleSystem.Valid())
	assert.True(t, RoleUser.Valid())
	assert.True(t, RoleAssistant.Valid())
	assert.True(t, RoleTool.Valid())
}

func TestRole_Valid_UnknownRoles(t *testing.T) {
	assert.False(t, Role("").Valid())
	assert.False(t, Role("admin").Valid())
	assert.False(t, Role("USER").Valid())
}

func TestRoleConstants_MatchExpectedValues(t *testing.T) {
	assert.Equal(t, Role("system"), RoleSystem)
	assert.Equal(t, Role("user"), RoleUser)
	assert.Equal(t, Role("assistant"), RoleAssistant)
	assert.Equal(t, Role("tool"), RoleTool)

	// Verify Role is a defined type — constants work in Message fields
	msg := Message{Role: RoleUser, Content: "hello"}
	assert.Equal(t, RoleUser, msg.Role)
}

func TestMessage_Valid_UserMessage(t *testing.T) {
	m := Message{Role: RoleUser, Content: "hello"}
	assert.NoError(t, m.Valid())
}

func TestMessage_Valid_AssistantWithToolCalls(t *testing.T) {
	m := Message{
		Role:      RoleAssistant,
		Content:   "calling tool",
		ToolCalls: []ToolCall{{ID: "1", Name: "bash", Args: "{}"}},
	}
	assert.NoError(t, m.Valid())
}

func TestMessage_Valid_ToolResult(t *testing.T) {
	m := Message{Role: RoleTool, Content: "output", ToolCallID: "1"}
	assert.NoError(t, m.Valid())
}

func TestMessage_Valid_InvalidRole(t *testing.T) {
	m := Message{Role: "admin", Content: "hello"}
	assert.Error(t, m.Valid())
}

func TestMessage_Valid_ToolCallsOnNonAssistant(t *testing.T) {
	m := Message{
		Role:      RoleUser,
		ToolCalls: []ToolCall{{ID: "1", Name: "bash", Args: "{}"}},
	}
	assert.Error(t, m.Valid())
}

func TestMessage_Valid_ToolCallIDOnNonTool(t *testing.T) {
	m := Message{Role: RoleAssistant, Content: "hi", ToolCallID: "1"}
	assert.Error(t, m.Valid())
}

func TestMessage_Valid_ToolMissingCallID(t *testing.T) {
	m := Message{Role: RoleTool, Content: "output"}
	assert.Error(t, m.Valid())
}

func TestStreamEvent_Valid_ContentDelta(t *testing.T) {
	e := StreamEvent{Type: EventContentDelta, Content: "hello"}
	assert.NoError(t, e.Valid())
}

func TestStreamEvent_Valid_ContentDelta_Empty(t *testing.T) {
	e := StreamEvent{Type: EventContentDelta}
	assert.NoError(t, e.Valid(), "empty content delta is valid (whitespace-only deltas)")
}

func TestStreamEvent_Valid_ToolCall(t *testing.T) {
	e := StreamEvent{Type: EventToolCall, ToolCall: &ToolCall{ID: "1", Name: "bash"}}
	assert.NoError(t, e.Valid())
}

func TestStreamEvent_Valid_ToolCall_MissingToolCall(t *testing.T) {
	e := StreamEvent{Type: EventToolCall}
	err := e.Valid()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "missing ToolCall")
}

func TestStreamEvent_Valid_Finish(t *testing.T) {
	e := StreamEvent{Type: EventFinish, Meta: &Metadata{InputTokens: 10}}
	assert.NoError(t, e.Valid())
}

func TestStreamEvent_Valid_Finish_MissingMeta(t *testing.T) {
	e := StreamEvent{Type: EventFinish}
	err := e.Valid()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "missing Meta")
}

func TestStreamEvent_Valid_Error(t *testing.T) {
	e := StreamEvent{Type: EventError, Error: fmt.Errorf("fail")}
	assert.NoError(t, e.Valid())
}

func TestStreamEvent_Valid_Error_MissingError(t *testing.T) {
	e := StreamEvent{Type: EventError}
	err := e.Valid()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "missing Error")
}

func TestStreamEvent_Valid_UnknownType(t *testing.T) {
	e := StreamEvent{Type: EventType(99)}
	err := e.Valid()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown event type")
}

func TestStreamEvent_Constructors(t *testing.T) {
	e1 := NewContentDelta("hello")
	assert.Equal(t, EventContentDelta, e1.Type)
	assert.Equal(t, "hello", e1.Content)

	tc := &ToolCall{ID: "1", Name: "bash", Args: "{}"}
	e2 := NewToolCallEvent(tc)
	assert.Equal(t, EventToolCall, e2.Type)
	assert.Equal(t, tc, e2.ToolCall)

	meta := &Metadata{InputTokens: 10}
	e3 := NewFinishEvent(meta)
	assert.Equal(t, EventFinish, e3.Type)
	assert.Equal(t, meta, e3.Meta)

	e4 := NewErrorEvent(fmt.Errorf("fail"))
	assert.Equal(t, EventError, e4.Type)
	assert.Equal(t, "fail", e4.Error.Error())
}
