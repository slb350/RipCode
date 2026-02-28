package tool

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTodo_ImplementsTool(t *testing.T) {
	var _ Tool = NewTodoTool()
}

func TestTodo_WriteAndRead(t *testing.T) {
	td := NewTodoTool()
	ctx := newTestCtx(t)

	// Write items
	result := td.Execute(ctx, `{"action":"write","items":[{"subject":"Fix bug","status":"pending"},{"subject":"Add tests","status":"pending"}]}`)
	require.NoError(t, result.Error)
	assert.Contains(t, result.Output, "2 items")

	// Read items back
	result = td.Execute(ctx, `{"action":"read"}`)
	require.NoError(t, result.Error)
	assert.Contains(t, result.Output, "Fix bug")
	assert.Contains(t, result.Output, "Add tests")
	assert.Contains(t, result.Output, "[ ]")
}

func TestTodo_UpdateStatus(t *testing.T) {
	td := NewTodoTool()
	ctx := newTestCtx(t)

	// Write items
	td.Execute(ctx, `{"action":"write","items":[{"subject":"Task A","status":"pending"},{"subject":"Task B","status":"pending"}]}`)

	// Update status
	result := td.Execute(ctx, `{"action":"write","items":[{"subject":"Task A","status":"completed"},{"subject":"Task B","status":"pending"}]}`)
	require.NoError(t, result.Error)

	// Verify
	result = td.Execute(ctx, `{"action":"read"}`)
	require.NoError(t, result.Error)
	assert.Contains(t, result.Output, "[x]")
}

func TestTodo_ReadEmpty(t *testing.T) {
	td := NewTodoTool()
	ctx := newTestCtx(t)

	result := td.Execute(ctx, `{"action":"read"}`)
	require.NoError(t, result.Error)
	assert.Contains(t, result.Output, "no items")
}

func TestTodo_InvalidAction(t *testing.T) {
	td := NewTodoTool()
	ctx := newTestCtx(t)

	result := td.Execute(ctx, `{"action":"delete"}`)
	assert.Error(t, result.Error)
	assert.Contains(t, result.Error.Error(), "unknown action")
}

func TestTodo_InvalidJSON(t *testing.T) {
	td := NewTodoTool()
	ctx := newTestCtx(t)

	result := td.Execute(ctx, `{bad}`)
	assert.Error(t, result.Error)
}

func TestTodo_Parameters(t *testing.T) {
	td := NewTodoTool()
	params := td.Parameters()

	props := params["properties"].(map[string]any)
	assert.Contains(t, props, "action")
	assert.Contains(t, props, "items")
}
