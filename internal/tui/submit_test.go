package tui

import (
	"strings"
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

func TestApp_SubmitPrompt_AddedToHistory(t *testing.T) {
	app := NewApp()
	app.SetProvider(&modelListProvider{})
	app.SetRegistry(tool.NewRegistry())
	app.SetSession(session.New(t.TempDir()))
	app.SetAgent(agent.BuildAgent())

	model, _ := app.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	a := model.(App)

	// Submit a slash command (doesn't start streaming, doesn't open dialog)
	model, _ = a.Update(components.InputSubmitMsg{Value: "/details"})
	a = model.(App)

	// Up arrow should recall "/details"
	a.input.SetValue("")
	model, _ = a.Update(tea.KeyPressMsg{Code: tea.KeyUp})
	a = model.(App)
	assert.Equal(t, "/details", a.input.Value())
}

// --- Shell Mode tests ---

func TestApp_ShellMode_ExclamationEntersShellMode(t *testing.T) {
	app := NewApp()
	app.SetProvider(&modelListProvider{})
	app.SetRegistry(tool.NewRegistry())
	app.SetSession(session.New(t.TempDir()))
	app.SetAgent(agent.BuildAgent())

	model, _ := app.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	a := model.(App)
	a.state = StateSession

	// Type "!"
	model, _ = a.Update(tea.KeyPressMsg{Text: "!"})
	a = model.(App)
	assert.True(t, a.shellMode, "typing ! should enter shell mode")
}

func TestApp_ShellMode_BadgeShowsShell(t *testing.T) {
	app := NewApp()
	app.SetProvider(&modelListProvider{})
	app.SetRegistry(tool.NewRegistry())
	app.SetSession(session.New(t.TempDir()))
	app.SetAgent(agent.BuildAgent())

	model, _ := app.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	a := model.(App)
	a.state = StateSession
	a.shellMode = true
	a.input.SetShellMode(true)

	view := a.View()
	assert.Contains(t, view.Content, "Shell")
}

func TestApp_ShellMode_BackspacePastBang_ExitsShellMode(t *testing.T) {
	app := NewApp()
	app.SetProvider(&modelListProvider{})
	app.SetRegistry(tool.NewRegistry())
	app.SetSession(session.New(t.TempDir()))
	app.SetAgent(agent.BuildAgent())

	model, _ := app.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	a := model.(App)
	a.state = StateSession

	// Type "!" then backspace
	model, _ = a.Update(tea.KeyPressMsg{Text: "!"})
	a = model.(App)
	assert.True(t, a.shellMode)

	model, _ = a.Update(tea.KeyPressMsg{Code: tea.KeyBackspace})
	a = model.(App)
	assert.False(t, a.shellMode, "backspacing past ! should exit shell mode")
}

func TestApp_ShellMode_ExclamationMidText_NoShellMode(t *testing.T) {
	app := NewApp()
	app.SetProvider(&modelListProvider{})
	app.SetRegistry(tool.NewRegistry())
	app.SetSession(session.New(t.TempDir()))
	app.SetAgent(agent.BuildAgent())

	model, _ := app.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	a := model.(App)
	a.state = StateSession

	// Type "hello!" — should NOT enter shell mode
	a.input.SetValue("hello")
	model, _ = a.Update(tea.KeyPressMsg{Text: "!"})
	a = model.(App)
	assert.False(t, a.shellMode, "! mid-text should not enter shell mode")
}

func TestApp_ShellMode_EmptyCommand_ShowsErrorToast(t *testing.T) {
	app := NewApp()
	app.SetProvider(&modelListProvider{})
	app.SetRegistry(tool.NewRegistry())
	app.SetSession(session.New(t.TempDir()))
	app.SetAgent(agent.BuildAgent())

	model, _ := app.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	a := model.(App)
	a.state = StateSession
	a.shellMode = true

	model, cmd := a.Update(components.InputSubmitMsg{Value: "!"})
	a = model.(App)
	assert.NotNil(t, cmd, "empty shell command should return toast dismiss cmd")
	assert.NotNil(t, a.toasts.Current(), "should show error toast")
	assert.False(t, a.shellMode, "shell mode should be cleared after submit")
}

func TestApp_ShellMode_ClearsShellModeAfterSubmit(t *testing.T) {
	app := NewApp()
	app.SetProvider(&modelListProvider{})
	reg := tool.NewRegistry()
	app.SetRegistry(reg)
	app.SetSession(session.New(t.TempDir()))
	app.SetAgent(agent.BuildAgent())

	model, _ := app.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	a := model.(App)
	a.state = StateSession
	a.shellMode = true

	model, _ = a.Update(components.InputSubmitMsg{Value: "!echo hello"})
	a = model.(App)
	assert.False(t, a.shellMode, "shell mode should be cleared after submit")
}

func TestApp_ShellMode_Submit_AddsToHistory(t *testing.T) {
	app := NewApp()
	app.SetProvider(&modelListProvider{})
	app.SetRegistry(tool.NewRegistry())
	app.SetSession(session.New(t.TempDir()))
	app.SetAgent(agent.BuildAgent())

	model, _ := app.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	a := model.(App)
	a.state = StateSession
	a.shellMode = true

	model, _ = a.Update(components.InputSubmitMsg{Value: "!echo test"})
	a = model.(App)

	// Shell result clears streaming and refocuses input
	model, _ = a.Update(ShellResultMsg{Command: "echo test", Output: "test"})
	a = model.(App)

	// Up arrow should recall "!echo test"
	model, _ = a.Update(tea.KeyPressMsg{Code: tea.KeyUp})
	a = model.(App)
	assert.Equal(t, "!echo test", a.input.Value())
}

// --- Command Registry Integration tests ---

func TestApp_SlashCompact_EmptySession_ShowsWarning(t *testing.T) {
	t.Setenv("RIPCODE_DIR", t.TempDir())
	app := NewApp()
	app.SetProvider(&modelListProvider{})
	app.SetRegistry(tool.NewRegistry())
	app.SetSession(session.New(t.TempDir()))
	app.SetAgent(agent.BuildAgent())

	model, _ := app.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	a := model.(App)
	a.state = StateSession

	model, cmd := a.Update(components.InputSubmitMsg{Value: "/compact"})
	a = model.(App)
	assert.NotNil(t, cmd, "/compact should return toast dismiss cmd")
	assert.NotNil(t, a.toasts.Current())
	assert.Contains(t, a.toasts.Current().Message, "Nothing to compact")
}

func TestApp_SlashDetails_TogglesShowDetails(t *testing.T) {
	app := NewApp()
	app.SetProvider(&modelListProvider{})
	app.SetRegistry(tool.NewRegistry())
	app.SetSession(session.New(t.TempDir()))
	app.SetAgent(agent.BuildAgent())

	model, _ := app.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	a := model.(App)
	a.state = StateSession

	assert.False(t, a.chat.ShowDetails())
	model, _ = a.Update(components.InputSubmitMsg{Value: "/details"})
	a = model.(App)
	assert.True(t, a.chat.ShowDetails())
	model, _ = a.Update(components.InputSubmitMsg{Value: "/details"})
	a = model.(App)
	assert.False(t, a.chat.ShowDetails())
}

func TestApp_SlashThinking_TogglesShowThinking(t *testing.T) {
	app := NewApp()
	app.SetProvider(&modelListProvider{})
	app.SetRegistry(tool.NewRegistry())
	app.SetSession(session.New(t.TempDir()))
	app.SetAgent(agent.BuildAgent())

	model, _ := app.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	a := model.(App)
	a.state = StateSession

	assert.False(t, a.chat.ShowThinking())
	model, _ = a.Update(components.InputSubmitMsg{Value: "/thinking"})
	a = model.(App)
	assert.True(t, a.chat.ShowThinking())
}

func TestApp_SlashTimestamps_TogglesShowTimestamps(t *testing.T) {
	app := NewApp()
	app.SetProvider(&modelListProvider{})
	app.SetRegistry(tool.NewRegistry())
	app.SetSession(session.New(t.TempDir()))
	app.SetAgent(agent.BuildAgent())

	model, _ := app.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	a := model.(App)
	a.state = StateSession

	assert.False(t, a.chat.ShowTimestamps())
	model, _ = a.Update(components.InputSubmitMsg{Value: "/timestamps"})
	a = model.(App)
	assert.True(t, a.chat.ShowTimestamps())
}

func TestApp_SlashRename_OpensDialog(t *testing.T) {
	app := NewApp()
	app.SetProvider(&modelListProvider{})
	app.SetRegistry(tool.NewRegistry())
	app.SetSession(session.New(t.TempDir()))
	app.SetAgent(agent.BuildAgent())

	model, _ := app.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	a := model.(App)
	a.state = StateSession

	model, _ = a.Update(components.InputSubmitMsg{Value: "/rename"})
	a = model.(App)
	assert.True(t, a.renameDialog.open)
}

func TestApp_UnknownSlashCommand_ShowsError(t *testing.T) {
	app := NewApp()
	app.SetProvider(&modelListProvider{})
	app.SetRegistry(tool.NewRegistry())
	app.SetSession(session.New(t.TempDir()))
	app.SetAgent(agent.BuildAgent())

	model, _ := app.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	a := model.(App)

	// Unknown slash commands should be handled locally with an error.
	model, _ = a.Update(components.InputSubmitMsg{Value: "/modelsxyz"})
	a = model.(App)
	assert.False(t, a.streaming)
	assert.Contains(t, a.View().Content, "Unknown command")
}

func TestApp_AgentSlashCommand_SwitchesMode(t *testing.T) {
	app := NewApp()
	app.SetProvider(&modelListProvider{})
	app.SetRegistry(tool.NewRegistry())
	app.SetSession(session.New(t.TempDir()))
	app.SetAgent(agent.BuildAgent())

	model, _ := app.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	a := model.(App)

	model, _ = a.Update(components.InputSubmitMsg{Value: "/agent plan"})
	a = model.(App)

	assert.Equal(t, "plan", a.agent.Name)
	assert.Contains(t, a.View().Content, `Agent switched to "plan".`)
}

func TestApp_AgentSlashCommand_NoArgs_OpensDialog(t *testing.T) {
	app := NewApp()
	app.SetProvider(&modelListProvider{})
	app.SetRegistry(tool.NewRegistry())
	app.SetSession(session.New(t.TempDir()))
	app.SetAgent(agent.BuildAgent())

	model, _ := app.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	a := model.(App)

	model, _ = a.Update(components.InputSubmitMsg{Value: "/agent"})
	a = model.(App)

	assert.True(t, a.agentDialog.open, "/agent with no args should open agent picker dialog")
}

func TestApp_ClearCommand_ResetsSession(t *testing.T) {
	workDir := t.TempDir()
	app := NewApp()
	app.SetProvider(&modelListProvider{})
	app.SetRegistry(tool.NewRegistry())
	sess := session.New(workDir)
	sess.AddUser("hello")
	sess.AddTokens(500, 200)
	app.SetSession(sess)
	app.SetAgent(agent.BuildAgent())

	model, _ := app.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	a := model.(App)

	oldID := a.session.ID
	model, _ = a.Update(components.InputSubmitMsg{Value: "/clear"})
	a = model.(App)

	assert.Empty(t, a.session.Records(), "/clear should reset session messages")
	assert.Equal(t, 0, a.session.TokenCount().Input, "/clear should reset token count")
	assert.Equal(t, 0, a.session.TokenCount().Output)
	assert.NotEqual(t, oldID, a.session.ID, "/clear should generate new session ID")
	assert.Contains(t, a.View().Content, "Conversation cleared.")
}

func TestApp_NewCommand_ResetsSession(t *testing.T) {
	workDir := t.TempDir()
	app := NewApp()
	app.SetProvider(&modelListProvider{})
	app.SetRegistry(tool.NewRegistry())
	sess := session.New(workDir)
	sess.AddUser("hello")
	sess.AddTokens(1000, 300)
	app.SetSession(sess)
	app.SetAgent(agent.BuildAgent())

	model, _ := app.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	a := model.(App)

	model, _ = a.Update(components.InputSubmitMsg{Value: "/new"})
	a = model.(App)

	assert.Empty(t, a.session.Records())
	assert.Equal(t, 0, a.session.TokenCount().Input)
	assert.Contains(t, a.View().Content, "Conversation cleared.")
}

func TestApp_ExitCommand_Quits(t *testing.T) {
	app := NewApp()
	app.SetProvider(&modelListProvider{})
	app.SetRegistry(tool.NewRegistry())
	app.SetSession(session.New(t.TempDir()))
	app.SetAgent(agent.BuildAgent())

	model, _ := app.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	a := model.(App)

	_, cmd := a.Update(components.InputSubmitMsg{Value: "/exit"})
	assert.NotNil(t, cmd, "/exit should return a quit command")

	_, cmd = a.Update(components.InputSubmitMsg{Value: "/quit"})
	assert.NotNil(t, cmd, "/quit should return a quit command")

	_, cmd = a.Update(components.InputSubmitMsg{Value: "/q"})
	assert.NotNil(t, cmd, "/q should return a quit command")
}

// --- Copy + Compact tests ---

func TestApp_CopyCommand_NoAssistant_ShowsWarning(t *testing.T) {
	a := makeSessionApp(t)
	// No assistant messages in chat
	a.chat.Clear()
	model, cmd := a.Update(components.InputSubmitMsg{Value: "/copy"})
	a = model.(App)
	assert.NotNil(t, cmd)
	assert.NotNil(t, a.toasts.Current())
	assert.Contains(t, a.toasts.Current().Message, "No assistant response")
}

func TestApp_CopyCommand_ShowsSuccessToast(t *testing.T) {
	a := makeSessionApp(t)
	a.chat.Clear()
	a.chat.AddEntry(components.ChatEntry{Role: components.RoleAssistant, Content: "Hello world"})
	model, cmd := a.Update(components.InputSubmitMsg{Value: "/copy"})
	a = model.(App)
	// May succeed or fail depending on clipboard availability in test env
	assert.NotNil(t, cmd)
	assert.NotNil(t, a.toasts.Current())
}

func TestApp_CopyCommand_MixedPartsNotTreatedAsEmpty(t *testing.T) {
	a := makeSessionApp(t)
	a.chat.Clear()
	a.chat.StreamPart(components.PartReasoning, "thinking")
	a.chat.StreamPart(components.PartText, "visible answer")
	a.chat.CommitStream()

	model, cmd := a.Update(components.InputSubmitMsg{Value: "/copy"})
	a = model.(App)
	assert.NotNil(t, cmd)
	toast := a.toasts.Current()
	assert.NotNil(t, toast)
	assert.NotContains(t, toast.Message, "No assistant response")
}

func TestApp_CopyCommand_AllReasoningEntry_IsCopyable(t *testing.T) {
	a := makeSessionApp(t)
	a.chat.Clear()
	// Add an entry with only reasoning parts (no text parts)
	a.chat.AddEntry(components.ChatEntry{
		Role: components.RoleAssistant,
		Parts: []components.MessagePart{
			{Type: components.PartReasoning, Content: "deep reasoning only"},
		},
	})

	model, cmd := a.Update(components.InputSubmitMsg{Value: "/copy"})
	a = model.(App)
	assert.NotNil(t, cmd)
	toast := a.toasts.Current()
	assert.NotNil(t, toast)
	// Should NOT warn about no response — CopyableContent() falls back to parts
	assert.NotContains(t, toast.Message, "No assistant response")
}

func TestApp_CompactCommand_ShowsToast(t *testing.T) {
	a := makeSessionAppWithHistory(t)
	model, _ := a.Update(components.InputSubmitMsg{Value: "/compact"})
	a = model.(App)
	toast := a.toasts.Current()
	assert.NotNil(t, toast)
	assert.Contains(t, toast.Message, "Compacted")
}

func TestApp_CompactCommand_ReducesMessages(t *testing.T) {
	a := makeSessionAppWithHistory(t)
	assert.Len(t, a.session.Records(), 4)
	model, _ := a.Update(components.InputSubmitMsg{Value: "/compact"})
	a = model.(App)
	// After compact, session should have fewer messages (just the summary)
	assert.Less(t, a.session.Len(), 4)
}

func TestApp_CompactCommand_PersistsSession(t *testing.T) {
	t.Setenv("RIPCODE_DIR", t.TempDir())
	a := makeSessionAppWithHistory(t)
	require.NoError(t, store.Save(a.session))
	origID := a.session.ID

	model, _ := a.Update(components.InputSubmitMsg{Value: "/compact"})
	a = model.(App)
	require.Len(t, a.session.Records(), 2)

	loaded, err := store.Load(origID)
	require.NoError(t, err)
	assert.Len(t, loaded.Records(), 2, "compacted session should be persisted to disk")
}

// --- Editor command tests ---

func TestApp_EditorCommand_NoEditorVar_ShowsWarning(t *testing.T) {
	a := makeSessionApp(t)
	t.Setenv("EDITOR", "")
	t.Setenv("VISUAL", "")
	model, _ := a.Update(components.InputSubmitMsg{Value: "/editor"})
	a = model.(App)
	toast := a.toasts.Current()
	assert.NotNil(t, toast)
	assert.Contains(t, toast.Message, "EDITOR")
}

// --- Skills command tests ---

func TestApp_SkillsCommand_ShowsToolList(t *testing.T) {
	a := makeSessionApp(t)
	model, _ := a.Update(components.InputSubmitMsg{Value: "/skills"})
	a = model.(App)
	// Should add entries to chat listing tools
	found := false
	for _, e := range a.chat.Entries() {
		if e.Role == components.RoleSystem && strings.Contains(e.Content, "Available tools") {
			found = true
			break
		}
	}
	assert.True(t, found, "should show available tools")
}

// --- Stash command tests ---

func TestApp_StashCommand_SavesAndClearsInput(t *testing.T) {
	app := makeSessionApp(t)
	app.input.SetValue("my draft prompt")
	model, _ := app.Update(components.InputSubmitMsg{Value: "/stash"})
	a := model.(App)
	// Stash should have one entry
	assert.Len(t, a.stash.List(), 1)
	assert.Equal(t, "my draft prompt", a.stash.List()[0].Content)
}

func TestApp_StashCommand_EmptyInput_ShowsWarning(t *testing.T) {
	app := makeSessionApp(t)
	// input is empty by default
	model, _ := app.Update(components.InputSubmitMsg{Value: "/stash"})
	a := model.(App)
	toast := a.toasts.Current()
	assert.NotNil(t, toast)
	assert.Contains(t, toast.Message, "Nothing to stash")
}

func TestApp_StashCommand_UsesSlashArgsWhenInputCleared(t *testing.T) {
	app := makeSessionApp(t)
	// Simulate submitted slash command after input reset.
	model, _ := app.Update(components.InputSubmitMsg{Value: "/stash draft from slash args"})
	a := model.(App)
	assert.Len(t, a.stash.List(), 1)
	assert.Equal(t, "draft from slash args", a.stash.List()[0].Content)
}

func TestApp_StashPopCommand_RestoresToInput(t *testing.T) {
	app := makeSessionApp(t)
	app.stash.Push("saved prompt")
	model, _ := app.Update(components.InputSubmitMsg{Value: "/stash-pop"})
	a := model.(App)
	assert.Equal(t, "saved prompt", a.input.Value())
}

func TestApp_StashPopCommand_EmptyStash_ShowsWarning(t *testing.T) {
	app := makeSessionApp(t)
	model, _ := app.Update(components.InputSubmitMsg{Value: "/stash-pop"})
	a := model.(App)
	toast := a.toasts.Current()
	assert.NotNil(t, toast)
	assert.Contains(t, toast.Message, "Stash is empty")
}

// --- Undo/Redo tests ---

func TestApp_UndoCommand_RevertsLastExchange(t *testing.T) {
	a := makeSessionAppWithHistory(t)
	assert.Len(t, a.session.Records(), 4)
	model, _ := a.Update(components.InputSubmitMsg{Value: "/undo"})
	a = model.(App)
	assert.Len(t, a.session.Records(), 2)
}

func TestApp_UndoCommand_RestoresPromptToInput(t *testing.T) {
	a := makeSessionAppWithHistory(t)
	model, _ := a.Update(components.InputSubmitMsg{Value: "/undo"})
	a = model.(App)
	assert.Equal(t, "second question", a.input.Value())
}

func TestApp_RedoCommand_RestoresRevertedMessages(t *testing.T) {
	a := makeSessionAppWithHistory(t)
	model, _ := a.Update(components.InputSubmitMsg{Value: "/undo"})
	a = model.(App)
	assert.Len(t, a.session.Records(), 2)
	model, _ = a.Update(components.InputSubmitMsg{Value: "/redo"})
	a = model.(App)
	assert.Len(t, a.session.Records(), 4)
}

func TestApp_RedoCommand_DisabledWhenNoRevert(t *testing.T) {
	a := makeSessionAppWithHistory(t)
	// No prior undo — redo should show warning
	model, _ := a.Update(components.InputSubmitMsg{Value: "/redo"})
	a = model.(App)
	// Session unchanged
	assert.Len(t, a.session.Records(), 4)
	// Should have a warning toast
	toast := a.toasts.Current()
	assert.NotNil(t, toast)
	assert.Contains(t, toast.Message, "Nothing to redo")
}

func TestApp_UndoCommand_EmptySession_ShowsWarning(t *testing.T) {
	app := makeSessionApp(t)
	app.session.ClearMessages()
	app.chat.Clear()
	model, _ := app.Update(components.InputSubmitMsg{Value: "/undo"})
	a := model.(App)
	toast := a.toasts.Current()
	assert.NotNil(t, toast)
	assert.Contains(t, toast.Message, "Nothing to undo")
}

func TestApp_UndoCommand_BlockedWhileStreaming(t *testing.T) {
	a := makeSessionAppWithHistory(t)
	a.streaming = true
	model, _ := a.Update(components.InputSubmitMsg{Value: "/undo"})
	a = model.(App)
	assert.Len(t, a.session.Records(), 4, "messages should not be reverted while streaming")
	toast := a.toasts.Current()
	assert.NotNil(t, toast)
	assert.Contains(t, toast.Message, "busy")
}

func TestApp_UndoCommand_AddsRevertMarkerToChat(t *testing.T) {
	a := makeSessionAppWithHistory(t)
	entriesBefore := len(a.chat.Entries())
	model, _ := a.Update(components.InputSubmitMsg{Value: "/undo"})
	a = model.(App)
	// After undo, chat is rebuilt with a revert marker
	found := false
	for _, e := range a.chat.Entries() {
		if e.Role == components.RoleSystem && strings.Contains(e.Content, "reverted") {
			found = true
			break
		}
	}
	assert.True(t, found, "should have revert marker in chat")
	_ = entriesBefore // used for reference
}

func TestApp_UndoCommand_PersistsSession(t *testing.T) {
	a := makeSessionAppWithHistory(t)
	require.NoError(t, store.Save(a.session))
	sessID := a.session.ID

	model, _ := a.Update(components.InputSubmitMsg{Value: "/undo"})
	a = model.(App)
	require.Len(t, a.session.Records(), 2)

	loaded, err := store.Load(sessID)
	require.NoError(t, err)
	assert.Len(t, loaded.Records(), 2, "undo should be persisted to disk")
}

func TestApp_RedoCommand_PersistsSession(t *testing.T) {
	a := makeSessionAppWithHistory(t)
	require.NoError(t, store.Save(a.session))
	sessID := a.session.ID

	model, _ := a.Update(components.InputSubmitMsg{Value: "/undo"})
	a = model.(App)
	require.Len(t, a.session.Records(), 2)
	// Simulate app state where undo was already persisted.
	require.NoError(t, store.Save(a.session))

	model, _ = a.Update(components.InputSubmitMsg{Value: "/redo"})
	a = model.(App)
	require.Len(t, a.session.Records(), 4)

	loaded, err := store.Load(sessID)
	require.NoError(t, err)
	assert.Len(t, loaded.Records(), 4, "redo should be persisted to disk")
}

// --- Modified files cleared on /new ---

func TestApp_ModifiedFiles_ClearedOnNewSession(t *testing.T) {
	a := makeSessionApp(t)
	a.modifiedFiles.add("/tmp/foo.go")
	a.modifiedFiles.add("/tmp/bar.go")

	cmd := a.cmdRegistry.Get("new")
	require.NotNil(t, cmd)
	cmd.Handler(&a)

	assert.Empty(t, a.modifiedFiles.paths())
}

func TestApp_ShellSubmit_SetsCancelAndStreaming(t *testing.T) {
	app := NewApp()
	app.SetProvider(&modelListProvider{})
	app.SetRegistry(tool.NewRegistry())
	app.SetSession(session.New(t.TempDir()))
	app.SetAgent(agent.BuildAgent())

	model, _ := app.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	a := model.(App)
	a.state = StateSession
	a.shellMode = true

	model, cmd := a.Update(components.InputSubmitMsg{Value: "!echo hello"})
	a = model.(App)
	assert.True(t, a.streaming, "shell submit should set streaming")
	assert.NotNil(t, a.cancel, "shell submit should set cancel function")
	assert.NotNil(t, cmd, "should return async command")
}

func TestApp_ShellResult_ClearsStreaming(t *testing.T) {
	app := NewApp()
	app.SetProvider(&modelListProvider{})
	app.SetRegistry(tool.NewRegistry())
	app.SetSession(session.New(t.TempDir()))
	app.SetAgent(agent.BuildAgent())
	app.state = StateSession
	app.streaming = true

	model, _ := app.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	a := model.(App)
	a.streaming = true

	model, _ = a.Update(ShellResultMsg{Command: "echo hello", Output: "hello"})
	a = model.(App)
	assert.False(t, a.streaming, "shell result should clear streaming state")
}

func TestApp_Submit_SetsReasoningEffort(t *testing.T) {
	app := NewApp()
	app.SetProvider(&modelListProvider{})
	app.SetRegistry(tool.NewRegistry())
	app.SetSession(session.New(t.TempDir()))
	app.SetAgent(agent.BuildAgent())

	model, _ := app.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	a := model.(App)
	a.activeVariant = "high"

	// Submit a normal prompt (will start streaming)
	model, _ = a.Update(components.InputSubmitMsg{Value: "hello"})
	a = model.(App)

	p := a.provider.(*modelListProvider)
	assert.Equal(t, "high", p.reasoningEffort, "submit should set reasoning effort before dispatching loop")
}
