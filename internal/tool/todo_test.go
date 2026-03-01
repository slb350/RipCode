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

func TestTodoTool_Items_Empty(t *testing.T) {
	td := NewTodoTool()
	items := td.Items()
	assert.Empty(t, items)
}

func TestTodoTool_Items_AfterWrite(t *testing.T) {
	td := NewTodoTool()
	ctx := newTestCtx(t)
	td.Execute(ctx, `{"action":"write","items":[{"subject":"Task A","status":"pending"},{"subject":"Task B","status":"completed"}]}`)

	items := td.Items()
	require.Len(t, items, 2)
	assert.Equal(t, "Task A", items[0].Subject)
	assert.Equal(t, "pending", items[0].Status)
	assert.Equal(t, "Task B", items[1].Subject)
	assert.Equal(t, "completed", items[1].Status)
}

func TestTodoTool_Items_ReturnsCopy(t *testing.T) {
	td := NewTodoTool()
	ctx := newTestCtx(t)
	td.Execute(ctx, `{"action":"write","items":[{"subject":"Task A","status":"pending"}]}`)

	items := td.Items()
	items[0].Subject = "mutated"

	fresh := td.Items()
	assert.Equal(t, "Task A", fresh[0].Subject)
}
