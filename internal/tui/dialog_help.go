package tui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
)

func (a App) handleHelpDialogKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case msg.Code == tea.KeyEscape:
		a.helpDialogOpen = false
		a.input.Focus()
		return a, nil

	case msg.Code == tea.KeyEnter:
		a.helpDialogOpen = false
		a.input.Focus()
		return a, nil

	case msg.Code == tea.KeyTab:
		a.helpDialogTab = (a.helpDialogTab + 1) % 2
		a.helpDialogSelect = 0
		return a, nil

	case msg.Code == tea.KeyUp:
		if a.helpDialogSelect > 0 {
			a.helpDialogSelect--
		}
		return a, nil

	case msg.Code == tea.KeyDown:
		a.helpDialogSelect++
		return a, nil

	case msg.Code == tea.KeyBackspace:
		a.helpDialogQuery = backspaceRune(a.helpDialogQuery)
		a.helpDialogSelect = 0
		return a, nil

	default:
		if msg.Text != "" {
			a.helpDialogQuery += msg.Text
			a.helpDialogSelect = 0
		}
		return a, nil
	}
}

var helpKeybinds = []struct {
	Key  string
	Desc string
}{
	{"Ctrl+A", "Move to line start"},
	{"Ctrl+E", "Move to line end"},
	{"Ctrl+U", "Delete to line start"},
	{"Ctrl+K", "Delete to line end / Command palette"},
	{"Ctrl+W", "Delete word left"},
	{"Alt+D", "Delete word right"},
	{"Ctrl+D", "Delete char right"},
	{"Alt+B / Ctrl+Left", "Move word left"},
	{"Alt+F / Ctrl+Right", "Move word right"},
	{"Ctrl+P / Ctrl+K", "Command palette"},
	{"Ctrl+B", "Toggle sidebar"},
	{"Tab / Shift+Tab", "Cycle agent mode"},
	{"PageUp / PageDown", "Scroll chat"},
	{"Ctrl+G", "Scroll to top"},
	{"Ctrl+Alt+G", "Scroll to bottom"},
	{"Ctrl+Alt+U / D", "Half page up/down"},
	{"Ctrl+Alt+Y / E", "Line up/down"},
	{"Up / Down", "History navigation"},
	{"Esc", "Cancel / Quit"},
}

func (a App) renderHelpDialog() string {
	var sb strings.Builder
	query := strings.TrimSpace(a.helpDialogQuery)

	tabs := []string{"Commands", "Keybinds"}
	for i, t := range tabs {
		if i == a.helpDialogTab {
			sb.WriteString("[" + t + "]")
		} else {
			sb.WriteString(" " + t + " ")
		}
		sb.WriteString("  ")
	}
	sb.WriteString("(Tab switch, Esc close)")
	if query != "" {
		sb.WriteString(" filter: " + query)
	}

	q := strings.ToLower(query)

	if a.helpDialogTab == 0 {
		cmds := a.cmdRegistry.All()
		idx := 0
		for _, c := range cmds {
			if q != "" {
				haystack := strings.ToLower(c.Name + " " + c.Description)
				if !strings.Contains(haystack, q) {
					continue
				}
			}
			prefix := "  "
			if idx == a.helpDialogSelect {
				prefix = "> "
			}
			line := fmt.Sprintf("%-16s %s", "/"+c.Name, c.Description)
			if c.Keybind != "" {
				line += "  [" + c.Keybind + "]"
			}
			sb.WriteString("\n" + prefix + line)
			idx++
		}
		if idx == 0 {
			sb.WriteString("\n  no matches")
		}
	} else {
		idx := 0
		for _, kb := range helpKeybinds {
			if q != "" {
				haystack := strings.ToLower(kb.Key + " " + kb.Desc)
				if !strings.Contains(haystack, q) {
					continue
				}
			}
			prefix := "  "
			if idx == a.helpDialogSelect {
				prefix = "> "
			}
			sb.WriteString(fmt.Sprintf("\n%s%-24s %s", prefix, kb.Key, kb.Desc))
			idx++
		}
		if idx == 0 {
			sb.WriteString("\n  no matches")
		}
	}

	return sb.String()
}
