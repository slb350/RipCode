package tool

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeTool is a minimal Tool implementation for testing the registry.
type fakeTool struct {
	id          string
	description string
	params      map[string]any
	execFn      func(ctx Context, args string) Result
}

func (f *fakeTool) ID() string              { return f.id }
func (f *fakeTool) Description() string      { return f.description }
func (f *fakeTool) Parameters() map[string]any { return f.params }
func (f *fakeTool) Execute(ctx Context, args string) Result {
	if f.execFn != nil {
		return f.execFn(ctx, args)
	}
	return Result{Output: "ok"}
}

func TestRegistry_RegisterAndGet(t *testing.T) {
	reg := NewRegistry()
	tool := &fakeTool{id: "test", description: "A test tool"}
	reg.Register(tool)

	got, ok := reg.Get("test")
	require.True(t, ok)
	assert.Equal(t, "test", got.ID())
}

func TestRegistry_GetMissing(t *testing.T) {
	reg := NewRegistry()

	_, ok := reg.Get("nonexistent")
	assert.False(t, ok)
}

func TestRegistry_List(t *testing.T) {
	reg := NewRegistry()
	reg.Register(&fakeTool{id: "alpha", description: "A"})
	reg.Register(&fakeTool{id: "beta", description: "B"})

	list := reg.List()
	assert.Len(t, list, 2)

	ids := make(map[string]bool)
	for _, t := range list {
		ids[t.ID()] = true
	}
	assert.True(t, ids["alpha"])
	assert.True(t, ids["beta"])
}

func TestRegistry_Definitions(t *testing.T) {
	reg := NewRegistry()
	reg.Register(&fakeTool{
		id:          "read",
		description: "Read a file",
		params: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path": map[string]any{"type": "string"},
			},
		},
	})

	defs := reg.Definitions()
	require.Len(t, defs, 1)
	assert.Equal(t, "read", defs[0].Name)
	assert.Equal(t, "Read a file", defs[0].Description)
	assert.Contains(t, defs[0].Parameters, "properties")
}

func TestContext_HasRequiredFields(t *testing.T) {
	ctx := Context{
		SessionID: "sess-1",
		WorkDir:   "/tmp/test",
		Abort:     context.Background(),
	}

	assert.Equal(t, "sess-1", ctx.SessionID)
	assert.Equal(t, "/tmp/test", ctx.WorkDir)
	assert.NotNil(t, ctx.Abort)
}

func TestResult_WithError(t *testing.T) {
	r := Result{
		Output: "",
		Error:  assert.AnError,
	}

	assert.Error(t, r.Error)
}
