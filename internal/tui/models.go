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
	a.fullModelID = modelID
	// Track in recents
	if a.modelPrefs != nil {
		ref := store.ModelRef{
			ProviderID: provider.ModelInfo{ID: modelID}.ProviderName(),
			ModelID:    modelID,
		}
		a.modelPrefs.AddRecent(ref)
		a.warnOnErr(a.modelPrefs.Save(), "model recents")
	}
	// Clear variant if new model doesn't support it
	if a.activeVariant != "" {
		variants := provider.VariantsFor(modelID)
		if len(variants) == 0 {
			a.activeVariant = ""
			a.statusbar.SetVariantBadge("")
		}
	}
	a.chat.AddEntry(components.ChatEntry{
		Role:    "system",
		Content: fmt.Sprintf(`Model switched to "%s".`, modelID),
	})
	return true
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
