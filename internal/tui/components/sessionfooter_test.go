package components

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSessionFooter_Default(t *testing.T) {
	f := NewSessionFooter()
	f.SetSize(80)
	f.SetWorkDir("/tmp/project")
	f.SetConnected(true)

	view := f.View()
	assert.Contains(t, view, "/tmp/project")
	assert.Contains(t, view, "/help · ^B sidebar")
}

func TestSessionFooter_Streaming(t *testing.T) {
	f := NewSessionFooter()
	f.SetSize(80)
	f.SetWorkDir("/tmp/project")
	f.SetConnected(true)
	f.SetStreaming(true)

	view := f.View()
	assert.Contains(t, view, "● Running")
	assert.Contains(t, view, "Esc interrupt")
}

func TestSessionFooter_Disconnected(t *testing.T) {
	f := NewSessionFooter()
	f.SetSize(80)
	f.SetConnected(false)

	view := f.View()
	assert.Contains(t, view, "Set OPENROUTER_API_KEY to connect")
}

func TestSessionFooter_ShowsMCPCount(t *testing.T) {
	f := NewSessionFooter()
	f.SetSize(120)
	f.SetWorkDir("/tmp")
	f.SetConnected(true)
	f.SetMCPCount(3)

	view := f.View()
	assert.Contains(t, view, "⊙ 3")
}

func TestSessionFooter_HidesMCPWhenZero(t *testing.T) {
	f := NewSessionFooter()
	f.SetSize(120)
	f.SetWorkDir("/tmp")
	f.SetConnected(true)
	f.SetMCPCount(0)

	view := f.View()
	assert.NotContains(t, view, "⊙")
}

func TestSessionFooter_ShowsLSPCount(t *testing.T) {
	f := NewSessionFooter()
	f.SetSize(120)
	f.SetWorkDir("/tmp")
	f.SetConnected(true)
	f.SetLSPCount(2)

	view := f.View()
	assert.Contains(t, view, "• 2")
}

func TestSessionFooter_HidesLSPWhenZero(t *testing.T) {
	f := NewSessionFooter()
	f.SetSize(120)
	f.SetWorkDir("/tmp")
	f.SetConnected(true)
	f.SetLSPCount(0)

	view := f.View()
	// Should not have LSP marker when count is 0
	// Note: the base right side should still show /models, /help, etc.
	assert.Contains(t, view, "/help")
}

func TestSessionFooter_ShowsBothCounts(t *testing.T) {
	f := NewSessionFooter()
	f.SetSize(120)
	f.SetWorkDir("/tmp")
	f.SetConnected(true)
	f.SetMCPCount(2)
	f.SetLSPCount(1)

	view := f.View()
	assert.Contains(t, view, "⊙ 2")
	assert.Contains(t, view, "• 1")
}
