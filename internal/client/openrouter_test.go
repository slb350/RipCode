package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stephenbrandon/ripcode/internal/provider"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpenRouter_ImplementsProvider(t *testing.T) {
	var _ provider.Provider = &OpenRouter{}
}

func TestOpenRouter_Name(t *testing.T) {
	c := NewOpenRouter("key", "model")
	assert.Equal(t, "openrouter", c.Name())
}

// sseResponse builds a raw SSE response body from lines of data payloads.
func sseResponse(chunks ...string) string {
	var sb strings.Builder
	for _, chunk := range chunks {
		sb.WriteString("data: ")
		sb.WriteString(chunk)
		sb.WriteString("\n\n")
	}
	sb.WriteString("data: [DONE]\n\n")
	return sb.String()
}

// chatChunk builds a JSON chunk for a streaming chat completion.
func chatChunk(content string, finishReason string) string {
	delta := map[string]any{}
	if content != "" {
		delta["content"] = content
	}

	choice := map[string]any{
		"index": 0,
		"delta": delta,
	}
	if finishReason != "" {
		choice["finish_reason"] = finishReason
	}

	resp := map[string]any{
		"id":      "gen-123",
		"model":   "anthropic/claude-sonnet-4",
		"choices": []any{choice},
	}

	b, _ := json.Marshal(resp)
	return string(b)
}

// chatChunkWithUsage builds a final chunk that includes usage data.
func chatChunkWithUsage(finishReason string, inputTokens, outputTokens int) string {
	choice := map[string]any{
		"index":         0,
		"delta":         map[string]any{},
		"finish_reason": finishReason,
	}

	resp := map[string]any{
		"id":      "gen-123",
		"model":   "anthropic/claude-sonnet-4",
		"choices": []any{choice},
		"usage": map[string]any{
			"prompt_tokens":     inputTokens,
			"completion_tokens": outputTokens,
		},
	}

	b, _ := json.Marshal(resp)
	return string(b)
}

// toolCallChunk builds a streaming chunk containing a tool call delta.
func toolCallChunk(index int, id, name, argsFragment string) string {
	tc := map[string]any{
		"index": index,
	}
	if id != "" {
		tc["id"] = id
	}
	if name != "" {
		tc["function"] = map[string]any{
			"name":      name,
			"arguments": argsFragment,
		}
	} else {
		tc["function"] = map[string]any{
			"arguments": argsFragment,
		}
	}

	choice := map[string]any{
		"index": 0,
		"delta": map[string]any{
			"tool_calls": []any{tc},
		},
	}

	resp := map[string]any{
		"id":      "gen-123",
		"model":   "anthropic/claude-sonnet-4",
		"choices": []any{choice},
	}

	b, _ := json.Marshal(resp)
	return string(b)
}

func TestOpenRouter_StreamsContentDeltas(t *testing.T) {
	body := sseResponse(
		chatChunk("Hello", ""),
		chatChunk(" world", ""),
		chatChunkWithUsage("stop", 10, 5),
	)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, body)
	}))
	defer srv.Close()

	c := NewOpenRouter("test-key", "test-model")
	c.baseURL = srv.URL

	ch, err := c.Chat(context.Background(), []provider.Message{
		{Role: "user", Content: "hi"},
	}, nil)
	require.NoError(t, err)

	var events []provider.StreamEvent
	for e := range ch {
		events = append(events, e)
	}

	// Should get content deltas then finish
	require.GreaterOrEqual(t, len(events), 2)

	var content strings.Builder
	var finish *provider.StreamEvent
	for i := range events {
		switch events[i].Type {
		case provider.EventContentDelta:
			content.WriteString(events[i].Content)
		case provider.EventFinish:
			finish = &events[i]
		}
	}

	assert.Equal(t, "Hello world", content.String())
	require.NotNil(t, finish)
	assert.Equal(t, 10, finish.Meta.InputTokens)
	assert.Equal(t, 5, finish.Meta.OutputTokens)
	assert.Equal(t, "stop", finish.Meta.FinishReason)
}

func TestOpenRouter_ParsesToolCalls(t *testing.T) {
	body := sseResponse(
		chatChunk("Let me check.", ""),
		toolCallChunk(0, "call_abc", "bash", `{"comma`),
		toolCallChunk(0, "", "", `nd":"ls"}`),
		chatChunkWithUsage("tool_calls", 15, 20),
	)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, body)
	}))
	defer srv.Close()

	c := NewOpenRouter("test-key", "test-model")
	c.baseURL = srv.URL

	ch, err := c.Chat(context.Background(), []provider.Message{
		{Role: "user", Content: "list files"},
	}, nil)
	require.NoError(t, err)

	var toolEvents []provider.StreamEvent
	for e := range ch {
		if e.Type == provider.EventToolCall {
			toolEvents = append(toolEvents, e)
		}
	}

	require.Len(t, toolEvents, 1)
	tc := toolEvents[0].ToolCall
	assert.Equal(t, "call_abc", tc.ID)
	assert.Equal(t, "bash", tc.Name)
	assert.Equal(t, `{"command":"ls"}`, tc.Args)
}

func TestOpenRouter_HandlesAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, `{"error":{"message":"Invalid API key"}}`)
	}))
	defer srv.Close()

	c := NewOpenRouter("bad-key", "test-model")
	c.baseURL = srv.URL

	_, err := c.Chat(context.Background(), []provider.Message{
		{Role: "user", Content: "hi"},
	}, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "401")
}

func TestOpenRouter_ContextCancellation(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		// Write one chunk then hang — context cancel should handle it
		fmt.Fprint(w, "data: "+chatChunk("start", "")+"\n\n")
		w.(http.Flusher).Flush()
		// Block until request context is done
		<-r.Context().Done()
	}))
	defer srv.Close()

	c := NewOpenRouter("test-key", "test-model")
	c.baseURL = srv.URL

	ctx, cancel := context.WithCancel(context.Background())
	ch, err := c.Chat(ctx, []provider.Message{
		{Role: "user", Content: "hi"},
	}, nil)
	require.NoError(t, err)

	// Read the first event
	e := <-ch
	assert.Equal(t, provider.EventContentDelta, e.Type)

	// Cancel context
	cancel()

	// Channel should close
	for range ch {
		// drain
	}
}

func TestOpenRouter_SendsToolDefinitions(t *testing.T) {
	var receivedBody map[string]any

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		receivedBody = body
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, sseResponse(chatChunkWithUsage("stop", 5, 3)))
	}))
	defer srv.Close()

	c := NewOpenRouter("test-key", "test-model")
	c.baseURL = srv.URL

	tools := []provider.ToolDef{
		{
			Name:        "read",
			Description: "Read a file",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"file_path": map[string]any{
						"type": "string",
					},
				},
				"required": []string{"file_path"},
			},
		},
	}

	ch, err := c.Chat(context.Background(), []provider.Message{
		{Role: "user", Content: "read go.mod"},
	}, tools)
	require.NoError(t, err)
	for range ch {
	}

	// Verify tools were sent in the request body
	require.NotNil(t, receivedBody)
	sentTools, ok := receivedBody["tools"].([]any)
	require.True(t, ok)
	require.Len(t, sentTools, 1)

	tool := sentTools[0].(map[string]any)
	assert.Equal(t, "function", tool["type"])
	fn := tool["function"].(map[string]any)
	assert.Equal(t, "read", fn["name"])
}

func TestOpenRouter_SendsToolResults(t *testing.T) {
	var receivedBody map[string]any

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		receivedBody = body
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, sseResponse(chatChunkWithUsage("stop", 5, 3)))
	}))
	defer srv.Close()

	c := NewOpenRouter("test-key", "test-model")
	c.baseURL = srv.URL

	msgs := []provider.Message{
		{Role: "user", Content: "list files"},
		{
			Role:    "assistant",
			Content: "Let me check.",
			ToolCalls: []provider.ToolCall{
				{ID: "call_1", Name: "bash", Args: `{"command":"ls"}`},
			},
		},
		{Role: "tool", ToolCallID: "call_1", Content: "file1.go\nfile2.go"},
	}

	ch, err := c.Chat(context.Background(), msgs, nil)
	require.NoError(t, err)
	for range ch {
	}

	// Verify message structure includes tool_calls and tool results
	require.NotNil(t, receivedBody)
	sentMsgs := receivedBody["messages"].([]any)
	require.Len(t, sentMsgs, 3)

	// Assistant message should have tool_calls
	assistantMsg := sentMsgs[1].(map[string]any)
	assert.Equal(t, "assistant", assistantMsg["role"])
	toolCalls := assistantMsg["tool_calls"].([]any)
	require.Len(t, toolCalls, 1)

	// Tool result should have tool_call_id
	toolMsg := sentMsgs[2].(map[string]any)
	assert.Equal(t, "tool", toolMsg["role"])
	assert.Equal(t, "call_1", toolMsg["tool_call_id"])
}

func TestOpenRouter_ListModels(t *testing.T) {
	body := `{"data":[{"id":"anthropic/claude-sonnet-4","name":"Claude Sonnet 4"},{"id":"openai/gpt-4o","name":"GPT-4o"}]}`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, body)
	}))
	defer srv.Close()

	c := NewOpenRouter("test-key", "test-model")
	c.modelsURL = srv.URL

	models, err := c.ListModels(context.Background())
	require.NoError(t, err)
	require.Len(t, models, 2)

	assert.Equal(t, "anthropic/claude-sonnet-4", models[0].ID)
	assert.Equal(t, "Claude Sonnet 4", models[0].Name)
	assert.Equal(t, "openai/gpt-4o", models[1].ID)
}

func TestOpenRouter_ListModels_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		fmt.Fprint(w, `{"error":"unavailable"}`)
	}))
	defer srv.Close()

	c := NewOpenRouter("test-key", "test-model")
	c.modelsURL = srv.URL

	_, err := c.ListModels(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "503")
}
