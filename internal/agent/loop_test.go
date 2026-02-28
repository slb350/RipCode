package agent

import (
	"context"
	"fmt"
	"testing"

	"github.com/stephenbrandon/ripcode/internal/provider"
	"github.com/stephenbrandon/ripcode/internal/session"
	"github.com/stephenbrandon/ripcode/internal/tool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockProvider is a test double that returns pre-configured responses.
type mockProvider struct {
	// responses is consumed in order; each call to Chat pops the first entry.
	responses []mockResponse
	callCount int
}

type mockResponse struct {
	events []provider.StreamEvent
	err    error
}

func (m *mockProvider) Chat(_ context.Context, _ []provider.Message, _ []provider.ToolDef) (<-chan provider.StreamEvent, error) {
	if m.callCount >= len(m.responses) {
		return nil, fmt.Errorf("no more mock responses")
	}

	resp := m.responses[m.callCount]
	m.callCount++

	if resp.err != nil {
		return nil, resp.err
	}

	ch := make(chan provider.StreamEvent, len(resp.events))
	for _, e := range resp.events {
		ch <- e
	}
	close(ch)
	return ch, nil
}

func (m *mockProvider) Name() string { return "mock" }

// echoTool is a simple tool that echoes its args.
type echoTool struct{}

func (e *echoTool) ID() string              { return "echo" }
func (e *echoTool) Description() string      { return "Echo args" }
func (e *echoTool) Parameters() map[string]any { return map[string]any{"type": "object"} }
func (e *echoTool) Execute(_ tool.Context, args string) tool.Result {
	return tool.Result{Output: "echoed: " + args}
}

// failTool is a tool that always returns an error.
type failTool struct{}

func (f *failTool) ID() string              { return "fail" }
func (f *failTool) Description() string      { return "Always fails" }
func (f *failTool) Parameters() map[string]any { return map[string]any{"type": "object"} }
func (f *failTool) Execute(_ tool.Context, _ string) tool.Result {
	return tool.Result{Error: fmt.Errorf("tool failed")}
}

func newTestRegistry(tools ...tool.Tool) *tool.Registry {
	reg := tool.NewRegistry()
	for _, t := range tools {
		reg.Register(t)
	}
	return reg
}

func collectEvents(ch <-chan Event) []Event {
	var events []Event
	for e := range ch {
		events = append(events, e)
	}
	return events
}

func TestLoop_SingleTurn_NoToolCalls(t *testing.T) {
	p := &mockProvider{
		responses: []mockResponse{
			{events: []provider.StreamEvent{
				{Type: provider.EventContentDelta, Content: "Hello "},
				{Type: provider.EventContentDelta, Content: "world"},
				{Type: provider.EventFinish, Meta: &provider.Metadata{
					InputTokens: 10, OutputTokens: 5, FinishReason: "stop",
				}},
			}},
		},
	}

	reg := newTestRegistry(&echoTool{})
	sess := session.New("/tmp")
	loop := NewLoop(p, reg, sess, BuildAgent(), 10)

	events := collectEvents(loop.Run(context.Background(), "hi"))

	// Should have content deltas and done
	var content string
	var gotDone bool
	for _, e := range events {
		switch e.Type {
		case EventContentDelta:
			content += e.Content
		case EventDone:
			gotDone = true
		}
	}

	assert.Equal(t, "Hello world", content)
	assert.True(t, gotDone)
	assert.Equal(t, 1, p.callCount)
}

func TestLoop_ToolCallRoundTrip(t *testing.T) {
	p := &mockProvider{
		responses: []mockResponse{
			// First response: tool call
			{events: []provider.StreamEvent{
				{Type: provider.EventContentDelta, Content: "Let me check."},
				{Type: provider.EventToolCall, ToolCall: &provider.ToolCall{
					ID: "call_1", Name: "echo", Args: `{"msg":"hello"}`,
				}},
				{Type: provider.EventFinish, Meta: &provider.Metadata{
					InputTokens: 10, OutputTokens: 5, FinishReason: "tool_calls",
				}},
			}},
			// Second response: final answer
			{events: []provider.StreamEvent{
				{Type: provider.EventContentDelta, Content: "Done!"},
				{Type: provider.EventFinish, Meta: &provider.Metadata{
					InputTokens: 20, OutputTokens: 8, FinishReason: "stop",
				}},
			}},
		},
	}

	reg := newTestRegistry(&echoTool{})
	sess := session.New("/tmp")
	loop := NewLoop(p, reg, sess, BuildAgent(), 10)

	events := collectEvents(loop.Run(context.Background(), "do something"))

	// Should see: content, tool start, tool end, content, done
	var types []EventType
	for _, e := range events {
		types = append(types, e.Type)
	}

	assert.Contains(t, types, EventContentDelta)
	assert.Contains(t, types, EventToolStart)
	assert.Contains(t, types, EventToolEnd)
	assert.Contains(t, types, EventDone)

	// Verify tool events
	for _, e := range events {
		if e.Type == EventToolStart {
			assert.Equal(t, "echo", e.Tool.Name)
			assert.Equal(t, "call_1", e.Tool.ID)
		}
		if e.Type == EventToolEnd {
			assert.Contains(t, e.Tool.Output, "echoed")
		}
	}

	assert.Equal(t, 2, p.callCount)
}

func TestLoop_ToolError_ContinuesLoop(t *testing.T) {
	p := &mockProvider{
		responses: []mockResponse{
			// Tool call to failing tool
			{events: []provider.StreamEvent{
				{Type: provider.EventToolCall, ToolCall: &provider.ToolCall{
					ID: "call_1", Name: "fail", Args: "{}",
				}},
				{Type: provider.EventFinish, Meta: &provider.Metadata{
					FinishReason: "tool_calls",
				}},
			}},
			// Final response
			{events: []provider.StreamEvent{
				{Type: provider.EventContentDelta, Content: "Tool failed, sorry."},
				{Type: provider.EventFinish, Meta: &provider.Metadata{
					FinishReason: "stop",
				}},
			}},
		},
	}

	reg := newTestRegistry(&failTool{})
	sess := session.New("/tmp")
	loop := NewLoop(p, reg, sess, BuildAgent(), 10)

	events := collectEvents(loop.Run(context.Background(), "try it"))

	// Should complete successfully despite tool error
	var gotDone bool
	for _, e := range events {
		if e.Type == EventDone {
			gotDone = true
		}
		if e.Type == EventToolEnd {
			assert.NotEmpty(t, e.Tool.Error)
		}
	}
	assert.True(t, gotDone)
	assert.Equal(t, 2, p.callCount)
}

func TestLoop_MaxSteps(t *testing.T) {
	// Provider always returns tool calls — should hit max steps
	makeResp := func() mockResponse {
		return mockResponse{events: []provider.StreamEvent{
			{Type: provider.EventToolCall, ToolCall: &provider.ToolCall{
				ID: "call_x", Name: "echo", Args: "{}",
			}},
			{Type: provider.EventFinish, Meta: &provider.Metadata{
				FinishReason: "tool_calls",
			}},
		}}
	}

	p := &mockProvider{
		responses: []mockResponse{makeResp(), makeResp(), makeResp(), makeResp()},
	}

	reg := newTestRegistry(&echoTool{})
	sess := session.New("/tmp")
	loop := NewLoop(p, reg, sess, BuildAgent(), 3) // max 3 steps

	events := collectEvents(loop.Run(context.Background(), "loop forever"))

	var gotError bool
	for _, e := range events {
		if e.Type == EventError {
			gotError = true
			assert.Contains(t, e.Error.Error(), "max steps")
		}
	}
	assert.True(t, gotError)
}

func TestLoop_ContextCancellation(t *testing.T) {
	p := &mockProvider{
		responses: []mockResponse{
			{events: []provider.StreamEvent{
				{Type: provider.EventContentDelta, Content: "start"},
				{Type: provider.EventFinish, Meta: &provider.Metadata{FinishReason: "stop"}},
			}},
		},
	}

	reg := newTestRegistry()
	sess := session.New("/tmp")
	loop := NewLoop(p, reg, sess, BuildAgent(), 10)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	events := collectEvents(loop.Run(ctx, "cancelled"))

	// Should get an error event or just close cleanly
	var hasError bool
	for _, e := range events {
		if e.Type == EventError {
			hasError = true
		}
	}
	// With immediate cancellation, we should get an error
	assert.True(t, hasError)
}

func TestLoop_SessionUpdated(t *testing.T) {
	p := &mockProvider{
		responses: []mockResponse{
			{events: []provider.StreamEvent{
				{Type: provider.EventContentDelta, Content: "response"},
				{Type: provider.EventFinish, Meta: &provider.Metadata{
					InputTokens: 10, OutputTokens: 5, FinishReason: "stop",
				}},
			}},
		},
	}

	reg := newTestRegistry()
	sess := session.New("/tmp")
	loop := NewLoop(p, reg, sess, BuildAgent(), 10)

	events := collectEvents(loop.Run(context.Background(), "test input"))
	require.NotEmpty(t, events)

	// Session should have user + assistant messages
	assert.Len(t, sess.Messages, 2)
	assert.Equal(t, "user", sess.Messages[0].Role)
	assert.Equal(t, "test input", sess.Messages[0].Content)
	assert.Equal(t, "assistant", sess.Messages[1].Role)
	assert.Equal(t, "response", sess.Messages[1].Content)

	// Tokens should be tracked
	assert.Equal(t, 10, sess.Tokens.Input)
	assert.Equal(t, 5, sess.Tokens.Output)
}

func TestLoop_UnknownTool(t *testing.T) {
	p := &mockProvider{
		responses: []mockResponse{
			{events: []provider.StreamEvent{
				{Type: provider.EventToolCall, ToolCall: &provider.ToolCall{
					ID: "call_1", Name: "nonexistent", Args: "{}",
				}},
				{Type: provider.EventFinish, Meta: &provider.Metadata{
					FinishReason: "tool_calls",
				}},
			}},
			// After unknown tool error, provider gives final response
			{events: []provider.StreamEvent{
				{Type: provider.EventContentDelta, Content: "Sorry."},
				{Type: provider.EventFinish, Meta: &provider.Metadata{FinishReason: "stop"}},
			}},
		},
	}

	reg := newTestRegistry(&echoTool{})
	sess := session.New("/tmp")
	loop := NewLoop(p, reg, sess, BuildAgent(), 10)

	events := collectEvents(loop.Run(context.Background(), "use unknown"))

	// Should handle gracefully — tool end with error, then continue
	var toolEndError string
	for _, e := range events {
		if e.Type == EventToolEnd && e.Tool.Error != "" {
			toolEndError = e.Tool.Error
		}
	}
	assert.Contains(t, toolEndError, "unknown tool")
}
