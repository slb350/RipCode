package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/stephenbrandon/ripcode/internal/agent"
	"github.com/stephenbrandon/ripcode/internal/tool"
	"github.com/stephenbrandon/ripcode/internal/tui/components"
)

func (a App) handleSubmit(input string) (tea.Model, tea.Cmd) {
	// Transition from home to session on first submit
	if a.state == StateHome {
		a.state = StateSession
		a.commandOpen = false
		a.modelDialogOpen = false
		a.sidebarOverlay = false
	}
	a.inlineOpen = false

	mode := "normal"
	if a.shellMode {
		mode = "shell"
	}
	a.promptHistory.PushWithMode(input, mode)
	persistHistory(a.promptHistory)

	// Shell mode: ! prefix executes bash directly
	if a.shellMode && strings.HasPrefix(strings.TrimSpace(input), "!") {
		return a.handleShellSubmit(input)
	}

	trimmed := strings.TrimSpace(input)
	if strings.HasPrefix(trimmed, "/") {
		next, cmd, handled := a.handleSlashCommand(trimmed)
		if handled {
			return next, cmd
		}
	}

	if a.provider == nil || a.registry == nil || a.session == nil {
		a.chat.AddEntry(components.ChatEntry{
			Role:    "error",
			Content: "Not configured — missing provider, registry, or session",
		})
		return a, nil
	}

	a.chat.AddEntry(components.ChatEntry{Role: "user", Content: input})
	a.setStreaming(true)
	a.responseStart = time.Now()
	a.input.Blur()

	ctx, cancel := context.WithCancel(context.Background())
	a.cancel = cancel

	loop := agent.NewLoop(a.provider, a.registry, a.session, a.agent, a.maxSteps)
	a.eventCh = loop.Run(ctx, input)

	return a, listenForEvents(a.eventCh)
}

func (a App) handleShellSubmit(input string) (tea.Model, tea.Cmd) {
	a.shellMode = false
	a.input.SetShellMode(false)

	cmd := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(input), "!"))
	if cmd == "" {
		toastCmd := a.ShowToast("Empty shell command", components.ToastWarning)
		return a, toastCmd
	}

	a.chat.AddEntry(components.ChatEntry{Role: "user", Content: "! " + cmd})

	// Execute async via tea.Cmd
	shellCmd := cmd
	workDir := ""
	if a.session != nil {
		workDir = a.session.WorkDir
	}
	return a, func() tea.Msg {
		bashTool := tool.NewBashTool()
		argsJSON := fmt.Sprintf(`{"command":%q}`, shellCmd)
		result := bashTool.Execute(tool.Context{
			WorkDir: workDir,
			Abort:   context.Background(),
		}, argsJSON)
		if result.Error != nil {
			return ShellResultMsg{Command: shellCmd, Error: result.Error.Error()}
		}
		return ShellResultMsg{Command: shellCmd, Output: result.Output}
	}
}

func (a App) handleSlashCommand(input string) (App, tea.Cmd, bool) {
	parts := strings.Fields(input)
	if len(parts) == 0 {
		return a, nil, false
	}

	name := strings.TrimPrefix(strings.ToLower(parts[0]), "/")

	// Special cases that need args or special handling
	switch name {
	case "models":
		model, cmd := a.handleModelsCommand(input)
		return model.(App), cmd, true
	case "agent":
		return a.handleAgentCommand(input, parts[1:]), nil, true
	case "model":
		if len(parts) == 1 {
			model, cmd := a.handleModelsCommand("/models")
			return model.(App), cmd, true
		}
		return a.handleModelCommand(input, parts[1:]), nil, true
	}

	// Registry lookup
	cmd := a.cmdRegistry.Get(name)
	if cmd == nil {
		a.chat.AddEntry(components.ChatEntry{Role: "user", Content: input})
		a.chat.AddEntry(components.ChatEntry{
			Role:    "error",
			Content: `Unknown command. Try "/help".`,
		})
		return a, nil, true
	}

	a.chat.AddEntry(components.ChatEntry{Role: "user", Content: input})
	teaCmd := cmd.Handler(&a)
	return a, teaCmd, true
}

func (a App) handleAgentCommand(input string, args []string) App {
	a.chat.AddEntry(components.ChatEntry{Role: "user", Content: input})
	if len(args) != 1 {
		a.chat.AddEntry(components.ChatEntry{
			Role:    "error",
			Content: fmt.Sprintf("Usage: /agent %s|%s", agent.NameBuild, agent.NamePlan),
		})
		return a
	}

	switch strings.ToLower(args[0]) {
	case agent.NameBuild:
		a.SetAgent(agent.BuildAgent())
		a.chat.AddEntry(components.ChatEntry{
			Role:    "system",
			Content: fmt.Sprintf(`Agent switched to "%s".`, agent.NameBuild),
		})
	case agent.NamePlan:
		a.SetAgent(agent.PlanAgent())
		a.chat.AddEntry(components.ChatEntry{
			Role:    "system",
			Content: fmt.Sprintf(`Agent switched to "%s".`, agent.NamePlan),
		})
	default:
		a.chat.AddEntry(components.ChatEntry{
			Role:    "error",
			Content: fmt.Sprintf(`Unknown agent. Use "%s" or "%s".`, agent.NameBuild, agent.NamePlan),
		})
	}
	return a
}

func (a App) handleModelCommand(input string, args []string) App {
	a.chat.AddEntry(components.ChatEntry{Role: "user", Content: input})
	if len(args) == 0 {
		a.chat.AddEntry(components.ChatEntry{
			Role:    "system",
			Content: fmt.Sprintf(`Current model: %s`, a.model),
		})
		a.chat.AddEntry(components.ChatEntry{
			Role:    "system",
			Content: `Usage: /model <provider/model-id>`,
		})
		return a
	}

	a.switchModel(strings.Join(args, " "))
	return a
}
