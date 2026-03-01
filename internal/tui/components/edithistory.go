package components

// EditState captures the input state for undo/redo.
type EditState struct {
	Value   string
	CursorX int
	CursorY int
}

// EditHistory is a bounded undo/redo buffer.
type EditHistory struct {
	states  []EditState
	current int // index of the "current" state; -1 means empty
	maxSize int
}

// NewEditHistory creates an EditHistory with the given max capacity.
func NewEditHistory(maxSize int) *EditHistory {
	if maxSize < 1 {
		maxSize = 1
	}
	return &EditHistory{
		current: -1,
		maxSize: maxSize,
	}
}

// Push adds a new state, clearing any redo future.
func (h *EditHistory) Push(s EditState) {
	// Truncate any future states beyond current position.
	h.states = h.states[:h.current+1]

	h.states = append(h.states, s)
	h.current = len(h.states) - 1

	// Enforce max size by dropping oldest.
	if len(h.states) > h.maxSize {
		drop := len(h.states) - h.maxSize
		h.states = h.states[drop:]
		h.current = len(h.states) - 1
	}
}

// Undo returns the previous state, or nil if at the start.
func (h *EditHistory) Undo() *EditState {
	if h.current <= 0 {
		return nil
	}
	h.current--
	s := h.states[h.current]
	return &s
}

// Redo returns the next state, or nil if at the end.
func (h *EditHistory) Redo() *EditState {
	if h.current >= len(h.states)-1 {
		return nil
	}
	h.current++
	s := h.states[h.current]
	return &s
}

// CanUndo reports whether there is a previous state.
func (h *EditHistory) CanUndo() bool {
	return h.current > 0
}

// CanRedo reports whether there is a next state.
func (h *EditHistory) CanRedo() bool {
	return h.current < len(h.states)-1
}

// PushIfChanged pushes s only if it differs from the current top state.
func (h *EditHistory) PushIfChanged(s EditState) {
	if h.current >= 0 && h.current < len(h.states) {
		c := h.states[h.current]
		if c.Value == s.Value && c.CursorX == s.CursorX && c.CursorY == s.CursorY {
			return
		}
	}
	h.Push(s)
}

// Clear resets the history.
func (h *EditHistory) Clear() {
	h.states = nil
	h.current = -1
}
