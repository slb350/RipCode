package components

// PromptHistory stores previously submitted prompts for recall with Up/Down.
type PromptHistory struct {
	prompts  []string
	position int // index into prompts; len(prompts) = at draft
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

// Push adds a prompt to history and resets navigation position.
// Consecutive duplicate prompts are deduplicated.
func (h *PromptHistory) Push(prompt string) {
	if len(h.prompts) > 0 && h.prompts[len(h.prompts)-1] == prompt {
		h.Reset()
		return
	}
	h.prompts = append(h.prompts, prompt)
	if len(h.prompts) > h.maxSize {
		h.prompts = h.prompts[len(h.prompts)-h.maxSize:]
	}
	h.Reset()
}

// Previous navigates to the older prompt.
func (h *PromptHistory) Previous() (string, bool) {
	if len(h.prompts) == 0 {
		return "", false
	}
	if h.position <= 0 {
		return "", false
	}
	h.position--
	return h.prompts[h.position], true
}

// Next navigates to the newer prompt or draft.
func (h *PromptHistory) Next() (string, bool) {
	if h.position >= len(h.prompts) {
		return "", false
	}
	h.position++
	if h.position >= len(h.prompts) {
		return h.draft, true
	}
	return h.prompts[h.position], true
}

// SaveDraft stores the current input text before history navigation.
func (h *PromptHistory) SaveDraft(text string) {
	h.draft = text
}

// Reset moves position back to the draft (newest).
func (h *PromptHistory) Reset() {
	h.position = len(h.prompts)
	h.draft = ""
}

// AtOldest reports whether we're at the oldest entry.
func (h *PromptHistory) AtOldest() bool {
	return h.position <= 0
}

// AtNewest reports whether we're at the draft position.
func (h *PromptHistory) AtNewest() bool {
	return h.position >= len(h.prompts)
}
