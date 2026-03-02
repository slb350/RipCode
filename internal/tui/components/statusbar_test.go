package components

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStatusBar_View(t *testing.T) {
	sb := NewStatusBar()
	sb.SetSize(80)
	sb.SetTitle("Session sess-1234")
	sb.SetModel("test-model")
	sb.SetMode("build")
	sb.SetTokens(1500)

	view := sb.View()
	assert.Contains(t, view, "# Session sess-1234")
	assert.Contains(t, view, "build")
	assert.Contains(t, view, "test-model")
	assert.Contains(t, view, "1.5K")
}

func TestStatusBar_Spinning(t *testing.T) {
	sb := NewStatusBar()
	sb.SetSize(80)
	sb.SetSpinning(true)

	view := sb.View()
	assert.Contains(t, view, "●")
}

func TestStatusBar_NoHotkeys(t *testing.T) {
	sb := NewStatusBar()
	sb.SetSize(120)

	view := sb.View()
	assert.NotContains(t, view, "^C quit")
	assert.NotContains(t, view, "Esc cancel")
}

func TestStatusBar_NarrowStillShowsHeader(t *testing.T) {
	sb := NewStatusBar()
	sb.SetTitle("Session abc")
	sb.SetSize(30) // too narrow for hotkeys

	view := sb.View()
	assert.Contains(t, view, "Session abc")
}
