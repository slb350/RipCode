package agent

import (
	"github.com/stephenbrandon/ripcode/internal/provider"
	"github.com/stephenbrandon/ripcode/internal/tool"
)

// Mode represents an agent's operational mode.
type Mode int

const (
	ModeBuild Mode = iota // All tools enabled
	ModePlan              // Read-only tools only
)

func (m Mode) String() string {
	switch m {
	case ModeBuild:
		return NameBuild
	case ModePlan:
		return NamePlan
	default:
		return "unknown"
	}
}

// Agent name constants.
const (
	NameBuild = "build"
	NamePlan  = "plan"
)

// Agent defines an agent configuration with a name, mode, and tool restrictions.
type Agent struct {
	Name         string
	Mode         Mode
	SystemPrompt string
	AllowedTools []string // tool IDs; empty means all tools allowed
}

// BuildAgent creates an agent with full tool access.
func BuildAgent() Agent {
	return Agent{
		Name:         NameBuild,
		Mode:         ModeBuild,
		SystemPrompt: buildSystemPrompt,
	}
}

// PlanAgent creates an agent with read-only tool access.
func PlanAgent() Agent {
	return Agent{
		Name:         NamePlan,
		Mode:         ModePlan,
		SystemPrompt: planSystemPrompt,
		// NOTE: these must match tool IDs registered in cmd/ripcode/main.go.
		// TestPlanAgent_AllowedToolsMatchRegistry validates this at test time.
		AllowedTools: []string{"read", "glob", "grep", "ls", "todo"},
	}
}

// FilterRegistry returns tool definitions filtered by the agent's allowed tools.
func (a Agent) FilterRegistry(reg *tool.Registry) []provider.ToolDef {
	if len(a.AllowedTools) == 0 {
		return reg.Definitions()
	}

	allowed := make(map[string]bool, len(a.AllowedTools))
	for _, id := range a.AllowedTools {
		allowed[id] = true
	}

	all := reg.Definitions()
	filtered := make([]provider.ToolDef, 0, len(a.AllowedTools))
	for _, def := range all {
		if allowed[def.Name] {
			filtered = append(filtered, def)
		}
	}
	return filtered
}

// AgentInfo describes an agent for display in the agent picker dialog.
type AgentInfo struct {
	Name        string
	Description string
	Native      bool // true for built-in agents (build/plan), false for MCP-provided agents
}

// AllAgents returns info about all available agents.
func AllAgents() []AgentInfo {
	return []AgentInfo{
		{Name: NameBuild, Description: "Full tool access for coding tasks", Native: true},
		{Name: NamePlan, Description: "Read-only analysis and planning", Native: true},
	}
}

const buildSystemPrompt = `You are ripcode, an AI coding assistant in the terminal.
You have access to tools for reading, writing, and executing code.
Use tools to explore the codebase and make changes as requested.
Always explain what you're doing before taking action.`

const planSystemPrompt = `You are ripcode in plan mode.
You can read and explore the codebase but cannot make changes.
Use the available tools to analyze code and propose a plan.
Describe your findings and recommendations clearly.`
