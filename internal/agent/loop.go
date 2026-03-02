package agent

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/stephenbrandon/ripcode/internal/provider"
	"github.com/stephenbrandon/ripcode/internal/session"
	"github.com/stephenbrandon/ripcode/internal/store"
	"github.com/stephenbrandon/ripcode/internal/tool"
)

// EventType identifies the kind of agent loop event.
// Ordinals differ from provider.EventType — these are separate iota sequences.
type EventType int

const (
	EventContentDelta EventType = iota
	EventReasoningDelta
	EventToolStart
	EventToolEnd
	EventDone
	EventError
)

// ToolEvent carries tool execution information.
type ToolEvent struct {
	ID     string
	Name   string
	Args   string
	Output string
	Error  string
	Diff   *tool.DiffInfo // optional, for write/edit tool results
}

// Event is a single event emitted by the agentic loop.
type Event struct {
	Type    EventType
	Content string     // for EventContentDelta
	Tool    *ToolEvent // for EventToolStart, EventToolEnd
	Meta    *provider.Metadata
	Error   error // for EventError
}

// Valid checks that the Event has a recognized type and required fields.
func (e Event) Valid() error {
	switch e.Type {
	case EventContentDelta:
		// Content may be empty (whitespace-only deltas are valid in SSE streams)
	case EventReasoningDelta:
		// Content may be empty (whitespace-only deltas are valid in SSE streams)
	case EventToolStart:
		if e.Tool == nil {
			return fmt.Errorf("tool start event missing Tool")
		}
	case EventToolEnd:
		if e.Tool == nil {
			return fmt.Errorf("tool end event missing Tool")
		}
	case EventDone:
		// No required fields
	case EventError:
		if e.Error == nil {
			return fmt.Errorf("error event missing Error")
		}
	default:
		return fmt.Errorf("unknown event type: %d", e.Type)
	}
	return nil
}

// NewContentEvent creates a content delta event.
func NewContentEvent(content string) Event {
	return Event{Type: EventContentDelta, Content: content}
}

// NewReasoningEvent creates a reasoning delta event.
func NewReasoningEvent(content string) Event {
	return Event{Type: EventReasoningDelta, Content: content}
}

// NewToolStartEvent creates a tool start event.
func NewToolStartEvent(t *ToolEvent) Event {
	return Event{Type: EventToolStart, Tool: t}
}

// NewToolEndEvent creates a tool end event.
func NewToolEndEvent(t *ToolEvent) Event {
	return Event{Type: EventToolEnd, Tool: t}
}

// NewDoneEvent creates a done event with optional metadata.
func NewDoneEvent(meta *provider.Metadata) Event {
	return Event{Type: EventDone, Meta: meta}
}

// NewErrorEvent creates an error event.
func NewErrorEvent(err error) Event {
	return Event{Type: EventError, Error: err}
}

// Loop orchestrates the agent's prompt-stream-tool cycle.
type Loop struct {
	provider provider.Provider
	registry *tool.Registry
	session  *session.Session
	agent    Agent
	maxSteps int
}

// NewLoop creates a new agentic loop.
func NewLoop(p provider.Provider, r *tool.Registry, s *session.Session, a Agent, maxSteps int) *Loop {
	return &Loop{
		provider: p,
		registry: r,
		session:  s,
		agent:    a,
		maxSteps: maxSteps,
	}
}

// Run processes user input through the agentic loop, returning events on a channel.
func (l *Loop) Run(ctx context.Context, input string) <-chan Event {
	ch := make(chan Event, 64)

	go func() {
		defer close(ch)
		l.run(ctx, input, ch)
	}()

	return ch
}

func (l *Loop) run(ctx context.Context, input string, ch chan<- Event) {
	l.session.AddUser(input)

	for step := 0; step < l.maxSteps; step++ {
		select {
		case <-ctx.Done():
			ch <- NewErrorEvent(ctx.Err())
			return
		default:
		}

		toolDefs := l.agent.FilterRegistry(l.registry)
		streamCh, err := l.provider.Chat(ctx, l.session.History(), toolDefs)
		if err != nil {
			ch <- NewErrorEvent(fmt.Errorf("provider error: %w", err))
			return
		}

		streamStart := time.Now()
		content, toolCalls, meta := l.consumeStream(ctx, streamCh, ch)

		// If cancelled mid-stream, discard partial tool calls to avoid
		// executing incomplete instructions. Still record any content.
		if ctx.Err() != nil {
			toolCalls = nil
		}

		var am *session.AssistantMeta
		if meta != nil {
			am = &session.AssistantMeta{
				Model:        meta.Model,
				InputTokens:  meta.InputTokens,
				OutputTokens: meta.OutputTokens,
				FinishReason: meta.FinishReason,
				Duration:     time.Since(streamStart),
			}
		}
		l.session.AddAssistant(content, toolCalls, am)
		if meta != nil {
			l.session.AddTokens(meta.InputTokens, meta.OutputTokens)
		}

		if len(toolCalls) == 0 {
			ch <- NewDoneEvent(meta)
			return
		}

		// Execute tool calls
		for _, tc := range toolCalls {
			l.executeTool(ctx, tc, ch)
		}
	}

	ch <- NewErrorEvent(fmt.Errorf("max steps reached (%d)", l.maxSteps))
}

// consumeStream reads all events from the provider stream, forwarding content
// deltas and collecting tool calls.
func (l *Loop) consumeStream(ctx context.Context, streamCh <-chan provider.StreamEvent, ch chan<- Event) (string, []provider.ToolCall, *provider.Metadata) {
	var content strings.Builder
	var toolCalls []provider.ToolCall
	var meta *provider.Metadata

	for event := range streamCh {
		select {
		case <-ctx.Done():
			return content.String(), toolCalls, meta
		default:
		}

		switch event.Type {
		case provider.EventContentDelta:
			content.WriteString(event.Content)
			ch <- NewContentEvent(event.Content)

		case provider.EventReasoningDelta:
			ch <- NewReasoningEvent(event.Content)

		case provider.EventToolCall:
			if event.ToolCall != nil {
				toolCalls = append(toolCalls, *event.ToolCall)
			}

		case provider.EventFinish:
			meta = event.Meta

		case provider.EventError:
			ch <- NewErrorEvent(event.Error)
			return content.String(), toolCalls, meta

		default:
			store.LogErrorf("loop: unknown provider event type %d", event.Type)
		}
	}

	return content.String(), toolCalls, meta
}

// executeTool runs a single tool call and emits start/end events.
func (l *Loop) executeTool(ctx context.Context, tc provider.ToolCall, ch chan<- Event) {
	ch <- NewToolStartEvent(&ToolEvent{
		ID:   tc.ID,
		Name: tc.Name,
		Args: tc.Args,
	})

	t, ok := l.registry.Get(tc.Name)
	if !ok {
		l.emitToolError(tc, fmt.Sprintf("unknown tool %q", tc.Name), ch)
		return
	}

	toolCtx := tool.Context{
		SessionID: l.session.ID,
		WorkDir:   l.session.WorkDir,
		Abort:     ctx,
	}
	if err := toolCtx.Valid(); err != nil {
		l.emitToolError(tc, fmt.Sprintf("invalid tool context: %v", err), ch)
		return
	}

	result := t.Execute(toolCtx, tc.Args)

	output := result.Output
	errStr := ""
	if result.Error != nil {
		errStr = result.Error.Error()
		if output != "" {
			output += "\nerror: " + errStr
		} else {
			output = "error: " + errStr
		}
	}

	l.session.AddToolResult(tc.ID, output)

	ch <- NewToolEndEvent(&ToolEvent{
		ID:     tc.ID,
		Name:   tc.Name,
		Args:   tc.Args,
		Output: output,
		Error:  errStr,
		Diff:   result.Diff,
	})
}

// emitToolError records a tool error result and emits an EventToolEnd with the error.
func (l *Loop) emitToolError(tc provider.ToolCall, errMsg string, ch chan<- Event) {
	l.session.AddToolResult(tc.ID, "error: "+errMsg)
	ch <- NewToolEndEvent(&ToolEvent{
		ID:    tc.ID,
		Name:  tc.Name,
		Error: errMsg,
	})
}
