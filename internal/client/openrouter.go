package client

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/stephenbrandon/ripcode/internal/provider"
	"github.com/stephenbrandon/ripcode/internal/store"
)

const defaultBaseURL = "https://openrouter.ai/api/v1/chat/completions"
const defaultModelsURL = "https://openrouter.ai/api/v1/models"

// OpenRouter implements provider.Provider using the OpenRouter API.
type OpenRouter struct {
	mu              sync.RWMutex
	apiKey          string
	model           string
	reasoningEffort string
	baseURL         string
	modelsURL       string
	httpClient      *http.Client
}

// NewOpenRouter creates a new OpenRouter client.
func NewOpenRouter(apiKey, model string) *OpenRouter {
	return &OpenRouter{
		apiKey:     apiKey,
		model:      model,
		baseURL:    defaultBaseURL,
		modelsURL:  defaultModelsURL,
		httpClient: http.DefaultClient,
	}
}

func (c *OpenRouter) Name() string { return "openrouter" }

// SetModel updates the model used for chat completions.
// Safe to call concurrently with Chat/buildRequest.
func (c *OpenRouter) SetModel(model string) {
	if model == "" {
		return
	}
	c.mu.Lock()
	c.model = model
	c.mu.Unlock()
}

// SetReasoningEffort sets the reasoning effort level for thinking models.
// Effort values: "low", "medium", "high" or "" to disable.
func (c *OpenRouter) SetReasoningEffort(effort string) {
	c.mu.Lock()
	c.reasoningEffort = effort
	c.mu.Unlock()
}

// ListModels fetches the available models from OpenRouter.
func (c *OpenRouter) ListModels(ctx context.Context) ([]provider.ModelInfo, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.modelsURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, apiErrorFromResponse(resp)
	}

	var parsed apiModelsResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, fmt.Errorf("decode models response: %w", err)
	}

	seen := make(map[string]bool, len(parsed.Data))
	models := make([]provider.ModelInfo, 0, len(parsed.Data))
	for _, m := range parsed.Data {
		if m.ID == "" || seen[m.ID] {
			continue
		}
		seen[m.ID] = true
		name := strings.TrimSpace(m.Name)
		if name == "" {
			name = m.ID
		}
		info := provider.ModelInfo{
			ID:            m.ID,
			Name:          name,
			Description:   m.Description,
			ContextLength: m.ContextLength,
		}
		if m.Pricing != nil {
			prompt, errP := strconv.ParseFloat(m.Pricing.Prompt, 64)
			completion, errC := strconv.ParseFloat(m.Pricing.Completion, 64)
			if errP != nil {
				store.LogError("pricing prompt "+m.ID, errP)
			}
			if errC != nil {
				store.LogError("pricing completion "+m.ID, errC)
			}
			if errP == nil && errC == nil {
				info.Pricing = &provider.ModelPricing{
					PromptPerMillion:     prompt,
					CompletionPerMillion: completion,
				}
			} else {
				info.PricingUnknown = true
			}
		}
		models = append(models, info)
	}

	sort.Slice(models, func(i, j int) bool {
		return models[i].ID < models[j].ID
	})
	return models, nil
}

// Chat sends messages to the OpenRouter API and streams response events.
func (c *OpenRouter) Chat(ctx context.Context, msgs []provider.Message, tools []provider.ToolDef) (<-chan provider.StreamEvent, error) {
	model, effort := c.snapshotSettings()
	body, err := c.buildRequestWithSettings(msgs, tools, model, effort)
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
		err := apiErrorFromResponse(resp)
		resp.Body.Close()
		return nil, err
	}

	ch := make(chan provider.StreamEvent, 64)
	go c.streamResponse(ctx, resp, ch, model)
	return ch, nil
}

// buildRequest constructs the JSON request body using the current model settings.
// Used by tests; production code calls snapshotSettings + buildRequestWithSettings directly.
func (c *OpenRouter) buildRequest(msgs []provider.Message, tools []provider.ToolDef) ([]byte, error) {
	model, effort := c.snapshotSettings()
	return c.buildRequestWithSettings(msgs, tools, model, effort)
}

func (c *OpenRouter) snapshotSettings() (model, effort string) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.model, c.reasoningEffort
}

func (c *OpenRouter) buildRequestWithSettings(msgs []provider.Message, tools []provider.ToolDef, model, effort string) ([]byte, error) {
	apiMsgs := make([]apiMessage, 0, len(msgs))
	for _, m := range msgs {
		am := apiMessage{
			Role:    string(m.Role),
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
		Model:    model,
		Messages: apiMsgs,
		Stream:   true,
	}

	if effort != "" {
		req.Reasoning = &apiReasoning{Effort: effort}
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
func (c *OpenRouter) streamResponse(ctx context.Context, resp *http.Response, ch chan<- provider.StreamEvent, requestModel string) {
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
	meta.Model = requestModel

	// send emits an event on ch, returning false if the context was cancelled.
	send := func(evt provider.StreamEvent) bool {
		select {
		case ch <- evt:
			return true
		case <-ctx.Done():
			return false
		}
	}

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024) // 1MB limit for large tool call args
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
			send(provider.NewErrorEvent(fmt.Errorf("parse SSE chunk: %w", err)))
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
			if !send(provider.NewContentDelta(choice.Delta.Content)) {
				return
			}
		}

		// Reasoning deltas
		for _, rd := range choice.Delta.ReasoningDetails {
			var text string
			switch rd.Type {
			case reasoningTypeText:
				text = rd.Text
			case reasoningTypeSummary:
				text = rd.Summary
			case reasoningTypeEncrypted:
				text = "[REDACTED]"
			default:
				store.LogErrorf("unknown reasoning detail type %q from model %s", rd.Type, requestModel)
				continue
			}
			if text != "" {
				if !send(provider.NewReasoningDelta(text)) {
					return
				}
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

	// Check for scanner errors (e.g. network disconnect mid-stream).
	// Discard partial tool calls — they shouldn't be executed from a truncated stream.
	if err := scanner.Err(); err != nil {
		send(provider.NewErrorEvent(fmt.Errorf("stream read error: %w", err)))
		return
	}

	// Emit accumulated tool calls. SSE delivers tool call fields incrementally
	// (index, name chunk, args chunk across multiple events). We accumulate
	// fragments by index key during streaming and emit complete calls here.
	// Incomplete tool calls (missing ID or Name) are discarded — executing
	// a partial call could cause errors or data corruption downstream.
	for i := 0; i < len(toolCalls); i++ {
		acc := toolCalls[i]
		if acc.id == "" || acc.name == "" {
			send(provider.NewErrorEvent(fmt.Errorf("incomplete tool call from provider: id=%q name=%q", acc.id, acc.name)))
			return
		}
		if !send(provider.NewToolCallEvent(&provider.ToolCall{
			ID:   acc.id,
			Name: acc.name,
			Args: acc.args.String(),
		})) {
			return
		}
	}

	send(provider.NewFinishEvent(&meta))
}

// apiErrorFromResponse reads up to 2KB from the response body and returns a
// formatted error including the HTTP status code and body excerpt.
// Does not close the body — caller is responsible for cleanup.
func apiErrorFromResponse(resp *http.Response) error {
	body, err := io.ReadAll(io.LimitReader(resp.Body, 2048))
	if err != nil {
		return fmt.Errorf("openrouter API error: %d (body unreadable: %v)", resp.StatusCode, err)
	}
	msg := strings.TrimSpace(string(body))
	if msg == "" {
		return fmt.Errorf("openrouter API error: %d", resp.StatusCode)
	}
	return fmt.Errorf("openrouter API error: %d: %s", resp.StatusCode, msg)
}

// --- API types ---

type apiRequest struct {
	Model     string        `json:"model"`
	Messages  []apiMessage  `json:"messages"`
	Stream    bool          `json:"stream"`
	Tools     []apiTool     `json:"tools,omitempty"`
	Reasoning *apiReasoning `json:"reasoning,omitempty"`
}

type apiReasoning struct {
	Effort string `json:"effort"`
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
	Content          string               `json:"content,omitempty"`
	ToolCalls        []apiToolCallDelta   `json:"tool_calls,omitempty"`
	ReasoningDetails []apiReasoningDetail `json:"reasoning_details,omitempty"`
}

// Reasoning detail type constants from the OpenRouter streaming API.
const (
	reasoningTypeText      = "reasoning.text"
	reasoningTypeSummary   = "reasoning.summary"
	reasoningTypeEncrypted = "reasoning.encrypted"
)

type apiReasoningDetail struct {
	Type    string `json:"type"`
	Text    string `json:"text,omitempty"`    // for reasoningTypeText
	Summary string `json:"summary,omitempty"` // for reasoningTypeSummary
	Data    string `json:"data,omitempty"`    // for reasoningTypeEncrypted
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

type apiModelsResponse struct {
	Data []apiModel `json:"data"`
}

type apiModel struct {
	ID            string      `json:"id"`
	Name          string      `json:"name"`
	Description   string      `json:"description"`
	ContextLength int         `json:"context_length"`
	Pricing       *apiPricing `json:"pricing"`
}

type apiPricing struct {
	Prompt     string `json:"prompt"`
	Completion string `json:"completion"`
}
