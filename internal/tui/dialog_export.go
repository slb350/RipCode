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
	if a.exportDialog.focusedField == 3 {
		switch {
		case msg.Code == tea.KeyEscape:
			a.exportDialog.open = false
			a.input.Focus()
			return a, nil
		case msg.Code == tea.KeyEnter:
			a.exportDialog.open = false
			a.input.Focus()
			return a, a.executeExport()
		case msg.Code == tea.KeyUp:
			a.exportDialog.focusedField--
			return a, nil
		case msg.Code == tea.KeyBackspace:
			a.exportDialog.filename = backspaceRune(a.exportDialog.filename)
			return a, nil
		default:
			if msg.Text != "" {
				a.exportDialog.filename += msg.Text
			}
			return a, nil
		}
	}

	switch {
	case msg.Code == tea.KeyEscape:
		a.exportDialog.open = false
		a.input.Focus()
		return a, nil

	case msg.Code == tea.KeyEnter:
		a.exportDialog.open = false
		a.input.Focus()
		return a, a.executeExport()

	case msg.Code == tea.KeyUp:
		if a.exportDialog.focusedField > 0 {
			a.exportDialog.focusedField--
		}
		return a, nil

	case msg.Code == tea.KeyDown:
		if a.exportDialog.focusedField < 3 {
			a.exportDialog.focusedField++
		}
		return a, nil

	case msg.Text == " ":
		switch a.exportDialog.focusedField {
		case 0:
			a.exportDialog.includeTools = !a.exportDialog.includeTools
		case 1:
			a.exportDialog.includeMeta = !a.exportDialog.includeMeta
		case 2:
			a.exportDialog.includeThinking = !a.exportDialog.includeThinking
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
			if a.exportDialog.includeTools {
				sb.WriteString(fmt.Sprintf("**Tool: %s** (%s)\n", e.ToolName, e.ToolStatus))
				sb.WriteString(e.Content + "\n\n")
			}
		case "complete":
			if a.exportDialog.includeMeta && e.Meta != nil {
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
	exportPath := filepath.Join(workDir, a.exportDialog.filename)
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
		if idx == a.exportDialog.focusedField {
			return "> "
		}
		return "  "
	}

	sb.WriteString(fmt.Sprintf("\n%s%s Include tool calls", marker(0), check(a.exportDialog.includeTools)))
	sb.WriteString(fmt.Sprintf("\n%s%s Include metadata (model, tokens, duration)", marker(1), check(a.exportDialog.includeMeta)))
	sb.WriteString(fmt.Sprintf("\n%s%s Include thinking blocks", marker(2), check(a.exportDialog.includeThinking)))

	fnMarker := "  "
	if a.exportDialog.focusedField == 3 {
		fnMarker = "> "
	}
	sb.WriteString(fmt.Sprintf("\n\n%sFilename: %s", fnMarker, a.exportDialog.filename))
	if a.exportDialog.focusedField == 3 {
		sb.WriteString("_")
	}
	sb.WriteString("\n\n  [Enter] export  [Space] toggle  [Esc] cancel")
	return sb.String()
}
