package client

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/stephenbrandon/ripcode/internal/provider"
)

const defaultBaseURL = "https://openrouter.ai/api/v1/chat/completions"

// OpenRouter implements provider.Provider using the OpenRouter API.
type OpenRouter struct {
	apiKey     string
	model      string
	baseURL    string
	httpClient *http.Client
}

// NewOpenRouter creates a new OpenRouter client.
func NewOpenRouter(apiKey, model string) *OpenRouter {
	return &OpenRouter{
		apiKey:     apiKey,
		model:      model,
		baseURL:    defaultBaseURL,
		httpClient: http.DefaultClient,
	}
}

func (c *OpenRouter) Name() string { return "openrouter" }

// Chat sends messages to the OpenRouter API and streams response events.
func (c *OpenRouter) Chat(ctx context.Context, msgs []provider.Message, tools []provider.ToolDef) (<-chan provider.StreamEvent, error) {
	body, err := c.buildRequest(msgs, tools)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("openrouter API error: %d", resp.StatusCode)
	}

	ch := make(chan provider.StreamEvent, 64)
	go c.streamResponse(ctx, resp, ch)
	return ch, nil
}

// buildRequest constructs the JSON request body.
func (c *OpenRouter) buildRequest(msgs []provider.Message, tools []provider.ToolDef) ([]byte, error) {
	apiMsgs := make([]apiMessage, 0, len(msgs))
	for _, m := range msgs {
		am := apiMessage{
			Role:    m.Role,
			Content: m.Content,
		}
		if m.ToolCallID != "" {
			am.ToolCallID = m.ToolCallID
		}
		if len(m.ToolCalls) > 0 {
			am.ToolCalls = make([]apiToolCall, len(m.ToolCalls))
			for i, tc := range m.ToolCalls {
				am.ToolCalls[i] = apiToolCall{
					ID:   tc.ID,
					Type: "function",
					Function: apiFunction{
						Name:      tc.Name,
						Arguments: tc.Args,
					},
				}
			}
		}
		apiMsgs = append(apiMsgs, am)
	}

	req := apiRequest{
		Model:    c.model,
		Messages: apiMsgs,
		Stream:   true,
	}

	if len(tools) > 0 {
		apiTools := make([]apiTool, len(tools))
		for i, t := range tools {
			apiTools[i] = apiTool{
				Type: "function",
				Function: apiToolDef{
					Name:        t.Name,
					Description: t.Description,
					Parameters:  t.Parameters,
				},
			}
		}
		req.Tools = apiTools
	}

	return json.Marshal(req)
}

// streamResponse reads SSE lines and emits events on the channel.
func (c *OpenRouter) streamResponse(ctx context.Context, resp *http.Response, ch chan<- provider.StreamEvent) {
	defer close(ch)
	defer resp.Body.Close()

	// Track accumulated tool calls across chunks
	type toolCallAcc struct {
		id   string
		name string
		args strings.Builder
	}
	toolCalls := map[int]*toolCallAcc{}

	var meta provider.Metadata
	meta.Model = c.model

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return
		default:
		}

		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			break
		}

		var chunk apiChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			ch <- provider.StreamEvent{
				Type:  provider.EventError,
				Error: fmt.Errorf("parse SSE chunk: %w", err),
			}
			return
		}

		if chunk.Usage != nil {
			meta.InputTokens = chunk.Usage.PromptTokens
			meta.OutputTokens = chunk.Usage.CompletionTokens
		}

		if len(chunk.Choices) == 0 {
			continue
		}

		choice := chunk.Choices[0]

		// Content delta
		if choice.Delta.Content != "" {
			ch <- provider.StreamEvent{
				Type:    provider.EventContentDelta,
				Content: choice.Delta.Content,
			}
		}

		// Tool call deltas — accumulate fragments
		for _, tcDelta := range choice.Delta.ToolCalls {
			acc, exists := toolCalls[tcDelta.Index]
			if !exists {
				acc = &toolCallAcc{}
				toolCalls[tcDelta.Index] = acc
			}
			if tcDelta.ID != "" {
				acc.id = tcDelta.ID
			}
			if tcDelta.Function.Name != "" {
				acc.name = tcDelta.Function.Name
			}
			acc.args.WriteString(tcDelta.Function.Arguments)
		}

		// Finish reason
		if choice.FinishReason != "" {
			meta.FinishReason = choice.FinishReason
		}
	}

	// Emit accumulated tool calls
	for i := 0; i < len(toolCalls); i++ {
		acc := toolCalls[i]
		ch <- provider.StreamEvent{
			Type: provider.EventToolCall,
			ToolCall: &provider.ToolCall{
				ID:   acc.id,
				Name: acc.name,
				Args: acc.args.String(),
			},
		}
	}

	// Emit finish
	ch <- provider.StreamEvent{
		Type: provider.EventFinish,
		Meta: &meta,
	}
}

// --- API types ---

type apiRequest struct {
	Model    string       `json:"model"`
	Messages []apiMessage `json:"messages"`
	Stream   bool         `json:"stream"`
	Tools    []apiTool    `json:"tools,omitempty"`
}

type apiMessage struct {
	Role       string        `json:"role"`
	Content    string        `json:"content"`
	ToolCalls  []apiToolCall `json:"tool_calls,omitempty"`
	ToolCallID string        `json:"tool_call_id,omitempty"`
}

type apiToolCall struct {
	ID       string      `json:"id"`
	Type     string      `json:"type"`
	Function apiFunction `json:"function"`
}

type apiFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type apiTool struct {
	Type     string     `json:"type"`
	Function apiToolDef `json:"function"`
}

type apiToolDef struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"`
}

type apiChunk struct {
	ID      string      `json:"id"`
	Model   string      `json:"model"`
	Choices []apiChoice `json:"choices"`
	Usage   *apiUsage   `json:"usage,omitempty"`
}

type apiChoice struct {
	Index        int      `json:"index"`
	Delta        apiDelta `json:"delta"`
	FinishReason string   `json:"finish_reason,omitempty"`
}

type apiDelta struct {
	Content   string             `json:"content,omitempty"`
	ToolCalls []apiToolCallDelta `json:"tool_calls,omitempty"`
}

type apiToolCallDelta struct {
	Index    int            `json:"index"`
	ID       string         `json:"id,omitempty"`
	Function apiPartialFunc `json:"function,omitempty"`
}

type apiPartialFunc struct {
	Name      string `json:"name,omitempty"`
	Arguments string `json:"arguments,omitempty"`
}

type apiUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
}
