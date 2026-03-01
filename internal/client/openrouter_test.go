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
	assert.Contains(t, err.Error(), "Invalid API key")
}

func TestOpenRouter_ChatErrorIncludesBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, `{"error":{"message":"model not found","code":400}}`)
	}))
	defer srv.Close()

	c := NewOpenRouter("test-key", "bad-model")
	c.baseURL = srv.URL

	_, err := c.Chat(context.Background(), []provider.Message{
		{Role: "user", Content: "hi"},
	}, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "400")
	assert.Contains(t, err.Error(), "model not found")
}

func TestOpenRouter_ChatErrorEmptyBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := NewOpenRouter("test-key", "test-model")
	c.baseURL = srv.URL

	_, err := c.Chat(context.Background(), []provider.Message{
		{Role: "user", Content: "hi"},
	}, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "500")
}

func TestOpenRouter_SetModelRejectsEmpty(t *testing.T) {
	c := NewOpenRouter("test-key", "original-model")
	c.SetModel("")

	c.mu.RLock()
	model := c.model
	c.mu.RUnlock()
	assert.Equal(t, "original-model", model, "empty model should be rejected")
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

func TestListModels_ParsesPricing(t *testing.T) {
	body := `{"data":[{"id":"anthropic/claude-4","name":"Claude 4","pricing":{"prompt":"0.003","completion":"0.015"}}]}`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, body)
	}))
	defer srv.Close()

	c := NewOpenRouter("test-key", "test-model")
	c.modelsURL = srv.URL

	models, err := c.ListModels(context.Background())
	require.NoError(t, err)
	require.Len(t, models, 1)
	require.NotNil(t, models[0].Pricing)
	assert.InDelta(t, 0.003, models[0].Pricing.PromptPerMillion, 0.0001)
	assert.InDelta(t, 0.015, models[0].Pricing.CompletionPerMillion, 0.0001)
}

func TestListModels_ParsesContextLength(t *testing.T) {
	body := `{"data":[{"id":"anthropic/claude-4","name":"Claude 4","context_length":200000}]}`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, body)
	}))
	defer srv.Close()

	c := NewOpenRouter("test-key", "test-model")
	c.modelsURL = srv.URL

	models, err := c.ListModels(context.Background())
	require.NoError(t, err)
	require.Len(t, models, 1)
	assert.Equal(t, 200000, models[0].ContextLength)
}

func TestListModels_ParsesDescription(t *testing.T) {
	body := `{"data":[{"id":"anthropic/claude-4","name":"Claude 4","description":"A powerful model"}]}`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, body)
	}))
	defer srv.Close()

	c := NewOpenRouter("test-key", "test-model")
	c.modelsURL = srv.URL

	models, err := c.ListModels(context.Background())
	require.NoError(t, err)
	require.Len(t, models, 1)
	assert.Equal(t, "A powerful model", models[0].Description)
}

func TestListModels_NullPricing_NilModelPricing(t *testing.T) {
	body := `{"data":[{"id":"free/model","name":"Free Model"}]}`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, body)
	}))
	defer srv.Close()

	c := NewOpenRouter("test-key", "test-model")
	c.modelsURL = srv.URL

	models, err := c.ListModels(context.Background())
	require.NoError(t, err)
	require.Len(t, models, 1)
	assert.Nil(t, models[0].Pricing)
}

func TestListModels_MalformedPricing_Fallback(t *testing.T) {
	body := `{"data":[{"id":"test/model","name":"Test","pricing":{"prompt":"free","completion":"free"}}]}`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, body)
	}))
	defer srv.Close()

	c := NewOpenRouter("test-key", "test-model")
	c.modelsURL = srv.URL

	models, err := c.ListModels(context.Background())
	require.NoError(t, err)
	require.Len(t, models, 1)
	assert.Nil(t, models[0].Pricing, "malformed pricing should result in nil")
}

func TestStreamResponse_ScannerError_EmitsErrorEvent(t *testing.T) {
	// Create a server that sends a partial response then closes abruptly
	// by writing a line that exceeds the default scanner buffer (64KB)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		// Send one valid content chunk
		fmt.Fprint(w, "data: "+chatChunk("partial", "")+"\n\n")
		w.(http.Flusher).Flush()
		// Write a single line exceeding bufio.Scanner's max token size
		// to force a scanner error
		huge := "data: " + strings.Repeat("x", 1024*1024) // 1MB line
		fmt.Fprint(w, huge)
		w.(http.Flusher).Flush()
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

	// Should get the content delta, then an error event
	hasContent := false
	hasError := false
	for _, e := range events {
		if e.Type == provider.EventContentDelta {
			hasContent = true
		}
		if e.Type == provider.EventError {
			hasError = true
			assert.Contains(t, e.Error.Error(), "stream read error")
		}
	}
	assert.True(t, hasContent, "should have received partial content")
	assert.True(t, hasError, "should have received error event from scanner failure")
	// Should NOT have a finish event since scanner errored
	for _, e := range events {
		assert.NotEqual(t, provider.EventFinish, e.Type, "should not emit finish after scanner error")
	}
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

func TestOpenRouter_ConcurrentSetModelAndChat(t *testing.T) {
	body := sseResponse(
		chatChunk("ok", ""),
		chatChunkWithUsage("stop", 1, 1),
	)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, body)
	}))
	defer srv.Close()

	c := NewOpenRouter("test-key", "test-model")
	c.baseURL = srv.URL

	const goroutines = 10
	done := make(chan struct{})

	// 10 goroutines calling SetModel concurrently
	for i := 0; i < goroutines; i++ {
		go func(n int) {
			defer func() { done <- struct{}{} }()
			c.SetModel(fmt.Sprintf("model-%d", n))
		}(i)
	}

	// 10 goroutines calling Chat concurrently
	for i := 0; i < goroutines; i++ {
		go func() {
			defer func() { done <- struct{}{} }()
			ch, err := c.Chat(context.Background(), []provider.Message{
				{Role: "user", Content: "hi"},
			}, nil)
			if err != nil {
				return
			}
			for range ch {
			}
		}()
	}

	// Wait for all goroutines
	for i := 0; i < goroutines*2; i++ {
		<-done
	}
}

func TestOpenRouter_StreamMetadataUsesRequestModel(t *testing.T) {
	firstChunkSent := make(chan struct{})
	allowFinish := make(chan struct{})

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")

		// Send one chunk so the stream starts, then wait while model changes.
		fmt.Fprint(w, "data: "+chatChunk("hello", "")+"\n\n")
		w.(http.Flusher).Flush()
		close(firstChunkSent)

		<-allowFinish
		fmt.Fprint(w, "data: "+chatChunkWithUsage("stop", 2, 1)+"\n\n")
		fmt.Fprint(w, "data: [DONE]\n\n")
		w.(http.Flusher).Flush()
	}))
	defer srv.Close()

	c := NewOpenRouter("test-key", "model-a")
	c.baseURL = srv.URL

	ch, err := c.Chat(context.Background(), []provider.Message{
		{Role: provider.RoleUser, Content: "hi"},
	}, nil)
	require.NoError(t, err)

	<-firstChunkSent
	c.SetModel("model-b")
	close(allowFinish)

	var finish *provider.StreamEvent
	for e := range ch {
		if e.Type == provider.EventFinish {
			evt := e
			finish = &evt
		}
	}

	require.NotNil(t, finish)
	assert.Equal(t, "model-a", finish.Meta.Model, "finish metadata should reflect the request model")
}

func TestOpenRouter_ImplementsReasoningEffortSetter(t *testing.T) {
	var _ provider.ReasoningEffortSetter = &OpenRouter{}
}

func TestOpenRouter_SetReasoningEffort(t *testing.T) {
	c := NewOpenRouter("key", "model")
	c.SetReasoningEffort("high")
	c.mu.RLock()
	assert.Equal(t, "high", c.reasoningEffort)
	c.mu.RUnlock()
}

func TestOpenRouter_ReasoningEffortInRequest(t *testing.T) {
	c := NewOpenRouter("key", "model")
	c.SetReasoningEffort("low")

	msgs := []provider.Message{{Role: provider.RoleUser, Content: "hello"}}
	body, err := c.buildRequest(msgs, nil)
	require.NoError(t, err)

	var req map[string]any
	require.NoError(t, json.Unmarshal(body, &req))
	reasoning, ok := req["reasoning"].(map[string]any)
	require.True(t, ok, "request should have reasoning field")
	assert.Equal(t, "low", reasoning["effort"])
}

func TestOpenRouter_ReasoningEffortEmpty_OmitsField(t *testing.T) {
	c := NewOpenRouter("key", "model")
	// No SetReasoningEffort call — effort is empty

	msgs := []provider.Message{{Role: provider.RoleUser, Content: "hello"}}
	body, err := c.buildRequest(msgs, nil)
	require.NoError(t, err)

	var req map[string]any
	require.NoError(t, json.Unmarshal(body, &req))
	_, ok := req["reasoning"]
	assert.False(t, ok, "request should not have reasoning field when effort is empty")
}
