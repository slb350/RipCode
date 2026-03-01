package tui

import (
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/stephenbrandon/ripcode/internal/agent"
	"github.com/stephenbrandon/ripcode/internal/session"
	"github.com/stephenbrandon/ripcode/internal/tool"
	"github.com/stephenbrandon/ripcode/internal/tui/components"
	"github.com/stretchr/testify/assert"
)

func TestApp_EscQuits_InHome(t *testing.T) {
	app := NewApp()
	_, cmd := app.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	assert.NotNil(t, cmd)
}

func TestApp_CtrlCQuits_WhenInputEmpty(t *testing.T) {
	app := NewApp()
	_, cmd := app.Update(tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl})
	assert.NotNil(t, cmd, "ctrl+c with empty input should quit")
}

func TestApp_CtrlC_ClearsInputWhenNonEmpty(t *testing.T) {
	app := NewApp()
	app.SetProvider(&modelListProvider{})
	app.SetRegistry(tool.NewRegistry())
	app.SetSession(session.New(t.TempDir()))
	app.SetAgent(agent.BuildAgent())

	model, _ := app.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	a := model.(App)
	a.state = StateSession
	a.input.SetValue("some text")

	model, cmd := a.Update(tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl})
	a = model.(App)

	assert.Nil(t, cmd, "ctrl+c with non-empty input should not quit")
	assert.Equal(t, "", a.input.Value(), "ctrl+c should clear input")
}

func TestApp_CtrlC_QuitsWhenInputEmpty(t *testing.T) {
	app := NewApp()
	app.SetProvider(&modelListProvider{})
	app.SetRegistry(tool.NewRegistry())
	app.SetSession(session.New(t.TempDir()))
	app.SetAgent(agent.BuildAgent())

	model, _ := app.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	a := model.(App)
	a.state = StateSession
	// Input is empty

	_, cmd := a.Update(tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl})
	assert.NotNil(t, cmd, "ctrl+c with empty input should quit")
}

func TestApp_UpArrow_AtFirstLine_RecallsPreviousPrompt(t *testing.T) {
	app := NewApp()
	app.SetProvider(&modelListProvider{})
	app.SetRegistry(tool.NewRegistry())
	app.SetSession(session.New(t.TempDir()))
	app.SetAgent(agent.BuildAgent())

	model, _ := app.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	a := model.(App)

	model, _ = a.Update(components.InputSubmitMsg{Value: "/details"})
	a = model.(App)
	model, _ = a.Update(components.InputSubmitMsg{Value: "/new"})
	a = model.(App)

	// Up at first line recalls "/new"
	model, _ = a.Update(tea.KeyPressMsg{Code: tea.KeyUp})
	a = model.(App)
	assert.Equal(t, "/new", a.input.Value())

	// Up again recalls "/details"
	model, _ = a.Update(tea.KeyPressMsg{Code: tea.KeyUp})
	a = model.(App)
	assert.Equal(t, "/details", a.input.Value())
}

func TestApp_DownArrow_AtLastLine_RecallsNextOrDraft(t *testing.T) {
	app := NewApp()
	app.SetProvider(&modelListProvider{})
	app.SetRegistry(tool.NewRegistry())
	app.SetSession(session.New(t.TempDir()))
	app.SetAgent(agent.BuildAgent())

	model, _ := app.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	a := model.(App)

	model, _ = a.Update(components.InputSubmitMsg{Value: "/details"})
	a = model.(App)

	// Type something as draft
	a.input.SetValue("my draft")

	// Up to recall "/details"
	model, _ = a.Update(tea.KeyPressMsg{Code: tea.KeyUp})
	a = model.(App)
	assert.Equal(t, "/details", a.input.Value())

	// Down to restore draft
	model, _ = a.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	a = model.(App)
	assert.Equal(t, "my draft", a.input.Value())
}

func TestApp_UpArrow_AtMiddleLine_MovesUpNormally(t *testing.T) {
	app := NewApp()
	app.SetProvider(&modelListProvider{})
	app.SetRegistry(tool.NewRegistry())
	app.SetSession(session.New(t.TempDir()))
	app.SetAgent(agent.BuildAgent())

	model, _ := app.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	a := model.(App)
	a.state = StateSession

	// Multi-line input: cursor on line 1 (not first line)
	a.input.SetValue("line1\nline2")
	// SetValue moves cursor to end (line 1, pos 5)
	// Up should move cursor to line 0, not history
	model, _ = a.Update(tea.KeyPressMsg{Code: tea.KeyUp})
	a = model.(App)
	assert.Equal(t, "line1\nline2", a.input.Value(), "should not change value")
}

func TestApp_DownArrow_AtMiddleLine_MovesDownNormally(t *testing.T) {
	app := NewApp()
	app.SetProvider(&modelListProvider{})
	app.SetRegistry(tool.NewRegistry())
	app.SetSession(session.New(t.TempDir()))
	app.SetAgent(agent.BuildAgent())

	model, _ := app.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	a := model.(App)
	a.state = StateSession

	// Multi-line input: cursor on line 0 (not last line)
	a.input.SetValue("line1\nline2")
	a.input.SetCursorOffset(3) // mid line 0
	// Down should move to line 1, not history
	model, _ = a.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	a = model.(App)
	assert.Equal(t, "line1\nline2", a.input.Value(), "should not change value")
}

func TestApp_HistoryNavigation_SavesAndRestoresDraft(t *testing.T) {
	app := NewApp()
	app.SetProvider(&modelListProvider{})
	app.SetRegistry(tool.NewRegistry())
	app.SetSession(session.New(t.TempDir()))
	app.SetAgent(agent.BuildAgent())

	model, _ := app.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	a := model.(App)

	model, _ = a.Update(components.InputSubmitMsg{Value: "/details"})
	a = model.(App)

	a.input.SetValue("draft text")

	// Up: saves draft, shows /details
	model, _ = a.Update(tea.KeyPressMsg{Code: tea.KeyUp})
	a = model.(App)
	assert.Equal(t, "/details", a.input.Value())

	// Down: restores draft
	model, _ = a.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	a = model.(App)
	assert.Equal(t, "draft text", a.input.Value())
}

func TestApp_TabCyclesAgent(t *testing.T) {
	app := NewApp()
	app.SetProvider(&modelListProvider{})
	app.SetRegistry(tool.NewRegistry())
	app.SetSession(session.New(t.TempDir()))
	app.SetAgent(agent.BuildAgent())

	model, _ := app.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	a := model.(App)
	a.state = StateSession
	assert.Equal(t, "build", a.agent.Name)

	model, _ = a.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	a = model.(App)
	assert.Equal(t, "plan", a.agent.Name)

	model, _ = a.Update(tea.KeyPressMsg{Code: tea.KeyTab, Mod: tea.ModShift})
	a = model.(App)
	assert.Equal(t, "build", a.agent.Name)
}

func TestApp_CtrlK_KillsToEndOfLine_WhenInputFocused(t *testing.T) {
	app := NewApp()
	app.SetProvider(&modelListProvider{})
	app.SetRegistry(tool.NewRegistry())
	app.SetSession(session.New(t.TempDir()))
	app.SetAgent(agent.BuildAgent())

	model, _ := app.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	a := model.(App)
	a.state = StateSession

	// Type text and move cursor to middle
	a.input.SetValue("hello world")
	a.input.SetCursorOffset(5) // cursor after "hello"

	// Ctrl+K should kill to end of line, not open command palette
	model, _ = a.Update(tea.KeyPressMsg{Code: 'k', Mod: tea.ModCtrl})
	a = model.(App)

	assert.Equal(t, "hello", a.input.Value(), "Ctrl+K should kill to end of line")
	assert.False(t, a.commandPalette.open, "Ctrl+K should not open command palette when input has text")
}

func TestApp_CtrlK_OpensPalette_WhenInputEmpty(t *testing.T) {
	app := NewApp()
	app.SetProvider(&modelListProvider{})
	app.SetRegistry(tool.NewRegistry())
	app.SetSession(session.New(t.TempDir()))
	app.SetAgent(agent.BuildAgent())

	model, _ := app.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	a := model.(App)
	a.state = StateSession
	// Input is empty

	model, _ = a.Update(tea.KeyPressMsg{Code: 'k', Mod: tea.ModCtrl})
	a = model.(App)

	assert.True(t, a.commandPalette.open, "Ctrl+K should open palette when input is empty")
}

func TestApp_CommandPalette_OpensWithCtrlK(t *testing.T) {
	app := NewApp()
	app.SetProvider(&modelListProvider{})
	app.SetRegistry(tool.NewRegistry())
	app.SetSession(session.New(t.TempDir()))
	app.SetAgent(agent.BuildAgent())

	model, _ := app.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	a := model.(App)
	a.state = StateSession

	model, _ = a.Update(tea.KeyPressMsg{Code: 'k', Mod: tea.ModCtrl})
	a = model.(App)

	assert.True(t, a.commandPalette.open)
	assert.Contains(t, a.View().Content, "Commands (Ctrl+P/Ctrl+K")
}

func TestApp_CommandPalette_OpensWithCtrlP(t *testing.T) {
	app := NewApp()
	app.SetProvider(&modelListProvider{})
	app.SetRegistry(tool.NewRegistry())
	app.SetSession(session.New(t.TempDir()))
	app.SetAgent(agent.BuildAgent())

	model, _ := app.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	a := model.(App)
	a.state = StateSession

	model, _ = a.Update(tea.KeyPressMsg{Code: 'p', Mod: tea.ModCtrl})
	a = model.(App)

	assert.True(t, a.commandPalette.open)
	assert.Contains(t, a.View().Content, "Commands (Ctrl+P/Ctrl+K")
}

// --- Scroll navigation tests ---

func TestApp_PageUp_ScrollsChat(t *testing.T) {
	a := makeSessionApp(t)
	a.chat.ScrollToBottom()
	prev := a.chat.ScrollPos()
	model, _ := a.Update(tea.KeyPressMsg{Code: tea.KeyPgUp})
	a = model.(App)
	assert.Less(t, a.chat.ScrollPos(), prev)
}

func TestApp_PageDown_ScrollsChat(t *testing.T) {
	a := makeSessionApp(t)
	a.chat.ScrollToTop()
	model, _ := a.Update(tea.KeyPressMsg{Code: tea.KeyPgDown})
	a = model.(App)
	assert.Greater(t, a.chat.ScrollPos(), 0)
}

func TestApp_CtrlG_ScrollsToTop(t *testing.T) {
	a := makeSessionApp(t)
	a.chat.ScrollToBottom()
	model, _ := a.Update(tea.KeyPressMsg{Code: 'g', Mod: tea.ModCtrl})
	a = model.(App)
	assert.Equal(t, 0, a.chat.ScrollPos())
}

func TestApp_CtrlAltG_ScrollsToBottom(t *testing.T) {
	a := makeSessionApp(t)
	a.chat.ScrollToTop()
	model, _ := a.Update(tea.KeyPressMsg{Code: 'g', Mod: tea.ModCtrl | tea.ModAlt})
	a = model.(App)
	assert.Greater(t, a.chat.ScrollPos(), 0)
}

func TestApp_HomeKey_ScrollsToTop(t *testing.T) {
	a := makeSessionApp(t)
	a.chat.ScrollToBottom()
	model, _ := a.Update(tea.KeyPressMsg{Code: tea.KeyHome})
	a = model.(App)
	assert.Equal(t, 0, a.chat.ScrollPos())
}

func TestApp_EndKey_ScrollsToBottom(t *testing.T) {
	a := makeSessionApp(t)
	a.chat.ScrollToTop()
	model, _ := a.Update(tea.KeyPressMsg{Code: tea.KeyEnd})
	a = model.(App)
	assert.Greater(t, a.chat.ScrollPos(), 0)
}

func TestApp_NextUserMessage_Keybind(t *testing.T) {
	a := makeSessionAppWithHistory(t)
	a.chat.ScrollToTop()
	prevPos := a.chat.ScrollPos()
	// Ctrl+Alt+N for next user message
	model, _ := a.Update(tea.KeyPressMsg{Code: 'n', Mod: tea.ModCtrl | tea.ModAlt})
	a = model.(App)
	assert.GreaterOrEqual(t, a.chat.ScrollPos(), prevPos)
}

func TestApp_PrevUserMessage_Keybind(t *testing.T) {
	a := makeSessionAppWithHistory(t)
	a.chat.ScrollToBottom()
	prevPos := a.chat.ScrollPos()
	// Ctrl+Alt+P for prev user message
	model, _ := a.Update(tea.KeyPressMsg{Code: 'p', Mod: tea.ModCtrl | tea.ModAlt})
	a = model.(App)
	assert.LessOrEqual(t, a.chat.ScrollPos(), prevPos)
}

// --- Leader key prefix tests ---

func TestApp_CtrlX_SetsLeaderPending(t *testing.T) {
	a := makeSessionApp(t)
	model, _ := a.Update(tea.KeyPressMsg{Code: 'x', Mod: tea.ModCtrl})
	a = model.(App)
	assert.True(t, a.leaderPending)
}

func TestApp_LeaderPending_FollowedByKey_DispatchesLeader(t *testing.T) {
	a := makeSessionApp(t)
	// Press ctrl+x to enter leader mode
	model, _ := a.Update(tea.KeyPressMsg{Code: 'x', Mod: tea.ModCtrl})
	a = model.(App)
	assert.True(t, a.leaderPending)
	// Press 'a' — should clear leader pending (dispatch happens, regardless of handler)
	model, _ = a.Update(tea.KeyPressMsg{Code: 'a', Text: "a"})
	a = model.(App)
	assert.False(t, a.leaderPending)
}

func TestApp_LeaderPending_Escape_Cancels(t *testing.T) {
	a := makeSessionApp(t)
	model, _ := a.Update(tea.KeyPressMsg{Code: 'x', Mod: tea.ModCtrl})
	a = model.(App)
	assert.True(t, a.leaderPending)
	model, _ = a.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	a = model.(App)
	assert.False(t, a.leaderPending)
}

func TestApp_LeaderPending_UnrecognizedKey_Cancels(t *testing.T) {
	a := makeSessionApp(t)
	model, _ := a.Update(tea.KeyPressMsg{Code: 'x', Mod: tea.ModCtrl})
	a = model.(App)
	assert.True(t, a.leaderPending)
	// Press an unrecognized key like 'z'
	model, _ = a.Update(tea.KeyPressMsg{Code: 'z', Text: "z"})
	a = model.(App)
	assert.False(t, a.leaderPending)
}

func TestApp_LeaderPending_NotInDialog(t *testing.T) {
	a := makeSessionApp(t)
	a.helpDialog.open = true
	model, _ := a.Update(tea.KeyPressMsg{Code: 'x', Mod: tea.ModCtrl})
	a = model.(App)
	assert.False(t, a.leaderPending, "leader key should be ignored when dialog is open")
}

func TestApp_LeaderPending_StatusBarHint(t *testing.T) {
	a := makeSessionApp(t)
	model, _ := a.Update(tea.KeyPressMsg{Code: 'x', Mod: tea.ModCtrl})
	a = model.(App)
	view := a.View()
	assert.Contains(t, view.Content, "ctrl+x…")
}

func TestApp_LeaderPending_CmdBetweenKeys_StillWorks(t *testing.T) {
	a := makeSessionApp(t)
	// Press ctrl+x
	model, _ := a.Update(tea.KeyPressMsg{Code: 'x', Mod: tea.ModCtrl})
	a = model.(App)
	assert.True(t, a.leaderPending)
	// Simulate an unrelated msg (e.g. window resize) between ctrl+x and follow-up
	model, _ = a.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	a = model.(App)
	// Leader pending should survive non-key messages
	assert.True(t, a.leaderPending)
	// Now press the follow-up key
	model, _ = a.Update(tea.KeyPressMsg{Code: 'a', Text: "a"})
	a = model.(App)
	assert.False(t, a.leaderPending)
}

func TestApp_LeaderA_OpensAgentDialog(t *testing.T) {
	a := makeSessionApp(t)
	// ctrl+x then 'a'
	model, _ := a.Update(tea.KeyPressMsg{Code: 'x', Mod: tea.ModCtrl})
	a = model.(App)
	model, _ = a.Update(tea.KeyPressMsg{Code: 'a', Text: "a"})
	a = model.(App)
	assert.True(t, a.agentDialog.open)
}
