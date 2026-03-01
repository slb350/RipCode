package tui

import (
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/stephenbrandon/ripcode/internal/agent"
	"github.com/stretchr/testify/assert"
)

func TestApp_ContentDelta_ContinuesListening(t *testing.T) {
	app := NewApp()
	ch := make(chan agent.Event, 1)
	app.streaming = true
	app.state = StateSession
	app.eventCh = ch

	model, cmd := app.Update(AgentEventMsg{
		Event: agent.Event{Type: agent.EventContentDelta, Content: "hello"},
	})
	a := model.(App)

	assert.True(t, a.streaming)
	assert.NotNil(t, cmd, "non-terminal event must return a listen command")
}

func TestApp_ToolStart_ContinuesListening(t *testing.T) {
	app := NewApp()
	ch := make(chan agent.Event, 1)
	app.streaming = true
	app.state = StateSession
	app.eventCh = ch

	model, cmd := app.Update(AgentEventMsg{
		Event: agent.Event{
			Type: agent.EventToolStart,
			Tool: &agent.ToolEvent{ID: "1", Name: "bash", Args: `{"command":"ls"}`},
		},
	})
	a := model.(App)

	assert.True(t, a.streaming)
	assert.NotNil(t, cmd, "non-terminal event must return a listen command")
}

func TestApp_ToolEnd_ContinuesListening(t *testing.T) {
	app := NewApp()
	ch := make(chan agent.Event, 1)
	app.streaming = true
	app.state = StateSession
	app.eventCh = ch

	model, cmd := app.Update(AgentEventMsg{
		Event: agent.Event{
			Type: agent.EventToolEnd,
			Tool: &agent.ToolEvent{ID: "1", Name: "bash", Output: "file.txt"},
		},
	})
	a := model.(App)

	assert.True(t, a.streaming)
	assert.NotNil(t, cmd, "non-terminal event must return a listen command")
}

func TestApp_AgentEventDone(t *testing.T) {
	app := NewApp()
	ch := make(chan agent.Event, 1)
	app.streaming = true
	app.state = StateSession
	app.eventCh = ch

	model, cmd := app.Update(AgentEventMsg{
		Event: agent.Event{Type: agent.EventDone},
	})
	a := model.(App)

	assert.False(t, a.streaming)
	assert.Nil(t, a.eventCh, "Done must clear the event channel")
	assert.Nil(t, cmd, "terminal event must not return a command")
}

func TestApp_AgentEventError(t *testing.T) {
	app := NewApp()
	ch := make(chan agent.Event, 1)
	app.streaming = true
	app.state = StateSession
	app.eventCh = ch

	model, cmd := app.Update(AgentEventMsg{
		Event: agent.Event{
			Type:  agent.EventError,
			Error: assert.AnError,
		},
	})
	a := model.(App)

	assert.False(t, a.streaming)
	assert.Nil(t, a.eventCh, "Error must clear the event channel")
	assert.Nil(t, cmd, "terminal event must not return a command")
}

func TestApp_EscCancel_ClearsChannel(t *testing.T) {
	app := NewApp()
	ch := make(chan agent.Event, 1)
	app.streaming = true
	app.state = StateSession
	app.eventCh = ch
	app.cancel = func() {} // no-op cancel

	model, _ := app.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	a := model.(App)

	assert.False(t, a.streaming)
	assert.Nil(t, a.eventCh, "Esc cancel must clear the event channel")
}

// --- Modified Files Tracking tests ---

func TestApp_ModifiedFiles_TracksWriteEvent(t *testing.T) {
	app := NewApp()
	app.state = StateSession
	ch := make(chan agent.Event, 1)
	app.streaming = true
	app.eventCh = ch

	model, _ := app.Update(AgentEventMsg{
		Event: agent.Event{
			Type: agent.EventToolEnd,
			Tool: &agent.ToolEvent{
				ID:     "1",
				Name:   "write",
				Args:   `{"file_path":"/tmp/foo.go","content":"package main"}`,
				Output: "wrote /tmp/foo.go",
			},
		},
	})
	a := model.(App)
	assert.Contains(t, a.modifiedFiles, "/tmp/foo.go")
}

func TestApp_ModifiedFiles_TracksEditEvent(t *testing.T) {
	app := NewApp()
	app.state = StateSession
	ch := make(chan agent.Event, 1)
	app.streaming = true
	app.eventCh = ch

	model, _ := app.Update(AgentEventMsg{
		Event: agent.Event{
			Type: agent.EventToolEnd,
			Tool: &agent.ToolEvent{
				ID:     "1",
				Name:   "edit",
				Args:   `{"file_path":"/tmp/bar.go","old_string":"foo","new_string":"bar"}`,
				Output: "edited /tmp/bar.go",
			},
		},
	})
	a := model.(App)
	assert.Contains(t, a.modifiedFiles, "/tmp/bar.go")
}

func TestApp_ModifiedFiles_Deduplicates(t *testing.T) {
	app := NewApp()
	app.state = StateSession
	ch := make(chan agent.Event, 1)
	app.streaming = true
	app.eventCh = ch

	// First write
	model, _ := app.Update(AgentEventMsg{
		Event: agent.Event{
			Type: agent.EventToolEnd,
			Tool: &agent.ToolEvent{
				ID:     "1",
				Name:   "write",
				Args:   `{"file_path":"/tmp/foo.go"}`,
				Output: "ok",
			},
		},
	})
	a := model.(App)

	// Second write to same file
	a.eventCh = ch
	model, _ = a.Update(AgentEventMsg{
		Event: agent.Event{
			Type: agent.EventToolEnd,
			Tool: &agent.ToolEvent{
				ID:     "2",
				Name:   "write",
				Args:   `{"file_path":"/tmp/foo.go"}`,
				Output: "ok",
			},
		},
	})
	a = model.(App)

	count := 0
	for _, f := range a.modifiedFiles {
		if f == "/tmp/foo.go" {
			count++
		}
	}
	assert.Equal(t, 1, count)
}
