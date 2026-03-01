package components

import (
	"strings"
	"sync/atomic"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/stephenbrandon/ripcode/internal/tui/styles"
)

// ToastVariant determines the visual style.
type ToastVariant string

const (
	ToastInfo    ToastVariant = "info"
	ToastSuccess ToastVariant = "success"
	ToastWarning ToastVariant = "warning"
	ToastError   ToastVariant = "error"
)

var toastIDCounter atomic.Int64

// Toast is a temporary notification.
type Toast struct {
	ID       int64
	Variant  ToastVariant
	Title    string
	Message  string
	Duration time.Duration
	Created  time.Time
}

// Expired reports whether the toast has exceeded its duration.
func (t Toast) Expired() bool {
	return time.Since(t.Created) >= t.Duration
}

// ToastManager manages the current toast.
type ToastManager struct {
	current *Toast
	nextID  int64
	width   int
	theme   *styles.Theme
}

// NewToastManager creates a ToastManager.
func NewToastManager() ToastManager {
	return ToastManager{theme: styles.DefaultTheme}
}

// SetWidth sets the render width.
func (m *ToastManager) SetWidth(w int) { m.width = w }

// Show displays a toast, replacing any existing one. Returns the toast ID.
func (m *ToastManager) Show(message string, variant ToastVariant, duration time.Duration) int64 {
	id := toastIDCounter.Add(1)
	m.current = &Toast{
		ID:       id,
		Variant:  variant,
		Message:  message,
		Duration: duration,
		Created:  time.Now(),
	}
	return id
}

// Dismiss removes the toast only if its ID matches.
func (m *ToastManager) Dismiss(id int64) {
	if m.current != nil && m.current.ID == id {
		m.current = nil
	}
}

// Current returns the current toast, or nil.
func (m *ToastManager) Current() *Toast { return m.current }

// View renders the toast. Returns empty string if no toast.
func (m ToastManager) View() string {
	if m.current == nil {
		return ""
	}

	t := m.theme
	if t == nil {
		t = styles.DefaultTheme
	}

	var style lipgloss.Style
	switch m.current.Variant {
	case ToastSuccess:
		style = t.SuccessStyle
	case ToastWarning:
		style = t.WarningStyle
	case ToastError:
		style = t.ErrorStyle
	default:
		style = t.TextMutedStyle
	}

	var sb strings.Builder
	if m.current.Title != "" {
		sb.WriteString(style.Bold(true).Render(m.current.Title))
		sb.WriteString(" ")
	}
	sb.WriteString(style.Render(m.current.Message))
	return sb.String()
}
