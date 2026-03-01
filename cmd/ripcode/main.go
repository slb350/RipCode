package main

import (
	"fmt"
	"os"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/stephenbrandon/ripcode/internal/agent"
	"github.com/stephenbrandon/ripcode/internal/client"
	"github.com/stephenbrandon/ripcode/internal/config"
	"github.com/stephenbrandon/ripcode/internal/session"
	"github.com/stephenbrandon/ripcode/internal/tool"
	"github.com/stephenbrandon/ripcode/internal/tui"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}
	for _, w := range cfg.Warnings {
		fmt.Fprintf(os.Stderr, "Warning: %s\n", w)
	}

	// Provider
	provider := client.NewOpenRouter(cfg.APIKey, cfg.Model)

	// Tool registry
	registry := tool.NewRegistry()
	registry.Register(tool.NewBashTool())
	registry.Register(tool.NewReadTool())
	registry.Register(tool.NewWriteTool())
	registry.Register(tool.NewEditTool())
	registry.Register(tool.NewGlobTool())
	registry.Register(tool.NewGrepTool())
	registry.Register(tool.NewLsTool())
	registry.Register(tool.NewTodoTool())

	// Agent
	ag := agent.BuildAgent()

	// Session
	sess := session.New(cfg.WorkDir)
	sess.SetSystemPrompt(ag.SystemPrompt)

	// TUI
	app := tui.NewApp()
	app.SetProvider(provider)
	app.SetRegistry(registry)
	app.SetSession(sess)
	app.SetAgent(ag)
	app.SetModel(shortModel(cfg.Model))
	app.SetFullModelID(cfg.Model)
	app.SetMaxSteps(cfg.MaxSteps)

	p := tea.NewProgram(app)
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}
}

// shortModel extracts a readable name from a provider/model string.
func shortModel(model string) string {
	parts := strings.Split(model, "/")
	if len(parts) > 1 {
		return parts[len(parts)-1]
	}
	return model
}
