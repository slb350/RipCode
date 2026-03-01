package tui

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/stephenbrandon/ripcode/internal/agent"
	"github.com/stephenbrandon/ripcode/internal/provider"
	"github.com/stephenbrandon/ripcode/internal/session"
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
}

type commandEntry struct {
	Title       string
	Command     string
	Description string
	Execute     bool
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
	return App{
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
	}
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

	switch strings.ToLower(parts[0]) {
	case "/models":
		model, cmd := a.handleModelsCommand(input)
		return model.(App), cmd, true

	case "/help", "/commands":
		a.chat.AddEntry(components.ChatEntry{Role: "user", Content: input})
		var lines []string
		lines = append(lines, "Commands:")
		for _, c := range a.commandEntries() {
			lines = append(lines, c.Command+" - "+c.Description)
		}
		a.chat.AddEntry(components.ChatEntry{
			Role:    "system",
			Content: strings.Join(lines, "\n"),
		})
		return a, nil, true

	case "/clear", "/new":
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
		return a, nil, true

	case "/exit", "/quit", "/q":
		return a, tea.Quit, true

	case "/sidebar":
		a = a.toggleSidebar()
		state := "hidden"
		if a.sidebarVisible() {
			state = "shown"
		}
		a.chat.AddEntry(components.ChatEntry{Role: "user", Content: input})
		a.chat.AddEntry(components.ChatEntry{
			Role:    "system",
			Content: fmt.Sprintf("Sidebar %s.", state),
		})
		return a, nil, true

	case "/agent":
		return a.handleAgentCommand(input, parts[1:]), nil, true

	case "/model":
		if len(parts) == 1 {
			model, cmd := a.handleModelsCommand("/models")
			return model.(App), cmd, true
		}
		return a.handleModelCommand(input, parts[1:]), nil, true

	default:
		a.chat.AddEntry(components.ChatEntry{Role: "user", Content: input})
		a.chat.AddEntry(components.ChatEntry{
			Role:    "error",
			Content: `Unknown command. Try "/help".`,
		})
		return a, nil, true
	}
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

func (a App) commandEntries() []commandEntry {
	return []commandEntry{
		{
			Title:       "Help",
			Command:     "/help",
			Description: "Show available commands",
			Execute:     true,
		},
		{
			Title:       "New session",
			Command:     "/new",
			Description: "Clear chat and tool history",
			Execute:     true,
		},
		{
			Title:       "Select model",
			Command:     "/models",
			Description: "Search and switch models",
			Execute:     true,
		},
		{
			Title:       "Switch to build agent",
			Command:     "/agent build",
			Description: "Enable full tool access mode",
			Execute:     true,
		},
		{
			Title:       "Switch to plan agent",
			Command:     "/agent plan",
			Description: "Enable read-only planning mode",
			Execute:     true,
		},
		{
			Title:       "Toggle sidebar",
			Command:     "/sidebar",
			Description: "Show or hide session sidebar",
			Execute:     true,
		},
		{
			Title:       "Set model",
			Command:     "/model ",
			Description: "Type full provider/model-id and submit",
			Execute:     false,
		},
		{
			Title:       "Exit",
			Command:     "/exit",
			Description: "Quit ripcode",
			Execute:     true,
		},
	}
}

func (a App) filteredCommands() []commandEntry {
	all := a.commandEntries()
	if strings.TrimSpace(a.commandQuery) == "" {
		return all
	}

	q := strings.ToLower(strings.TrimSpace(a.commandQuery))
	out := make([]commandEntry, 0, len(all))
	for _, c := range all {
		haystack := strings.ToLower(c.Title + " " + c.Command + " " + c.Description)
		if strings.Contains(haystack, q) {
			out = append(out, c)
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
		commands := a.commandEntries()
		out := make([]inlineEntry, 0, len(commands))
		for _, cmd := range commands {
			name := strings.TrimSpace(strings.TrimPrefix(cmd.Command, "/"))
			haystack := strings.ToLower(name + " " + cmd.Description)
			if query != "" && !strings.Contains(haystack, query) {
				continue
			}
			out = append(out, inlineEntry{
				Display:     cmd.Command,
				Insert:      cmd.Command,
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
		entries := a.filteredCommands()
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
			return a.handleSubmit(entry.Command)
		}
		a.input.SetValue(entry.Command)
		cacheCmd := a.updateInlineSuggestions()
		return a, cacheCmd

	case msg.Code == tea.KeyUp:
		entries := a.filteredCommands()
		if len(entries) == 0 {
			return a, nil
		}
		a.commandSelect--
		if a.commandSelect < 0 {
			a.commandSelect = len(entries) - 1
		}
		return a, nil

	case msg.Code == tea.KeyDown:
		entries := a.filteredCommands()
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
	entries := a.filteredCommands()
	query := strings.TrimSpace(a.commandQuery)
	if query == "" {
		if len(entries) == 0 {
			query = "(empty)"
		} else {
			query = "all"
		}
	}
	header := "Commands (Ctrl+P/Ctrl+K, Esc close) - filter: " + query

	items := make([]pickerItem, len(entries))
	for i, e := range entries {
		items[i] = pickerItem{Label: e.Command, Description: e.Description}
	}
	return renderPickerList(header, items, a.commandSelect, 7)
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
