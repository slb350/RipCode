package tui

import (
	"context"
	"strings"

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
	provider  provider.Provider
	registry  *tool.Registry
	session   *session.Session
	agent     agent.Agent
	maxSteps  int
	streaming bool
	cancel    context.CancelFunc
}

// NewApp creates the initial application model.
func NewApp() App {
	return App{
		chat:      components.NewChat(),
		input:     components.NewInput(),
		statusbar: components.NewStatusBar(),
		toolpanel: components.NewToolPanel(),
		agent:     agent.BuildAgent(),
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

// SetModel updates the displayed model name.
func (a *App) SetModel(model string) {
	a.statusbar.SetModel(model)
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
			// Cancel streaming
			if a.cancel != nil {
				a.cancel()
				a.cancel = nil
			}
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
	a.statusbar.SetSpinning(true)
	a.input.Blur()

	ctx, cancel := context.WithCancel(context.Background())
	a.cancel = cancel

	loop := agent.NewLoop(a.provider, a.registry, a.session, a.agent, a.maxSteps)
	eventCh := loop.Run(ctx, input)

	return a, listenForEvents(eventCh)
}

func (a App) handleAgentEvent(event agent.Event) (tea.Model, tea.Cmd) {
	switch event.Type {
	case agent.EventContentDelta:
		a.chat.StreamContent(event.Content)

	case agent.EventToolStart:
		if event.Tool != nil {
			a.toolpanel.AddEvent(agent.ToolEvent{
				ID:   event.Tool.ID,
				Name: event.Tool.Name,
				Args: event.Tool.Args,
			})
		}

	case agent.EventToolEnd:
		if event.Tool != nil {
			a.toolpanel.AddEvent(agent.ToolEvent{
				ID:     event.Tool.ID,
				Name:   event.Tool.Name,
				Output: event.Tool.Output,
				Error:  event.Tool.Error,
			})
		}

	case agent.EventDone:
		a.streaming = false
		a.statusbar.SetSpinning(false)
		a.chat.CommitStream()
		a.input.Focus()
		if a.session != nil {
			a.statusbar.SetTokens(a.session.Tokens.Input + a.session.Tokens.Output)
		}
		return a, nil

	case agent.EventError:
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

	// Keep listening for more events while streaming
	return a, nil
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
	statusH := 1
	inputH := 3
	toolH := 0
	if tv := a.toolpanel.View(); tv != "" {
		toolH = strings.Count(tv, "\n") + 1
	}

	chatH := a.height - statusH - inputH - toolH
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
	if tv := a.toolpanel.View(); tv != "" {
		sb.WriteString(tv)
	}
	sb.WriteString(a.input.View())

	v := tea.NewView(sb.String())
	v.AltScreen = true
	v.WindowTitle = "ripcode"
	return v
}
