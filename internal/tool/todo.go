package tool

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
)

// TodoItem represents a single task.
type TodoItem struct {
	Subject string `json:"subject"`
	Status  string `json:"status"` // "pending", "in_progress", "completed"
}

// TodoTool provides session-scoped task tracking.
type TodoTool struct {
	mu    sync.Mutex
	items []TodoItem
}

func NewTodoTool() *TodoTool { return &TodoTool{} }

func (td *TodoTool) ID() string { return "todo" }

func (td *TodoTool) Description() string {
	return "Track tasks for the current session. Use 'read' to list tasks and 'write' to set the full task list."
}

func (td *TodoTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"action": map[string]any{
				"type":        "string",
				"description": "Action to perform: 'read' or 'write'",
				"enum":        []string{"read", "write"},
			},
			"items": map[string]any{
				"type":        "array",
				"description": "Full list of todo items (required for 'write' action)",
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"subject": map[string]any{"type": "string"},
						"status":  map[string]any{"type": "string"},
					},
				},
			},
		},
		"required": []string{"action"},
	}
}

type todoArgs struct {
	Action string     `json:"action"`
	Items  []TodoItem `json:"items,omitempty"`
}

func (td *TodoTool) Execute(ctx Context, argsJSON string) Result {
	var args todoArgs
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return Result{Error: fmt.Errorf("parse args: %w", err)}
	}

	switch args.Action {
	case "read":
		return td.read()
	case "write":
		return td.write(args.Items)
	default:
		return Result{Error: fmt.Errorf("unknown action %q — use 'read' or 'write'", args.Action)}
	}
}

func (td *TodoTool) read() Result {
	td.mu.Lock()
	defer td.mu.Unlock()

	if len(td.items) == 0 {
		return Result{Output: "(no items)"}
	}

	var sb strings.Builder
	for i, item := range td.items {
		marker := "[ ]"
		switch item.Status {
		case "completed":
			marker = "[x]"
		case "in_progress":
			marker = "[~]"
		}
		fmt.Fprintf(&sb, "%d. %s %s\n", i+1, marker, item.Subject)
	}
	return Result{Output: sb.String()}
}

// Items returns a copy of the current todo items.
func (td *TodoTool) Items() []TodoItem {
	td.mu.Lock()
	defer td.mu.Unlock()
	out := make([]TodoItem, len(td.items))
	copy(out, td.items)
	return out
}

func (td *TodoTool) write(items []TodoItem) Result {
	td.mu.Lock()
	defer td.mu.Unlock()

	td.items = make([]TodoItem, len(items))
	copy(td.items, items)

	return Result{Output: fmt.Sprintf("Updated: %d items", len(items))}
}
