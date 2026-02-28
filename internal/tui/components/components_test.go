package components

import (
	"testing"

	"github.com/stephenbrandon/ripcode/internal/agent"
	"github.com/stretchr/testify/assert"
)

// --- Chat tests ---

func TestChat_AddEntry(t *testing.T) {
	c := NewChat()
	c.SetSize(80, 20)

	c.AddEntry(ChatEntry{Role: "user", Content: "hello"})
	view := c.View()
	assert.Contains(t, view, "hello")
}

func TestChat_StreamAndCommit(t *testing.T) {
	c := NewChat()
	c.SetSize(80, 20)

	c.StreamContent("Hello ")
	c.StreamContent("world")
	view := c.View()
	assert.Contains(t, view, "Hello world")

	c.CommitStream()
	view = c.View()
	assert.Contains(t, view, "Hello world")
}

func TestChat_Clear(t *testing.T) {
	c := NewChat()
	c.SetSize(80, 20)

	c.AddEntry(ChatEntry{Role: "user", Content: "hello"})
	c.Clear()

	view := c.View()
	assert.NotContains(t, view, "hello")
}

// --- Input tests ---

func TestInput_Value(t *testing.T) {
	i := NewInput()
	assert.Equal(t, "", i.Value())
}

func TestInput_Reset(t *testing.T) {
	i := NewInput()
	i.value = []string{"some text"}
	i.cursorX = 5
	i.Reset()
	assert.Equal(t, "", i.Value())
	assert.Equal(t, 0, i.cursorX)
}

func TestInput_View(t *testing.T) {
	i := NewInput()
	view := i.View()
	assert.Contains(t, view, ">")
}

// --- StatusBar tests ---

func TestStatusBar_View(t *testing.T) {
	sb := NewStatusBar()
	sb.SetSize(80)
	sb.SetModel("test-model")
	sb.SetMode("build")
	sb.SetTokens(1500)

	view := sb.View()
	assert.Contains(t, view, "ripcode")
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

// --- ToolPanel tests ---

func TestToolPanel_AddEvent(t *testing.T) {
	tp := NewToolPanel()
	tp.SetSize(80)

	tp.AddEvent(agent.ToolEvent{
		Name:   "bash",
		Output: "file1.go\nfile2.go",
	})

	view := tp.View()
	assert.Contains(t, view, "bash")
}

func TestToolPanel_ErrorEvent(t *testing.T) {
	tp := NewToolPanel()
	tp.SetSize(80)

	tp.AddEvent(agent.ToolEvent{
		Name:  "bash",
		Error: "command not found",
	})

	view := tp.View()
	assert.Contains(t, view, "command not found")
}

func TestToolPanel_Clear(t *testing.T) {
	tp := NewToolPanel()
	tp.AddEvent(agent.ToolEvent{Name: "bash", Output: "ok"})
	tp.Clear()

	view := tp.View()
	assert.Empty(t, view)
}
