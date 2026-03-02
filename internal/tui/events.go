package tui

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/stephenbrandon/ripcode/internal/agent"
	"github.com/stephenbrandon/ripcode/internal/store"
	"github.com/stephenbrandon/ripcode/internal/tui/components"
)

func (a App) handleAgentEvent(event agent.Event) (tea.Model, tea.Cmd) {
	switch event.Type {
	case agent.EventContentDelta:
		a.chat.StreamPart(components.PartText, event.Content)
		return a, listenForEvents(a.eventCh)

	case agent.EventReasoningDelta:
		a.chat.StreamPart(components.PartReasoning, event.Content)
		return a, listenForEvents(a.eventCh)

	case agent.EventToolStart:
		// Tool boundaries split assistant output into distinct messages.
		// This prevents pre-tool deltas from being merged into post-tool content.
		a.chat.CommitStream()
		if event.Tool != nil {
			a.chat.AddEntry(components.ChatEntry{
				Role:       components.RoleTool,
				Content:    toolSummary(event.Tool),
				ToolID:     event.Tool.ID,
				ToolName:   event.Tool.Name,
				ToolStatus: components.StatusPending,
			})
			a.toolpanel.AddEvent(agent.ToolEvent{
				ID:   event.Tool.ID,
				Name: event.Tool.Name,
				Args: event.Tool.Args,
			})
		}
		return a, listenForEvents(a.eventCh)

	case agent.EventToolEnd:
		if event.Tool != nil {
			status := components.StatusSuccess
			if event.Tool.Error != "" {
				status = components.StatusError
			}
			content := event.Tool.Output
			if status == components.StatusError {
				content = event.Tool.Error
			}
			var diff *components.DiffInfo
			if event.Tool.Diff != nil {
				diff = &components.DiffInfo{
					Path:   event.Tool.Diff.Path,
					Before: event.Tool.Diff.Before,
					After:  event.Tool.Diff.After,
					Binary: event.Tool.Diff.Binary,
				}
			}
			a.chat.UpdateLastTool(event.Tool.ID, components.ChatEntry{
				Role:       components.RoleTool,
				Content:    content,
				ToolID:     event.Tool.ID,
				ToolName:   event.Tool.Name,
				ToolStatus: status,
				Diff:       diff,
			})
			a.toolpanel.AddEvent(agent.ToolEvent{
				ID:     event.Tool.ID,
				Name:   event.Tool.Name,
				Output: event.Tool.Output,
				Error:  event.Tool.Error,
			})
			if (event.Tool.Name == "write" || event.Tool.Name == "edit") && event.Tool.Error == "" {
				var parsed struct {
					FilePath string `json:"file_path"`
				}
				if err := json.Unmarshal([]byte(event.Tool.Args), &parsed); err != nil {
					store.LogError(fmt.Sprintf("events: parse tool args for %s: args=%q", event.Tool.Name, event.Tool.Args), err)
				} else if parsed.FilePath != "" {
					a.trackModifiedFile(parsed.FilePath)
				}
			}
		}
		return a, listenForEvents(a.eventCh)

	case agent.EventDone:
		a.eventCh = nil
		a.setStreaming(false)
		a.chat.CommitStream()
		a.input.Focus()
		if a.session != nil {
			tc := a.session.TokenCount()
			a.statusbar.SetTokens(tc.Input + tc.Output)
			a.warnOnErr(store.Save(a.session), "session")
		}
		dur := time.Since(a.responseStart)
		modeName := a.agent.Name
		if modeName == "" {
			modeName = agent.NameBuild
		}
		a.chat.AddEntry(components.ChatEntry{
			Role: components.RoleComplete,
			Meta: &components.CompleteMeta{
				Mode:     modeName,
				Model:    a.model,
				Duration: dur,
			},
		})
		return a, nil

	case agent.EventError:
		a.eventCh = nil
		a.setStreaming(false)
		a.chat.CommitStream()
		a.input.Focus()
		if a.session != nil {
			a.warnOnErr(store.Save(a.session), "session")
		}
		if event.Error != nil {
			a.chat.AddEntry(components.ChatEntry{
				Role:    components.RoleError,
				Content: event.Error.Error(),
			})
		}
		return a, nil

	default:
		store.LogErrorf("events: unhandled agent event type %d", event.Type)
		return a, listenForEvents(a.eventCh)
	}
}

// toolSummary extracts a short summary from tool args.
func toolSummary(te *agent.ToolEvent) string {
	if te.Args != "" {
		s := te.Args
		if idx := strings.IndexByte(s, '\n'); idx >= 0 {
			s = s[:idx]
		}
		if len(s) > 80 {
			s = s[:77] + "..."
		}
		return s
	}
	return te.Name
}

// listenForEvents returns a cmd that reads events from the channel.
func listenForEvents(ch <-chan agent.Event) tea.Cmd {
	return func() tea.Msg {
		event, ok := <-ch
		if !ok {
			return AgentEventMsg{Event: agent.NewDoneEvent(nil)}
		}
		return AgentEventMsg{Event: event}
	}
}
