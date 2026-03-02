package tests

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stephenbrandon/ripcode/internal/agent"
	"github.com/stephenbrandon/ripcode/internal/provider"
	"github.com/stephenbrandon/ripcode/internal/session"
	"github.com/stephenbrandon/ripcode/internal/tool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockProvider returns pre-configured responses in sequence.
type mockProvider struct {
	responses [][]provider.StreamEvent
	callIdx   int
}

func (m *mockProvider) Chat(_ context.Context, _ []provider.Message, _ []provider.ToolDef) (<-chan provider.StreamEvent, error) {
	if m.callIdx >= len(m.responses) {
		return nil, fmt.Errorf("no more responses")
	}
	events := m.responses[m.callIdx]
	m.callIdx++

	ch := make(chan provider.StreamEvent, len(events))
	for _, e := range events {
		ch <- e
	}
	close(ch)
	return ch, nil
}

func (m *mockProvider) Name() string { return "mock" }

// newFullRegistry creates a registry with all tools.
func newFullRegistry() *tool.Registry {
	reg := tool.NewRegistry()
	reg.Register(tool.NewBashTool())
	reg.Register(tool.NewReadTool())
	reg.Register(tool.NewWriteTool())
	reg.Register(tool.NewEditTool())
	reg.Register(tool.NewGlobTool())
	reg.Register(tool.NewGrepTool())
	reg.Register(tool.NewLsTool())
	reg.Register(tool.NewTodoTool())
	return reg
}

func collectEvents(ch <-chan agent.Event) []agent.Event {
	var events []agent.Event
	for e := range ch {
		events = append(events, e)
	}
	return events
}

// TestIntegration_SingleTurnConversation verifies a simple user→assistant exchange.
func TestIntegration_SingleTurnConversation(t *testing.T) {
	p := &mockProvider{
		responses: [][]provider.StreamEvent{
			{
				{Type: provider.EventContentDelta, Content: "Hello! "},
				{Type: provider.EventContentDelta, Content: "How can I help?"},
				{Type: provider.EventFinish, Meta: &provider.Metadata{
					InputTokens: 15, OutputTokens: 8, FinishReason: "stop",
				}},
			},
		},
	}

	reg := newFullRegistry()
	sess := session.New(t.TempDir())
	loop := agent.NewLoop(p, reg, sess, agent.BuildAgent(), 10)

	events := collectEvents(loop.Run(context.Background(), "Hello"))

	// Verify content streamed
	var content strings.Builder
	for _, e := range events {
		if e.Type == agent.EventContentDelta {
			content.WriteString(e.Content)
		}
	}
	assert.Equal(t, "Hello! How can I help?", content.String())

	// Session should have 2 messages
	assert.Len(t, sess.Records(), 2)
	assert.Equal(t, 15, sess.TokenCount().Input)
	assert.Equal(t, 8, sess.TokenCount().Output)
}

// TestIntegration_ToolExecution verifies the agent can execute tools and continue.
func TestIntegration_ToolExecution(t *testing.T) {
	dir := t.TempDir()
	testFile := filepath.Join(dir, "hello.txt")
	require.NoError(t, os.WriteFile(testFile, []byte("Hello from file!\n"), 0644))

	p := &mockProvider{
		responses: [][]provider.StreamEvent{
			// Turn 1: agent decides to read a file
			{
				{Type: provider.EventContentDelta, Content: "Let me read that file."},
				{Type: provider.EventToolCall, ToolCall: &provider.ToolCall{
					ID:   "call_1",
					Name: "read",
					Args: fmt.Sprintf(`{"file_path":"%s"}`, testFile),
				}},
				{Type: provider.EventFinish, Meta: &provider.Metadata{
					InputTokens: 20, OutputTokens: 15, FinishReason: "tool_calls",
				}},
			},
			// Turn 2: agent responds with file contents
			{
				{Type: provider.EventContentDelta, Content: "The file says: Hello from file!"},
				{Type: provider.EventFinish, Meta: &provider.Metadata{
					InputTokens: 30, OutputTokens: 10, FinishReason: "stop",
				}},
			},
		},
	}

	reg := newFullRegistry()
	sess := session.New(dir)
	loop := agent.NewLoop(p, reg, sess, agent.BuildAgent(), 10)

	events := collectEvents(loop.Run(context.Background(), "Read hello.txt"))

	// Should see tool start and end
	var toolNames []string
	for _, e := range events {
		if e.Type == agent.EventToolStart {
			toolNames = append(toolNames, e.Tool.Name)
		}
		if e.Type == agent.EventToolEnd {
			assert.Contains(t, e.Tool.Output, "Hello from file!")
		}
	}
	assert.Contains(t, toolNames, "read")

	// Session should have: user, assistant+tool_call, tool_result, assistant
	assert.Len(t, sess.Records(), 4)
	assert.Equal(t, 50, sess.TokenCount().Input)
	assert.Equal(t, 25, sess.TokenCount().Output)
}

// TestIntegration_BashToolSafety verifies blocked commands are rejected.
func TestIntegration_BashToolSafety(t *testing.T) {
	p := &mockProvider{
		responses: [][]provider.StreamEvent{
			// Agent tries a dangerous command
			{
				{Type: provider.EventToolCall, ToolCall: &provider.ToolCall{
					ID:   "call_1",
					Name: "bash",
					Args: `{"command":"rm -rf /"}`,
				}},
				{Type: provider.EventFinish, Meta: &provider.Metadata{
					FinishReason: "tool_calls",
				}},
			},
			// Agent acknowledges the error
			{
				{Type: provider.EventContentDelta, Content: "That command was blocked."},
				{Type: provider.EventFinish, Meta: &provider.Metadata{FinishReason: "stop"}},
			},
		},
	}

	reg := newFullRegistry()
	sess := session.New(t.TempDir())
	loop := agent.NewLoop(p, reg, sess, agent.BuildAgent(), 10)

	events := collectEvents(loop.Run(context.Background(), "delete everything"))

	// Tool should fail with blocked error
	for _, e := range events {
		if e.Type == agent.EventToolEnd {
			assert.Contains(t, e.Tool.Error, "blocked")
		}
	}
}

// TestIntegration_PlanMode verifies plan mode restricts tools.
func TestIntegration_PlanMode(t *testing.T) {
	planAgent := agent.PlanAgent()
	reg := newFullRegistry()

	defs := planAgent.FilterRegistry(reg)
	names := make(map[string]bool)
	for _, d := range defs {
		names[d.Name] = true
	}

	// Plan mode should have read-only tools
	assert.True(t, names["read"])
	assert.True(t, names["glob"])
	assert.True(t, names["grep"])
	assert.True(t, names["ls"])
	assert.True(t, names["todo"])

	// Should NOT have write tools
	assert.False(t, names["bash"])
	assert.False(t, names["write"])
	assert.False(t, names["edit"])
}

// TestIntegration_MultiToolRound verifies multi-tool execution in one turn.
func TestIntegration_MultiToolRound(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "a.go"), []byte("package a\n"), 0644))

	p := &mockProvider{
		responses: [][]provider.StreamEvent{
			// Agent uses two tools
			{
				{Type: provider.EventToolCall, ToolCall: &provider.ToolCall{
					ID: "call_1", Name: "ls", Args: fmt.Sprintf(`{"path":"%s"}`, dir),
				}},
				{Type: provider.EventToolCall, ToolCall: &provider.ToolCall{
					ID: "call_2", Name: "glob", Args: fmt.Sprintf(`{"pattern":"*.go","path":"%s"}`, dir),
				}},
				{Type: provider.EventFinish, Meta: &provider.Metadata{FinishReason: "tool_calls"}},
			},
			// Final response
			{
				{Type: provider.EventContentDelta, Content: "Found one Go file."},
				{Type: provider.EventFinish, Meta: &provider.Metadata{FinishReason: "stop"}},
			},
		},
	}

	reg := newFullRegistry()
	sess := session.New(dir)
	loop := agent.NewLoop(p, reg, sess, agent.BuildAgent(), 10)

	events := collectEvents(loop.Run(context.Background(), "list Go files"))

	toolStartCount := 0
	toolEndCount := 0
	for _, e := range events {
		if e.Type == agent.EventToolStart {
			toolStartCount++
		}
		if e.Type == agent.EventToolEnd {
			toolEndCount++
		}
	}
	assert.Equal(t, 2, toolStartCount)
	assert.Equal(t, 2, toolEndCount)
}
