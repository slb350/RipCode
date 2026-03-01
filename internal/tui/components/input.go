package components

import (
	"strings"
	"unicode/utf8"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/stephenbrandon/ripcode/internal/tui/styles"
)

// InputSubmitMsg is sent when the user submits their input.
type InputSubmitMsg struct {
	Value string
}

// Input is a multi-line text input component with accent border and agent badge.
type Input struct {
	value   []string
	cursorX int
	cursorY int
	width   int
	height  int
	focused bool
	mode    string
	model   string
	theme   *styles.Theme
	history *EditHistory
	lastOp  inputOp // tracks last operation type for undo grouping
}

// inputOp classifies the last operation for undo grouping.
type inputOp int

const (
	opNone   inputOp = iota
	opInsert         // character insertion (grouped into one undo step)
	opOther          // any non-insert operation
)

// NewInput creates a new input component.
func NewInput() Input {
	return Input{
		value:   []string{""},
		focused: true,
		mode:    "build",
		model:   "",
		theme:   styles.DefaultTheme,
		history: NewEditHistory(100),
	}
}

// pushUndo snapshots current state onto the undo stack.
func (i *Input) pushUndo() {
	i.history.Push(EditState{
		Value:   i.Value(),
		CursorX: i.cursorX,
		CursorY: i.cursorY,
	})
}

// applyState restores an EditState to the input.
func (i *Input) applyState(s *EditState) {
	if s == nil {
		return
	}
	i.value = strings.Split(s.Value, "\n")
	if len(i.value) == 0 {
		i.value = []string{""}
	}
	i.cursorX = s.CursorX
	i.cursorY = s.CursorY
}

// SetSize updates the input dimensions.
func (i *Input) SetSize(width, height int) {
	i.width = width
	i.height = height
}

// SetMode sets the agent mode displayed in the badge.
func (i *Input) SetMode(mode string) { i.mode = mode }

// SetModel sets the model name displayed in the badge.
func (i *Input) SetModel(model string) { i.model = model }

// SetTheme sets the theme used for rendering.
func (i *Input) SetTheme(t *styles.Theme) { i.theme = t }

// Focus sets focus state.
func (i *Input) Focus() { i.focused = true }

// Blur removes focus.
func (i *Input) Blur() { i.focused = false }

// Value returns the current input text.
func (i Input) Value() string {
	return strings.Join(i.value, "\n")
}

// Reset clears the input.
func (i *Input) Reset() {
	i.value = []string{""}
	i.cursorX = 0
	i.cursorY = 0
}

// SetValue replaces the current input text and moves the cursor to the end.
func (i *Input) SetValue(v string) {
	i.value = strings.Split(v, "\n")
	if len(i.value) == 0 {
		i.value = []string{""}
	}
	i.cursorY = len(i.value) - 1
	i.cursorX = utf8.RuneCountInString(i.value[i.cursorY])
}

// CursorOffset returns the cursor rune offset in the full input string.
func (i Input) CursorOffset() int {
	offset := 0
	for y := 0; y < i.cursorY; y++ {
		offset += utf8.RuneCountInString(i.value[y]) + 1 // +1 for newline
	}
	offset += i.cursorX
	return offset
}

// SetCursorOffset moves the cursor to a rune offset in the full input string.
func (i *Input) SetCursorOffset(offset int) {
	text := i.Value()
	runeCount := utf8.RuneCountInString(text)
	if offset < 0 {
		offset = 0
	}
	if offset > runeCount {
		offset = runeCount
	}

	lines := strings.Split(text, "\n")
	if len(lines) == 0 {
		lines = []string{""}
	}
	i.value = lines

	remaining := offset
	for y, line := range lines {
		lineRunes := utf8.RuneCountInString(line)
		if remaining <= lineRunes {
			i.cursorY = y
			i.cursorX = remaining
			return
		}
		remaining -= lineRunes + 1
	}

	i.cursorY = len(lines) - 1
	i.cursorX = utf8.RuneCountInString(lines[i.cursorY])
}

// ReplaceRange replaces rune range [start,end) in the full input string
// and sets cursor after replacement.
func (i *Input) ReplaceRange(start, end int, replacement string) {
	runes := []rune(i.Value())
	if start < 0 {
		start = 0
	}
	if end < 0 {
		end = 0
	}
	if start > len(runes) {
		start = len(runes)
	}
	if end > len(runes) {
		end = len(runes)
	}
	if start > end {
		start, end = end, start
	}

	next := string(runes[:start]) + replacement + string(runes[end:])
	i.SetValue(next)
	i.SetCursorOffset(start + utf8.RuneCountInString(replacement))
}

// Update handles key events.
func (i *Input) Update(msg tea.Msg) tea.Cmd {
	if !i.focused {
		return nil
	}

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case msg.Code == '-' && msg.Mod == tea.ModCtrl:
			// Snapshot current state so we can redo back to it.
			i.history.PushIfChanged(EditState{
				Value: i.Value(), CursorX: i.cursorX, CursorY: i.cursorY,
			})
			s := i.history.Undo()
			if s != nil {
				i.applyState(s)
				i.lastOp = opOther
			}
			return nil

		case msg.Code == '.' && msg.Mod == tea.ModCtrl:
			s := i.history.Redo()
			if s != nil {
				i.applyState(s)
				i.lastOp = opOther
			}
			return nil

		case msg.Code == tea.KeyEnter && msg.Mod == 0:
			val := i.Value()
			if strings.TrimSpace(val) == "" {
				return nil
			}
			i.Reset()
			i.lastOp = opNone
			return func() tea.Msg { return InputSubmitMsg{Value: val} }

		case msg.Code == tea.KeyEnter && msg.Mod == tea.ModShift:
			i.pushUndo()
			i.lastOp = opOther
			runes := []rune(i.value[i.cursorY])
			before := string(runes[:i.cursorX])
			after := string(runes[i.cursorX:])

			newLines := make([]string, 0, len(i.value)+1)
			newLines = append(newLines, i.value[:i.cursorY]...)
			newLines = append(newLines, before)
			newLines = append(newLines, after)
			newLines = append(newLines, i.value[i.cursorY+1:]...)
			i.value = newLines
			i.cursorY++
			i.cursorX = 0

		case msg.Code == tea.KeyBackspace:
			i.pushUndo()
			i.lastOp = opOther
			if i.cursorX > 0 {
				runes := []rune(i.value[i.cursorY])
				i.value[i.cursorY] = string(runes[:i.cursorX-1]) + string(runes[i.cursorX:])
				i.cursorX--
			} else if i.cursorY > 0 {
				prevLine := i.value[i.cursorY-1]
				i.cursorX = utf8.RuneCountInString(prevLine)
				i.value[i.cursorY-1] = prevLine + i.value[i.cursorY]
				i.value = append(i.value[:i.cursorY], i.value[i.cursorY+1:]...)
				i.cursorY--
			}

		case msg.Code == tea.KeyLeft:
			if i.cursorX > 0 {
				i.cursorX--
			}

		case msg.Code == tea.KeyRight:
			if i.cursorX < utf8.RuneCountInString(i.value[i.cursorY]) {
				i.cursorX++
			}

		case msg.Code == tea.KeyUp:
			if i.cursorY > 0 {
				i.cursorY--
				if i.cursorX > utf8.RuneCountInString(i.value[i.cursorY]) {
					i.cursorX = utf8.RuneCountInString(i.value[i.cursorY])
				}
			}

		case msg.Code == tea.KeyDown:
			if i.cursorY < len(i.value)-1 {
				i.cursorY++
				if i.cursorX > utf8.RuneCountInString(i.value[i.cursorY]) {
					i.cursorX = utf8.RuneCountInString(i.value[i.cursorY])
				}
			}

		default:
			if msg.Text != "" {
				// Debounce: only push undo on first char after non-insert op.
				if i.lastOp != opInsert {
					i.pushUndo()
				}
				i.lastOp = opInsert
				runes := []rune(i.value[i.cursorY])
				i.value[i.cursorY] = string(runes[:i.cursorX]) + msg.Text + string(runes[i.cursorX:])
				i.cursorX += utf8.RuneCountInString(msg.Text)
			}
		}
	}

	return nil
}

// View renders the input with accent border, placeholder, badge, and hints.
func (i Input) View() string {
	t := i.theme
	if t == nil {
		t = styles.DefaultTheme
	}

	modeColor := t.ModeColor(i.mode)
	accentStyle := lipgloss.NewStyle().Foreground(modeColor)
	mutedStyle := t.TextMutedStyle

	var sb strings.Builder

	// Line 1: accent border + content (or placeholder)
	accent := accentStyle.Render("┃")
	hasContent := strings.TrimSpace(i.Value()) != ""
	if hasContent {
		sb.WriteString(accent + " " + i.renderContent())
	} else {
		sb.WriteString(accent + " " + mutedStyle.Render("What do you want to do?"))
		if i.focused {
			sb.WriteString("█")
		}
	}
	sb.WriteByte('\n')

	// Line 2: bottom cap
	sb.WriteString(accentStyle.Render("╹"))
	sb.WriteByte('\n')

	// Line 3: agent badge
	modeLabel := "Build"
	if len(i.mode) > 0 {
		modeLabel = strings.ToUpper(i.mode[:1]) + i.mode[1:]
	}
	badge := "    " + accentStyle.Render("▣") + " " + mutedStyle.Render(modeLabel)
	if i.model != "" {
		badge += mutedStyle.Render(" · " + i.model)
	}
	sb.WriteString(badge)
	sb.WriteByte('\n')

	// Line 4: keyboard hints
	sb.WriteString("    " + mutedStyle.Render("Enter send · Shift+Enter newline · / @ autocomplete · Ctrl+P commands · Tab agents · Esc cancel"))

	return sb.String()
}

// renderContent renders the multi-line text content with cursor.
func (i Input) renderContent() string {
	var sb strings.Builder
	for lineIdx, line := range i.value {
		if lineIdx > 0 {
			sb.WriteByte('\n')
			sb.WriteString("  ") // indent continuation lines
		}

		if i.focused && lineIdx == i.cursorY {
			runes := []rune(line)
			before := string(runes[:i.cursorX])
			after := ""
			if i.cursorX < len(runes) {
				after = string(runes[i.cursorX:])
			}
			sb.WriteString(before + "█" + after)
		} else {
			sb.WriteString(line)
		}
	}
	return sb.String()
}
