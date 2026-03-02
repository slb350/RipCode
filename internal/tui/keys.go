package tui

import (
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/stephenbrandon/ripcode/internal/agent"
)

func (a App) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case msg.Mod == tea.ModCtrl && msg.Code == 'c':
		if a.state == StateSession && !a.streaming && a.input.Value() != "" {
			a.input.Reset()
			return a, nil
		}
		return a, tea.Quit

	case a.helpDialog.open:
		return a.handleHelpDialogKey(msg)

	case a.statusDialog.open:
		return a.handleStatusDialogKey(msg)

	case a.messageActions.open:
		return a.handleMessageActionsDialogKey(msg)

	case a.stashDialog.open:
		return a.handleStashDialogKey(msg)

	case a.agentDialog.open:
		return a.handleAgentDialogKey(msg)

	case a.connectDialog.open:
		return a.handleConnectDialogKey(msg)

	case a.mcpDialog.open:
		return a.handleMCPDialogKey(msg)

	case a.themesDialog.open:
		return a.handleThemesDialogKey(msg)

	case a.forkDialog.open:
		return a.handleForkDialogKey(msg)

	case a.timelineDialog.open:
		return a.handleTimelineDialogKey(msg)

	case a.sessionsDialog.open:
		return a.handleSessionsDialogKey(msg)

	case a.renameDialog.open:
		return a.handleRenameDialogKey(msg)

	case a.exportDialog.open:
		return a.handleExportDialogKey(msg)

	case a.modelDialog.open:
		return a.handleModelDialogKey(msg)

	case a.commandPalette.open:
		return a.handleCommandPaletteKey(msg)

	case a.inline.open:
		return a.handleInlineKey(msg)

	case a.sidebarOverlayActive():
		return a.handleSidebarOverlayKey(msg)

	case a.leaderPending:
		return a.handleLeaderKey(msg)

	case msg.Code == tea.KeyEscape:
		if a.streaming {
			if a.cancel != nil {
				a.cancel()
				a.cancel = nil
			}
			a.eventCh = nil
			a.setStreaming(false)
			a.chat.CommitStream()
			return a, nil
		}
		return a, tea.Quit

	case msg.Mod == tea.ModCtrl && msg.Code == 'l':
		if a.state == StateSession {
			a.chat.Clear()
			a.toolpanel.Clear()
		}
		return a, nil

	case msg.Mod == tea.ModCtrl && msg.Code == 'b':
		if a.state == StateSession {
			return a.toggleSidebar(), nil
		}
		return a, nil

	case msg.Mod == tea.ModCtrl && msg.Code == 'r':
		if !a.streaming && a.state == StateSession {
			if cmd := a.cmdRegistry.Get("rename"); cmd != nil && cmd.Handler != nil {
				return a, cmd.Handler(&a)
			}
		}
		return a, nil

	case msg.Mod == tea.ModCtrl && msg.Code == 's':
		if !a.streaming && a.state == StateSession {
			if cmd := a.cmdRegistry.Get("stash"); cmd != nil && cmd.Handler != nil {
				return a, cmd.Handler(&a)
			}
		}
		return a, nil

	case msg.Mod == tea.ModCtrl && msg.Code == 'p':
		if !a.streaming && a.state == StateSession {
			a.commandPalette.open = true
			a.commandPalette.query = ""
			a.commandPalette.selected = 0
			a.modelDialog.open = false
			a.inline.open = false
			a.input.Blur()
		}
		return a, nil

	case msg.Mod == tea.ModCtrl && msg.Code == 'k':
		if !a.streaming && a.state == StateSession {
			// If input has text, forward Ctrl+K to input for kill-to-end-of-line
			if a.input.Value() != "" {
				cmd := a.input.Update(msg)
				return a, cmd
			}
			a.commandPalette.open = true
			a.commandPalette.query = ""
			a.commandPalette.selected = 0
			a.modelDialog.open = false
			a.inline.open = false
			a.input.Blur()
		}
		return a, nil

	case msg.Code == tea.KeyTab:
		if !a.streaming && a.state == StateSession {
			return a.cycleAgent(msg.Mod&tea.ModShift != 0), nil
		}
		return a, nil

	// Message navigation keybinds
	case msg.Code == tea.KeyPgUp:
		if a.state == StateSession {
			a.chat.PageUp()
		}
		return a, nil

	case msg.Code == tea.KeyPgDown:
		if a.state == StateSession {
			a.chat.PageDown()
		}
		return a, nil

	case msg.Mod == tea.ModCtrl|tea.ModAlt && msg.Code == 'u':
		if a.state == StateSession {
			a.chat.HalfPageUp()
		}
		return a, nil

	case msg.Mod == tea.ModCtrl|tea.ModAlt && msg.Code == 'd':
		if a.state == StateSession {
			a.chat.HalfPageDown()
		}
		return a, nil

	case msg.Mod == tea.ModCtrl|tea.ModAlt && msg.Code == 'y':
		if a.state == StateSession {
			a.chat.LineUp()
		}
		return a, nil

	case msg.Mod == tea.ModCtrl|tea.ModAlt && msg.Code == 'e':
		if a.state == StateSession {
			a.chat.LineDown()
		}
		return a, nil

	case msg.Mod == tea.ModCtrl && msg.Code == 'g':
		if a.state == StateSession {
			a.chat.ScrollToTop()
		}
		return a, nil

	case msg.Mod == tea.ModCtrl|tea.ModAlt && msg.Code == 'g':
		if a.state == StateSession {
			a.chat.ScrollToBottom()
		}
		return a, nil

	case msg.Code == tea.KeyHome && msg.Mod == 0:
		if a.state == StateSession {
			a.chat.ScrollToTop()
		}
		return a, nil

	case msg.Code == tea.KeyEnd && msg.Mod == 0:
		if a.state == StateSession {
			a.chat.ScrollToBottom()
		}
		return a, nil

	case msg.Mod == tea.ModCtrl|tea.ModAlt && msg.Code == 'n':
		if a.state == StateSession {
			a.chat.NextUserMessage()
		}
		return a, nil

	case msg.Mod == tea.ModCtrl|tea.ModAlt && msg.Code == 'p':
		if a.state == StateSession {
			a.chat.PrevUserMessage()
		}
		return a, nil

	case msg.Code == tea.KeyF2:
		if !a.streaming && a.state == StateSession {
			return a.cycleRecentModel(msg.Mod&tea.ModShift != 0)
		}
		return a, nil

	case msg.Mod == tea.ModCtrl && msg.Code == 't':
		if !a.streaming && a.state == StateSession {
			return a.handleVariantCycle()
		}
		return a, nil

	case msg.Mod == tea.ModCtrl && msg.Code == 'x':
		if !a.streaming && a.state == StateSession {
			a.leaderPending = true
			a.statusbar.SetLeaderHint("ctrl+x…")
		}
		return a, nil

	default:
		if !a.streaming {
			if a.state == StateHome {
				cmd := a.home.Input().Update(msg)
				return a, cmd
			}

			// Prompt history: Up at first line, Down at last line
			if msg.Code == tea.KeyUp && msg.Mod == 0 && a.input.CursorY() == 0 {
				if a.promptHistory.AtNewest() {
					a.promptHistory.SaveDraft(a.input.Value())
				}
				if p, ok := a.promptHistory.Previous(); ok {
					a.input.SetValue(p)
				}
				return a, nil
			}
			if msg.Code == tea.KeyDown && msg.Mod == 0 && a.input.CursorY() == a.input.LineCount()-1 {
				if p, ok := a.promptHistory.Next(); ok {
					a.input.SetValue(p)
				}
				return a, nil
			}

			cmd := a.input.Update(msg)

			// Detect shell mode based on leading "!"
			val := a.input.Value()
			if strings.HasPrefix(val, "!") {
				if !a.shellMode {
					a.shellMode = true
					a.input.SetShellMode(true)
				}
			} else if a.shellMode {
				a.shellMode = false
				a.input.SetShellMode(false)
			}

			cacheCmd := a.updateInlineSuggestions()
			return a, tea.Batch(cmd, cacheCmd)
		}
	}

	return a, nil
}

func (a App) handleLeaderKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	a.leaderPending = false
	a.statusbar.SetLeaderHint("")

	switch {
	case msg.Code == tea.KeyEscape:
		return a, nil
	case msg.Code == 'a':
		a.closeAllDialogs()
		a.agentDialog.open = true
		a.agentDialog.query = ""
		a.agentDialog.selected = 0
		return a, nil
	case msg.Code == 'h':
		if cmd := a.cmdRegistry.Get("conceal"); cmd != nil && cmd.Handler != nil {
			return a, cmd.Handler(&a)
		}
		return a, nil
	default:
		return a, nil
	}
}

func (a App) cycleAgent(reverse bool) App {
	current := strings.ToLower(strings.TrimSpace(a.agent.Name))
	switch {
	case reverse:
		if current == agent.NamePlan {
			a.SetAgent(agent.BuildAgent())
			return a
		}
		a.SetAgent(agent.PlanAgent())
		return a
	default:
		if current == agent.NameBuild {
			a.SetAgent(agent.PlanAgent())
			return a
		}
		a.SetAgent(agent.BuildAgent())
		return a
	}
}
