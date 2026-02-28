package components

import (
	"math/rand/v2"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/stephenbrandon/ripcode/internal/tui/styles"
)

var logo = []string{
	"██████╗ ██╗██████╗ ",
	"██╔══██╗██║██╔══██╗",
	"██████╔╝██║██████╔╝",
	"██╔══██╗██║██╔═══╝ ",
	"██║  ██║██║██║     ",
	"╚═╝  ╚═╝╚═╝╚═╝     ",
}

var tips = []string{
	"Use Ctrl+L to clear the conversation",
	"Press Esc to cancel a streaming response",
	"Use Shift+Enter for multi-line input",
	"Ctrl+C quits at any time",
}

// Home renders the startup home screen with logo, input, tip, and footer.
type Home struct {
	width   int
	height  int
	input   Input
	workDir string
	version string
	tip     string
	theme   *styles.Theme
}

// NewHome creates a new home screen component.
func NewHome() Home {
	return Home{
		input:   NewInput(),
		version: "v0.1.0",
		tip:     tips[rand.IntN(len(tips))],
		theme:   styles.DefaultTheme,
	}
}

// SetSize updates the home screen dimensions.
func (h *Home) SetSize(width, height int) {
	h.width = width
	h.height = height
	h.input.SetSize(width, 5)
}

// SetWorkDir sets the working directory shown in the footer.
func (h *Home) SetWorkDir(dir string) { h.workDir = dir }

// SetVersion sets the version shown in the footer.
func (h *Home) SetVersion(v string) { h.version = v }

// SetMode sets the mode on the embedded input.
func (h *Home) SetMode(mode string) { h.input.SetMode(mode) }

// SetModel sets the model on the embedded input.
func (h *Home) SetModel(model string) { h.input.SetModel(model) }

// SetTheme sets the theme.
func (h *Home) SetTheme(t *styles.Theme) {
	h.theme = t
	h.input.SetTheme(t)
}

// Input returns a pointer to the embedded Input for event forwarding.
func (h *Home) Input() *Input { return &h.input }

// View renders the home screen.
func (h Home) View() string {
	t := h.theme
	if t == nil {
		t = styles.DefaultTheme
	}

	maxContentW := 75
	if h.width < maxContentW {
		maxContentW = h.width
	}

	primaryStyle := lipgloss.NewStyle().Foreground(t.ModeColor("build"))
	mutedStyle := t.TextMutedStyle

	var content []string

	// Logo
	for _, line := range logo {
		content = append(content, primaryStyle.Render(line))
	}

	// Subtitle
	content = append(content, mutedStyle.Render("       code"))
	content = append(content, "")

	// Input
	inputLines := strings.Split(h.input.View(), "\n")
	content = append(content, inputLines...)
	content = append(content, "")

	// Tip
	content = append(content, mutedStyle.Render("● Tip: "+h.tip))

	// Calculate vertical centering
	contentH := len(content)
	footerH := 1
	available := h.height - footerH
	topPad := (available - contentH) / 2
	if topPad < 1 {
		topPad = 1
	}

	// Build final view
	var sb strings.Builder

	// Top padding
	for i := 0; i < topPad; i++ {
		sb.WriteByte('\n')
	}

	// Centered content
	for _, line := range content {
		centered := centerLine(line, h.width, maxContentW)
		sb.WriteString(centered)
		sb.WriteByte('\n')
	}

	// Bottom padding to push footer down
	bottomPad := available - topPad - contentH
	for i := 0; i < bottomPad; i++ {
		sb.WriteByte('\n')
	}

	// Footer
	sb.WriteString(h.renderFooter(mutedStyle))

	return sb.String()
}

func (h Home) renderFooter(mutedStyle lipgloss.Style) string {
	left := h.workDir
	if left == "" {
		left = "."
	}
	right := "ripcode " + h.version

	gap := h.width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 1 {
		gap = 1
	}

	return mutedStyle.Render(left) + strings.Repeat(" ", gap) + mutedStyle.Render(right)
}

// centerLine horizontally centers a line within the given width.
func centerLine(line string, totalW, maxW int) string {
	lineW := lipgloss.Width(line)
	if lineW >= totalW {
		return line
	}
	leftPad := (totalW - min(lineW, maxW)) / 2
	if leftPad < 0 {
		leftPad = 0
	}
	return strings.Repeat(" ", leftPad) + line
}
