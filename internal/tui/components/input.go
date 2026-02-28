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
}

// NewInput creates a new input component.
func NewInput() Input {
	return Input{
		value:   []string{""},
		focused: true,
		mode:    "build",
		model:   "",
		theme:   styles.DefaultTheme,
	}
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

// Update handles key events.
func (i *Input) Update(msg tea.Msg) tea.Cmd {
	if !i.focused {
		return nil
	}

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case msg.Code == tea.KeyEnter && msg.Mod == 0:
			val := i.Value()
			if strings.TrimSpace(val) == "" {
				return nil
			}
			i.Reset()
			return func() tea.Msg { return InputSubmitMsg{Value: val} }

		case msg.Code == tea.KeyEnter && msg.Mod == tea.ModShift:
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
			if msg.Text != "" {
				line := i.value[i.cursorY]
				i.value[i.cursorY] = line[:i.cursorX] + msg.Text + line[i.cursorX:]
				i.cursorX += len(msg.Text)
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
	modeLabel := strings.ToUpper(i.mode[:1]) + i.mode[1:]
	badge := "    " + accentStyle.Render("▣") + " " + mutedStyle.Render(modeLabel)
	if i.model != "" {
		badge += mutedStyle.Render(" · " + i.model)
	}
	sb.WriteString(badge)
	sb.WriteByte('\n')

	// Line 4: keyboard hints
	sb.WriteString("    " + mutedStyle.Render("Enter send · Shift+Enter newline · Esc cancel"))

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
			before := line[:i.cursorX]
			after := ""
			if i.cursorX < len(line) {
				after = line[i.cursorX:]
			}
			sb.WriteString(before + "█" + after)
		} else {
			sb.WriteString(line)
		}
	}
	return sb.String()
}
