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

func (e *echoTool) ID() string                 { return "echo" }
func (e *echoTool) Description() string        { return "Echo args" }
func (e *echoTool) Parameters() map[string]any { return map[string]any{"type": "object"} }
func (e *echoTool) Execute(_ tool.Context, args string) tool.Result {
	return tool.Result{Output: "echoed: " + args}
}

// failTool is a tool that always returns an error.
type failTool struct{}

func (f *failTool) ID() string                 { return "fail" }
func (f *failTool) Description() string        { return "Always fails" }
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
	assert.Len(t, sess.Records(), 2)
	assert.Equal(t, provider.RoleUser, sess.Records()[0].Message.Role)
	assert.Equal(t, "test input", sess.Records()[0].Message.Content)
	assert.Equal(t, provider.RoleAssistant, sess.Records()[1].Message.Role)
	assert.Equal(t, "response", sess.Records()[1].Message.Content)

	// Assistant should have metadata
	require.NotNil(t, sess.Records()[1].Meta)
	assert.Equal(t, 10, sess.Records()[1].Meta.InputTokens)
	assert.Equal(t, 5, sess.Records()[1].Meta.OutputTokens)
	assert.Equal(t, "stop", sess.Records()[1].Meta.FinishReason)

	// Tokens should be tracked
	assert.Equal(t, 10, sess.TokenCount().Input)
	assert.Equal(t, 5, sess.TokenCount().Output)
}

func TestLoop_StreamEventError(t *testing.T) {
	// Provider returns EventError mid-stream — verify the loop emits EventError.
	p := &mockProvider{
		responses: []mockResponse{
			{events: []provider.StreamEvent{
				{Type: provider.EventContentDelta, Content: "partial"},
				{Type: provider.EventError, Error: fmt.Errorf("stream broke")},
			}},
		},
	}

	reg := newTestRegistry()
	sess := session.New("/tmp")
	loop := NewLoop(p, reg, sess, BuildAgent(), 10)

	events := collectEvents(loop.Run(context.Background(), "test"))

	var gotError bool
	var errMsg string
	for _, e := range events {
		if e.Type == EventError {
			gotError = true
			errMsg = e.Error.Error()
		}
	}
	assert.True(t, gotError, "should emit EventError for stream error")
	assert.Contains(t, errMsg, "stream broke")
}

// trackingTool records whether Execute was called.
type trackingTool struct {
	executed *bool
}

func (t *trackingTool) ID() string                 { return "track" }
func (t *trackingTool) Description() string        { return "Tracks execution" }
func (t *trackingTool) Parameters() map[string]any { return map[string]any{"type": "object"} }
func (t *trackingTool) Execute(_ tool.Context, _ string) tool.Result {
	*t.executed = true
	return tool.Result{Output: "executed"}
}

// slowTool blocks until the context is cancelled, simulating a long-running tool.
type slowTool struct {
	started chan struct{} // closed when Execute begins
}

func (s *slowTool) ID() string                 { return "slow" }
func (s *slowTool) Description() string        { return "Blocks until cancelled" }
func (s *slowTool) Parameters() map[string]any { return map[string]any{"type": "object"} }
func (s *slowTool) Execute(ctx tool.Context, _ string) tool.Result {
	close(s.started)
	<-ctx.Abort.Done()
	return tool.Result{Output: "cancelled"}
}

func TestLoop_CancelDuringToolExecution(t *testing.T) {
	st := &slowTool{started: make(chan struct{})}

	p := &mockProvider{
		responses: []mockResponse{
			{events: []provider.StreamEvent{
				{Type: provider.EventToolCall, ToolCall: &provider.ToolCall{
					ID: "call_slow", Name: "slow", Args: "{}",
				}},
				{Type: provider.EventFinish, Meta: &provider.Metadata{
					FinishReason: "tool_calls",
				}},
			}},
		},
	}

	reg := newTestRegistry(st)
	sess := session.New("/tmp")
	loop := NewLoop(p, reg, sess, BuildAgent(), 10)

	ctx, cancel := context.WithCancel(context.Background())
	ch := loop.Run(ctx, "run slow tool")

	// Wait for the slow tool to start executing
	<-st.started

	// Cancel while the tool is running
	cancel()

	// Drain events
	events := collectEvents(ch)

	// Should see a ToolStart event for "slow"
	var gotToolStart bool
	for _, e := range events {
		if e.Type == EventToolStart && e.Tool.Name == "slow" {
			gotToolStart = true
		}
	}
	assert.True(t, gotToolStart, "should have started the slow tool")

	// The loop should terminate (channel closed) — it shouldn't hang
	// The next iteration's ctx.Done() check prevents further provider calls
	assert.Equal(t, 1, p.callCount, "should only call provider once before cancellation stops the loop")
}

func TestLoop_CancelMidStream_DiscardsPartialToolCalls(t *testing.T) {
	// Provider returns tool calls, but we cancel the context before the loop
	// processes the tool execution. The loop should discard the tool calls
	// (set toolCalls = nil) and NOT execute the tool.
	toolExecuted := false
	trackTool := &trackingTool{executed: &toolExecuted}

	p := &mockProvider{
		responses: []mockResponse{
			{events: []provider.StreamEvent{
				{Type: provider.EventContentDelta, Content: "planning"},
				{Type: provider.EventToolCall, ToolCall: &provider.ToolCall{
					ID: "call_1", Name: "track", Args: "{}",
				}},
				{Type: provider.EventFinish, Meta: &provider.Metadata{
					FinishReason: "tool_calls",
				}},
			}},
		},
	}

	reg := newTestRegistry(trackTool)
	sess := session.New("/tmp")
	loop := NewLoop(p, reg, sess, BuildAgent(), 10)

	ctx, cancel := context.WithCancel(context.Background())

	// Cancel immediately so the loop sees ctx.Err() after consuming the stream
	cancel()

	events := collectEvents(loop.Run(ctx, "test cancel"))

	// Should get an error event (context cancelled)
	var hasError bool
	for _, e := range events {
		if e.Type == EventError {
			hasError = true
		}
	}
	assert.True(t, hasError, "should emit error on cancellation")
	assert.False(t, toolExecuted, "tool should NOT be executed when context is cancelled mid-stream")
}

func TestLoop_ReasoningDeltaEmitted(t *testing.T) {
	p := &mockProvider{
		responses: []mockResponse{
			{events: []provider.StreamEvent{
				{Type: provider.EventReasoningDelta, Content: "Let me think..."},
				{Type: provider.EventContentDelta, Content: "Answer"},
				{Type: provider.EventFinish, Meta: &provider.Metadata{
					InputTokens: 10, OutputTokens: 5, FinishReason: "stop",
				}},
			}},
		},
	}

	reg := newTestRegistry()
	sess := session.New("/tmp")
	loop := NewLoop(p, reg, sess, BuildAgent(), 10)

	events := collectEvents(loop.Run(context.Background(), "think"))

	var reasoning, content string
	var gotDone bool
	for _, e := range events {
		switch e.Type {
		case EventReasoningDelta:
			reasoning += e.Content
		case EventContentDelta:
			content += e.Content
		case EventDone:
			gotDone = true
		}
	}

	assert.Equal(t, "Let me think...", reasoning)
	assert.Equal(t, "Answer", content)
	assert.True(t, gotDone)
}

func TestLoop_ReasoningDelta_NotPersisted(t *testing.T) {
	p := &mockProvider{
		responses: []mockResponse{
			{events: []provider.StreamEvent{
				{Type: provider.EventReasoningDelta, Content: "secret thoughts"},
				{Type: provider.EventContentDelta, Content: "visible answer"},
				{Type: provider.EventFinish, Meta: &provider.Metadata{
					InputTokens: 10, OutputTokens: 5, FinishReason: "stop",
				}},
			}},
		},
	}

	reg := newTestRegistry()
	sess := session.New("/tmp")
	loop := NewLoop(p, reg, sess, BuildAgent(), 10)

	_ = collectEvents(loop.Run(context.Background(), "think"))

	// Session should only have user + assistant with content (no reasoning)
	records := sess.Records()
	assert.Len(t, records, 2)
	assert.Equal(t, "visible answer", records[1].Message.Content)
	// Content should NOT include reasoning text
	assert.NotContains(t, records[1].Message.Content, "secret thoughts")
}

func TestLoop_InterleavedReasoningAndContent(t *testing.T) {
	p := &mockProvider{
		responses: []mockResponse{
			{events: []provider.StreamEvent{
				{Type: provider.EventReasoningDelta, Content: "thinking "},
				{Type: provider.EventReasoningDelta, Content: "more"},
				{Type: provider.EventContentDelta, Content: "response "},
				{Type: provider.EventContentDelta, Content: "text"},
				{Type: provider.EventFinish, Meta: &provider.Metadata{
					InputTokens: 10, OutputTokens: 5, FinishReason: "stop",
				}},
			}},
		},
	}

	reg := newTestRegistry()
	sess := session.New("/tmp")
	loop := NewLoop(p, reg, sess, BuildAgent(), 10)

	events := collectEvents(loop.Run(context.Background(), "multi"))

	var types []EventType
	for _, e := range events {
		types = append(types, e.Type)
	}

	// Should see reasoning deltas before content deltas
	assert.Contains(t, types, EventReasoningDelta)
	assert.Contains(t, types, EventContentDelta)
	assert.Contains(t, types, EventDone)
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
