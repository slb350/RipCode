package tui

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/atotto/clipboard"
	"sort"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/stephenbrandon/ripcode/internal/agent"
	"github.com/stephenbrandon/ripcode/internal/provider"
	"github.com/stephenbrandon/ripcode/internal/session"
	"github.com/stephenbrandon/ripcode/internal/store"
	"github.com/stephenbrandon/ripcode/internal/tool"
	"github.com/stephenbrandon/ripcode/internal/tui/components"
	"github.com/stephenbrandon/ripcode/internal/tui/styles"
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
	footer    components.SessionFooter
	toolpanel components.ToolPanel
	home      components.Home

	// Agent state
	provider          provider.Provider
	registry          *tool.Registry
	session           *session.Session
	agent             agent.Agent
	model             string
	maxSteps          int
	streaming         bool
	modelsCache       []provider.ModelInfo
	modelsLoaded      bool
	commandOpen       bool
	commandQuery      string
	commandSelect     int
	inlineOpen        bool
	inlineMode        string // "/" or "@"
	inlineQuery       string
	inlineSelect      int
	inlineStart       int
	inlineEnd         int
	modelDialogOpen   bool
	modelDialogQuery  string
	modelDialogSelect int
	sidebarHidden     bool
	sidebarOverlay    bool
	fileCache         []string
	fileCacheLoaded   bool
	fileCacheLoading  bool
	responseStart     time.Time
	cancel            context.CancelFunc
	eventCh           <-chan agent.Event
	promptHistory     *components.PromptHistory
	toasts            components.ToastManager
	shellMode         bool
	cmdRegistry       *CommandRegistry
	showDetails       bool
	showThinking      bool
	showTimestamps    bool

	// Help dialog
	helpDialogOpen   bool
	helpDialogQuery  string
	helpDialogSelect int
	helpDialogTab    int // 0=commands, 1=keybinds

	// Status dialog
	statusDialogOpen bool

	// Export dialog
	exportDialogOpen   bool
	exportIncludeTools bool
	exportIncludeMeta  bool
	exportFilename     string
	exportFocusedField int // 0=tools, 1=meta

	// Rename dialog
	renameDialogOpen  bool
	renameDialogValue string

	// Sessions dialog
	sessionsDialogOpen    bool
	sessionsDialogQuery   string
	sessionsDialogSelect  int
	sessionsDialogConfirm bool
	sessionsDialogEntries []store.SessionSummary
	sessionsDialogLoaded  bool

	// Timeline dialog
	timelineDialogOpen   bool
	timelineDialogQuery  string
	timelineDialogSelect int

	// Prompt stash
	stash               *components.PromptStash
	stashDialogOpen     bool
	stashDialogSelect   int
	stashPendingContent string // content to stash (captured before input reset)
}

type inlineEntry struct {
	Display     string
	Insert      string
	Description string
	Execute     bool
}

// pickerItem is a generic display item for renderPickerList.
type pickerItem struct {
	Label       string
	Description string
}

// renderPickerList renders a windowed selection list with "> " marker.
// header is the title line, items are the display items, selected is the
// highlighted index, and maxRows is the visible window size.
// Returns empty-state text if items is empty.
func renderPickerList(header string, items []pickerItem, selected, maxRows int) string {
	if len(items) == 0 {
		return header + "\n  no matches"
	}

	selected = clamp(selected, 0, len(items)-1)
	start := 0
	if selected >= maxRows {
		start = selected - maxRows + 1
	}
	end := min(len(items), start+maxRows)

	var sb strings.Builder
	sb.WriteString(header)
	for i := start; i < end; i++ {
		prefix := "  "
		if i == selected {
			prefix = "> "
		}
		sb.WriteString("\n")
		sb.WriteString(prefix)
		sb.WriteString(items[i].Label)
		if items[i].Description != "" {
			sb.WriteString("  ")
			sb.WriteString(items[i].Description)
		}
	}
	if len(items) > end {
		sb.WriteString(fmt.Sprintf("\n  ... %d more", len(items)-end))
	}
	return sb.String()
}

// NewApp creates the initial application model.
func NewApp() App {
	a := App{
		chat:          components.NewChat(),
		input:         components.NewInput(),
		statusbar:     components.NewStatusBar(),
		footer:        components.NewSessionFooter(),
		toolpanel:     components.NewToolPanel(),
		home:          components.NewHome(),
		state:         StateHome,
		maxSteps:      100,
		promptHistory: components.NewPromptHistory(200),
		toasts:        components.NewToastManager(),
		stash:         components.NewPromptStash(),
	}
	a.initRegistry()
	return a
}

// closeAllDialogs closes every dialog and overlay, then blurs input.
func (a *App) closeAllDialogs() {
	a.commandOpen = false
	a.modelDialogOpen = false
	a.helpDialogOpen = false
	a.statusDialogOpen = false
	a.exportDialogOpen = false
	a.renameDialogOpen = false
	a.sessionsDialogOpen = false
	a.timelineDialogOpen = false
	a.stashDialogOpen = false
	a.inlineOpen = false
	a.input.Blur()
}

// stubToast returns a handler that shows a "Not yet implemented" toast.
func stubToast(name string) func(a *App) tea.Cmd {
	return func(a *App) tea.Cmd {
		return a.ShowToast(fmt.Sprintf("/%s: Not yet implemented", name), components.ToastInfo)
	}
}

func (a *App) initRegistry() {
	r := NewCommandRegistry()

	r.Register(Command{
		Name: "help", Aliases: []string{"commands"}, Category: CategorySystem,
		Title: "Help", Description: "Show available commands",
		Suggested: true, Execute: true,
		Handler: func(a *App) tea.Cmd {
			a.closeAllDialogs()
			a.helpDialogOpen = true
			a.helpDialogQuery = ""
			a.helpDialogSelect = 0
			a.helpDialogTab = 0
			return nil
		},
	})

	r.Register(Command{
		Name: "new", Aliases: []string{"clear"}, Category: CategorySession,
		Title: "New session", Description: "Clear chat and tool history",
		Suggested: true, Execute: true,
		Handler: func(a *App) tea.Cmd {
			a.chat.Clear()
			a.toolpanel.Clear()
			if a.session != nil {
				a.session.Reset()
				a.statusbar.SetTitle(shortSessionTitle(a.session.ID))
				a.statusbar.SetTokens(0)
			}
			a.chat.AddEntry(components.ChatEntry{
				Role:    "system",
				Content: "Conversation cleared.",
			})
			return nil
		},
	})

	r.Register(Command{
		Name: "models", Category: CategoryAgent,
		Title: "Select model", Description: "Search and switch models",
		Suggested: true, Execute: true,
		Handler: func(a *App) tea.Cmd {
			// Re-use existing models command logic
			return nil // special-cased in handleSlashCommand
		},
	})

	r.Register(Command{
		Name: "agent", Category: CategoryAgent,
		Title: "Switch agent", Description: "Toggle build/plan agent",
		Keybind: "Tab", Execute: true,
		Handler: func(a *App) tea.Cmd {
			return nil // special-cased (needs args)
		},
	})

	r.Register(Command{
		Name: "sidebar", Category: CategoryView,
		Title: "Toggle sidebar", Description: "Show or hide session sidebar",
		Keybind: "Ctrl+B", Execute: true,
		Handler: func(a *App) tea.Cmd {
			*a = a.toggleSidebar()
			state := "hidden"
			if a.sidebarVisible() {
				state = "shown"
			}
			a.chat.AddEntry(components.ChatEntry{
				Role:    "system",
				Content: fmt.Sprintf("Sidebar %s.", state),
			})
			return nil
		},
	})

	r.Register(Command{
		Name: "model", Category: CategoryAgent,
		Title: "Set model", Description: "Type full provider/model-id and submit",
		Execute: false,
		Handler: func(a *App) tea.Cmd { return nil },
	})

	r.Register(Command{
		Name: "exit", Aliases: []string{"quit", "q"}, Category: CategorySystem,
		Title: "Exit", Description: "Quit ripcode",
		Execute: true,
		Handler: func(_ *App) tea.Cmd { return tea.Quit },
	})

	r.Register(Command{
		Name: "compact", Aliases: []string{"summarize"}, Category: CategorySession,
		Title: "Compact", Description: "Compact session history",
		Execute: true, Handler: stubToast("compact"),
	})

	r.Register(Command{
		Name: "connect", Category: CategorySession,
		Title: "Connect", Description: "Connect to remote session",
		Execute: true, Handler: stubToast("connect"),
	})

	r.Register(Command{
		Name: "copy", Category: CategorySession,
		Title: "Copy", Description: "Copy last response to clipboard",
		Execute: true,
		Handler: func(a *App) tea.Cmd {
			entries := a.chat.Entries()
			var last string
			for i := len(entries) - 1; i >= 0; i-- {
				if entries[i].Role == "assistant" {
					last = entries[i].Content
					break
				}
			}
			if last == "" {
				return a.ShowToast("No assistant response to copy", components.ToastWarning)
			}
			if err := clipboard.WriteAll(last); err != nil {
				return a.ShowToast("Copy failed: "+err.Error(), components.ToastError)
			}
			return a.ShowToast("Copied to clipboard", components.ToastSuccess)
		},
	})

	r.Register(Command{
		Name: "details", Category: CategoryView,
		Title: "Details", Description: "Toggle tool detail display",
		Execute: true,
		Handler: func(a *App) tea.Cmd {
			a.showDetails = !a.showDetails
			state := "shown"
			if !a.showDetails {
				state = "hidden"
			}
			return a.ShowToast(fmt.Sprintf("Tool details %s", state), components.ToastInfo)
		},
	})

	r.Register(Command{
		Name: "editor", Category: CategorySession,
		Title: "Editor", Description: "Open in external editor",
		Execute: true, Handler: stubToast("editor"),
	})

	r.Register(Command{
		Name: "export", Category: CategorySession,
		Title: "Export", Description: "Export conversation transcript",
		Execute: true,
		Handler: func(a *App) tea.Cmd {
			entries := a.chat.Entries()
			if len(entries) == 0 {
				return a.ShowToast("Nothing to export", components.ToastWarning)
			}
			a.closeAllDialogs()
			a.exportDialogOpen = true
			a.exportIncludeTools = true
			a.exportIncludeMeta = false
			a.exportFilename = "session-export.md"
			a.exportFocusedField = 0
			return nil
		},
	})

	r.Register(Command{
		Name: "rename", Category: CategorySession,
		Title: "Rename", Description: "Rename current session",
		Keybind: "Ctrl+R",
		Execute: true,
		Handler: func(a *App) tea.Cmd {
			a.closeAllDialogs()
			a.renameDialogOpen = true
			a.renameDialogValue = ""
			if a.session != nil {
				a.renameDialogValue = a.session.Title
			}
			return nil
		},
	})

	r.Register(Command{
		Name: "sessions", Category: CategorySession,
		Title: "Sessions", Description: "List saved sessions",
		Suggested: true, Execute: true,
		Handler: func(a *App) tea.Cmd {
			a.closeAllDialogs()
			a.sessionsDialogOpen = true
			a.sessionsDialogQuery = ""
			a.sessionsDialogSelect = 0
			a.sessionsDialogConfirm = false
			if !a.sessionsDialogLoaded {
				return a.loadSessions()
			}
			return nil
		},
	})

	r.Register(Command{
		Name: "skills", Category: CategorySession,
		Title: "Skills", Description: "List available skills",
		Execute: true, Handler: stubToast("skills"),
	})

	r.Register(Command{
		Name: "status", Category: CategorySystem,
		Title: "Status", Description: "View system status",
		Execute: true,
		Handler: func(a *App) tea.Cmd {
			a.closeAllDialogs()
			a.statusDialogOpen = true
			return nil
		},
	})

	r.Register(Command{
		Name: "themes", Category: CategoryView,
		Title: "Themes", Description: "Switch color theme",
		Execute: true, Handler: stubToast("themes"),
	})

	r.Register(Command{
		Name: "thinking", Aliases: []string{"toggle-thinking"}, Category: CategoryView,
		Title: "Thinking", Description: "Toggle thinking block display",
		Execute: true,
		Handler: func(a *App) tea.Cmd {
			a.showThinking = !a.showThinking
			state := "shown"
			if !a.showThinking {
				state = "hidden"
			}
			return a.ShowToast(fmt.Sprintf("Thinking blocks %s", state), components.ToastInfo)
		},
	})

	r.Register(Command{
		Name: "timeline", Category: CategorySession,
		Title: "Timeline", Description: "View session timeline",
		Execute: true,
		Handler: func(a *App) tea.Cmd {
			a.closeAllDialogs()
			a.timelineDialogOpen = true
			a.timelineDialogQuery = ""
			a.timelineDialogSelect = 0
			return nil
		},
	})

	r.Register(Command{
		Name: "timestamps", Aliases: []string{"toggle-timestamps"}, Category: CategoryView,
		Title: "Timestamps", Description: "Toggle message timestamps",
		Execute: true,
		Handler: func(a *App) tea.Cmd {
			a.showTimestamps = !a.showTimestamps
			state := "shown"
			if !a.showTimestamps {
				state = "hidden"
			}
			return a.ShowToast(fmt.Sprintf("Timestamps %s", state), components.ToastInfo)
		},
	})

	r.Register(Command{
		Name: "undo", Category: CategorySession,
		Title: "Undo", Description: "Undo last exchange",
		Execute: true,
		Handler: func(a *App) tea.Cmd {
			if a.session == nil || !a.session.CanUndo() {
				return a.ShowToast("Nothing to undo", components.ToastWarning)
			}
			prompt, err := a.session.Revert()
			if err != nil {
				return a.ShowToast("Nothing to undo", components.ToastWarning)
			}
			a.rebuildChatFromSession()
			a.input.SetValue(prompt)
			return a.ShowToast("Reverted last exchange", components.ToastInfo)
		},
	})

	r.Register(Command{
		Name: "redo", Category: CategorySession,
		Title: "Redo", Description: "Redo last undone exchange",
		Execute: true,
		Handler: func(a *App) tea.Cmd {
			if a.session == nil || !a.session.CanRedo() {
				return a.ShowToast("Nothing to redo", components.ToastWarning)
			}
			if err := a.session.Unrevert(); err != nil {
				return a.ShowToast("Nothing to redo", components.ToastWarning)
			}
			a.rebuildChatFromSession()
			return a.ShowToast("Restored exchange", components.ToastInfo)
		},
	})

	r.Register(Command{
		Name: "stash", Category: CategorySession,
		Title: "Stash", Description: "Save current input draft",
		Execute: true,
		Handler: func(a *App) tea.Cmd {
			content := a.input.Value()
			// Also check stashPendingContent for when invoked via slash command
			if a.stashPendingContent != "" {
				content = a.stashPendingContent
				a.stashPendingContent = ""
			}
			if strings.TrimSpace(content) == "" {
				return a.ShowToast("Nothing to stash", components.ToastWarning)
			}
			a.stash.Push(content)
			a.input.Reset()
			return a.ShowToast("Stashed draft", components.ToastSuccess)
		},
	})

	r.Register(Command{
		Name: "stash-pop", Category: CategorySession,
		Title: "Stash Pop", Description: "Restore last stashed draft",
		Execute: true,
		Handler: func(a *App) tea.Cmd {
			entry, ok := a.stash.Pop()
			if !ok {
				return a.ShowToast("Stash is empty", components.ToastWarning)
			}
			a.input.SetValue(entry.Content)
			return a.ShowToast("Restored from stash", components.ToastSuccess)
		},
	})

	r.Register(Command{
		Name: "stash-list", Category: CategorySession,
		Title: "Stash List", Description: "View stashed drafts",
		Execute: true,
		Handler: func(a *App) tea.Cmd {
			if len(a.stash.List()) == 0 {
				return a.ShowToast("Stash is empty", components.ToastWarning)
			}
			a.closeAllDialogs()
			a.stashDialogOpen = true
			a.stashDialogSelect = 0
			return nil
		},
	})

	a.cmdRegistry = r
}

// SetProvider configures the LLM provider.
func (a *App) SetProvider(p provider.Provider) {
	a.provider = p
	a.footer.SetConnected(p != nil)
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
		a.footer.SetWorkDir(s.WorkDir)
		a.statusbar.SetTitle(shortSessionTitle(s.ID))
	}
}

// SetAgent configures the agent.
func (a *App) SetAgent(ag agent.Agent) {
	a.agent = ag
	a.input.SetMode(ag.Name)
	a.chat.SetMode(ag.Name)
	a.statusbar.SetMode(ag.Name)
	a.home.SetMode(ag.Name)
	if a.session != nil {
		a.session.SetSystemPrompt(ag.SystemPrompt)
	}
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

// ShowToast displays a toast notification and returns a dismiss command.
func (a *App) ShowToast(msg string, variant components.ToastVariant) tea.Cmd {
	id := a.toasts.Show(msg, variant, 3*time.Second)
	return func() tea.Msg {
		time.Sleep(3 * time.Second)
		return ToastDismissMsg{ID: id}
	}
}

// setStreaming updates the streaming state and keeps statusbar/footer in sync.
func (a *App) setStreaming(streaming bool) {
	a.streaming = streaming
	a.statusbar.SetSpinning(streaming)
	a.footer.SetStreaming(streaming)
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

	case FileCacheLoadedMsg:
		a.fileCache = msg.Files
		a.fileCacheLoaded = true
		a.fileCacheLoading = false
		return a, nil

	case SessionsLoadedMsg:
		a.sessionsDialogEntries = msg.Sessions
		a.sessionsDialogLoaded = true
		return a, nil

	case ToastDismissMsg:
		a.toasts.Dismiss(msg.ID)
		return a, nil

	case ShellResultMsg:
		if msg.Error != "" {
			a.chat.AddEntry(components.ChatEntry{Role: "error", Content: msg.Error})
			toastCmd := a.ShowToast("Command failed", components.ToastError)
			return a, toastCmd
		}
		output := msg.Output
		if len(output) > 2000 {
			output = output[:2000] + "\n... (truncated)"
		}
		if output != "" {
			a.chat.AddEntry(components.ChatEntry{Role: "system", Content: output})
		}
		return a, nil

	case AgentEventMsg:
		return a.handleAgentEvent(msg.Event)

	case tea.MouseClickMsg:
		if a.sidebarOverlayActive() {
			m := msg.Mouse()
			x, y, w, h := a.sidebarOverlayPanelRect()
			inside := m.X >= x && m.X < x+w && m.Y >= y && m.Y < y+h
			if !inside {
				a.sidebarOverlay = false
				if !a.streaming {
					a.input.Focus()
				}
			}
			return a, nil
		}
		return a, nil

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
		if a.state == StateSession && !a.streaming && a.input.Value() != "" {
			a.input.Reset()
			return a, nil
		}
		return a, tea.Quit

	case a.helpDialogOpen:
		return a.handleHelpDialogKey(msg)

	case a.statusDialogOpen:
		return a.handleStatusDialogKey(msg)

	case a.stashDialogOpen:
		return a.handleStashDialogKey(msg)

	case a.timelineDialogOpen:
		return a.handleTimelineDialogKey(msg)

	case a.sessionsDialogOpen:
		return a.handleSessionsDialogKey(msg)

	case a.renameDialogOpen:
		return a.handleRenameDialogKey(msg)

	case a.exportDialogOpen:
		return a.handleExportDialogKey(msg)

	case a.modelDialogOpen:
		return a.handleModelDialogKey(msg)

	case a.commandOpen:
		return a.handleCommandPaletteKey(msg)

	case a.inlineOpen:
		return a.handleInlineKey(msg)

	case a.sidebarOverlayActive():
		return a.handleSidebarOverlayKey(msg)

	case msg.Code == tea.KeyEscape:
		if a.streaming {
			if a.cancel != nil {
				a.cancel()
				a.cancel = nil
			}
			a.eventCh = nil
			a.setStreaming(false)
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

	case msg.Mod == tea.ModCtrl && msg.Code == 'b':
		if a.state == StateSession {
			return a.toggleSidebar(), nil
		}
		return a, nil

	case msg.Mod == tea.ModCtrl && msg.Code == 'r':
		if !a.streaming && a.state == StateSession {
			if cmd := a.cmdRegistry.Get("rename"); cmd != nil && cmd.Handler != nil {
				return a, cmd.Handler(&a)
			}
		}
		return a, nil

	case msg.Mod == tea.ModCtrl && (msg.Code == 'k' || msg.Code == 'p'):
		if !a.streaming && a.state == StateSession {
			a.commandOpen = true
			a.commandQuery = ""
			a.commandSelect = 0
			a.modelDialogOpen = false
			a.inlineOpen = false
			a.input.Blur()
		}
		return a, nil

	case msg.Code == tea.KeyTab:
		if !a.streaming && a.state == StateSession {
			return a.cycleAgent(msg.Mod&tea.ModShift != 0), nil
		}
		return a, nil

	// Message navigation keybinds
	case msg.Code == tea.KeyPgUp:
		if a.state == StateSession {
			a.chat.PageUp()
		}
		return a, nil

	case msg.Code == tea.KeyPgDown:
		if a.state == StateSession {
			a.chat.PageDown()
		}
		return a, nil

	case msg.Mod == tea.ModCtrl|tea.ModAlt && msg.Code == 'u':
		if a.state == StateSession {
			a.chat.HalfPageUp()
		}
		return a, nil

	case msg.Mod == tea.ModCtrl|tea.ModAlt && msg.Code == 'd':
		if a.state == StateSession {
			a.chat.HalfPageDown()
		}
		return a, nil

	case msg.Mod == tea.ModCtrl|tea.ModAlt && msg.Code == 'y':
		if a.state == StateSession {
			a.chat.LineUp()
		}
		return a, nil

	case msg.Mod == tea.ModCtrl|tea.ModAlt && msg.Code == 'e':
		if a.state == StateSession {
			a.chat.LineDown()
		}
		return a, nil

	case msg.Mod == tea.ModCtrl && msg.Code == 'g':
		if a.state == StateSession {
			a.chat.ScrollToTop()
		}
		return a, nil

	case msg.Mod == tea.ModCtrl|tea.ModAlt && msg.Code == 'g':
		if a.state == StateSession {
			a.chat.ScrollToBottom()
		}
		return a, nil

	default:
		if !a.streaming {
			if a.state == StateHome {
				cmd := a.home.Input().Update(msg)
				return a, cmd
			}

			// Prompt history: Up at first line, Down at last line
			if msg.Code == tea.KeyUp && msg.Mod == 0 && a.input.CursorY() == 0 {
				if a.promptHistory.AtNewest() {
					a.promptHistory.SaveDraft(a.input.Value())
				}
				if p, ok := a.promptHistory.Previous(); ok {
					a.input.SetValue(p)
				}
				return a, nil
			}
			if msg.Code == tea.KeyDown && msg.Mod == 0 && a.input.CursorY() == a.input.LineCount()-1 {
				if p, ok := a.promptHistory.Next(); ok {
					a.input.SetValue(p)
				}
				return a, nil
			}

			cmd := a.input.Update(msg)

			// Detect shell mode based on leading "!"
			val := a.input.Value()
			if strings.HasPrefix(val, "!") {
				if !a.shellMode {
					a.shellMode = true
					a.input.SetShellMode(true)
				}
			} else if a.shellMode {
				a.shellMode = false
				a.input.SetShellMode(false)
			}

			cacheCmd := a.updateInlineSuggestions()
			return a, tea.Batch(cmd, cacheCmd)
		}
	}

	return a, nil
}

func (a App) handleSubmit(input string) (tea.Model, tea.Cmd) {
	// Transition from home to session on first submit
	if a.state == StateHome {
		a.state = StateSession
		a.commandOpen = false
		a.modelDialogOpen = false
		a.sidebarOverlay = false
	}
	a.inlineOpen = false

	a.promptHistory.Push(input)

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

// switchModel attempts to switch to the given model ID via the provider.
// Returns true on success, false on error. Adds appropriate chat entries.
func (a *App) switchModel(modelID string) bool {
	if a.provider == nil {
		a.chat.AddEntry(components.ChatEntry{
			Role:    "error",
			Content: "Not configured — missing provider",
		})
		return false
	}

	setter, ok := a.provider.(provider.ModelSetter)
	if !ok {
		a.chat.AddEntry(components.ChatEntry{
			Role:    "error",
			Content: fmt.Sprintf(`Provider "%s" does not support runtime model switching.`, a.provider.Name()),
		})
		return false
	}

	setter.SetModel(modelID)
	a.SetModel(displayModelName(modelID))
	a.chat.AddEntry(components.ChatEntry{
		Role:    "system",
		Content: fmt.Sprintf(`Model switched to "%s".`, modelID),
	})
	return true
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

func displayModelName(model string) string {
	parts := strings.Split(model, "/")
	if len(parts) > 1 {
		return parts[len(parts)-1]
	}
	return model
}

func shortSessionTitle(id string) string {
	trim := strings.TrimSpace(id)
	if trim == "" {
		return "Session"
	}
	if len(trim) > 14 {
		return "Session " + trim[:14]
	}
	return "Session " + trim
}

func repeatStr(ch string, n int) string {
	if n <= 0 {
		return ""
	}
	return strings.Repeat(ch, n)
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func (a App) handleModelsCommand(input string) (tea.Model, tea.Cmd) {
	if a.provider == nil {
		a.chat.AddEntry(components.ChatEntry{
			Role:    "error",
			Content: "Not configured — missing provider",
		})
		return a, nil
	}

	query := parseModelsQuery(input)

	// Cache hit — open dialog synchronously, no HTTP call.
	if a.modelsLoaded {
		a = a.openModelDialog(query)
		return a, nil
	}

	// Cache miss — show spinner, fetch async.
	a.statusbar.SetSpinning(true)
	a.input.Blur()
	return a, loadModelsCmd(a.provider, query)
}

func (a App) handleModelsLoaded(msg ModelsLoadedMsg) (tea.Model, tea.Cmd) {
	a.statusbar.SetSpinning(false)

	if msg.Err != nil {
		a.chat.AddEntry(components.ChatEntry{
			Role:    "error",
			Content: msg.Err.Error(),
		})
		a.input.Focus()
		return a, nil
	}

	a.modelsCache = msg.Models
	a.modelsLoaded = true
	a = a.openModelDialog(msg.Query)
	return a, nil
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

// paletteEntries returns commands in the order displayed in the command palette.
// When filtering, this is the flat filtered list. When unfiltered, this is
// suggested commands first, then remaining commands grouped by category order.
func (a App) paletteEntries() []*Command {
	q := strings.TrimSpace(a.commandQuery)
	if q != "" {
		return a.cmdRegistry.Filter(q)
	}

	suggested := a.cmdRegistry.Suggested()
	suggestedSet := make(map[string]bool, len(suggested))
	for _, s := range suggested {
		suggestedSet[s.Name] = true
	}

	out := make([]*Command, 0, len(a.cmdRegistry.commands))
	out = append(out, suggested...)

	byCategory := a.cmdRegistry.ByCategory()
	for _, cat := range categoryOrder {
		for _, cmd := range byCategory[cat] {
			if !suggestedSet[cmd.Name] {
				out = append(out, cmd)
			}
		}
	}
	return out
}

func (a *App) requestFileCache() tea.Cmd {
	if a.fileCacheLoaded || a.fileCacheLoading || a.session == nil {
		return nil
	}
	a.fileCacheLoading = true
	root := a.session.WorkDir
	return loadFileCacheCmd(root)
}

func loadFileCacheCmd(root string) tea.Cmd {
	return func() tea.Msg {
		const maxFiles = 8000
		files := make([]string, 0, 2048)
		_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if d.IsDir() {
				switch d.Name() {
				case ".git", "node_modules", ".next", ".turbo":
					return filepath.SkipDir
				}
				return nil
			}
			// Skip symlinks to prevent path traversal.
			if d.Type()&os.ModeSymlink != 0 {
				return nil
			}

			rel, err := filepath.Rel(root, path)
			if err != nil {
				return nil
			}
			files = append(files, filepath.ToSlash(rel))
			if len(files) >= maxFiles {
				return fs.SkipAll
			}
			return nil
		})

		sort.Strings(files)
		return FileCacheLoadedMsg{Files: files}
	}
}

func containsWhitespace(s string) bool {
	for _, r := range s {
		if r == ' ' || r == '\t' || r == '\n' || r == '\r' {
			return true
		}
	}
	return false
}

func (a App) closeInlineSuggestions() App {
	a.inlineOpen = false
	a.inlineMode = ""
	a.inlineQuery = ""
	a.inlineSelect = 0
	a.inlineStart = 0
	a.inlineEnd = 0
	return a
}

func (a App) inlineEntries() []inlineEntry {
	if !a.inlineOpen {
		return nil
	}

	query := strings.ToLower(strings.TrimSpace(a.inlineQuery))
	if a.inlineMode == "/" {
		var commands []*Command
		if query == "" {
			commands = a.cmdRegistry.All()
		} else {
			commands = a.cmdRegistry.Filter(query)
		}
		out := make([]inlineEntry, 0, len(commands))
		for _, cmd := range commands {
			out = append(out, inlineEntry{
				Display:     "/" + cmd.Name,
				Insert:      "/" + cmd.Name,
				Description: cmd.Description,
				Execute:     cmd.Execute,
			})
		}
		return out
	}

	if a.inlineMode == "@" {
		out := make([]inlineEntry, 0, 10)
		for _, path := range a.fileCache {
			p := strings.ToLower(path)
			if query != "" && !strings.Contains(p, query) {
				continue
			}
			out = append(out, inlineEntry{
				Display: path,
				Insert:  "@" + path + " ",
			})
			if len(out) >= 10 {
				break
			}
		}
		return out
	}

	return nil
}

func (a *App) updateInlineSuggestions() tea.Cmd {
	if a.state != StateSession || a.streaming || a.commandOpen || a.modelDialogOpen {
		*a = a.closeInlineSuggestions()
		return nil
	}

	text := a.input.Value()
	runes := []rune(text)
	cursor := a.input.CursorOffset()
	if cursor < 0 {
		cursor = 0
	}
	if cursor > len(runes) {
		cursor = len(runes)
	}
	if len(runes) == 0 {
		*a = a.closeInlineSuggestions()
		return nil
	}

	prefix := string(runes[:cursor])
	if strings.HasPrefix(prefix, "/") && !containsWhitespace(prefix) {
		a.inlineOpen = true
		a.inlineMode = "/"
		a.inlineQuery = strings.TrimPrefix(prefix, "/")
		a.inlineStart = 0
		a.inlineEnd = cursor
		a.inlineSelect = 0
		return nil
	}

	idx := strings.LastIndex(prefix, "@")
	if idx != -1 {
		beforeOK := idx == 0 || containsWhitespace(prefix[idx-1:idx])
		between := prefix[idx+1:]
		if beforeOK && !containsWhitespace(between) {
			a.inlineOpen = true
			a.inlineMode = "@"
			a.inlineQuery = between
			a.inlineStart = idx
			a.inlineEnd = cursor
			a.inlineSelect = 0
			return a.requestFileCache()
		}
	}

	*a = a.closeInlineSuggestions()
	return nil
}

func (a App) handleInlineKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case msg.Code == tea.KeyEscape:
		return a.closeInlineSuggestions(), nil

	case msg.Code == tea.KeyUp || (msg.Mod == tea.ModCtrl && msg.Code == 'p'):
		entries := a.inlineEntries()
		if len(entries) == 0 {
			return a, nil
		}
		a.inlineSelect--
		if a.inlineSelect < 0 {
			a.inlineSelect = len(entries) - 1
		}
		return a, nil

	case msg.Code == tea.KeyDown || (msg.Mod == tea.ModCtrl && msg.Code == 'n'):
		entries := a.inlineEntries()
		if len(entries) == 0 {
			return a, nil
		}
		a.inlineSelect++
		if a.inlineSelect >= len(entries) {
			a.inlineSelect = 0
		}
		return a, nil

	case msg.Code == tea.KeyEnter || msg.Code == tea.KeyTab:
		entries := a.inlineEntries()
		if len(entries) == 0 {
			return a.closeInlineSuggestions(), nil
		}
		if a.inlineSelect < 0 {
			a.inlineSelect = 0
		}
		if a.inlineSelect >= len(entries) {
			a.inlineSelect = len(entries) - 1
		}

		choice := entries[a.inlineSelect]
		if a.inlineMode == "/" && choice.Execute {
			a.input.Reset()
			a = a.closeInlineSuggestions()
			return a.handleSubmit(choice.Insert)
		}

		a.input.ReplaceRange(a.inlineStart, a.inlineEnd, choice.Insert)
		a = a.closeInlineSuggestions()
		cacheCmd := a.updateInlineSuggestions()
		return a, cacheCmd

	default:
		cmd := a.input.Update(msg)
		cacheCmd := a.updateInlineSuggestions()
		return a, tea.Batch(cmd, cacheCmd)
	}
}

func (a App) renderInlineSuggestions() string {
	entries := a.inlineEntries()
	query := strings.TrimSpace(a.inlineQuery)
	if query == "" {
		query = "all"
	}
	header := "Autocomplete " + a.inlineMode + " (Enter select, Esc close) - filter: " + query

	items := make([]pickerItem, len(entries))
	for i, e := range entries {
		items[i] = pickerItem{Label: e.Display, Description: e.Description}
	}
	return renderPickerList(header, items, a.inlineSelect, 8)
}

func (a App) handleSidebarOverlayKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case msg.Code == tea.KeyEscape:
		a.sidebarOverlay = false
		if !a.streaming {
			a.input.Focus()
		}
		return a, nil
	case msg.Mod == tea.ModCtrl && msg.Code == 'b':
		a.sidebarOverlay = false
		if !a.streaming {
			a.input.Focus()
		}
		return a, nil
	default:
		return a, nil
	}
}

func (a App) toggleSidebar() App {
	if a.width >= 120 {
		a.sidebarHidden = !a.sidebarHidden
		if a.sidebarHidden {
			a.sidebarOverlay = false
		}
		return a
	}

	if a.sidebarOverlayActive() {
		a.sidebarOverlay = false
		if !a.streaming {
			a.input.Focus()
		}
		return a
	}

	a.sidebarHidden = false
	a.sidebarOverlay = true
	a.commandOpen = false
	a.inlineOpen = false
	a.modelDialogOpen = false
	a.input.Blur()
	return a
}

func (a App) openModelDialog(query string) App {
	a.modelDialogOpen = true
	a.modelDialogQuery = strings.TrimSpace(query)
	a.modelDialogSelect = 0
	a.commandOpen = false
	a.inlineOpen = false
	a.input.Blur()
	return a
}

func (a App) closeModelDialog() App {
	a.modelDialogOpen = false
	a.modelDialogQuery = ""
	a.modelDialogSelect = 0
	a.input.Focus()
	return a
}

func (a App) filteredModelsDialog() []provider.ModelInfo {
	return filterModels(a.modelsCache, strings.TrimSpace(a.modelDialogQuery))
}

func (a App) handleModelDialogKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case msg.Code == tea.KeyEscape:
		return a.closeModelDialog(), nil

	case msg.Code == tea.KeyEnter:
		models := a.filteredModelsDialog()
		if len(models) == 0 {
			return a.closeModelDialog(), nil
		}

		if a.modelDialogSelect < 0 {
			a.modelDialogSelect = 0
		}
		if a.modelDialogSelect >= len(models) {
			a.modelDialogSelect = len(models) - 1
		}

		selected := models[a.modelDialogSelect]
		a.switchModel(selected.ID)
		return a.closeModelDialog(), nil

	case msg.Code == tea.KeyUp:
		models := a.filteredModelsDialog()
		if len(models) == 0 {
			return a, nil
		}
		a.modelDialogSelect--
		if a.modelDialogSelect < 0 {
			a.modelDialogSelect = len(models) - 1
		}
		return a, nil

	case msg.Code == tea.KeyDown:
		models := a.filteredModelsDialog()
		if len(models) == 0 {
			return a, nil
		}
		a.modelDialogSelect++
		if a.modelDialogSelect >= len(models) {
			a.modelDialogSelect = 0
		}
		return a, nil

	case msg.Code == tea.KeyBackspace:
		if a.modelDialogQuery == "" {
			return a, nil
		}
		runes := []rune(a.modelDialogQuery)
		a.modelDialogQuery = string(runes[:len(runes)-1])
		a.modelDialogSelect = 0
		return a, nil

	default:
		if msg.Text != "" {
			a.modelDialogQuery += msg.Text
			a.modelDialogSelect = 0
		}
		return a, nil
	}
}

func (a App) renderModelDialog() string {
	models := a.filteredModelsDialog()
	query := strings.TrimSpace(a.modelDialogQuery)
	if query == "" {
		query = "all"
	}
	header := "Select model (type to filter, Enter choose, Esc close) - filter: " + query

	items := make([]pickerItem, len(models))
	for i, m := range models {
		label := modelLine(m)
		if displayModelName(m.ID) == a.model || m.ID == a.model {
			label = "● " + label
		}
		items[i] = pickerItem{Label: label}
	}
	return renderPickerList(header, items, a.modelDialogSelect, 9)
}

func (a App) cycleAgent(reverse bool) App {
	current := strings.ToLower(strings.TrimSpace(a.agent.Name))
	switch {
	case reverse:
		if current == agent.NamePlan {
			a.SetAgent(agent.BuildAgent())
			return a
		}
		a.SetAgent(agent.PlanAgent())
		return a
	default:
		if current == agent.NameBuild {
			a.SetAgent(agent.PlanAgent())
			return a
		}
		a.SetAgent(agent.BuildAgent())
		return a
	}
}

func (a App) handleCommandPaletteKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case msg.Code == tea.KeyEscape:
		a.commandOpen = false
		a.commandQuery = ""
		a.commandSelect = 0
		a.input.Focus()
		return a, nil

	case msg.Code == tea.KeyEnter:
		entries := a.paletteEntries()
		if len(entries) == 0 {
			a.commandOpen = false
			a.commandQuery = ""
			a.commandSelect = 0
			a.input.Focus()
			return a, nil
		}
		if a.commandSelect >= len(entries) {
			a.commandSelect = len(entries) - 1
		}
		if a.commandSelect < 0 {
			a.commandSelect = 0
		}
		entry := entries[a.commandSelect]
		a.commandOpen = false
		a.commandQuery = ""
		a.commandSelect = 0
		a.input.Focus()
		if entry.Execute {
			return a.handleSubmit("/" + entry.Name)
		}
		a.input.SetValue("/" + entry.Name + " ")
		cacheCmd := a.updateInlineSuggestions()
		return a, cacheCmd

	case msg.Code == tea.KeyUp:
		entries := a.paletteEntries()
		if len(entries) == 0 {
			return a, nil
		}
		a.commandSelect--
		if a.commandSelect < 0 {
			a.commandSelect = len(entries) - 1
		}
		return a, nil

	case msg.Code == tea.KeyDown:
		entries := a.paletteEntries()
		if len(entries) == 0 {
			return a, nil
		}
		a.commandSelect++
		if a.commandSelect >= len(entries) {
			a.commandSelect = 0
		}
		return a, nil

	case msg.Code == tea.KeyBackspace:
		if a.commandQuery == "" {
			return a, nil
		}
		runes := []rune(a.commandQuery)
		a.commandQuery = string(runes[:len(runes)-1])
		a.commandSelect = 0
		return a, nil

	default:
		if msg.Text != "" {
			a.commandQuery += msg.Text
			a.commandSelect = 0
		}
		return a, nil
	}
}

func (a App) renderCommandPalette() string {
	entries := a.paletteEntries()
	query := strings.TrimSpace(a.commandQuery)

	var sb strings.Builder
	sb.WriteString("Commands (Ctrl+P/Ctrl+K, Esc close)")
	if query != "" {
		sb.WriteString(" - filter: ")
		sb.WriteString(query)
	}

	if len(entries) == 0 {
		sb.WriteString("\n  no matches")
		return sb.String()
	}

	// When filtering, show a flat list (no categories).
	if query != "" {
		for i, e := range entries {
			a.writePaletteEntry(&sb, e, i)
		}
		return sb.String()
	}

	// Unfiltered: show Suggested, then categories with headers.
	numSuggested := len(a.cmdRegistry.Suggested())
	lastCategory := CommandCategory("")

	for i, e := range entries {
		if i < numSuggested {
			if i == 0 {
				sb.WriteString("\n\n  Suggested")
			}
		} else if e.Category != lastCategory {
			lastCategory = e.Category
			sb.WriteString("\n\n  ")
			sb.WriteString(string(e.Category))
		}
		a.writePaletteEntry(&sb, e, i)
	}

	return sb.String()
}

// writePaletteEntry writes a single palette row with selection marker and optional keybind.
func (a App) writePaletteEntry(sb *strings.Builder, e *Command, idx int) {
	prefix := "  "
	if idx == a.commandSelect {
		prefix = "> "
	}
	sb.WriteString("\n")
	sb.WriteString(prefix)
	sb.WriteString(fmt.Sprintf("%-20s %s", "/"+e.Name, e.Description))
	if e.Keybind != "" {
		sb.WriteString("  [")
		sb.WriteString(e.Keybind)
		sb.WriteString("]")
	}
}

// --- Help dialog ---

func (a App) handleHelpDialogKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case msg.Code == tea.KeyEscape:
		a.helpDialogOpen = false
		a.input.Focus()
		return a, nil

	case msg.Code == tea.KeyEnter:
		a.helpDialogOpen = false
		a.input.Focus()
		return a, nil

	case msg.Code == tea.KeyTab:
		a.helpDialogTab = (a.helpDialogTab + 1) % 2
		a.helpDialogSelect = 0
		return a, nil

	case msg.Code == tea.KeyUp:
		if a.helpDialogSelect > 0 {
			a.helpDialogSelect--
		}
		return a, nil

	case msg.Code == tea.KeyDown:
		a.helpDialogSelect++
		return a, nil

	case msg.Code == tea.KeyBackspace:
		if a.helpDialogQuery != "" {
			runes := []rune(a.helpDialogQuery)
			a.helpDialogQuery = string(runes[:len(runes)-1])
			a.helpDialogSelect = 0
		}
		return a, nil

	default:
		if msg.Text != "" {
			a.helpDialogQuery += msg.Text
			a.helpDialogSelect = 0
		}
		return a, nil
	}
}

var helpKeybinds = []struct {
	Key  string
	Desc string
}{
	{"Ctrl+A", "Move to line start"},
	{"Ctrl+E", "Move to line end"},
	{"Ctrl+U", "Delete to line start"},
	{"Ctrl+K", "Delete to line end / Command palette"},
	{"Ctrl+W", "Delete word left"},
	{"Alt+D", "Delete word right"},
	{"Ctrl+D", "Delete char right"},
	{"Alt+B / Ctrl+Left", "Move word left"},
	{"Alt+F / Ctrl+Right", "Move word right"},
	{"Ctrl+P / Ctrl+K", "Command palette"},
	{"Ctrl+B", "Toggle sidebar"},
	{"Tab / Shift+Tab", "Cycle agent mode"},
	{"PageUp / PageDown", "Scroll chat"},
	{"Ctrl+G", "Scroll to top"},
	{"Ctrl+Alt+G", "Scroll to bottom"},
	{"Ctrl+Alt+U / D", "Half page up/down"},
	{"Ctrl+Alt+Y / E", "Line up/down"},
	{"Up / Down", "History navigation"},
	{"Esc", "Cancel / Quit"},
}

func (a App) renderHelpDialog() string {
	var sb strings.Builder
	query := strings.TrimSpace(a.helpDialogQuery)

	tabs := []string{"Commands", "Keybinds"}
	for i, t := range tabs {
		if i == a.helpDialogTab {
			sb.WriteString("[" + t + "]")
		} else {
			sb.WriteString(" " + t + " ")
		}
		sb.WriteString("  ")
	}
	sb.WriteString("(Tab switch, Esc close)")
	if query != "" {
		sb.WriteString(" filter: " + query)
	}

	q := strings.ToLower(query)

	if a.helpDialogTab == 0 {
		cmds := a.cmdRegistry.All()
		idx := 0
		for _, c := range cmds {
			if q != "" {
				haystack := strings.ToLower(c.Name + " " + c.Description)
				if !strings.Contains(haystack, q) {
					continue
				}
			}
			prefix := "  "
			if idx == a.helpDialogSelect {
				prefix = "> "
			}
			line := fmt.Sprintf("%-16s %s", "/"+c.Name, c.Description)
			if c.Keybind != "" {
				line += "  [" + c.Keybind + "]"
			}
			sb.WriteString("\n" + prefix + line)
			idx++
		}
		if idx == 0 {
			sb.WriteString("\n  no matches")
		}
	} else {
		idx := 0
		for _, kb := range helpKeybinds {
			if q != "" {
				haystack := strings.ToLower(kb.Key + " " + kb.Desc)
				if !strings.Contains(haystack, q) {
					continue
				}
			}
			prefix := "  "
			if idx == a.helpDialogSelect {
				prefix = "> "
			}
			sb.WriteString(fmt.Sprintf("\n%s%-24s %s", prefix, kb.Key, kb.Desc))
			idx++
		}
		if idx == 0 {
			sb.WriteString("\n  no matches")
		}
	}

	return sb.String()
}

// --- Status dialog ---

func (a App) handleStatusDialogKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case msg.Code == tea.KeyEscape, msg.Code == tea.KeyEnter:
		a.statusDialogOpen = false
		a.input.Focus()
		return a, nil
	default:
		return a, nil
	}
}

func formatNumber(n int) string {
	if n < 1000 {
		return fmt.Sprintf("%d", n)
	}
	if n < 1_000_000 {
		thousands := n / 1000
		remainder := n % 1000
		return fmt.Sprintf("%d,%03d", thousands, remainder)
	}
	millions := n / 1_000_000
	remainder := (n % 1_000_000) / 1000
	return fmt.Sprintf("%d,%03d,%03d", millions, remainder, n%1000)
}

func (a App) renderStatusDialog() string {
	var sb strings.Builder
	sb.WriteString("Status                                    esc\n")

	modelName := a.model
	if modelName == "" {
		modelName = "(not set)"
	}
	agentName := a.agent.Name
	if agentName == "" {
		agentName = "build"
	}
	providerName := "(none)"
	if a.provider != nil {
		providerName = a.provider.Name()
	}

	sb.WriteString("\nSystem")
	sb.WriteString(fmt.Sprintf("\n  Model       %s", modelName))
	sb.WriteString(fmt.Sprintf("\n  Agent       %s", agentName))
	sb.WriteString(fmt.Sprintf("\n  Provider    %s", providerName))

	sb.WriteString("\n\nSession")
	msgCount := 0
	tokIn := 0
	tokOut := 0
	workDir := "(not set)"
	if a.session != nil {
		msgCount = len(a.session.Messages)
		tokIn = a.session.Tokens.Input
		tokOut = a.session.Tokens.Output
		workDir = a.session.WorkDir
	}
	sb.WriteString(fmt.Sprintf("\n  Messages    %d messages", msgCount))
	sb.WriteString(fmt.Sprintf("\n  Tokens      %s in / %s out", formatNumber(tokIn), formatNumber(tokOut)))
	sb.WriteString(fmt.Sprintf("\n  WorkDir     %s", workDir))

	return sb.String()
}

// --- Export dialog ---

func (a App) handleExportDialogKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case msg.Code == tea.KeyEscape:
		a.exportDialogOpen = false
		a.input.Focus()
		return a, nil

	case msg.Code == tea.KeyEnter:
		a.exportDialogOpen = false
		a.input.Focus()
		return a, a.executeExport()

	case msg.Code == tea.KeyUp:
		if a.exportFocusedField > 0 {
			a.exportFocusedField--
		}
		return a, nil

	case msg.Code == tea.KeyDown:
		if a.exportFocusedField < 1 {
			a.exportFocusedField++
		}
		return a, nil

	case msg.Text == " ":
		switch a.exportFocusedField {
		case 0:
			a.exportIncludeTools = !a.exportIncludeTools
		case 1:
			a.exportIncludeMeta = !a.exportIncludeMeta
		}
		return a, nil

	default:
		return a, nil
	}
}

func (a *App) executeExport() tea.Cmd {
	entries := a.chat.Entries()
	var sb strings.Builder

	sb.WriteString("# Session Export\n\n")
	for _, e := range entries {
		switch e.Role {
		case "user":
			sb.WriteString("## User\n\n")
			sb.WriteString(e.Content + "\n\n")
		case "assistant":
			sb.WriteString("## Assistant\n\n")
			sb.WriteString(e.Content + "\n\n")
		case "tool":
			if a.exportIncludeTools {
				sb.WriteString(fmt.Sprintf("**Tool: %s** (%s)\n", e.ToolName, e.ToolStatus))
				sb.WriteString(e.Content + "\n\n")
			}
		case "complete":
			if a.exportIncludeMeta && e.Meta != nil {
				sb.WriteString(fmt.Sprintf("*%s · %s · %.1fs*\n\n",
					e.Meta.Mode, e.Meta.Model, e.Meta.Duration.Seconds()))
			}
		case "error":
			sb.WriteString("**Error:** " + e.Content + "\n\n")
		case "system":
			sb.WriteString("*" + e.Content + "*\n\n")
		}
	}

	workDir := "."
	if a.session != nil {
		workDir = a.session.WorkDir
	}
	exportPath := filepath.Join(workDir, a.exportFilename)
	if err := os.WriteFile(exportPath, []byte(sb.String()), 0o644); err != nil {
		return a.ShowToast("Export failed: "+err.Error(), components.ToastError)
	}
	return a.ShowToast("Exported to "+exportPath, components.ToastSuccess)
}

func (a App) renderExportDialog() string {
	var sb strings.Builder
	sb.WriteString("Export transcript                          esc\n")

	check := func(v bool) string {
		if v {
			return "[x]"
		}
		return "[ ]"
	}
	marker := func(idx int) string {
		if idx == a.exportFocusedField {
			return "> "
		}
		return "  "
	}

	sb.WriteString(fmt.Sprintf("\n%s%s Include tool calls", marker(0), check(a.exportIncludeTools)))
	sb.WriteString(fmt.Sprintf("\n%s%s Include metadata (model, tokens, duration)", marker(1), check(a.exportIncludeMeta)))
	sb.WriteString(fmt.Sprintf("\n\n  Filename: %s", a.exportFilename))
	sb.WriteString("\n\n  [Enter] export  [Esc] cancel")
	return sb.String()
}

// --- Rename dialog ---

func (a App) handleRenameDialogKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case msg.Code == tea.KeyEscape:
		a.renameDialogOpen = false
		a.input.Focus()
		return a, nil

	case msg.Code == tea.KeyEnter:
		if strings.TrimSpace(a.renameDialogValue) == "" {
			// Stay open — empty title not allowed
			return a, nil
		}
		a.renameDialogOpen = false
		a.input.Focus()
		if a.session != nil {
			a.session.Title = a.renameDialogValue
			a.statusbar.SetTitle(a.renameDialogValue)
		}
		id := a.toasts.Show(fmt.Sprintf("Renamed to \"%s\"", a.renameDialogValue), components.ToastSuccess, 3*time.Second)
		return a, func() tea.Msg {
			time.Sleep(3 * time.Second)
			return ToastDismissMsg{ID: id}
		}

	case msg.Code == tea.KeyBackspace:
		if len(a.renameDialogValue) > 0 {
			a.renameDialogValue = a.renameDialogValue[:len(a.renameDialogValue)-1]
		}
		return a, nil

	default:
		if msg.Text != "" {
			a.renameDialogValue += msg.Text
		}
		return a, nil
	}
}

func (a App) renderRenameDialog() string {
	var sb strings.Builder
	sb.WriteString("Rename session                            esc\n")
	sb.WriteString(fmt.Sprintf("\n  > %s█", a.renameDialogValue))
	sb.WriteString("\n\n  [Enter] save  [Esc] cancel")
	return sb.String()
}

// --- Sessions dialog ---

func (a *App) loadSessions() tea.Cmd {
	return func() tea.Msg {
		summaries, err := store.List()
		if err != nil {
			return SessionsLoadedMsg{Err: err}
		}
		return SessionsLoadedMsg{Sessions: summaries}
	}
}

func (a App) handleSessionsDialogKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if a.sessionsDialogConfirm {
		return a.handleSessionsDeleteConfirm(msg)
	}

	switch {
	case msg.Code == tea.KeyEscape:
		a.sessionsDialogOpen = false
		a.input.Focus()
		return a, nil

	case msg.Code == tea.KeyEnter:
		filtered := a.filteredSessions()
		if len(filtered) == 0 || a.sessionsDialogSelect >= len(filtered) {
			return a, nil
		}
		entry := filtered[a.sessionsDialogSelect]
		return a.resumeSession(entry.ID)

	case msg.Code == tea.KeyUp:
		if a.sessionsDialogSelect > 0 {
			a.sessionsDialogSelect--
		}
		return a, nil

	case msg.Code == tea.KeyDown:
		filtered := a.filteredSessions()
		if a.sessionsDialogSelect < len(filtered)-1 {
			a.sessionsDialogSelect++
		}
		return a, nil

	case msg.Mod == tea.ModCtrl && msg.Code == 'd':
		filtered := a.filteredSessions()
		if len(filtered) > 0 {
			a.sessionsDialogConfirm = true
		}
		return a, nil

	case msg.Code == tea.KeyBackspace:
		if len(a.sessionsDialogQuery) > 0 {
			a.sessionsDialogQuery = a.sessionsDialogQuery[:len(a.sessionsDialogQuery)-1]
			a.sessionsDialogSelect = 0
		}
		return a, nil

	default:
		if msg.Text != "" {
			a.sessionsDialogQuery += msg.Text
			a.sessionsDialogSelect = 0
		}
		return a, nil
	}
}

func (a App) handleSessionsDeleteConfirm(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case msg.Code == tea.KeyEscape:
		a.sessionsDialogConfirm = false
		return a, nil

	case msg.Code == tea.KeyEnter:
		filtered := a.filteredSessions()
		if a.sessionsDialogSelect < len(filtered) {
			entry := filtered[a.sessionsDialogSelect]
			_ = store.Delete(entry.ID)
			// Remove from cached entries
			for i, e := range a.sessionsDialogEntries {
				if e.ID == entry.ID {
					a.sessionsDialogEntries = append(a.sessionsDialogEntries[:i], a.sessionsDialogEntries[i+1:]...)
					break
				}
			}
			if a.sessionsDialogSelect >= len(a.filteredSessions()) && a.sessionsDialogSelect > 0 {
				a.sessionsDialogSelect--
			}
		}
		a.sessionsDialogConfirm = false
		return a, nil

	default:
		return a, nil
	}
}

func (a App) filteredSessions() []store.SessionSummary {
	if a.sessionsDialogQuery == "" {
		return a.sessionsDialogEntries
	}
	q := strings.ToLower(a.sessionsDialogQuery)
	var out []store.SessionSummary
	for _, e := range a.sessionsDialogEntries {
		if strings.Contains(strings.ToLower(e.Title), q) ||
			strings.Contains(strings.ToLower(e.WorkDir), q) {
			out = append(out, e)
		}
	}
	return out
}

func (a App) resumeSession(id string) (tea.Model, tea.Cmd) {
	loaded, err := store.Load(id)
	if err != nil {
		a.sessionsDialogOpen = false
		a.input.Focus()
		id := a.toasts.Show("Failed to load session: "+err.Error(), components.ToastError, 3*time.Second)
		return a, func() tea.Msg {
			time.Sleep(3 * time.Second)
			return ToastDismissMsg{ID: id}
		}
	}

	a.session = loaded
	a.sessionsDialogOpen = false
	a.input.Focus()

	// Rebuild chat from session records
	a.rebuildChatFromSession()

	// Update status bar
	title := loaded.Title
	if title == "" {
		title = shortSessionTitle(loaded.ID)
	}
	a.statusbar.SetTitle(title)
	a.statusbar.SetTokens(loaded.Tokens.Input + loaded.Tokens.Output)

	toastID := a.toasts.Show("Session resumed", components.ToastSuccess, 3*time.Second)
	return a, func() tea.Msg {
		time.Sleep(3 * time.Second)
		return ToastDismissMsg{ID: toastID}
	}
}

func (a *App) rebuildChatFromSession() {
	a.chat.Clear()
	if a.session == nil {
		return
	}
	for _, rec := range a.session.Messages {
		switch rec.Message.Role {
		case "user":
			a.chat.AddEntry(components.ChatEntry{
				Role:    "user",
				Content: rec.Message.Content,
			})
		case "assistant":
			a.chat.AddEntry(components.ChatEntry{
				Role:    "assistant",
				Content: rec.Message.Content,
			})
		case "tool":
			a.chat.AddEntry(components.ChatEntry{
				Role:       "tool",
				Content:    rec.Message.Content,
				ToolID:     rec.Message.ToolCallID,
				ToolStatus: "success",
			})
		}
	}
}

func (a App) renderSessionsDialog() string {
	var sb strings.Builder
	sb.WriteString("Sessions (type to filter, Esc close)\n")

	if !a.sessionsDialogLoaded {
		sb.WriteString("\n  Loading...")
		return sb.String()
	}

	filtered := a.filteredSessions()
	if len(filtered) == 0 {
		sb.WriteString("\n  No sessions found")
		return sb.String()
	}

	if a.sessionsDialogConfirm && a.sessionsDialogSelect < len(filtered) {
		entry := filtered[a.sessionsDialogSelect]
		title := entry.Title
		if title == "" {
			title = entry.ID
		}
		sb.WriteString(fmt.Sprintf("\n  Delete \"%s\"?  [Enter] confirm  [Esc] cancel", title))
		return sb.String()
	}

	for i, entry := range filtered {
		marker := "  "
		if i == a.sessionsDialogSelect {
			marker = "> "
		}
		title := entry.Title
		if title == "" {
			title = entry.ID
		}
		sb.WriteString(fmt.Sprintf("\n%s%-30s  %d msgs", marker, title, entry.MessageCount))
	}

	sb.WriteString("\n\n  [Enter] resume  [Ctrl+D] delete  [Esc] close")
	return sb.String()
}

// --- Timeline dialog ---

type timelineEntry struct {
	ID      string
	Content string
	Time    time.Time
}

func (a App) timelineEntries() []timelineEntry {
	if a.session == nil {
		return nil
	}
	var entries []timelineEntry
	for _, rec := range a.session.Messages {
		if rec.Message.Role == "user" {
			content := rec.Message.Content
			if len(content) > 60 {
				content = content[:57] + "..."
			}
			entries = append(entries, timelineEntry{
				ID:      rec.ID,
				Content: content,
				Time:    rec.CreatedAt,
			})
		}
	}
	return entries
}

func (a App) handleTimelineDialogKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case msg.Code == tea.KeyEscape:
		a.timelineDialogOpen = false
		a.input.Focus()
		return a, nil

	case msg.Code == tea.KeyEnter:
		entries := a.timelineEntries()
		if a.timelineDialogSelect < len(entries) {
			// Scroll chat to the position of this user message
			a.scrollToUserMessage(a.timelineDialogSelect)
		}
		a.timelineDialogOpen = false
		a.input.Focus()
		return a, nil

	case msg.Code == tea.KeyUp:
		if a.timelineDialogSelect > 0 {
			a.timelineDialogSelect--
		}
		return a, nil

	case msg.Code == tea.KeyDown:
		entries := a.timelineEntries()
		if a.timelineDialogSelect < len(entries)-1 {
			a.timelineDialogSelect++
		}
		return a, nil

	default:
		return a, nil
	}
}

func (a *App) scrollToUserMessage(idx int) {
	userCount := 0
	linePos := 0
	for _, entry := range a.chat.Entries() {
		if entry.Role == "user" {
			if userCount == idx {
				a.chat.SetScrollPos(linePos)
				return
			}
			userCount++
		}
		linePos += 2 // approximate: entry + blank line
	}
}

func (a App) renderTimelineDialog() string {
	var sb strings.Builder
	sb.WriteString("Timeline (Enter jump, Esc close)\n")

	entries := a.timelineEntries()
	if len(entries) == 0 {
		sb.WriteString("\n  No user messages")
		return sb.String()
	}

	for i, entry := range entries {
		marker := "  "
		if i == a.timelineDialogSelect {
			marker = "> "
		}
		sb.WriteString(fmt.Sprintf("\n%s%s", marker, entry.Content))
	}

	return sb.String()
}

// --- Stash dialog ---

func (a App) handleStashDialogKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	entries := a.stash.List()

	switch {
	case msg.Code == tea.KeyEscape:
		a.stashDialogOpen = false
		a.input.Focus()
		return a, nil

	case msg.Code == tea.KeyEnter:
		if a.stashDialogSelect < len(entries) {
			// Restore selected entry to input and remove from stash
			entry := entries[len(entries)-1-a.stashDialogSelect] // displayed newest-first
			a.input.SetValue(entry.Content)
			a.stash.Delete(entry.ID)
		}
		a.stashDialogOpen = false
		a.input.Focus()
		return a, nil

	case msg.Code == tea.KeyUp:
		if a.stashDialogSelect > 0 {
			a.stashDialogSelect--
		}
		return a, nil

	case msg.Code == tea.KeyDown:
		if a.stashDialogSelect < len(entries)-1 {
			a.stashDialogSelect++
		}
		return a, nil

	case msg.Mod == tea.ModCtrl && msg.Code == 'd':
		if a.stashDialogSelect < len(entries) {
			entry := entries[len(entries)-1-a.stashDialogSelect]
			a.stash.Delete(entry.ID)
			remaining := a.stash.List()
			if a.stashDialogSelect >= len(remaining) && a.stashDialogSelect > 0 {
				a.stashDialogSelect--
			}
			if len(remaining) == 0 {
				a.stashDialogOpen = false
				a.input.Focus()
			}
		}
		return a, nil

	default:
		return a, nil
	}
}

func (a App) renderStashDialog() string {
	var sb strings.Builder
	sb.WriteString("Stash (Enter restore, Ctrl+D delete, Esc close)\n")

	entries := a.stash.List()
	if len(entries) == 0 {
		sb.WriteString("\n  Empty stash")
		return sb.String()
	}

	// Display newest first
	for i := len(entries) - 1; i >= 0; i-- {
		idx := len(entries) - 1 - i
		marker := "  "
		if idx == a.stashDialogSelect {
			marker = "> "
		}
		content := entries[i].Content
		if len(content) > 50 {
			content = content[:47] + "..."
		}
		sb.WriteString(fmt.Sprintf("\n%s\"%s\"", marker, content))
	}

	return sb.String()
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
		a.setStreaming(false)
		a.chat.CommitStream()
		a.input.Focus()
		if a.session != nil {
			a.statusbar.SetTokens(a.session.Tokens.Input + a.session.Tokens.Output)
		}
		dur := time.Since(a.responseStart)
		modeName := a.agent.Name
		if modeName == "" {
			modeName = agent.NameBuild
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
		a.setStreaming(false)
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
	footerH := 1

	mainW := a.mainContentWidth()
	chatH := a.height - statusH - inputH - footerH
	if chatH < 1 {
		chatH = 1
	}

	a.statusbar.SetSize(mainW)
	a.chat.SetSize(mainW, chatH)
	a.input.SetSize(mainW, inputH)
	a.footer.SetSize(mainW)
	if a.sidebarWideVisible() {
		a.toolpanel.SetSize(a.sidebarWidth())
	} else {
		a.toolpanel.SetSize(mainW)
	}
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

func (a App) sidebarWidth() int {
	return 42
}

func (a App) sidebarWideVisible() bool {
	if a.state != StateSession {
		return false
	}
	if a.sidebarHidden {
		return false
	}
	return a.width >= 120
}

func (a App) sidebarOverlayActive() bool {
	if a.state != StateSession {
		return false
	}
	if a.sidebarHidden {
		return false
	}
	if a.width >= 120 {
		return false
	}
	return a.sidebarOverlay
}

func (a App) sidebarVisible() bool {
	return a.sidebarWideVisible() || a.sidebarOverlayActive()
}

func (a App) mainContentWidth() int {
	if !a.sidebarWideVisible() {
		if a.width < 1 {
			return 1
		}
		return a.width
	}

	w := a.width - a.sidebarWidth() - 1
	if w < 40 {
		return a.width
	}
	return w
}

func renderSideBySide(main, side string, mainW, sideW int) string {
	mainLines := strings.Split(main, "\n")
	sideLines := strings.Split(side, "\n")
	n := max(len(mainLines), len(sideLines))

	mainCell := lipgloss.NewStyle().Width(mainW).MaxWidth(mainW)
	sideCell := lipgloss.NewStyle().Width(sideW).MaxWidth(sideW)

	var sb strings.Builder
	for i := 0; i < n; i++ {
		m := ""
		if i < len(mainLines) {
			m = mainLines[i]
		}
		s := ""
		if i < len(sideLines) {
			s = sideLines[i]
		}
		sb.WriteString(mainCell.Render(m))
		sb.WriteString(" ")
		sb.WriteString(sideCell.Render(s))
		if i < n-1 {
			sb.WriteByte('\n')
		}
	}
	return sb.String()
}

func (a App) renderSidebar() string {
	if !a.sidebarVisible() {
		return ""
	}

	muted := styles.DefaultTheme.TextMutedStyle
	text := styles.DefaultTheme.TextStyle
	success := styles.DefaultTheme.SuccessStyle
	errorStyle := styles.DefaultTheme.ErrorStyle
	warning := styles.DefaultTheme.WarningStyle
	modeColor := lipgloss.NewStyle().Foreground(styles.DefaultTheme.ModeColor(a.agent.Name)).Bold(true)

	var lines []string
	headerTitle := "Session"
	if a.session != nil {
		headerTitle = shortSessionTitle(a.session.ID)
	}
	lines = append(lines, text.Render(headerTitle))
	if a.session != nil {
		age := time.Since(a.session.CreatedAt).Round(time.Second)
		if age < 0 {
			age = 0
		}
		lines = append(lines, muted.Render(fmt.Sprintf("started %s ago", age)))
	}
	lines = append(lines, muted.Render(repeatStr("─", 34)))
	lines = append(lines, "")

	lines = append(lines, text.Render("Context"))
	tokens := 0
	if a.session != nil {
		tokens = a.session.Tokens.Input + a.session.Tokens.Output
	}
	lines = append(lines, muted.Render(fmt.Sprintf("%s tokens", components.FormatTokens(tokens))))
	const assumedContextLimit = 200_000
	percent := clamp((tokens*100)/assumedContextLimit, 0, 100)
	const barWidth = 20
	filled := clamp((percent*barWidth)/100, 0, barWidth)
	bar := "[" + repeatStr("■", filled) + repeatStr("·", barWidth-filled) + "]"
	lines = append(lines, muted.Render(bar)+" "+muted.Render(fmt.Sprintf("%d%%", percent)))
	modeName := strings.TrimSpace(a.agent.Name)
	if modeName == "" {
		modeName = agent.NameBuild
	}
	modeLabel := "Build"
	if len(modeName) > 0 {
		modeLabel = strings.ToUpper(modeName[:1]) + modeName[1:]
	}
	lines = append(lines, modeColor.Render(modeLabel)+" "+muted.Render("agent"))
	if a.model != "" {
		lines = append(lines, muted.Render(a.model))
	}
	if a.session != nil {
		msgCount := len(a.session.Messages)
		lines = append(lines, muted.Render(fmt.Sprintf("%d messages", msgCount)))
	}
	lines = append(lines, muted.Render(repeatStr("─", 34)))
	lines = append(lines, "")

	lines = append(lines, text.Render("Recent tools"))
	events := a.toolpanel.Events()
	successCount := 0
	errorCount := 0
	pendingCount := 0
	for _, ev := range events {
		switch {
		case ev.Error != "":
			errorCount++
		case ev.Output != "":
			successCount++
		default:
			pendingCount++
		}
	}
	summary := fmt.Sprintf("✓ %d", successCount)
	if errorCount > 0 {
		summary += fmt.Sprintf("  ✗ %d", errorCount)
	}
	if pendingCount > 0 {
		summary += fmt.Sprintf("  ⋯ %d", pendingCount)
	}
	lines = append(lines, muted.Render(summary))
	if len(events) == 0 {
		lines = append(lines, muted.Render("No tool activity yet"))
	} else {
		start := max(0, len(events)-8)
		for i := start; i < len(events); i++ {
			ev := events[i]
			switch {
			case ev.Error != "":
				lines = append(lines, errorStyle.Render("✗ "+ev.Name))
			case ev.Output != "":
				lines = append(lines, success.Render("✓ "+ev.Name))
			default:
				lines = append(lines, warning.Render("⋯ "+ev.Name))
			}
		}
	}
	lines = append(lines, muted.Render(repeatStr("─", 34)))
	lines = append(lines, "")

	lines = append(lines, text.Render("Quick actions"))
	lines = append(lines, muted.Render("/models"))
	lines = append(lines, muted.Render("/help"))
	lines = append(lines, muted.Render("^B toggle sidebar"))

	return strings.Join(lines, "\n")
}

func (a App) sidebarOverlayPanel() string {
	body := "Sidebar overlay (Esc close)\n\n" + a.renderSidebar()
	panelW := min(a.width-2, a.sidebarWidth()+2)
	if panelW < 30 {
		panelW = max(20, a.width)
	}
	return lipgloss.NewStyle().
		Width(panelW).
		MaxWidth(panelW).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(styles.DefaultTheme.Palette.Border)).
		Padding(0, 1).
		Render(body)
}

func (a App) sidebarOverlayPanelRect() (x, y, w, h int) {
	return a.panelRectFromRendered(a.sidebarOverlayPanel())
}

func (a App) panelRectFromRendered(panel string) (x, y, w, h int) {
	lines := strings.Split(panel, "\n")
	maxW := 0
	for _, ln := range lines {
		maxW = max(maxW, lipgloss.Width(ln))
	}
	h = len(lines)
	w = maxW
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}
	x = max(0, (a.width-w)/2)
	y = max(0, (a.height-h)/2)
	return
}

func (a App) renderSidebarOverlay(main string) string {
	dimmed := lipgloss.NewStyle().Faint(true).Render(main)
	mainLines := strings.Split(dimmed, "\n")
	panel := a.sidebarOverlayPanel()
	panelLines := strings.Split(panel, "\n")
	x, y, _, _ := a.panelRectFromRendered(panel)

	for i, ln := range panelLines {
		row := y + i
		if row < 0 || row >= len(mainLines) {
			continue
		}
		line := strings.Repeat(" ", x) + ln
		lineW := lipgloss.Width(line)
		if lineW < a.width {
			line += strings.Repeat(" ", a.width-lineW)
		}
		mainLines[row] = line
	}
	return strings.Join(mainLines, "\n")
}

func (a App) renderSessionView() string {
	var sb strings.Builder
	sb.WriteString(a.statusbar.View())
	sb.WriteByte('\n')
	toastView := a.toasts.View()
	if toastView != "" {
		sb.WriteString(toastView)
		sb.WriteByte('\n')
	}
	sb.WriteString(a.chat.View())
	sb.WriteByte('\n')
	if a.helpDialogOpen {
		sb.WriteString(a.renderHelpDialog())
		sb.WriteByte('\n')
	}
	if a.statusDialogOpen {
		sb.WriteString(a.renderStatusDialog())
		sb.WriteByte('\n')
	}
	if a.stashDialogOpen {
		sb.WriteString(a.renderStashDialog())
		sb.WriteByte('\n')
	}
	if a.timelineDialogOpen {
		sb.WriteString(a.renderTimelineDialog())
		sb.WriteByte('\n')
	}
	if a.sessionsDialogOpen {
		sb.WriteString(a.renderSessionsDialog())
		sb.WriteByte('\n')
	}
	if a.renameDialogOpen {
		sb.WriteString(a.renderRenameDialog())
		sb.WriteByte('\n')
	}
	if a.exportDialogOpen {
		sb.WriteString(a.renderExportDialog())
		sb.WriteByte('\n')
	}
	if a.modelDialogOpen {
		sb.WriteString(a.renderModelDialog())
		sb.WriteByte('\n')
	}
	if a.commandOpen {
		sb.WriteString(a.renderCommandPalette())
		sb.WriteByte('\n')
	}
	if a.inlineOpen {
		sb.WriteString(a.renderInlineSuggestions())
		sb.WriteByte('\n')
	}
	sb.WriteString(a.input.View())
	sb.WriteByte('\n')
	sb.WriteString(a.footer.View())
	main := sb.String()
	if a.sidebarOverlayActive() {
		return a.renderSidebarOverlay(main)
	}
	if !a.sidebarWideVisible() {
		return main
	}
	return renderSideBySide(main, a.renderSidebar(), a.mainContentWidth(), a.sidebarWidth())
}
