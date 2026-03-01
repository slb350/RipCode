package tui

import (
	"github.com/stephenbrandon/ripcode/internal/store"
	"github.com/stephenbrandon/ripcode/internal/tui/components"
)

const historyMaxSize = 200

func loadPromptHistory() *components.PromptHistory {
	h := components.NewPromptHistory(historyMaxSize)
	entries, err := store.LoadHistory()
	if err != nil || len(entries) == 0 {
		return h
	}
	items := make([]components.HistoryItem, len(entries))
	for i, e := range entries {
		items[i] = components.HistoryItem{Prompt: e.Prompt, Mode: e.Mode}
	}
	h.LoadItems(items)

	// Compact on load: if the file grew beyond maxSize, rewrite with only kept entries
	if len(entries) > historyMaxSize {
		kept := h.Items()
		compacted := make([]store.HistoryEntry, len(kept))
		for i, item := range kept {
			compacted[i] = store.HistoryEntry{Prompt: item.Prompt, Mode: item.Mode}
		}
		_ = store.SaveHistory(compacted)
	}
	return h
}

func loadPromptStash() *components.PromptStash {
	s := components.NewPromptStash()
	entries, err := store.LoadStash()
	if err != nil || len(entries) == 0 {
		return s
	}
	for _, e := range entries {
		s.PushWithID(e.ID, e.Content)
	}
	return s
}

func persistStash(s *components.PromptStash) {
	items := s.List()
	entries := make([]store.StashFileEntry, len(items))
	for i, item := range items {
		entries[i] = store.StashFileEntry{ID: item.ID, Content: item.Content}
	}
	_ = store.SaveStash(entries)
}

func persistHistory(h *components.PromptHistory) {
	items := h.Items()
	if len(items) == 0 {
		return
	}
	last := items[len(items)-1]
	_ = store.AppendHistory(store.HistoryEntry{Prompt: last.Prompt, Mode: last.Mode})
}
