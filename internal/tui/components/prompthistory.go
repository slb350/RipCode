package components

// HistoryItem is a prompt with its mode for persistence.
type HistoryItem struct {
	Prompt string
	Mode   string // "normal" or "shell"
}

// PromptHistory stores previously submitted prompts for recall with Up/Down.
type PromptHistory struct {
	items    []HistoryItem
	position int // index into items; len(items) = at draft
	draft    string
	maxSize  int
}

// NewPromptHistory creates a PromptHistory with the given max capacity.
func NewPromptHistory(maxSize int) *PromptHistory {
	if maxSize < 1 {
		maxSize = 1
	}
	return &PromptHistory{
		maxSize: maxSize,
	}
}

// Push adds a prompt to history with "normal" mode and resets navigation.
// Consecutive duplicate prompts are deduplicated.
func (h *PromptHistory) Push(prompt string) {
	h.PushWithMode(prompt, "normal")
}

// PushWithMode adds a prompt with the given mode and resets navigation.
// Consecutive duplicate prompts are deduplicated.
func (h *PromptHistory) PushWithMode(prompt, mode string) {
	if len(h.items) > 0 && h.items[len(h.items)-1].Prompt == prompt {
		h.Reset()
		return
	}
	h.items = append(h.items, HistoryItem{Prompt: prompt, Mode: mode})
	if len(h.items) > h.maxSize {
		h.items = h.items[len(h.items)-h.maxSize:]
	}
	h.Reset()
}

// Previous navigates to the older prompt.
func (h *PromptHistory) Previous() (string, bool) {
	if len(h.items) == 0 {
		return "", false
	}
	if h.position <= 0 {
		return "", false
	}
	h.position--
	return h.items[h.position].Prompt, true
}

// Next navigates to the newer prompt or draft.
func (h *PromptHistory) Next() (string, bool) {
	if h.position >= len(h.items) {
		return "", false
	}
	h.position++
	if h.position >= len(h.items) {
		return h.draft, true
	}
	return h.items[h.position].Prompt, true
}

// SaveDraft stores the current input text before history navigation.
func (h *PromptHistory) SaveDraft(text string) {
	h.draft = text
}

// Reset moves position back to the draft (newest).
func (h *PromptHistory) Reset() {
	h.position = len(h.items)
	h.draft = ""
}

// AtOldest reports whether we're at the oldest entry.
func (h *PromptHistory) AtOldest() bool {
	return h.position <= 0
}

// AtNewest reports whether we're at the draft position.
func (h *PromptHistory) AtNewest() bool {
	return h.position >= len(h.items)
}

// Items returns a copy of all history items for persistence.
func (h *PromptHistory) Items() []HistoryItem {
	out := make([]HistoryItem, len(h.items))
	copy(out, h.items)
	return out
}

// LoadItems replaces the history with the given items, respecting maxSize.
func (h *PromptHistory) LoadItems(items []HistoryItem) {
	if len(items) > h.maxSize {
		items = items[len(items)-h.maxSize:]
	}
	h.items = make([]HistoryItem, len(items))
	copy(h.items, items)
	h.Reset()
}
