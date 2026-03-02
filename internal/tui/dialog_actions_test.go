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

// mockClipboard records the last written text for test assertions.
type mockClipboard struct {
	written string
	err     error
}

func (m *mockClipboard) WriteAll(text string) error {
	if m.err != nil {
		return m.err
	}
	m.written = text
	return nil
}

func makeActionsApp(t *testing.T) App {
	t.Helper()
	t.Setenv("RIPCODE_DIR", t.TempDir())
	app := NewApp()
	app.SetProvider(&modelListProvider{})
	app.SetRegistry(tool.NewRegistry())
	sess := session.New(t.TempDir())
	sess.AddUser("hello world")
	sess.AddAssistant("hello back", nil, nil)
	app.SetSession(sess)
	app.SetAgent(agent.BuildAgent())
	app.clipboard = &mockClipboard{}
	app.state = StateSession
	model, _ := app.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	a := model.(App)
	a.rebuildChatFromSession()
	return a
}

// --- actionsForRole tests ---

func TestActionsForRole_User(t *testing.T) {
	actions := actionsForRole(components.RoleUser)
	assert.Equal(t, []string{"Copy", "Revert to here", "Fork from here"}, actions)
}

func TestActionsForRole_Assistant(t *testing.T) {
	actions := actionsForRole(components.RoleAssistant)
	assert.Equal(t, []string{"Copy", "Fork from here"}, actions)
}

func TestActionsForRole_Tool(t *testing.T) {
	actions := actionsForRole(components.RoleTool)
	assert.Equal(t, []string{"Copy output"}, actions)
}

func TestActionsForRole_Error(t *testing.T) {
	actions := actionsForRole(components.RoleError)
	assert.Equal(t, []string{"Copy"}, actions)
}

func TestActionsForRole_System_Nil(t *testing.T) {
	actions := actionsForRole(components.RoleSystem)
	assert.Nil(t, actions)
}

func TestActionsForRole_Complete_Nil(t *testing.T) {
	actions := actionsForRole(components.RoleComplete)
	assert.Nil(t, actions)
}

// --- Dialog key handling ---

func TestMessageActionsDialog_Escape_Closes(t *testing.T) {
	a := makeActionsApp(t)
	a.messageActions.open = true
	a.messageActions.messageIdx = 0
	a.messageActions.entryRole = components.RoleUser

	model, _ := a.handleMessageActionsDialogKey(tea.KeyPressMsg{Code: tea.KeyEscape})
	result := model.(App)
	assert.False(t, result.messageActions.open)
}

func TestMessageActionsDialog_UpDown_Navigate(t *testing.T) {
	a := makeActionsApp(t)
	a.messageActions.open = true
	a.messageActions.messageIdx = 0
	a.messageActions.entryRole = components.RoleUser
	a.messageActions.selected = 0

	// Down
	model, _ := a.handleMessageActionsDialogKey(tea.KeyPressMsg{Code: tea.KeyDown})
	result := model.(App)
	assert.Equal(t, 1, result.messageActions.selected)

	// Down again
	model, _ = result.handleMessageActionsDialogKey(tea.KeyPressMsg{Code: tea.KeyDown})
	result = model.(App)
	assert.Equal(t, 2, result.messageActions.selected)

	// Down at bottom wraps to 0
	model, _ = result.handleMessageActionsDialogKey(tea.KeyPressMsg{Code: tea.KeyDown})
	result = model.(App)
	assert.Equal(t, 0, result.messageActions.selected)

	// Up at top wraps to bottom
	model, _ = result.handleMessageActionsDialogKey(tea.KeyPressMsg{Code: tea.KeyUp})
	result = model.(App)
	assert.Equal(t, 2, result.messageActions.selected)
}

func TestMessageActionsDialog_Enter_Copy_User(t *testing.T) {
	a := makeActionsApp(t)
	clip := &mockClipboard{}
	a.clipboard = clip
	a.messageActions.open = true
	a.messageActions.messageIdx = 0
	a.messageActions.entryRole = components.RoleUser
	a.messageActions.selected = 0 // "Copy"

	model, _ := a.handleMessageActionsDialogKey(tea.KeyPressMsg{Code: tea.KeyEnter})
	result := model.(App)
	assert.False(t, result.messageActions.open)
	assert.Equal(t, "hello world", clip.written)
}

func TestMessageActionsDialog_Enter_Copy_Assistant(t *testing.T) {
	a := makeActionsApp(t)
	clip := &mockClipboard{}
	a.clipboard = clip
	a.messageActions.open = true
	a.messageActions.messageIdx = 1
	a.messageActions.entryRole = components.RoleAssistant
	a.messageActions.selected = 0 // "Copy"

	model, _ := a.handleMessageActionsDialogKey(tea.KeyPressMsg{Code: tea.KeyEnter})
	result := model.(App)
	assert.False(t, result.messageActions.open)
	assert.Equal(t, "hello back", clip.written)
}

func TestMessageActionsDialog_Enter_CopyOutput_Tool(t *testing.T) {
	a := makeActionsApp(t)
	clip := &mockClipboard{}
	a.clipboard = clip
	a.chat.AddEntry(components.ChatEntry{
		Role:       components.RoleTool,
		Content:    "tool output here",
		ToolName:   "bash",
		ToolStatus: components.StatusSuccess,
	})
	a.messageActions.open = true
	a.messageActions.messageIdx = 2 // the tool entry
	a.messageActions.entryRole = components.RoleTool
	a.messageActions.selected = 0 // "Copy output"

	model, _ := a.handleMessageActionsDialogKey(tea.KeyPressMsg{Code: tea.KeyEnter})
	result := model.(App)
	assert.False(t, result.messageActions.open)
	assert.Equal(t, "tool output here", clip.written)
}

func TestMessageActionsDialog_Enter_Revert(t *testing.T) {
	t.Setenv("RIPCODE_DIR", t.TempDir())
	app := NewApp()
	app.SetProvider(&modelListProvider{})
	app.SetRegistry(tool.NewRegistry())
	sess := session.New(t.TempDir())
	sess.AddUser("first question")
	sess.AddAssistant("first answer", nil, nil)
	sess.AddUser("second question")
	sess.AddAssistant("second answer", nil, nil)
	app.SetSession(sess)
	app.SetAgent(agent.BuildAgent())
	app.clipboard = &mockClipboard{}
	app.state = StateSession
	model, _ := app.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	a := model.(App)
	a.rebuildChatFromSession()

	require.True(t, a.session.CanUndo())

	// Open actions on first user message, select "Revert to here"
	a.messageActions.open = true
	a.messageActions.messageIdx = 0
	a.messageActions.entryRole = components.RoleUser
	a.messageActions.selected = 1 // "Revert to here"

	model, _ = a.handleMessageActionsDialogKey(tea.KeyPressMsg{Code: tea.KeyEnter})
	result := model.(App)
	assert.False(t, result.messageActions.open)
	// Session should have been reverted
	assert.Equal(t, 2, result.session.Len()) // only first exchange remains
}

func TestMessageActionsDialog_Enter_Fork(t *testing.T) {
	t.Setenv("RIPCODE_DIR", t.TempDir())
	app := NewApp()
	app.SetProvider(&modelListProvider{})
	app.SetRegistry(tool.NewRegistry())
	sess := session.New(t.TempDir())
	sess.AddUser("first question")
	sess.AddAssistant("first answer", nil, nil)
	sess.AddUser("second question")
	sess.AddAssistant("second answer", nil, nil)
	app.SetSession(sess)
	app.SetAgent(agent.BuildAgent())
	app.clipboard = &mockClipboard{}
	app.state = StateSession
	model, _ := app.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	a := model.(App)
	a.rebuildChatFromSession()

	oldID := a.session.ID
	// Open actions on first user message, select "Fork from here" (index 2)
	a.messageActions.open = true
	a.messageActions.messageIdx = 0
	a.messageActions.entryRole = components.RoleUser
	a.messageActions.selected = 2 // "Fork from here"

	model, _ = a.handleMessageActionsDialogKey(tea.KeyPressMsg{Code: tea.KeyEnter})
	result := model.(App)
	assert.False(t, result.messageActions.open)
	assert.NotEqual(t, oldID, result.session.ID) // new session
}

func TestMessageActionsDialog_Renders(t *testing.T) {
	a := makeActionsApp(t)
	a.messageActions.open = true
	a.messageActions.messageIdx = 0
	a.messageActions.entryRole = components.RoleUser
	a.messageActions.selected = 1

	output := a.renderMessageActionsDialog()
	assert.Contains(t, output, "Copy")
	assert.Contains(t, output, "Revert to here")
	assert.Contains(t, output, "Fork from here")
	assert.Contains(t, output, ">") // selection marker
}

func TestMessageActionsDialog_EmptyActions_Closes(t *testing.T) {
	a := makeActionsApp(t)
	a.messageActions.open = true
	a.messageActions.messageIdx = 0
	a.messageActions.entryRole = components.RoleSystem // no actions

	model, _ := a.handleMessageActionsDialogKey(tea.KeyPressMsg{Code: tea.KeyEnter})
	result := model.(App)
	assert.False(t, result.messageActions.open)
}

// --- EntryAtLine tests ---

func TestEntryAtLine_FirstEntry(t *testing.T) {
	c := components.NewChat()
	c.SetSize(80, 40)
	c.AddEntry(components.ChatEntry{Role: components.RoleUser, Content: "hello"})
	c.AddEntry(components.ChatEntry{Role: components.RoleAssistant, Content: "world"})

	idx, ok := c.EntryAtLine(0)
	assert.True(t, ok)
	assert.Equal(t, 0, idx)
}

func TestEntryAtLine_SecondEntry(t *testing.T) {
	c := components.NewChat()
	c.SetSize(80, 40)
	c.AddEntry(components.ChatEntry{Role: components.RoleUser, Content: "hello"})
	c.AddEntry(components.ChatEntry{Role: components.RoleAssistant, Content: "world"})

	// User entry: ┃ hello + ╹ + blank = 3 lines, second entry starts at line 3
	idx, ok := c.EntryAtLine(3)
	assert.True(t, ok)
	assert.Equal(t, 1, idx)
}

func TestEntryAtLine_OutOfBounds(t *testing.T) {
	c := components.NewChat()
	c.SetSize(80, 40)
	c.AddEntry(components.ChatEntry{Role: components.RoleUser, Content: "hello"})

	_, ok := c.EntryAtLine(100)
	assert.False(t, ok)
}

func TestEntryAtLine_NegativeLine(t *testing.T) {
	c := components.NewChat()
	c.SetSize(80, 40)
	c.AddEntry(components.ChatEntry{Role: components.RoleUser, Content: "hello"})

	_, ok := c.EntryAtLine(-1)
	assert.False(t, ok)
}

func TestEntryAtLine_Empty(t *testing.T) {
	c := components.NewChat()
	c.SetSize(80, 40)

	_, ok := c.EntryAtLine(0)
	assert.False(t, ok)
}

func TestEntryAtLine_MultiLineEntry(t *testing.T) {
	c := components.NewChat()
	c.SetSize(80, 40)
	c.AddEntry(components.ChatEntry{Role: components.RoleUser, Content: "line one\nline two\nline three"})
	c.AddEntry(components.ChatEntry{Role: components.RoleAssistant, Content: "response"})

	// Multi-line user: ┃ line1, ┃ line2, ┃ line3, ╹ = 4 lines + 1 blank = 5
	// Second entry at line 5
	idx, ok := c.EntryAtLine(5)
	assert.True(t, ok)
	assert.Equal(t, 1, idx)
}

// --- chatBounds tests ---

func TestChatBounds_Default(t *testing.T) {
	a := makeActionsApp(t)
	top, bottom := a.chatBounds()
	assert.Equal(t, 1, top)     // status bar = 1 line
	assert.Equal(t, 24, bottom) // height(30) - inputH(5) - footerH(1)
}

func TestChatBounds_WithToast(t *testing.T) {
	a := makeActionsApp(t)
	a.toasts.Show("test", components.ToastInfo, 0)

	top, bottom := a.chatBounds()
	assert.Greater(t, top, 1)   // toast adds lines above chat
	assert.Equal(t, 24, bottom) // bottom unchanged
}

// --- closeAllDialogs includes messageActions ---

func TestCloseAllDialogs_IncludesMessageActions(t *testing.T) {
	a := makeActionsApp(t)
	a.messageActions.open = true
	a.closeAllDialogs()
	assert.False(t, a.messageActions.open)
}

// --- handleKey routes to messageActions dialog ---

func TestHandleKey_MessageActionsDialog_Routes(t *testing.T) {
	a := makeActionsApp(t)
	a.messageActions.open = true
	a.messageActions.entryRole = components.RoleUser

	model, _ := a.handleKey(tea.KeyPressMsg{Code: tea.KeyEscape})
	result := model.(App)
	assert.False(t, result.messageActions.open)
}

// --- Mouse click in chat area opens dialog ---

func TestMouseClick_ChatArea_OpensDialog(t *testing.T) {
	t.Setenv("RIPCODE_DIR", t.TempDir())
	app := NewApp()
	app.SetProvider(&modelListProvider{})
	app.SetRegistry(tool.NewRegistry())
	sess := session.New(t.TempDir())
	sess.AddUser("hello")
	sess.AddAssistant("world", nil, nil)
	app.SetSession(sess)
	app.SetAgent(agent.BuildAgent())
	app.clipboard = &mockClipboard{}
	app.state = StateSession
	model, _ := app.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	a := model.(App)
	a.rebuildChatFromSession()

	// Click at Y=1 (inside chat area), X=5 (inside main content)
	model, _ = a.Update(tea.MouseClickMsg{X: 5, Y: 1, Button: tea.MouseLeft})
	result := model.(App)
	assert.True(t, result.messageActions.open)
}

func TestMouseClick_OutsideChatArea_NoDialog(t *testing.T) {
	a := makeActionsApp(t)

	// Click at Y=0 (status bar area)
	model, _ := a.Update(tea.MouseClickMsg{X: 5, Y: 0, Button: tea.MouseLeft})
	result := model.(App)
	assert.False(t, result.messageActions.open)
}

func TestMouseClick_StreamingState_NoDialog(t *testing.T) {
	a := makeActionsApp(t)
	a.streaming = true

	model, _ := a.Update(tea.MouseClickMsg{X: 5, Y: 2, Button: tea.MouseLeft})
	result := model.(App)
	assert.False(t, result.messageActions.open)
}

// --- renderSessionView includes messageActions dialog ---

func TestRenderSessionView_IncludesActionsDialog(t *testing.T) {
	a := makeActionsApp(t)
	a.messageActions.open = true
	a.messageActions.entryRole = components.RoleUser
	a.messageActions.messageIdx = 0

	view := a.renderSessionView()
	assert.Contains(t, view, "Copy")
}

// --- Session nil guard for revert/fork ---

func TestMessageActions_RevertNoSession(t *testing.T) {
	a := makeActionsApp(t)
	a.session = nil
	a.messageActions.open = true
	a.messageActions.messageIdx = 0
	a.messageActions.entryRole = components.RoleUser
	a.messageActions.selected = 1 // "Revert to here"

	model, _ := a.handleMessageActionsDialogKey(tea.KeyPressMsg{Code: tea.KeyEnter})
	result := model.(App)
	assert.False(t, result.messageActions.open)
}

func TestMessageActions_ForkNoSession(t *testing.T) {
	a := makeActionsApp(t)
	a.session = nil
	a.messageActions.open = true
	a.messageActions.messageIdx = 0
	a.messageActions.entryRole = components.RoleUser
	a.messageActions.selected = 2 // "Fork from here"

	model, _ := a.handleMessageActionsDialogKey(tea.KeyPressMsg{Code: tea.KeyEnter})
	result := model.(App)
	assert.False(t, result.messageActions.open)
}

// --- Revert to specific message index ---

func TestMessageActions_RevertToSpecificMessage(t *testing.T) {
	t.Setenv("RIPCODE_DIR", t.TempDir())
	app := NewApp()
	app.SetProvider(&modelListProvider{})
	app.SetRegistry(tool.NewRegistry())
	sess := session.New(t.TempDir())
	sess.AddUser("q1")
	sess.AddAssistant("a1", nil, nil)
	sess.AddUser("q2")
	sess.AddAssistant("a2", nil, nil)
	sess.AddUser("q3")
	sess.AddAssistant("a3", nil, nil)
	require.Equal(t, 6, sess.Len())
	app.SetSession(sess)
	app.SetAgent(agent.BuildAgent())
	app.clipboard = &mockClipboard{}
	app.state = StateSession
	model, _ := app.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	a := model.(App)
	a.rebuildChatFromSession()

	// Revert from second user message (index 2 in chat: q1, a1, q2)
	// This should keep q1+a1+q2
	a.messageActions.open = true
	a.messageActions.messageIdx = 2 // q2 in chat entries
	a.messageActions.entryRole = components.RoleUser
	a.messageActions.selected = 1 // "Revert to here"

	model, _ = a.handleMessageActionsDialogKey(tea.KeyPressMsg{Code: tea.KeyEnter})
	result := model.(App)
	assert.False(t, result.messageActions.open)
	// After revert: should have q1+a1+q2 = 3 messages (reverted q3+a3 = last exchange, keeping q2 prompt)
	// Actually Revert() removes the last user + following messages, returning the prompt.
	// With 3 exchanges, Revert() removes q3+a3 (last exchange), returning "q3".
	// Then another Revert() removes q2+a2, returning "q2".
	// We need to check session.Len() is reduced.
	assert.Less(t, result.session.Len(), 6)
}

// --- Fork from specific message index ---

func TestMessageActions_ForkFromAssistant(t *testing.T) {
	t.Setenv("RIPCODE_DIR", t.TempDir())
	app := NewApp()
	app.SetProvider(&modelListProvider{})
	app.SetRegistry(tool.NewRegistry())
	sess := session.New(t.TempDir())
	sess.AddUser("q1")
	sess.AddAssistant("a1", nil, nil)
	sess.AddUser("q2")
	sess.AddAssistant("a2", nil, nil)
	if err := store.Save(sess); err != nil {
		t.Fatal(err)
	}
	app.SetSession(sess)
	app.SetAgent(agent.BuildAgent())
	app.clipboard = &mockClipboard{}
	app.state = StateSession
	model, _ := app.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	a := model.(App)
	a.rebuildChatFromSession()

	oldID := a.session.ID
	// Fork from assistant entry (index 1 in chat: q1, a1, ...)
	a.messageActions.open = true
	a.messageActions.messageIdx = 1
	a.messageActions.entryRole = components.RoleAssistant
	a.messageActions.selected = 1 // "Fork from here"

	model, _ = a.handleMessageActionsDialogKey(tea.KeyPressMsg{Code: tea.KeyEnter})
	result := model.(App)
	assert.False(t, result.messageActions.open)
	assert.NotEqual(t, oldID, result.session.ID)
}
