package tui

import (
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/stephenbrandon/ripcode/internal/agent"
	"github.com/stephenbrandon/ripcode/internal/session"
	"github.com/stephenbrandon/ripcode/internal/store"
	"github.com/stephenbrandon/ripcode/internal/tool"
	"github.com/stephenbrandon/ripcode/internal/tui/components"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestListenForEvents_ChannelClosedUnexpectedly(t *testing.T) {
	ch := make(chan agent.Event)
	close(ch) // Close immediately without sending EventDone

	cmd := listenForEvents(ch)
	msg := cmd()
	agentMsg, ok := msg.(AgentEventMsg)
	assert.True(t, ok, "should produce AgentEventMsg")
	assert.Equal(t, agent.EventDone, agentMsg.Event.Type, "closed channel should produce EventDone")
}

func TestApp_AgentEventDone_PersistsSession(t *testing.T) {
	t.Setenv("RIPCODE_DIR", t.TempDir())
	app := NewApp()
	app.SetProvider(&modelListProvider{})
	app.SetRegistry(tool.NewRegistry())
	sess := session.New(t.TempDir())
	sess.AddUser("hello")
	sess.AddAssistant("world", nil, nil)
	app.SetSession(sess)
	app.SetAgent(agent.BuildAgent())
	app.state = StateSession
	model, _ := app.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	a := model.(App)

	ch := make(chan agent.Event, 1)
	a.streaming = true
	a.eventCh = ch

	model, _ = a.Update(AgentEventMsg{
		Event: agent.Event{Type: agent.EventDone},
	})
	_ = model.(App)

	// Session should now be on disk
	loaded, err := store.Load(sess.ID)
	require.NoError(t, err, "session should have been persisted to disk on EventDone")
	require.NotNil(t, loaded)
	assert.Equal(t, sess.ID, loaded.ID)
}

func TestApp_UnknownEventType_GracefullyIgnored(t *testing.T) {
	t.Setenv("RIPCODE_DIR", t.TempDir())
	app := NewApp()
	ch := make(chan agent.Event, 1)
	app.streaming = true
	app.state = StateSession
	app.eventCh = ch

	// Use an event type value beyond the known range
	model, cmd := app.Update(AgentEventMsg{
		Event: agent.Event{Type: agent.EventType(99), Content: "future event"},
	})
	a := model.(App)

	// Should not panic, should not crash
	assert.True(t, a.streaming, "streaming should remain active after unknown event")
	assert.Nil(t, cmd, "unknown event should not return a command")
}

// --- Reasoning Event tests ---

func TestApp_ReasoningDelta_ContinuesListening(t *testing.T) {
	app := NewApp()
	ch := make(chan agent.Event, 1)
	app.streaming = true
	app.state = StateSession
	app.eventCh = ch

	model, cmd := app.Update(AgentEventMsg{
		Event: agent.Event{Type: agent.EventReasoningDelta, Content: "thinking"},
	})
	a := model.(App)

	assert.True(t, a.streaming)
	assert.NotNil(t, cmd, "non-terminal event must return a listen command")
}

func TestApp_ContentDelta_UsesStreamPart(t *testing.T) {
	app := NewApp()
	app.state = StateSession
	ch := make(chan agent.Event, 1)
	app.streaming = true
	app.eventCh = ch

	// Size the chat so rendering works
	app.chat.SetSize(80, 20)

	model, _ := app.Update(AgentEventMsg{
		Event: agent.Event{Type: agent.EventContentDelta, Content: "hello"},
	})
	a := model.(App)

	// Should show in view (via StreamPart, not legacy StreamContent)
	view := a.chat.View()
	assert.Contains(t, view, "hello")
}

func TestApp_ReasoningDelta_StreamsPart(t *testing.T) {
	app := NewApp()
	app.state = StateSession
	ch := make(chan agent.Event, 1)
	app.streaming = true
	app.eventCh = ch

	// Size the chat and enable thinking
	app.chat.SetSize(80, 20)
	app.chat.SetShowThinking(true)

	model, _ := app.Update(AgentEventMsg{
		Event: agent.Event{Type: agent.EventReasoningDelta, Content: "deep thought"},
	})
	a := model.(App)

	view := a.chat.View()
	assert.Contains(t, view, "deep thought")
}

func TestApp_ToolStart_SegmentsAssistantStreamAcrossToolPhase(t *testing.T) {
	t.Setenv("RIPCODE_DIR", t.TempDir())
	app := NewApp()
	app.state = StateSession
	ch := make(chan agent.Event, 1)
	app.streaming = true
	app.eventCh = ch

	model, _ := app.Update(AgentEventMsg{
		Event: agent.Event{Type: agent.EventContentDelta, Content: "before "},
	})
	a := model.(App)

	model, _ = a.Update(AgentEventMsg{
		Event: agent.Event{
			Type: agent.EventToolStart,
			Tool: &agent.ToolEvent{ID: "1", Name: "read", Args: `{"file_path":"README.md"}`},
		},
	})
	a = model.(App)

	model, _ = a.Update(AgentEventMsg{
		Event: agent.Event{Type: agent.EventContentDelta, Content: "after"},
	})
	a = model.(App)

	model, _ = a.Update(AgentEventMsg{
		Event: agent.Event{Type: agent.EventDone},
	})
	a = model.(App)

	var assistant []components.ChatEntry
	for _, e := range a.chat.Entries() {
		if e.Role == components.RoleAssistant {
			assistant = append(assistant, e)
		}
	}

	require.Len(t, assistant, 2)
	assert.Equal(t, "before ", assistant[0].Content)
	assert.Equal(t, "after", assistant[1].Content)
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
	assert.Contains(t, a.modifiedFiles.paths(), "/tmp/foo.go")
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
	assert.Contains(t, a.modifiedFiles.paths(), "/tmp/bar.go")
}

func TestApp_ModifiedFiles_FailedWrite_NotTracked(t *testing.T) {
	app := NewApp()
	app.state = StateSession
	ch := make(chan agent.Event, 1)
	app.streaming = true
	app.eventCh = ch

	model, _ := app.Update(AgentEventMsg{
		Event: agent.Event{
			Type: agent.EventToolEnd,
			Tool: &agent.ToolEvent{
				ID:    "1",
				Name:  "write",
				Args:  `{"file_path":"/tmp/fail.go","content":"package main"}`,
				Error: "permission denied",
			},
		},
	})
	a := model.(App)
	assert.Empty(t, a.modifiedFiles.paths(), "failed write should not be tracked")
}

func TestApp_ModifiedFiles_FailedEdit_NotTracked(t *testing.T) {
	app := NewApp()
	app.state = StateSession
	ch := make(chan agent.Event, 1)
	app.streaming = true
	app.eventCh = ch

	model, _ := app.Update(AgentEventMsg{
		Event: agent.Event{
			Type: agent.EventToolEnd,
			Tool: &agent.ToolEvent{
				ID:    "1",
				Name:  "edit",
				Args:  `{"file_path":"/tmp/fail.go","old_string":"a","new_string":"b"}`,
				Error: "no match found",
			},
		},
	})
	a := model.(App)
	assert.Empty(t, a.modifiedFiles.paths(), "failed edit should not be tracked")
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
	for _, f := range a.modifiedFiles.paths() {
		if f == "/tmp/foo.go" {
			count++
		}
	}
	assert.Equal(t, 1, count)
}
