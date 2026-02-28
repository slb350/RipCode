package agent

import (
	"testing"

	"github.com/stephenbrandon/ripcode/internal/tool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildAgent(t *testing.T) {
	a := BuildAgent()

	assert.Equal(t, "build", a.Name)
	assert.Equal(t, ModeBuild, a.Mode)
	assert.NotEmpty(t, a.SystemPrompt)
	assert.Empty(t, a.AllowedTools) // all tools allowed
}

func TestPlanAgent(t *testing.T) {
	a := PlanAgent()

	assert.Equal(t, "plan", a.Name)
	assert.Equal(t, ModePlan, a.Mode)
	assert.NotEmpty(t, a.AllowedTools)
	// Plan mode should only allow read-only tools
	assert.Contains(t, a.AllowedTools, "read")
	assert.Contains(t, a.AllowedTools, "glob")
	assert.Contains(t, a.AllowedTools, "grep")
	assert.Contains(t, a.AllowedTools, "ls")
	assert.Contains(t, a.AllowedTools, "todo")
	// Should NOT allow write tools
	assert.NotContains(t, a.AllowedTools, "bash")
	assert.NotContains(t, a.AllowedTools, "write")
	assert.NotContains(t, a.AllowedTools, "edit")
}

func TestFilterRegistry_BuildMode(t *testing.T) {
	reg := tool.NewRegistry()
	reg.Register(tool.NewBashTool())
	reg.Register(tool.NewReadTool())
	reg.Register(tool.NewWriteTool())

	a := BuildAgent()
	defs := a.FilterRegistry(reg)

	// Build mode should include all tools
	assert.Len(t, defs, 3)
}

func TestFilterRegistry_PlanMode(t *testing.T) {
	reg := tool.NewRegistry()
	reg.Register(tool.NewBashTool())
	reg.Register(tool.NewReadTool())
	reg.Register(tool.NewWriteTool())
	reg.Register(tool.NewGlobTool())
	reg.Register(tool.NewGrepTool())
	reg.Register(tool.NewLsTool())
	reg.Register(tool.NewTodoTool())

	a := PlanAgent()
	defs := a.FilterRegistry(reg)

	// Plan mode should only include read-only tools
	names := make(map[string]bool)
	for _, d := range defs {
		names[d.Name] = true
	}

	assert.True(t, names["read"])
	assert.True(t, names["glob"])
	assert.True(t, names["grep"])
	assert.True(t, names["ls"])
	assert.True(t, names["todo"])
	assert.False(t, names["bash"])
	assert.False(t, names["write"])

	require.Len(t, defs, 5)
}

func TestMode_String(t *testing.T) {
	assert.Equal(t, "build", ModeBuild.String())
	assert.Equal(t, "plan", ModePlan.String())
}
