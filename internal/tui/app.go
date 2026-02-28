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

// App is the top-level Bubble Tea model for ripcode.
type App struct {
	width  int
	height int
	ready  bool

	// Core components
	chat      components.Chat
	input     components.Input
	statusbar components.StatusBar
	toolpanel components.ToolPanel

	// Agent state
	provider      provider.Provider
	registry      *tool.Registry
	session       *session.Session
	agent         agent.Agent
	model         string // model name for display
	maxSteps      int
	streaming     bool
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
}

// SetAgent configures the agent.
func (a *App) SetAgent(ag agent.Agent) {
	a.agent = ag
	a.input.SetMode(ag.Name)
	a.chat.SetMode(ag.Name)
	a.statusbar.SetMode(ag.Name)
}

// SetModel updates the displayed model name.
func (a *App) SetModel(model string) {
	a.model = model
	a.statusbar.SetModel(model)
	a.input.SetModel(model)
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
		firstReady := !a.ready
		a.width = msg.Width
		a.height = msg.Height
		a.ready = true
		a.layout()
		if firstReady {
			a.showWelcome()
		}
		return a, nil

	case tea.KeyPressMsg:
		return a.handleKey(msg)

	case components.InputSubmitMsg:
		return a.handleSubmit(msg.Value)

	case AgentEventMsg:
		return a.handleAgentEvent(msg.Event)

	case tea.MouseWheelMsg:
		a.chat.Update(msg)
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
		a.chat.Clear()
		a.toolpanel.Clear()
		return a, nil

	default:
		if !a.streaming {
			cmd := a.input.Update(msg)
			return a, cmd
		}
	}

	return a, nil
}

func (a App) handleSubmit(input string) (tea.Model, tea.Cmd) {
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

func (a App) handleAgentEvent(event agent.Event) (tea.Model, tea.Cmd) {
	switch event.Type {
	case agent.EventContentDelta:
		a.chat.StreamContent(event.Content)
		return a, listenForEvents(a.eventCh)

	case agent.EventToolStart:
		if event.Tool != nil {
			// Inline tool call in chat
			a.chat.AddEntry(components.ChatEntry{
				Role:       "tool",
				Content:    toolSummary(event.Tool),
				ToolID:     event.Tool.ID,
				ToolName:   event.Tool.Name,
				ToolStatus: "pending",
			})
			// Keep toolpanel in sync for backward compat
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
		// Add completion bar
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

func (a *App) showWelcome() {
	workDir := "."
	if a.session != nil {
		workDir = a.session.WorkDir
	}
	welcome := fmt.Sprintf("Welcome to ripcode.\nWorking in: %s\n\nHotkeys: Enter submit | Shift+Enter newline | Esc cancel | Ctrl+C quit | Ctrl+L clear", workDir)
	a.chat.AddEntry(components.ChatEntry{Role: "system", Content: welcome})
}

func (a *App) layout() {
	statusH := 1
	inputH := 5 // accent border + cap + badge + hints + spacing

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

	var sb strings.Builder
	sb.WriteString(a.statusbar.View())
	sb.WriteByte('\n')
	sb.WriteString(a.chat.View())
	sb.WriteByte('\n')
	sb.WriteString(a.input.View())

	v := tea.NewView(sb.String())
	v.AltScreen = true
	v.WindowTitle = "ripcode"
	return v
}
