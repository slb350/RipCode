package provider

import (
	"context"
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

	assert.Equal(t, "assistant", msg.Role)
	assert.Len(t, msg.ToolCalls, 1)
	assert.Equal(t, "bash", msg.ToolCalls[0].Name)
}

func TestMessage_ToolResult(t *testing.T) {
	msg := Message{
		Role:       "tool",
		Content:    "file1.go\nfile2.go\n",
		ToolCallID: "call_1",
	}

	assert.Equal(t, "tool", msg.Role)
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
