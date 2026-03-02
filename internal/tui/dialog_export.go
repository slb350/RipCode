package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/stephenbrandon/ripcode/internal/store"
	"github.com/stephenbrandon/ripcode/internal/tool"
	"github.com/stephenbrandon/ripcode/internal/tui/components"
)

var exportRename = os.Rename

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
		case components.RoleUser:
			sb.WriteString("## User\n\n")
			sb.WriteString(e.Content + "\n\n")
		case components.RoleAssistant:
			sb.WriteString("## Assistant\n\n")
			if a.exportDialog.includeThinking {
				sb.WriteString(e.FullContent() + "\n\n")
			} else {
				sb.WriteString(e.Content + "\n\n")
			}
		case components.RoleTool:
			if a.exportDialog.includeTools {
				sb.WriteString(fmt.Sprintf("**Tool: %s** (%s)\n", e.ToolName, e.ToolStatus))
				sb.WriteString(e.Content + "\n\n")
			}
		case components.RoleComplete:
			if a.exportDialog.includeMeta && e.Meta != nil {
				sb.WriteString(fmt.Sprintf("*%s · %s · %.1fs*\n\n",
					e.Meta.Mode, e.Meta.Model, e.Meta.Duration.Seconds()))
			}
		case components.RoleError:
			sb.WriteString("**Error:** " + e.Content + "\n\n")
		case components.RoleSystem:
			sb.WriteString("*" + e.Content + "*\n\n")
		default:
			store.LogErrorf("export: unknown entry role %q skipped", e.Role)
		}
	}

	workDir := "."
	if a.session != nil {
		workDir = a.session.WorkDir
	}
	exportPath, err := tool.ValidatePath(a.exportDialog.filename, workDir, false)
	if err != nil {
		return a.ShowToast("Export failed: invalid path", components.ToastError)
	}
	if err := writeExportFile(exportPath, []byte(sb.String())); err != nil {
		return a.ShowToast("Export failed: "+err.Error(), components.ToastError)
	}
	return a.ShowToast("Exported to "+exportPath, components.ToastSuccess)
}

func writeExportFile(path string, data []byte) error {
	// Keep symlink behavior explicit: exports should never write through links.
	if info, err := os.Lstat(path); err == nil {
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("refusing to follow symlink: %s", path)
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("stat export path: %w", err)
	}

	dir := filepath.Dir(path)
	f, err := os.CreateTemp(dir, filepath.Base(path)+".tmp.*")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmp := f.Name()
	cleanup := func() {
		if rmErr := os.Remove(tmp); rmErr != nil && !os.IsNotExist(rmErr) {
			store.LogError("export cleanup", rmErr)
		}
	}

	if _, err := f.Write(data); err != nil {
		_ = f.Close()
		cleanup()
		return fmt.Errorf("write temp file: %w", err)
	}
	if err := f.Close(); err != nil {
		cleanup()
		return fmt.Errorf("close temp file: %w", err)
	}
	if err := os.Chmod(tmp, 0o644); err != nil {
		cleanup()
		return fmt.Errorf("chmod temp file: %w", err)
	}
	if err := exportRename(tmp, path); err != nil {
		cleanup()
		return fmt.Errorf("rename temp file: %w", err)
	}

	return nil
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
