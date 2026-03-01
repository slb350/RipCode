package components

import (
	"testing"

	"github.com/stephenbrandon/ripcode/internal/agent"
	"github.com/stretchr/testify/assert"
)

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
