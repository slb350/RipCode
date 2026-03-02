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

func TestAllAgents_ReturnsBuildAndPlan(t *testing.T) {
	agents := AllAgents()
	assert.Len(t, agents, 2)
	names := make([]string, len(agents))
	for i, a := range agents {
		names[i] = a.Name
	}
	assert.Contains(t, names, NameBuild)
	assert.Contains(t, names, NamePlan)
}

// TestPlanAgent_AllowedToolsMatchRegistry verifies that every tool ID in
// PlanAgent().AllowedTools corresponds to a tool that would be registered
// in the full registry. This prevents silent drift if tools are renamed
// or removed.
func TestPlanAgent_AllowedToolsMatchRegistry(t *testing.T) {
	// Build a full registry matching cmd/ripcode/main.go
	reg := tool.NewRegistry()
	reg.Register(tool.NewBashTool())
	reg.Register(tool.NewReadTool())
	reg.Register(tool.NewWriteTool())
	reg.Register(tool.NewEditTool())
	reg.Register(tool.NewGlobTool())
	reg.Register(tool.NewGrepTool())
	reg.Register(tool.NewLsTool())
	reg.Register(tool.NewTodoTool())

	plan := PlanAgent()
	for _, id := range plan.AllowedTools {
		_, ok := reg.Get(id)
		assert.True(t, ok, "PlanAgent.AllowedTools contains %q but no such tool is registered — update agent.go or cmd/ripcode/main.go", id)
	}
	// Write-tool exclusion is verified in TestPlanAgent.
}

func TestAllAgents_HaveDescriptions(t *testing.T) {
	agents := AllAgents()
	for _, a := range agents {
		assert.NotEmpty(t, a.Description, "agent %s should have a description", a.Name)
	}
}
