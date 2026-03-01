package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/stephenbrandon/ripcode/internal/agent"
	"github.com/stephenbrandon/ripcode/internal/provider"
	"github.com/stephenbrandon/ripcode/internal/session"
	"github.com/stephenbrandon/ripcode/internal/tool"
	"github.com/stephenbrandon/ripcode/internal/tui/components"
)

// AppState represents which screen is active.
type AppState int

const (
	StateHome    AppState = iota // startup home screen with logo
	StateSession                 // active conversation session
)

// App is the top-level Bubble Tea model for ripcode.
type App struct {
	width  int
	height int
	ready  bool
	state  AppState

	// Core components
	chat      components.Chat
	input     components.Input
	statusbar components.StatusBar
	toolpanel components.ToolPanel
	home      components.Home

	// Agent state
	provider      provider.Provider
	registry      *tool.Registry
	session       *session.Session
	agent         agent.Agent
	model         string
	maxSteps      int
	streaming     bool
	modelsCache   []provider.ModelInfo
	modelsLoaded  bool
	responseStart time.Time
	cancel        context.CancelFunc
	eventCh       <-chan agent.Event
}

// NewApp creates the initial application model.
func NewApp() App {
	return App{
		chat:      components.NewChat(),
		input:     components.NewInput(),
		statusbar: components.NewStatusBar(),
		toolpanel: components.NewToolPanel(),
		home:      components.NewHome(),
		state:     StateHome,
		maxSteps:  100,
	}
}

// SetProvider configures the LLM provider.
func (a *App) SetProvider(p provider.Provider) {
	a.provider = p
}

// SetRegistry configures the tool registry.
func (a *App) SetRegistry(r *tool.Registry) {
	a.registry = r
}

// SetSession configures the session.
func (a *App) SetSession(s *session.Session) {
	a.session = s
	if s != nil {
		a.home.SetWorkDir(s.WorkDir)
	}
}

// SetAgent configures the agent.
func (a *App) SetAgent(ag agent.Agent) {
	a.agent = ag
	a.input.SetMode(ag.Name)
	a.chat.SetMode(ag.Name)
	a.statusbar.SetMode(ag.Name)
	a.home.SetMode(ag.Name)
}

// SetModel updates the displayed model name.
func (a *App) SetModel(model string) {
	a.model = model
	a.statusbar.SetModel(model)
	a.input.SetModel(model)
	a.home.SetModel(model)
}

// SetMaxSteps sets the max agent loop steps.
func (a *App) SetMaxSteps(n int) {
	a.maxSteps = n
}

// Init implements tea.Model.
func (a App) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model.
func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		a.ready = true
		a.layout()
		return a, nil

	case tea.KeyPressMsg:
		return a.handleKey(msg)

	case components.InputSubmitMsg:
		return a.handleSubmit(msg.Value)

	case ModelsLoadedMsg:
		return a.handleModelsLoaded(msg)

	case AgentEventMsg:
		return a.handleAgentEvent(msg.Event)

	case tea.MouseWheelMsg:
		if a.state == StateSession {
			a.chat.Update(msg)
		}
		return a, nil
	}

	return a, nil
}

func (a App) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case msg.Mod == tea.ModCtrl && msg.Code == 'c':
		return a, tea.Quit

	case msg.Code == tea.KeyEscape:
		if a.streaming {
			if a.cancel != nil {
				a.cancel()
				a.cancel = nil
			}
			a.eventCh = nil
			a.streaming = false
			a.statusbar.SetSpinning(false)
			a.chat.CommitStream()
			return a, nil
		}
		return a, tea.Quit

	case msg.Mod == tea.ModCtrl && msg.Code == 'l':
		if a.state == StateSession {
			a.chat.Clear()
			a.toolpanel.Clear()
		}
		return a, nil

	default:
		if !a.streaming {
			if a.state == StateHome {
				cmd := a.home.Input().Update(msg)
				return a, cmd
			}
			cmd := a.input.Update(msg)
			return a, cmd
		}
	}

	return a, nil
}

func (a App) handleSubmit(input string) (tea.Model, tea.Cmd) {
	// Transition from home to session on first submit
	if a.state == StateHome {
		a.state = StateSession
	}

	trimmed := strings.TrimSpace(input)
	if trimmed == "/models" || strings.HasPrefix(trimmed, "/models ") {
		return a.handleModelsCommand(trimmed)
	}

	if a.provider == nil || a.registry == nil || a.session == nil {
		a.chat.AddEntry(components.ChatEntry{
			Role:    "error",
			Content: "Not configured — missing provider, registry, or session",
		})
		return a, nil
	}

	a.chat.AddEntry(components.ChatEntry{Role: "user", Content: input})
	a.streaming = true
	a.responseStart = time.Now()
	a.statusbar.SetSpinning(true)
	a.input.Blur()

	ctx, cancel := context.WithCancel(context.Background())
	a.cancel = cancel

	loop := agent.NewLoop(a.provider, a.registry, a.session, a.agent, a.maxSteps)
	a.eventCh = loop.Run(ctx, input)

	return a, listenForEvents(a.eventCh)
}

func (a App) handleModelsCommand(input string) (tea.Model, tea.Cmd) {
	a.chat.AddEntry(components.ChatEntry{Role: "user", Content: input})

	if a.provider == nil {
		a.chat.AddEntry(components.ChatEntry{
			Role:    "error",
			Content: "Not configured — missing provider",
		})
		return a, nil
	}

	query := parseModelsQuery(input)

	// Cache hit — display synchronously, no HTTP call.
	if a.modelsLoaded {
		return a.displayModels(a.modelsCache, query), nil
	}

	// Cache miss — show spinner, fetch async.
	a.statusbar.SetSpinning(true)
	a.input.Blur()
	return a, loadModelsCmd(a.provider, query)
}

func (a App) handleModelsLoaded(msg ModelsLoadedMsg) (tea.Model, tea.Cmd) {
	a.statusbar.SetSpinning(false)
	a.input.Focus()

	if msg.Err != nil {
		a.chat.AddEntry(components.ChatEntry{
			Role:    "error",
			Content: msg.Err.Error(),
		})
		return a, nil
	}

	a.modelsCache = msg.Models
	a.modelsLoaded = true
	return a.displayModels(msg.Models, msg.Query), nil
}

// displayModels renders filtered model results into the chat.
func (a App) displayModels(models []provider.ModelInfo, query string) App {
	filtered := filterModels(models, query)
	if len(filtered) == 0 {
		msg := "No models found."
		if query != "" {
			msg = fmt.Sprintf("No models found for %q.", query)
		}
		a.chat.AddEntry(components.ChatEntry{
			Role:    "system",
			Content: msg,
		})
		return a
	}

	const maxDisplay = 120
	lines := make([]string, 0, min(len(filtered), maxDisplay)+2)
	if query == "" {
		lines = append(lines, fmt.Sprintf("Available models: %d total (showing up to %d)", len(filtered), maxDisplay))
	} else {
		lines = append(lines, fmt.Sprintf("Filtered models for %q: %d matches (showing up to %d)", query, len(filtered), maxDisplay))
	}

	display := filtered
	if len(display) > maxDisplay {
		display = display[:maxDisplay]
	}
	for _, m := range display {
		lines = append(lines, modelLine(m))
	}
	if len(filtered) > maxDisplay {
		lines = append(lines, fmt.Sprintf("... %d more matches not shown", len(filtered)-maxDisplay))
	}

	a.chat.AddEntry(components.ChatEntry{
		Role:    "system",
		Content: strings.Join(lines, "\n"),
	})
	return a
}

// loadModelsCmd returns a Cmd that fetches models from the provider in a goroutine.
func loadModelsCmd(p provider.Provider, query string) tea.Cmd {
	return func() tea.Msg {
		lister, ok := p.(provider.ModelLister)
		if !ok {
			return ModelsLoadedMsg{
				Err:   fmt.Errorf("provider %q does not support model listing", p.Name()),
				Query: query,
			}
		}

		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		models, err := lister.ListModels(ctx)
		if err != nil {
			return ModelsLoadedMsg{
				Err:   fmt.Errorf("list models: %w", err),
				Query: query,
			}
		}

		return ModelsLoadedMsg{
			Models: models,
			Query:  query,
		}
	}
}

func parseModelsQuery(input string) string {
	parts := strings.Fields(input)
	if len(parts) <= 1 {
		return ""
	}
	return strings.Join(parts[1:], " ")
}

func filterModels(models []provider.ModelInfo, query string) []provider.ModelInfo {
	if query == "" {
		return models
	}

	q := strings.ToLower(query)
	out := make([]provider.ModelInfo, 0, len(models))
	for _, m := range models {
		if strings.Contains(strings.ToLower(m.ID), q) || strings.Contains(strings.ToLower(m.Name), q) {
			out = append(out, m)
		}
	}
	return out
}

func modelLine(m provider.ModelInfo) string {
	if m.Name == "" || m.Name == m.ID {
		return m.ID
	}
	return m.ID + " - " + m.Name
}

func (a App) handleAgentEvent(event agent.Event) (tea.Model, tea.Cmd) {
	switch event.Type {
	case agent.EventContentDelta:
		a.chat.StreamContent(event.Content)
		return a, listenForEvents(a.eventCh)

	case agent.EventToolStart:
		if event.Tool != nil {
			a.chat.AddEntry(components.ChatEntry{
				Role:       "tool",
				Content:    toolSummary(event.Tool),
				ToolID:     event.Tool.ID,
				ToolName:   event.Tool.Name,
				ToolStatus: "pending",
			})
			a.toolpanel.AddEvent(agent.ToolEvent{
				ID:   event.Tool.ID,
				Name: event.Tool.Name,
				Args: event.Tool.Args,
			})
		}
		return a, listenForEvents(a.eventCh)

	case agent.EventToolEnd:
		if event.Tool != nil {
			status := "success"
			if event.Tool.Error != "" {
				status = "error"
			}
			content := event.Tool.Output
			if status == "error" {
				content = event.Tool.Error
			}
			a.chat.UpdateLastTool(event.Tool.ID, components.ChatEntry{
				Role:       "tool",
				Content:    content,
				ToolID:     event.Tool.ID,
				ToolName:   event.Tool.Name,
				ToolStatus: status,
			})
			a.toolpanel.AddEvent(agent.ToolEvent{
				ID:     event.Tool.ID,
				Name:   event.Tool.Name,
				Output: event.Tool.Output,
				Error:  event.Tool.Error,
			})
		}
		return a, listenForEvents(a.eventCh)

	case agent.EventDone:
		a.eventCh = nil
		a.streaming = false
		a.statusbar.SetSpinning(false)
		a.chat.CommitStream()
		a.input.Focus()
		if a.session != nil {
			a.statusbar.SetTokens(a.session.Tokens.Input + a.session.Tokens.Output)
		}
		dur := time.Since(a.responseStart)
		modeName := a.agent.Name
		if modeName == "" {
			modeName = "build"
		}
		a.chat.AddEntry(components.ChatEntry{
			Role: "complete",
			Meta: &components.CompleteMeta{
				Mode:     modeName,
				Model:    a.model,
				Duration: dur,
			},
		})
		return a, nil

	case agent.EventError:
		a.eventCh = nil
		a.streaming = false
		a.statusbar.SetSpinning(false)
		a.chat.CommitStream()
		a.input.Focus()
		if event.Error != nil {
			a.chat.AddEntry(components.ChatEntry{
				Role:    "error",
				Content: event.Error.Error(),
			})
		}
		return a, nil
	}

	return a, nil
}

// toolSummary extracts a short summary from tool args.
func toolSummary(te *agent.ToolEvent) string {
	if te.Args != "" {
		s := te.Args
		if idx := strings.IndexByte(s, '\n'); idx >= 0 {
			s = s[:idx]
		}
		if len(s) > 80 {
			s = s[:77] + "..."
		}
		return s
	}
	return te.Name
}

// listenForEvents returns a cmd that reads events from the channel.
func listenForEvents(ch <-chan agent.Event) tea.Cmd {
	return func() tea.Msg {
		event, ok := <-ch
		if !ok {
			return AgentEventMsg{Event: agent.Event{Type: agent.EventDone}}
		}
		return AgentEventMsg{Event: event}
	}
}

func (a *App) layout() {
	if a.state == StateHome {
		a.home.SetSize(a.width, a.height)
		return
	}

	statusH := 1
	inputH := 5

	chatH := a.height - statusH - inputH
	if chatH < 1 {
		chatH = 1
	}

	a.statusbar.SetSize(a.width)
	a.chat.SetSize(a.width, chatH)
	a.input.SetSize(a.width, inputH)
	a.toolpanel.SetSize(a.width)
}

// View implements tea.Model.
func (a App) View() tea.View {
	if !a.ready {
		return tea.NewView("Initializing...")
	}

	a.layout()

	var content string
	if a.state == StateHome {
		content = a.home.View()
	} else {
		content = a.renderSessionView()
	}

	v := tea.NewView(content)
	v.AltScreen = true
	v.WindowTitle = "ripcode"
	return v
}

func (a App) renderSessionView() string {
	var sb strings.Builder
	sb.WriteString(a.statusbar.View())
	sb.WriteByte('\n')
	sb.WriteString(a.chat.View())
	sb.WriteByte('\n')
	sb.WriteString(a.input.View())
	return sb.String()
}
