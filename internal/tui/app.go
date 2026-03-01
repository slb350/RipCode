package tui

import (
	"context"
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
	provider          provider.Provider
	registry          *tool.Registry
	session           *session.Session
	agent             agent.Agent
	model             string
	fullModelID       string
	activeVariant     string
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

	// Leader key prefix
	leaderPending bool

	// Help dialog
	helpDialogOpen   bool
	helpDialogQuery  string
	helpDialogSelect int
	helpDialogTab    int // 0=commands, 1=keybinds

	// Status dialog
	statusDialogOpen bool

	// Export dialog
	exportDialogOpen      bool
	exportIncludeTools    bool
	exportIncludeMeta     bool
	exportIncludeThinking bool
	exportFilename        string
	exportFocusedField    int // 0=tools, 1=meta, 2=thinking, 3=filename

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

	// Agent dialog
	agentDialogOpen   bool
	agentDialogQuery  string
	agentDialogSelect int

	// Connect dialog
	connectDialogOpen  bool
	connectDialogInput string

	// Model provider filter (sub-mode within model dialog)
	modelDialogProviderMode bool
	modelProviderFilter     string

	// Themes dialog
	themesDialogOpen   bool
	themesDialogSelect int

	// Timeline dialog
	timelineDialogOpen   bool
	timelineDialogQuery  string
	timelineDialogSelect int

	// Fork dialog
	forkDialogOpen   bool
	forkDialogSelect int

	// MCP dialog
	mcpDialogOpen   bool
	mcpDialogSelect int

	// Prompt stash
	stash               *components.PromptStash
	stashDialogOpen     bool
	stashDialogSelect   int
	stashPendingContent string // content to stash (captured before input reset)

	// Modified files tracking
	modifiedFiles []string

	// Startup warnings (e.g. corrupted config files)
	startupWarnings      []string
	startupWarningsShown bool
}

// NewApp creates the initial application model.
func NewApp() App {
	var warnings []string
	prefs, err := store.LoadModelPrefs()
	if err != nil {
		warnings = append(warnings, "model preferences corrupted, using defaults")
	}
	mcpCfg, err := store.LoadMCPConfig()
	if err != nil {
		warnings = append(warnings, "MCP config corrupted, using defaults")
	}
	lspCfg, err := store.LoadLSPConfig()
	if err != nil {
		warnings = append(warnings, "LSP config corrupted, using defaults")
	}
	uiPrefs, err := store.LoadUIPrefs()
	if err != nil {
		warnings = append(warnings, "UI preferences corrupted, using defaults")
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
		warnings = append(warnings, "prompt history corrupted, using defaults")
	}
	stash, err := loadPromptStash()
	if err != nil {
		warnings = append(warnings, "prompt stash corrupted, using defaults")
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
	a.commandOpen = false
	a.modelDialogOpen = false
	a.helpDialogOpen = false
	a.statusDialogOpen = false
	a.exportDialogOpen = false
	a.renameDialogOpen = false
	a.sessionsDialogOpen = false
	a.agentDialogOpen = false
	a.connectDialogOpen = false
	a.themesDialogOpen = false
	a.timelineDialogOpen = false
	a.forkDialogOpen = false
	a.stashDialogOpen = false
	a.mcpDialogOpen = false
	a.inlineOpen = false
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
func (a *App) warnOnErr(err error, what string) {
	if err != nil {
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
	if a.sessionsDialogLoaded && len(a.sessionsDialogEntries) > 0 {
		return false
	}
	return true
}

// trackModifiedFile adds a file path to the modified files list, deduplicating.
func (a *App) trackModifiedFile(path string) {
	for _, f := range a.modifiedFiles {
		if f == path {
			return
		}
	}
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
