package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/stephenbrandon/ripcode/internal/tui/components"
)

func (a App) handleExportDialogKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	// When editing filename, handle text input
	if a.exportFocusedField == 3 {
		switch {
		case msg.Code == tea.KeyEscape:
			a.exportDialogOpen = false
			a.input.Focus()
			return a, nil
		case msg.Code == tea.KeyEnter:
			a.exportDialogOpen = false
			a.input.Focus()
			return a, a.executeExport()
		case msg.Code == tea.KeyUp:
			a.exportFocusedField--
			return a, nil
		case msg.Code == tea.KeyBackspace:
			a.exportFilename = backspaceRune(a.exportFilename)
			return a, nil
		default:
			if msg.Text != "" {
				a.exportFilename += msg.Text
			}
			return a, nil
		}
	}

	switch {
	case msg.Code == tea.KeyEscape:
		a.exportDialogOpen = false
		a.input.Focus()
		return a, nil

	case msg.Code == tea.KeyEnter:
		a.exportDialogOpen = false
		a.input.Focus()
		return a, a.executeExport()

	case msg.Code == tea.KeyUp:
		if a.exportFocusedField > 0 {
			a.exportFocusedField--
		}
		return a, nil

	case msg.Code == tea.KeyDown:
		if a.exportFocusedField < 3 {
			a.exportFocusedField++
		}
		return a, nil

	case msg.Text == " ":
		switch a.exportFocusedField {
		case 0:
			a.exportIncludeTools = !a.exportIncludeTools
		case 1:
			a.exportIncludeMeta = !a.exportIncludeMeta
		case 2:
			a.exportIncludeThinking = !a.exportIncludeThinking
		}
		return a, nil

	default:
		return a, nil
	}
}

func (a *App) executeExport() tea.Cmd {
	entries := a.chat.Entries()
	var sb strings.Builder

	sb.WriteString("# Session Export\n\n")
	for _, e := range entries {
		switch e.Role {
		case "user":
			sb.WriteString("## User\n\n")
			sb.WriteString(e.Content + "\n\n")
		case "assistant":
			sb.WriteString("## Assistant\n\n")
			sb.WriteString(e.Content + "\n\n")
		case "tool":
			if a.exportIncludeTools {
				sb.WriteString(fmt.Sprintf("**Tool: %s** (%s)\n", e.ToolName, e.ToolStatus))
				sb.WriteString(e.Content + "\n\n")
			}
		case "complete":
			if a.exportIncludeMeta && e.Meta != nil {
				sb.WriteString(fmt.Sprintf("*%s · %s · %.1fs*\n\n",
					e.Meta.Mode, e.Meta.Model, e.Meta.Duration.Seconds()))
			}
		case "error":
			sb.WriteString("**Error:** " + e.Content + "\n\n")
		case "system":
			sb.WriteString("*" + e.Content + "*\n\n")
		}
	}

	workDir := "."
	if a.session != nil {
		workDir = a.session.WorkDir
	}
	exportPath := filepath.Join(workDir, a.exportFilename)
	if err := os.WriteFile(exportPath, []byte(sb.String()), 0o644); err != nil {
		return a.ShowToast("Export failed: "+err.Error(), components.ToastError)
	}
	return a.ShowToast("Exported to "+exportPath, components.ToastSuccess)
}

func (a App) renderExportDialog() string {
	var sb strings.Builder
	sb.WriteString("Export transcript                          esc\n")

	check := func(v bool) string {
		if v {
			return "[x]"
		}
		return "[ ]"
	}
	marker := func(idx int) string {
		if idx == a.exportFocusedField {
			return "> "
		}
		return "  "
	}

	sb.WriteString(fmt.Sprintf("\n%s%s Include tool calls", marker(0), check(a.exportIncludeTools)))
	sb.WriteString(fmt.Sprintf("\n%s%s Include metadata (model, tokens, duration)", marker(1), check(a.exportIncludeMeta)))
	sb.WriteString(fmt.Sprintf("\n%s%s Include thinking blocks", marker(2), check(a.exportIncludeThinking)))

	fnMarker := "  "
	if a.exportFocusedField == 3 {
		fnMarker = "> "
	}
	sb.WriteString(fmt.Sprintf("\n\n%sFilename: %s", fnMarker, a.exportFilename))
	if a.exportFocusedField == 3 {
		sb.WriteString("_")
	}
	sb.WriteString("\n\n  [Enter] export  [Space] toggle  [Esc] cancel")
	return sb.String()
}
