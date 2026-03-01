package tui

import (
	"fmt"
	"sort"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/stephenbrandon/ripcode/internal/provider"
	"github.com/stephenbrandon/ripcode/internal/store"
	"github.com/stephenbrandon/ripcode/internal/tui/components"
)

func (a App) openModelDialog(query string) App {
	a.modelDialog.open = true
	a.modelDialog.query = strings.TrimSpace(query)
	a.modelDialog.selected = 0
	a.commandPalette.open = false
	a.inline.open = false
	a.input.Blur()
	return a
}

func (a App) closeModelDialog() App {
	a.modelDialog.open = false
	a.modelDialog.query = ""
	a.modelDialog.selected = 0
	a.modelDialog.providerMode = false
	a.modelDialog.providerFilter = ""
	a.input.Focus()
	return a
}

func (a App) filteredModelsDialog() []provider.ModelInfo {
	// Apply provider filter first
	source := a.modelsCache
	if a.modelDialog.providerFilter != "" {
		source = make([]provider.ModelInfo, 0)
		for _, m := range a.modelsCache {
			if m.ProviderName() == a.modelDialog.providerFilter {
				source = append(source, m)
			}
		}
	}
	filtered := filterModels(source, strings.TrimSpace(a.modelDialog.query))
	query := strings.TrimSpace(a.modelDialog.query)
	// When filtering, return flat fuzzy-sorted list (no grouping)
	if query != "" {
		return filtered
	}
	// When unfiltered, put favorites first, then recents, then rest
	if a.modelPrefs == nil {
		return filtered
	}
	var favs, recents, rest []provider.ModelInfo
	recentSet := make(map[string]bool)
	for _, r := range a.modelPrefs.Recent {
		recentSet[r.ModelID] = true
	}
	for _, m := range filtered {
		ref := store.ModelRef{ProviderID: m.ProviderName(), ModelID: m.ID}
		if a.modelPrefs.IsFavorite(ref) {
			favs = append(favs, m)
		} else if recentSet[m.ID] {
			recents = append(recents, m)
		} else {
			rest = append(rest, m)
		}
	}
	result := make([]provider.ModelInfo, 0, len(filtered))
	result = append(result, favs...)
	result = append(result, recents...)
	result = append(result, rest...)
	return result
}

func (a App) uniqueProviders() []string {
	seen := make(map[string]bool)
	var providers []string
	for _, m := range a.modelsCache {
		p := m.ProviderName()
		if !seen[p] {
			seen[p] = true
			providers = append(providers, p)
		}
	}
	sort.Strings(providers)
	return providers
}

func (a App) handleModelDialogKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	// Provider sub-mode: Escape clears filter and exits mode
	if a.modelDialog.providerMode {
		switch {
		case msg.Code == tea.KeyEscape:
			a.modelDialog.providerMode = false
			a.modelDialog.providerFilter = ""
			a.modelDialog.selected = 0
			return a, nil
		case msg.Code == tea.KeyEnter:
			providers := a.uniqueProviders()
			if a.modelDialog.selected == 0 {
				// "All" option
				a.modelDialog.providerFilter = ""
			} else if a.modelDialog.selected > 0 && a.modelDialog.selected <= len(providers) {
				a.modelDialog.providerFilter = providers[a.modelDialog.selected-1]
			}
			a.modelDialog.providerMode = false
			a.modelDialog.selected = 0
			return a, nil
		case msg.Code == tea.KeyUp:
			providers := a.uniqueProviders()
			total := len(providers) + 1 // +1 for "All"
			a.modelDialog.selected--
			if a.modelDialog.selected < 0 {
				a.modelDialog.selected = total - 1
			}
			return a, nil
		case msg.Code == tea.KeyDown:
			providers := a.uniqueProviders()
			total := len(providers) + 1 // +1 for "All"
			a.modelDialog.selected++
			if a.modelDialog.selected >= total {
				a.modelDialog.selected = 0
			}
			return a, nil
		default:
			return a, nil
		}
	}

	switch {
	case msg.Code == tea.KeyEscape:
		return a.closeModelDialog(), nil

	case msg.Mod == tea.ModCtrl && msg.Code == 'a':
		a.modelDialog.providerMode = true
		a.modelDialog.selected = 0
		return a, nil

	case msg.Mod == tea.ModCtrl && msg.Code == 'f':
		if a.modelPrefs == nil {
			return a, nil
		}
		models := a.filteredModelsDialog()
		if len(models) == 0 {
			return a, nil
		}
		sel := clamp(a.modelDialog.selected, 0, len(models)-1)
		selected := models[sel]
		ref := store.ModelRef{ProviderID: selected.ProviderName(), ModelID: selected.ID}
		isFav := a.modelPrefs.ToggleFavorite(ref)
		a.warnOnErr(a.modelPrefs.Save(), "favorites")
		// Re-sort and find the same model
		newModels := a.filteredModelsDialog()
		for i, m := range newModels {
			if m.ID == selected.ID {
				a.modelDialog.selected = i
				break
			}
		}
		msg := "Removed from favorites"
		if isFav {
			msg = "Added to favorites"
		}
		return a, a.ShowToast(msg, components.ToastSuccess)

	case msg.Code == tea.KeyEnter:
		models := a.filteredModelsDialog()
		if len(models) == 0 {
			return a.closeModelDialog(), nil
		}

		if a.modelDialog.selected < 0 {
			a.modelDialog.selected = 0
		}
		if a.modelDialog.selected >= len(models) {
			a.modelDialog.selected = len(models) - 1
		}

		selected := models[a.modelDialog.selected]
		a.switchModel(selected.ID)
		return a.closeModelDialog(), nil

	case msg.Code == tea.KeyUp:
		models := a.filteredModelsDialog()
		if len(models) == 0 {
			return a, nil
		}
		a.modelDialog.selected--
		if a.modelDialog.selected < 0 {
			a.modelDialog.selected = len(models) - 1
		}
		return a, nil

	case msg.Code == tea.KeyDown:
		models := a.filteredModelsDialog()
		if len(models) == 0 {
			return a, nil
		}
		a.modelDialog.selected++
		if a.modelDialog.selected >= len(models) {
			a.modelDialog.selected = 0
		}
		return a, nil

	case msg.Code == tea.KeyBackspace:
		a.modelDialog.query = backspaceRune(a.modelDialog.query)
		a.modelDialog.selected = 0
		return a, nil

	default:
		if msg.Text != "" {
			a.modelDialog.query += msg.Text
			a.modelDialog.selected = 0
		}
		return a, nil
	}
}

func (a App) renderModelDialog() string {
	models := a.filteredModelsDialog()
	query := strings.TrimSpace(a.modelDialog.query)
	if query == "" {
		query = "all"
	}
	header := "Select model (Enter choose, Esc close, ^F fav) - filter: " + query

	isFiltering := strings.TrimSpace(a.modelDialog.query) != ""

	// Build items with badges
	items := make([]pickerItem, len(models))
	for i, m := range models {
		label := modelLine(m)
		// Current model marker
		if displayModelName(m.ID) == a.model || m.ID == a.model {
			label = "● " + label
		}
		// Favorite star
		if a.modelPrefs != nil {
			ref := store.ModelRef{ProviderID: m.ProviderName(), ModelID: m.ID}
			if a.modelPrefs.IsFavorite(ref) {
				label = "★ " + label
			}
		}
		// Free badge
		if m.IsFree() {
			label += " [free]"
		}
		// Context length
		if ctx := formatContextLength(m.ContextLength); ctx != "" {
			label += " " + ctx
		}
		items[i] = pickerItem{Label: label}
	}

	// When filtering, use flat list (no section headers)
	if isFiltering || len(models) == 0 {
		return renderPickerList(header, items, a.modelDialog.selected, 9)
	}

	// When unfiltered, render with section headers
	var sb strings.Builder
	sb.WriteString(header)

	// Determine section boundaries
	type section struct {
		header string
		start  int
		end    int
	}
	var sections []section

	// Find favorites section
	favEnd := 0
	if a.modelPrefs != nil {
		for i, m := range models {
			ref := store.ModelRef{ProviderID: m.ProviderName(), ModelID: m.ID}
			if a.modelPrefs.IsFavorite(ref) {
				favEnd = i + 1
			} else {
				break
			}
		}
	}
	if favEnd > 0 {
		sections = append(sections, section{header: "── Favorites ──", start: 0, end: favEnd})
	}

	// Find recents section
	recentEnd := favEnd
	recentSet := make(map[string]bool)
	if a.modelPrefs != nil {
		for _, r := range a.modelPrefs.Recent {
			recentSet[r.ModelID] = true
		}
	}
	for i := favEnd; i < len(models); i++ {
		if recentSet[models[i].ID] {
			recentEnd = i + 1
		} else {
			break
		}
	}
	if recentEnd > favEnd {
		sections = append(sections, section{header: "── Recent ──", start: favEnd, end: recentEnd})
	}

	// Provider groups for remaining models
	restStart := recentEnd
	if restStart < len(models) {
		providerGroups := make(map[string][]int)
		var providerOrder []string
		for i := restStart; i < len(models); i++ {
			prov := models[i].ProviderName()
			if _, exists := providerGroups[prov]; !exists {
				providerOrder = append(providerOrder, prov)
			}
			providerGroups[prov] = append(providerGroups[prov], i)
		}
		sort.Strings(providerOrder)
		for _, prov := range providerOrder {
			indices := providerGroups[prov]
			sections = append(sections, section{
				header: "── " + prov + " ──",
				start:  indices[0],
				end:    indices[len(indices)-1] + 1,
			})
		}
	}

	// Render with windowing
	maxRows := 9
	selected := clamp(a.modelDialog.selected, 0, len(models)-1)
	start := 0
	if selected >= maxRows {
		start = selected - maxRows + 1
	}
	end := min(len(models), start+maxRows)

	sectionIdx := 0
	for i := start; i < end; i++ {
		// Check if we need a section header
		for sectionIdx < len(sections) && sections[sectionIdx].start <= i && i < sections[sectionIdx].end {
			if sections[sectionIdx].start == i {
				sb.WriteString("\n  ")
				sb.WriteString(sections[sectionIdx].header)
			}
			break
		}
		// Advance section index if past current section
		for sectionIdx < len(sections) && i >= sections[sectionIdx].end {
			sectionIdx++
		}
		// Check again after advancing
		if sectionIdx < len(sections) && sections[sectionIdx].start == i {
			sb.WriteString("\n  ")
			sb.WriteString(sections[sectionIdx].header)
		}

		prefix := "  "
		if i == selected {
			prefix = "> "
		}
		sb.WriteString("\n")
		sb.WriteString(prefix)
		sb.WriteString(items[i].Label)
	}
	if len(models) > end {
		sb.WriteString(fmt.Sprintf("\n  ... %d more", len(models)-end))
	}
	return sb.String()
}

func (a App) cycleRecentModel(reverse bool) (tea.Model, tea.Cmd) {
	if a.modelPrefs == nil || len(a.modelPrefs.Recent) < 2 {
		return a, a.ShowToast("No recent models to cycle", components.ToastWarning)
	}

	// Find current model in recents
	current := a.fullModelID
	idx := -1
	for i, r := range a.modelPrefs.Recent {
		if r.ModelID == current {
			idx = i
			break
		}
	}

	var next store.ModelRef
	if idx < 0 {
		// Current not in recents, go to first
		next = a.modelPrefs.Recent[0]
	} else if reverse {
		next = a.modelPrefs.Recent[(idx-1+len(a.modelPrefs.Recent))%len(a.modelPrefs.Recent)]
	} else {
		next = a.modelPrefs.Recent[(idx+1)%len(a.modelPrefs.Recent)]
	}

	a.switchModel(next.ModelID)
	return a, a.ShowToast("Model: "+displayModelName(next.ModelID), components.ToastInfo)
}

func (a App) handleVariantCycle() (tea.Model, tea.Cmd) {
	variants := provider.VariantsFor(a.fullModelID)
	if len(variants) == 0 {
		return a, a.ShowToast("No variants for this model", components.ToastWarning)
	}
	next := provider.CycleVariant(a.fullModelID, a.activeVariant)
	a.applyVariant(next)
	if a.modelPrefs != nil {
		a.modelPrefs.SetVariant(a.fullModelID, next)
		a.warnOnErr(a.modelPrefs.Save(), "variant")
	}
	if next == "" {
		return a, a.ShowToast("Variant: none", components.ToastInfo)
	}
	return a, a.ShowToast("Variant: "+next, components.ToastInfo)
}
