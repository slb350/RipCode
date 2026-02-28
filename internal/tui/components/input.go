package components

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/stephenbrandon/ripcode/internal/tui/styles"
)

// InputSubmitMsg is sent when the user submits their input.
type InputSubmitMsg struct {
	Value string
}

// Input is a multi-line text input component.
type Input struct {
	value   []string
	cursorX int
	cursorY int
	width   int
	height  int
	focused bool
	prompt  string
}

// NewInput creates a new input component.
func NewInput() Input {
	return Input{
		value:   []string{""},
		focused: true,
		prompt:  "> ",
	}
}

// SetSize updates the input dimensions.
func (i *Input) SetSize(width, height int) {
	i.width = width
	i.height = height
}

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

// Update handles key events.
func (i *Input) Update(msg tea.Msg) tea.Cmd {
	if !i.focused {
		return nil
	}

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case msg.Code == tea.KeyEnter && msg.Mod == 0:
			// Submit on Enter
			val := i.Value()
			if strings.TrimSpace(val) == "" {
				return nil
			}
			i.Reset()
			return func() tea.Msg { return InputSubmitMsg{Value: val} }

		case msg.Code == tea.KeyEnter && msg.Mod == tea.ModShift:
			// Newline on Shift+Enter
			line := i.value[i.cursorY]
			before := line[:i.cursorX]
			after := line[i.cursorX:]

			newLines := make([]string, 0, len(i.value)+1)
			newLines = append(newLines, i.value[:i.cursorY]...)
			newLines = append(newLines, before)
			newLines = append(newLines, after)
			newLines = append(newLines, i.value[i.cursorY+1:]...)
			i.value = newLines
			i.cursorY++
			i.cursorX = 0

		case msg.Code == tea.KeyBackspace:
			if i.cursorX > 0 {
				line := i.value[i.cursorY]
				i.value[i.cursorY] = line[:i.cursorX-1] + line[i.cursorX:]
				i.cursorX--
			} else if i.cursorY > 0 {
				// Join with previous line
				prevLine := i.value[i.cursorY-1]
				i.cursorX = len(prevLine)
				i.value[i.cursorY-1] = prevLine + i.value[i.cursorY]
				i.value = append(i.value[:i.cursorY], i.value[i.cursorY+1:]...)
				i.cursorY--
			}

		case msg.Code == tea.KeyLeft:
			if i.cursorX > 0 {
				i.cursorX--
			}

		case msg.Code == tea.KeyRight:
			if i.cursorX < len(i.value[i.cursorY]) {
				i.cursorX++
			}

		case msg.Code == tea.KeyUp:
			if i.cursorY > 0 {
				i.cursorY--
				if i.cursorX > len(i.value[i.cursorY]) {
					i.cursorX = len(i.value[i.cursorY])
				}
			}

		case msg.Code == tea.KeyDown:
			if i.cursorY < len(i.value)-1 {
				i.cursorY++
				if i.cursorX > len(i.value[i.cursorY]) {
					i.cursorX = len(i.value[i.cursorY])
				}
			}

		default:
			// Regular character input
			if msg.Text != "" {
				line := i.value[i.cursorY]
				i.value[i.cursorY] = line[:i.cursorX] + msg.Text + line[i.cursorX:]
				i.cursorX += len(msg.Text)
			}
		}
	}

	return nil
}

// View renders the input.
func (i Input) View() string {
	promptStyle := styles.Muted
	if i.focused {
		promptStyle = lipgloss.NewStyle().Foreground(styles.ColorGreen)
	}

	var sb strings.Builder
	for lineIdx, line := range i.value {
		prompt := "  "
		if lineIdx == 0 {
			prompt = promptStyle.Render(i.prompt)
		}

		if i.focused && lineIdx == i.cursorY {
			// Show cursor position
			before := line[:i.cursorX]
			after := ""
			if i.cursorX < len(line) {
				after = line[i.cursorX:]
			}
			sb.WriteString(prompt + before + "█" + after)
		} else {
			sb.WriteString(prompt + line)
		}

		if lineIdx < len(i.value)-1 {
			sb.WriteByte('\n')
		}
	}

	return sb.String()
}
