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
	"github.com/stephenbrandon/ripcode/internal/store"
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
	footer    components.SessionFooter
	toolpanel components.ToolPanel
	home      components.Home

	// Preferences and config
	modelPrefs *store.ModelPrefs
	mcpConfig  *store.MCPConfig
	lspConfig  *store.LSPConfig
	uiPrefs    *store.UIPrefs
	todoTool   *tool.TodoTool

	// Agent state
	provider      provider.Provider
	registry      *tool.Registry
	session       *session.Session
	agent         agent.Agent
	model         string
	fullModelID   string
	activeVariant string
	maxSteps      int
	streaming     bool
	modelsCache   []provider.ModelInfo
	modelsLoaded  bool

	// Dialog and overlay states
	commandPalette commandPaletteState
	inline         inlineState
	modelDialog    modelDialogState
	helpDialog     helpDialogState
	statusDialog   statusDialogState
	exportDialog   exportDialogState
	renameDialog   renameDialogState
	sessionsDialog sessionsDialogState
	agentDialog    agentDialogState
	connectDialog  connectDialogState
	themesDialog   themesDialogState
	timelineDialog timelineDialogState
	forkDialog     forkDialogState
	mcpDialog      mcpDialogState
	stashDialog    stashDialogState

	sidebarHidden    bool
	sidebarOverlay   bool
	fileCache        []string
	fileCacheLoaded  bool
	fileCacheLoading bool
	responseStart    time.Time
	cancel           context.CancelFunc
	eventCh          <-chan agent.Event
	promptHistory    *components.PromptHistory
	toasts           components.ToastManager
	shellMode        bool
	cmdRegistry      *CommandRegistry
	showDetails      bool
	showThinking     bool
	showTimestamps   bool

	// Leader key prefix
	leaderPending bool

	// Prompt stash (the stash object; dialog state is in stashDialog)
	stash *components.PromptStash

	// Modified files tracking
	modifiedFiles    []string
	modifiedFilesSet map[string]bool

	// Startup warnings (e.g. corrupted config files)
	startupWarnings      []string
	startupWarningsShown bool
}

// NewApp creates the initial application model.
func NewApp() App {
	var warnings []string
	prefs, err := store.LoadModelPrefs()
	if err != nil {
		warnings = append(warnings, fmt.Sprintf("model preferences: %v, using defaults", err))
	}
	mcpCfg, err := store.LoadMCPConfig()
	if err != nil {
		warnings = append(warnings, fmt.Sprintf("MCP config: %v, using defaults", err))
	}
	lspCfg, err := store.LoadLSPConfig()
	if err != nil {
		warnings = append(warnings, fmt.Sprintf("LSP config: %v, using defaults", err))
	}
	uiPrefs, err := store.LoadUIPrefs()
	if err != nil {
		warnings = append(warnings, fmt.Sprintf("UI preferences: %v, using defaults", err))
	}

	footer := components.NewSessionFooter()
	if mcpCfg != nil {
		footer.SetMCPCount(mcpCfg.CountEnabled())
	}
	if lspCfg != nil {
		footer.SetLSPCount(lspCfg.CountEnabled())
	}

	history, err := loadPromptHistory()
	if err != nil {
		warnings = append(warnings, fmt.Sprintf("prompt history: %v, using defaults", err))
	}
	stash, err := loadPromptStash()
	if err != nil {
		warnings = append(warnings, fmt.Sprintf("prompt stash: %v, using defaults", err))
	}

	a := App{
		chat:            components.NewChat(),
		input:           components.NewInput(),
		statusbar:       components.NewStatusBar(),
		footer:          footer,
		toolpanel:       components.NewToolPanel(),
		home:            components.NewHome(),
		state:           StateHome,
		maxSteps:        100,
		promptHistory:   history,
		toasts:          components.NewToastManager(),
		stash:           stash,
		modelPrefs:      prefs,
		mcpConfig:       mcpCfg,
		lspConfig:       lspCfg,
		uiPrefs:         uiPrefs,
		startupWarnings: warnings,
	}
	a.initRegistry()
	return a
}

// closeAllDialogs closes every dialog and overlay, then blurs input.
func (a *App) closeAllDialogs() {
	a.commandPalette.open = false
	a.modelDialog.open = false
	a.helpDialog.open = false
	a.statusDialog.open = false
	a.exportDialog.open = false
	a.renameDialog.open = false
	a.sessionsDialog.open = false
	a.agentDialog.open = false
	a.connectDialog.open = false
	a.themesDialog.open = false
	a.timelineDialog.open = false
	a.forkDialog.open = false
	a.stashDialog.open = false
	a.mcpDialog.open = false
	a.inline.open = false
	a.input.Blur()
}

// SetProvider configures the LLM provider.
func (a *App) SetProvider(p provider.Provider) {
	a.provider = p
	a.footer.SetConnected(p != nil)
}

// SetRegistry configures the tool registry.
func (a *App) SetRegistry(r *tool.Registry) {
	a.registry = r
	if t, ok := r.Get("todo"); ok {
		if td, ok := t.(*tool.TodoTool); ok {
			a.todoTool = td
		}
	}
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

// SetFullModelID updates the full model ID (e.g. "anthropic/claude-4").
func (a *App) SetFullModelID(id string) {
	a.fullModelID = id
}

// SetMaxSteps sets the max agent loop steps.
func (a *App) SetMaxSteps(n int) {
	a.maxSteps = n
}

// warnOnErr shows a toast warning if err is non-nil. Used for non-fatal
// save/persist failures where the operation can continue with stale state.
// Also logs the error to disk for later diagnosis.
func (a *App) warnOnErr(err error, what string) {
	if err != nil {
		store.LogError("save "+what, err)
		a.toasts.Show("Failed to save "+what, components.ToastWarning, 3*time.Second)
	}
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

// showGettingStarted returns whether the getting started card should appear.
func (a App) showGettingStarted() bool {
	if a.uiPrefs == nil || a.uiPrefs.GettingStartedDismissed {
		return false
	}
	// Hide when user has session history
	if a.sessionsDialog.loaded && len(a.sessionsDialog.entries) > 0 {
		return false
	}
	return true
}

// trackModifiedFile adds a file path to the modified files list, deduplicating.
func (a *App) trackModifiedFile(path string) {
	if a.modifiedFilesSet[path] {
		return
	}
	if a.modifiedFilesSet == nil {
		a.modifiedFilesSet = make(map[string]bool)
	}
	a.modifiedFilesSet[path] = true
	a.modifiedFiles = append(a.modifiedFiles, path)
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
		if len(a.startupWarnings) > 0 && !a.startupWarningsShown {
			a.startupWarningsShown = true
			combined := strings.Join(a.startupWarnings, "; ")
			if len(combined) > 120 {
				combined = combined[:117] + "..."
			}
			return a, a.ShowToast(combined, components.ToastWarning)
		}
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
		if msg.Err != nil {
			a.sessionsDialog.loaded = true
			return a, a.ShowToast("Failed to load sessions", components.ToastError)
		}
		a.sessionsDialog.entries = msg.Sessions
		a.sessionsDialog.loaded = true
		if len(msg.Corrupted) > 0 {
			return a, a.ShowToast(
				fmt.Sprintf("%d corrupted session file(s) skipped", len(msg.Corrupted)),
				components.ToastWarning,
			)
		}
		return a, nil

	case ToastDismissMsg:
		a.toasts.Dismiss(msg.ID)
		return a, nil

	case ShellResultMsg:
		a.setStreaming(false)
		a.input.Focus()
		if msg.Error != "" {
			a.chat.AddEntry(components.ChatEntry{Role: components.RoleError, Content: msg.Error})
			toastCmd := a.ShowToast("Command failed", components.ToastError)
			return a, toastCmd
		}
		output := msg.Output
		if len(output) > 2000 {
			output = output[:2000] + "\n... (truncated)"
		}
		if output != "" {
			a.chat.AddEntry(components.ChatEntry{Role: components.RoleSystem, Content: output})
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
