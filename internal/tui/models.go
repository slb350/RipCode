package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/sahilm/fuzzy"

	"github.com/stephenbrandon/ripcode/internal/provider"
	"github.com/stephenbrandon/ripcode/internal/store"
	"github.com/stephenbrandon/ripcode/internal/tui/components"
)

// --- Dialog state types ---

type commandPaletteState struct {
	open     bool
	query    string
	selected int
}

// Inline suggestion mode constants.
const (
	inlineModeCommand = "/"
	inlineModeFile    = "@"
)

type inlineState struct {
	open     bool
	mode     string // inlineModeCommand or inlineModeFile
	query    string
	selected int
	start    int
	end      int
}

type modelDialogState struct {
	open           bool
	query          string
	selected       int
	providerMode   bool
	providerFilter string
}

type helpDialogState struct {
	open     bool
	query    string
	selected int
	tab      int // 0=commands, 1=keybinds
}

type statusDialogState struct {
	open bool
}

type exportDialogState struct {
	open            bool
	includeTools    bool
	includeMeta     bool
	includeThinking bool
	filename        string
	focusedField    int // 0=tools, 1=meta, 2=thinking, 3=filename
}

type renameDialogState struct {
	open  bool
	value string
}

type sessionsDialogState struct {
	open     bool
	query    string
	selected int
	confirm  bool
	entries  []store.SessionSummary
	loaded   bool
}

type agentDialogState struct {
	open     bool
	query    string
	selected int
}

type connectDialogState struct {
	open  bool
	input string
}

type themesDialogState struct {
	open     bool
	selected int
}

type timelineDialogState struct {
	open     bool
	query    string
	selected int
}

type forkDialogState struct {
	open     bool
	selected int
}

type mcpDialogState struct {
	open     bool
	selected int
}

type stashDialogState struct {
	open           bool
	selected       int
	pendingContent string // content to stash (captured before input reset)
}

// applyVariant updates activeVariant, the status bar badge, and the provider's
// reasoning effort in one place.
func (a *App) applyVariant(variant string) {
	a.activeVariant = variant
	a.statusbar.SetVariantBadge(provider.VariantBadge(variant))
	if rs, ok := a.provider.(provider.ReasoningEffortSetter); ok {
		rs.SetReasoningEffort(variant)
	}
}

// switchModel attempts to switch to the given model ID via the provider.
// Returns true on success, false on error. Adds appropriate chat entries.
func (a *App) switchModel(modelID string) bool {
	if a.provider == nil {
		a.chat.AddEntry(components.ChatEntry{
			Role:    components.RoleError,
			Content: "Not configured — missing provider",
		})
		return false
	}

	setter, ok := a.provider.(provider.ModelSetter)
	if !ok {
		a.chat.AddEntry(components.ChatEntry{
			Role:    components.RoleError,
			Content: fmt.Sprintf(`Provider "%s" does not support runtime model switching.`, a.provider.Name()),
		})
		return false
	}

	setter.SetModel(modelID)
	a.SetModel(displayModelName(modelID))
	a.SetFullModelID(modelID)
	// Track in recents
	if a.modelPrefs != nil {
		ref := store.ModelRef{
			ProviderID: provider.ModelInfo{ID: modelID}.ProviderName(),
			ModelID:    modelID,
		}
		a.modelPrefs.AddRecent(ref)
		a.warnOnErr(a.modelPrefs.Save(), "model recents")
	}
	a.chat.AddEntry(components.ChatEntry{
		Role:    components.RoleSystem,
		Content: fmt.Sprintf(`Model switched to "%s".`, modelID),
	})
	return true
}

func (a App) handleModelsCommand(input string) (tea.Model, tea.Cmd) {
	if a.provider == nil {
		a.chat.AddEntry(components.ChatEntry{
			Role:    components.RoleError,
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
			Role:    components.RoleError,
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

// modelSearchStrings implements fuzzy.Source for model searching.
type modelSearchStrings []provider.ModelInfo

func (m modelSearchStrings) String(i int) string {
	return m[i].ID + " " + m[i].Name
}

func (m modelSearchStrings) Len() int {
	return len(m)
}

func filterModels(models []provider.ModelInfo, query string) []provider.ModelInfo {
	if query == "" {
		return models
	}

	matches := fuzzy.FindFrom(strings.ToLower(query), modelSearchStrings(models))
	out := make([]provider.ModelInfo, len(matches))
	for i, match := range matches {
		out[i] = models[match.Index]
	}
	return out
}

func modelLine(m provider.ModelInfo) string {
	if m.Name == "" || m.Name == m.ID {
		return m.ID
	}
	return m.ID + " - " + m.Name
}

func formatContextLength(n int) string {
	if n <= 0 {
		return ""
	}
	if n >= 999999 {
		return fmt.Sprintf("%dM", (n+500000)/1000000)
	}
	return fmt.Sprintf("%dK", n/1000)
}

// fileTracker maintains an ordered, deduplicated list of file paths.
type fileTracker struct {
	list []string
	set  map[string]bool
}

// add records a file path if not already tracked.
func (ft *fileTracker) add(path string) {
	if ft.set[path] {
		return
	}
	if ft.set == nil {
		ft.set = make(map[string]bool)
	}
	ft.set[path] = true
	ft.list = append(ft.list, path)
}

// reset clears all tracked files.
func (ft *fileTracker) reset() {
	ft.list = nil
	ft.set = nil
}

// paths returns the ordered list of tracked file paths.
func (ft *fileTracker) paths() []string {
	return ft.list
}

// len returns the number of tracked files.
func (ft *fileTracker) len() int {
	return len(ft.list)
}
