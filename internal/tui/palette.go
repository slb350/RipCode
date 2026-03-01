package tui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
)

// paletteEntries returns commands in the order displayed in the command palette.
// When filtering, this is the flat filtered list. When unfiltered, this is
// suggested commands first, then remaining commands grouped by category order.
func (a App) paletteEntries() []*Command {
	q := strings.TrimSpace(a.commandPalette.query)
	if q != "" {
		return a.cmdRegistry.Filter(q)
	}

	suggested := a.cmdRegistry.Suggested()
	suggestedSet := make(map[string]bool, len(suggested))
	for _, s := range suggested {
		suggestedSet[s.Name] = true
	}

	out := make([]*Command, 0, len(a.cmdRegistry.commands))
	out = append(out, suggested...)

	byCategory := a.cmdRegistry.ByCategory()
	for _, cat := range categoryOrder {
		for _, cmd := range byCategory[cat] {
			if !suggestedSet[cmd.Name] {
				out = append(out, cmd)
			}
		}
	}
	return out
}

func (a App) handleCommandPaletteKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case msg.Code == tea.KeyEscape:
		a.commandPalette.open = false
		a.commandPalette.query = ""
		a.commandPalette.selected = 0
		a.input.Focus()
		return a, nil

	case msg.Code == tea.KeyEnter:
		entries := a.paletteEntries()
		if len(entries) == 0 {
			a.commandPalette.open = false
			a.commandPalette.query = ""
			a.commandPalette.selected = 0
			a.input.Focus()
			return a, nil
		}
		if a.commandPalette.selected >= len(entries) {
			a.commandPalette.selected = len(entries) - 1
		}
		if a.commandPalette.selected < 0 {
			a.commandPalette.selected = 0
		}
		entry := entries[a.commandPalette.selected]
		a.commandPalette.open = false
		a.commandPalette.query = ""
		a.commandPalette.selected = 0
		a.input.Focus()
		if entry.Execute {
			return a.handleSubmit("/" + entry.Name)
		}
		a.input.SetValue("/" + entry.Name + " ")
		cacheCmd := a.updateInlineSuggestions()
		return a, cacheCmd

	case msg.Code == tea.KeyUp:
		entries := a.paletteEntries()
		if len(entries) == 0 {
			return a, nil
		}
		a.commandPalette.selected--
		if a.commandPalette.selected < 0 {
			a.commandPalette.selected = len(entries) - 1
		}
		return a, nil

	case msg.Code == tea.KeyDown:
		entries := a.paletteEntries()
		if len(entries) == 0 {
			return a, nil
		}
		a.commandPalette.selected++
		if a.commandPalette.selected >= len(entries) {
			a.commandPalette.selected = 0
		}
		return a, nil

	case msg.Code == tea.KeyBackspace:
		a.commandPalette.query = backspaceRune(a.commandPalette.query)
		a.commandPalette.selected = 0
		return a, nil

	default:
		if msg.Text != "" {
			a.commandPalette.query += msg.Text
			a.commandPalette.selected = 0
		}
		return a, nil
	}
}

func (a App) renderCommandPalette() string {
	entries := a.paletteEntries()
	query := strings.TrimSpace(a.commandPalette.query)

	var sb strings.Builder
	sb.WriteString("Commands (Ctrl+P/Ctrl+K, Esc close)")
	if query != "" {
		sb.WriteString(" - filter: ")
		sb.WriteString(query)
	}

	if len(entries) == 0 {
		sb.WriteString("\n  no matches")
		return sb.String()
	}

	// When filtering, show a flat list (no categories).
	if query != "" {
		for i, e := range entries {
			a.writePaletteEntry(&sb, e, i)
		}
		return sb.String()
	}

	// Unfiltered: show Suggested, then categories with headers.
	numSuggested := len(a.cmdRegistry.Suggested())
	lastCategory := CommandCategory("")

	for i, e := range entries {
		if i < numSuggested {
			if i == 0 {
				sb.WriteString("\n\n  Suggested")
			}
		} else if e.Category != lastCategory {
			lastCategory = e.Category
			sb.WriteString("\n\n  ")
			sb.WriteString(string(e.Category))
		}
		a.writePaletteEntry(&sb, e, i)
	}

	return sb.String()
}

// writePaletteEntry writes a single palette row with selection marker and optional keybind.
func (a App) writePaletteEntry(sb *strings.Builder, e *Command, idx int) {
	prefix := "  "
	if idx == a.commandPalette.selected {
		prefix = "> "
	}
	sb.WriteString("\n")
	sb.WriteString(prefix)
	sb.WriteString(fmt.Sprintf("%-20s %s", "/"+e.Name, e.Description))
	if e.Keybind != "" {
		sb.WriteString("  [")
		sb.WriteString(e.Keybind)
		sb.WriteString("]")
	}
}
