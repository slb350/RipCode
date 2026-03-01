package tui

import (
	"github.com/stephenbrandon/ripcode/internal/store"
	"github.com/stephenbrandon/ripcode/internal/tui/components"
)

const historyMaxSize = 200

func loadPromptHistory() (*components.PromptHistory, error) {
	h := components.NewPromptHistory(historyMaxSize)
	entries, err := store.LoadHistory()
	if err != nil {
		return h, err
	}
	if len(entries) == 0 {
		return h, nil
	}
	items := make([]components.HistoryItem, len(entries))
	for i, e := range entries {
		items[i] = components.HistoryItem{Prompt: e.Prompt, Mode: e.Mode}
	}
	h.LoadItems(items)

	// Best-effort compaction; if save fails, uncompacted history still works
	if len(entries) > historyMaxSize {
		kept := h.Items()
		compacted := make([]store.HistoryEntry, len(kept))
		for i, item := range kept {
			compacted[i] = store.HistoryEntry{Prompt: item.Prompt, Mode: item.Mode}
		}
		_ = store.SaveHistory(compacted)
	}
	return h, nil
}

func loadPromptStash() (*components.PromptStash, error) {
	s := components.NewPromptStash()
	entries, err := store.LoadStash()
	if err != nil {
		return s, err
	}
	if len(entries) == 0 {
		return s, nil
	}
	for _, e := range entries {
		s.PushWithID(e.ID, e.Content)
	}
	return s, nil
}

func persistStash(s *components.PromptStash) error {
	items := s.List()
	entries := make([]store.StashFileEntry, len(items))
	for i, item := range items {
		entries[i] = store.StashFileEntry{ID: item.ID, Content: item.Content}
	}
	return store.SaveStash(entries)
}

func persistHistory(h *components.PromptHistory) error {
	items := h.Items()
	if len(items) == 0 {
		return nil
	}
	last := items[len(items)-1]
	return store.AppendHistory(store.HistoryEntry{Prompt: last.Prompt, Mode: last.Mode})
}
