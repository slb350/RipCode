package tui

import (
	"fmt"
	"os"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/atotto/clipboard"

	"github.com/stephenbrandon/ripcode/internal/tui/components"
)

// stubToast returns a handler that shows a "Not yet implemented" toast.
func stubToast(name string) func(a *App) tea.Cmd {
	return func(a *App) tea.Cmd {
		return a.ShowToast(fmt.Sprintf("/%s: Not yet implemented", name), components.ToastInfo)
	}
}

func (a *App) initRegistry() {
	r := NewCommandRegistry()

	r.Register(Command{
		Name: "help", Aliases: []string{"commands"}, Category: CategorySystem,
		Title: "Help", Description: "Show available commands",
		Suggested: true, Execute: true,
		Handler: func(a *App) tea.Cmd {
			a.closeAllDialogs()
			a.helpDialogOpen = true
			a.helpDialogQuery = ""
			a.helpDialogSelect = 0
			a.helpDialogTab = 0
			return nil
		},
	})

	r.Register(Command{
		Name: "new", Aliases: []string{"clear"}, Category: CategorySession,
		Title: "New session", Description: "Clear chat and tool history",
		Suggested: true, Execute: true,
		Handler: func(a *App) tea.Cmd {
			a.chat.Clear()
			a.toolpanel.Clear()
			a.modifiedFiles = nil
			if a.session != nil {
				a.session.Reset()
				a.statusbar.SetTitle(shortSessionTitle(a.session.ID))
				a.statusbar.SetTokens(0)
			}
			a.chat.AddEntry(components.ChatEntry{
				Role:    "system",
				Content: "Conversation cleared.",
			})
			return nil
		},
	})

	r.Register(Command{
		Name: "models", Category: CategoryAgent,
		Title: "Select model", Description: "Search and switch models",
		Suggested: true, Execute: true,
		Handler: func(a *App) tea.Cmd {
			// Re-use existing models command logic
			return nil // special-cased in handleSlashCommand
		},
	})

	r.Register(Command{
		Name: "agent", Category: CategoryAgent,
		Title: "Switch agent", Description: "Toggle build/plan agent or open agent picker",
		Keybind: "ctrl+x a", Execute: true,
		Handler: func(a *App) tea.Cmd {
			// Opens agent dialog when invoked without args
			a.closeAllDialogs()
			a.agentDialogOpen = true
			a.agentDialogQuery = ""
			a.agentDialogSelect = 0
			return nil
		},
	})

	r.Register(Command{
		Name: "sidebar", Category: CategoryView,
		Title: "Toggle sidebar", Description: "Show or hide session sidebar",
		Keybind: "Ctrl+B", Execute: true,
		Handler: func(a *App) tea.Cmd {
			*a = a.toggleSidebar()
			state := "hidden"
			if a.sidebarVisible() {
				state = "shown"
			}
			a.chat.AddEntry(components.ChatEntry{
				Role:    "system",
				Content: fmt.Sprintf("Sidebar %s.", state),
			})
			return nil
		},
	})

	r.Register(Command{
		Name: "model", Category: CategoryAgent,
		Title: "Set model", Description: "Type full provider/model-id and submit",
		Execute: false,
		Handler: func(a *App) tea.Cmd { return nil },
	})

	r.Register(Command{
		Name: "exit", Aliases: []string{"quit", "q"}, Category: CategorySystem,
		Title: "Exit", Description: "Quit ripcode",
		Execute: true,
		Handler: func(_ *App) tea.Cmd { return tea.Quit },
	})

	r.Register(Command{
		Name: "compact", Aliases: []string{"summarize"}, Category: CategorySession,
		Title: "Compact", Description: "Compact session history",
		Execute: true,
		Handler: func(a *App) tea.Cmd {
			if a.session == nil || len(a.session.Messages) == 0 {
				return a.ShowToast("Nothing to compact", components.ToastWarning)
			}
			// Build summary from messages
			var summary strings.Builder
			summary.WriteString("Session compacted. Previous conversation covered:\n")
			userCount := 0
			for _, rec := range a.session.Messages {
				if rec.Message.Role == "user" {
					userCount++
					content := rec.Message.Content
					if len(content) > 80 {
						content = content[:77] + "..."
					}
					summary.WriteString(fmt.Sprintf("- %s\n", content))
				}
			}
			// Replace session messages with a single summary
			a.session.Messages = nil
			a.session.AddUser("[compacted: " + fmt.Sprintf("%d", userCount) + " exchanges]")
			a.session.AddAssistant(summary.String(), nil, nil)
			a.rebuildChatFromSession()
			return a.ShowToast("Compacted session history", components.ToastSuccess)
		},
	})

	r.Register(Command{
		Name: "connect", Category: CategorySystem,
		Title: "Connect", Description: "Provider connection info",
		Execute: true,
		Handler: func(a *App) tea.Cmd {
			a.closeAllDialogs()
			a.connectDialogOpen = true
			a.connectDialogInput = ""
			return nil
		},
	})

	r.Register(Command{
		Name: "mcp", Category: CategorySystem,
		Title: "MCP Servers", Description: "Manage MCP server connections",
		Execute: true,
		Handler: func(a *App) tea.Cmd {
			a.closeAllDialogs()
			a.mcpDialogOpen = true
			a.mcpDialogSelect = 0
			return nil
		},
	})

	r.Register(Command{
		Name: "copy", Category: CategorySession,
		Title: "Copy", Description: "Copy last response to clipboard",
		Execute: true,
		Handler: func(a *App) tea.Cmd {
			entries := a.chat.Entries()
			var last string
			for i := len(entries) - 1; i >= 0; i-- {
				if entries[i].Role == "assistant" {
					last = entries[i].Content
					break
				}
			}
			if last == "" {
				return a.ShowToast("No assistant response to copy", components.ToastWarning)
			}
			if err := clipboard.WriteAll(last); err != nil {
				return a.ShowToast("Copy failed: "+err.Error(), components.ToastError)
			}
			return a.ShowToast("Copied to clipboard", components.ToastSuccess)
		},
	})

	r.Register(Command{
		Name: "details", Category: CategoryView,
		Title: "Details", Description: "Toggle tool detail display",
		Execute: true,
		Handler: func(a *App) tea.Cmd {
			a.showDetails = !a.showDetails
			state := "shown"
			if !a.showDetails {
				state = "hidden"
			}
			return a.ShowToast(fmt.Sprintf("Tool details %s", state), components.ToastInfo)
		},
	})

	r.Register(Command{
		Name: "editor", Category: CategorySession,
		Title: "Editor", Description: "Compose prompt in $EDITOR",
		Execute: true,
		Handler: func(a *App) tea.Cmd {
			editor := os.Getenv("EDITOR")
			if editor == "" {
				editor = os.Getenv("VISUAL")
			}
			if editor == "" {
				return a.ShowToast("Set $EDITOR to use this command", components.ToastWarning)
			}
			return a.ShowToast("Editor: use $EDITOR ("+editor+") to compose", components.ToastInfo)
		},
	})

	r.Register(Command{
		Name: "export", Category: CategorySession,
		Title: "Export", Description: "Export conversation transcript",
		Execute: true,
		Handler: func(a *App) tea.Cmd {
			entries := a.chat.Entries()
			if len(entries) == 0 {
				return a.ShowToast("Nothing to export", components.ToastWarning)
			}
			a.closeAllDialogs()
			a.exportDialogOpen = true
			a.exportIncludeTools = true
			a.exportIncludeMeta = false
			a.exportIncludeThinking = false
			a.exportFilename = "session-export.md"
			a.exportFocusedField = 0
			return nil
		},
	})

	r.Register(Command{
		Name: "fork", Category: CategorySession,
		Title: "Fork", Description: "Fork session from a message",
		Execute: true,
		Handler: func(a *App) tea.Cmd {
			if a.session == nil || a.session.MessageCount("user") == 0 {
				return a.ShowToast("Nothing to fork", components.ToastWarning)
			}
			a.closeAllDialogs()
			a.forkDialogOpen = true
			a.forkDialogSelect = 0
			return nil
		},
	})

	r.Register(Command{
		Name: "rename", Category: CategorySession,
		Title: "Rename", Description: "Rename current session",
		Keybind: "Ctrl+R",
		Execute: true,
		Handler: func(a *App) tea.Cmd {
			a.closeAllDialogs()
			a.renameDialogOpen = true
			a.renameDialogValue = ""
			if a.session != nil {
				a.renameDialogValue = a.session.Title
			}
			return nil
		},
	})

	r.Register(Command{
		Name: "sessions", Category: CategorySession,
		Title: "Sessions", Description: "List saved sessions",
		Suggested: true, Execute: true,
		Handler: func(a *App) tea.Cmd {
			a.closeAllDialogs()
			a.sessionsDialogOpen = true
			a.sessionsDialogQuery = ""
			a.sessionsDialogSelect = 0
			a.sessionsDialogConfirm = false
			if !a.sessionsDialogLoaded {
				return a.loadSessions()
			}
			return nil
		},
	})

	r.Register(Command{
		Name: "skills", Category: CategorySession,
		Title: "Skills", Description: "List available tools",
		Execute: true,
		Handler: func(a *App) tea.Cmd {
			var sb strings.Builder
			sb.WriteString("Available tools:\n")
			if a.registry != nil {
				for _, t := range a.registry.List() {
					sb.WriteString(fmt.Sprintf("  - %s: %s\n", t.ID(), t.Description()))
				}
			}
			if sb.Len() == len("Available tools:\n") {
				sb.WriteString("  (none registered)")
			}
			a.chat.AddEntry(components.ChatEntry{
				Role:    "system",
				Content: sb.String(),
			})
			return nil
		},
	})

	r.Register(Command{
		Name: "status", Category: CategorySystem,
		Title: "Status", Description: "View system status",
		Execute: true,
		Handler: func(a *App) tea.Cmd {
			a.closeAllDialogs()
			a.statusDialogOpen = true
			return nil
		},
	})

	r.Register(Command{
		Name: "themes", Category: CategoryView,
		Title: "Themes", Description: "Switch color theme",
		Execute: true,
		Handler: func(a *App) tea.Cmd {
			a.closeAllDialogs()
			a.themesDialogOpen = true
			a.themesDialogSelect = 0
			return nil
		},
	})

	r.Register(Command{
		Name: "thinking", Aliases: []string{"toggle-thinking"}, Category: CategoryView,
		Title: "Thinking", Description: "Toggle thinking block display",
		Execute: true,
		Handler: func(a *App) tea.Cmd {
			a.showThinking = !a.showThinking
			state := "shown"
			if !a.showThinking {
				state = "hidden"
			}
			return a.ShowToast(fmt.Sprintf("Thinking blocks %s", state), components.ToastInfo)
		},
	})

	r.Register(Command{
		Name: "timeline", Category: CategorySession,
		Title: "Timeline", Description: "View session timeline",
		Execute: true,
		Handler: func(a *App) tea.Cmd {
			a.closeAllDialogs()
			a.timelineDialogOpen = true
			a.timelineDialogQuery = ""
			a.timelineDialogSelect = 0
			return nil
		},
	})

	r.Register(Command{
		Name: "timestamps", Aliases: []string{"toggle-timestamps"}, Category: CategoryView,
		Title: "Timestamps", Description: "Toggle message timestamps",
		Execute: true,
		Handler: func(a *App) tea.Cmd {
			a.showTimestamps = !a.showTimestamps
			state := "shown"
			if !a.showTimestamps {
				state = "hidden"
			}
			return a.ShowToast(fmt.Sprintf("Timestamps %s", state), components.ToastInfo)
		},
	})

	r.Register(Command{
		Name: "undo", Category: CategorySession,
		Title: "Undo", Description: "Undo last exchange",
		Execute: true,
		Handler: func(a *App) tea.Cmd {
			if a.streaming {
				return a.ShowToast("Session busy — stop generation first", components.ToastWarning)
			}
			if a.session == nil || !a.session.CanUndo() {
				return a.ShowToast("Nothing to undo", components.ToastWarning)
			}
			prompt, err := a.session.Revert()
			if err != nil {
				return a.ShowToast("Nothing to undo", components.ToastWarning)
			}
			a.rebuildChatFromSession()
			a.chat.AddEntry(components.ChatEntry{
				Role:    "system",
				Content: "--- reverted ---",
			})
			a.input.SetValue(prompt)
			return a.ShowToast("Reverted last exchange", components.ToastInfo)
		},
	})

	r.Register(Command{
		Name: "redo", Category: CategorySession,
		Title: "Redo", Description: "Redo last undone exchange",
		Execute: true,
		Handler: func(a *App) tea.Cmd {
			if a.session == nil || !a.session.CanRedo() {
				return a.ShowToast("Nothing to redo", components.ToastWarning)
			}
			if err := a.session.Unrevert(); err != nil {
				return a.ShowToast("Nothing to redo", components.ToastWarning)
			}
			a.rebuildChatFromSession()
			return a.ShowToast("Restored exchange", components.ToastInfo)
		},
	})

	r.Register(Command{
		Name: "stash", Category: CategorySession,
		Title: "Stash", Description: "Save current input draft",
		Execute: true,
		Handler: func(a *App) tea.Cmd {
			content := a.input.Value()
			// Also check stashPendingContent for when invoked via slash command
			if a.stashPendingContent != "" {
				content = a.stashPendingContent
				a.stashPendingContent = ""
			}
			if strings.TrimSpace(content) == "" {
				return a.ShowToast("Nothing to stash", components.ToastWarning)
			}
			a.stash.Push(content)
			a.warnOnErr(persistStash(a.stash), "stash")
			a.input.Reset()
			return a.ShowToast("Stashed draft", components.ToastSuccess)
		},
	})

	r.Register(Command{
		Name: "stash-pop", Category: CategorySession,
		Title: "Stash Pop", Description: "Restore last stashed draft",
		Execute: true,
		Handler: func(a *App) tea.Cmd {
			entry, ok := a.stash.Pop()
			if !ok {
				return a.ShowToast("Stash is empty", components.ToastWarning)
			}
			a.warnOnErr(persistStash(a.stash), "stash")
			a.input.SetValue(entry.Content)
			return a.ShowToast("Restored from stash", components.ToastSuccess)
		},
	})

	r.Register(Command{
		Name: "stash-list", Category: CategorySession,
		Title: "Stash List", Description: "View stashed drafts",
		Execute: true,
		Handler: func(a *App) tea.Cmd {
			if len(a.stash.List()) == 0 {
				return a.ShowToast("Stash is empty", components.ToastWarning)
			}
			a.closeAllDialogs()
			a.stashDialogOpen = true
			a.stashDialogSelect = 0
			return nil
		},
	})

	r.Register(Command{
		Name: "recent-model", Category: CategoryAgent,
		Title: "Cycle recent model", Description: "Cycle through recently used models",
		Keybind: "F2", Execute: true, Hidden: true,
		Handler: func(a *App) tea.Cmd {
			m, cmd := (*a).cycleRecentModel(false)
			*a = m.(App)
			return cmd
		},
	})

	r.Register(Command{
		Name: "variant", Category: CategoryAgent,
		Title: "Cycle variant", Description: "Cycle thinking budget variant",
		Keybind: "Ctrl+T", Execute: true, Hidden: true,
		Handler: func(a *App) tea.Cmd {
			m, cmd := (*a).handleVariantCycle()
			*a = m.(App)
			return cmd
		},
	})

	a.cmdRegistry = r
}
