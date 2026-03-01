package tui

import (
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"

	tea "charm.land/bubbletea/v2"
)

func (a *App) requestFileCache() tea.Cmd {
	if a.fileCacheLoaded || a.fileCacheLoading || a.session == nil {
		return nil
	}
	a.fileCacheLoading = true
	root := a.session.WorkDir
	return loadFileCacheCmd(root)
}

func loadFileCacheCmd(root string) tea.Cmd {
	return func() tea.Msg {
		const maxFiles = 8000
		files := make([]string, 0, 2048)
		// Best-effort file collection for @ autocomplete.
		// Individual entry errors (permission denied, broken symlinks) are
		// skipped so the cache populates with accessible files.
		_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if d.IsDir() {
				switch d.Name() {
				case ".git", "node_modules", ".next", ".turbo":
					return filepath.SkipDir
				}
				return nil
			}
			// Skip symlinks to prevent path traversal.
			if d.Type()&os.ModeSymlink != 0 {
				return nil
			}

			rel, err := filepath.Rel(root, path)
			if err != nil {
				return nil
			}
			files = append(files, filepath.ToSlash(rel))
			if len(files) >= maxFiles {
				return fs.SkipAll
			}
			return nil
		})

		sort.Strings(files)
		return FileCacheLoadedMsg{Files: files}
	}
}

func (a App) closeInlineSuggestions() App {
	a.inlineOpen = false
	a.inlineMode = ""
	a.inlineQuery = ""
	a.inlineSelect = 0
	a.inlineStart = 0
	a.inlineEnd = 0
	return a
}

func (a App) inlineEntries() []inlineEntry {
	if !a.inlineOpen {
		return nil
	}

	query := strings.ToLower(strings.TrimSpace(a.inlineQuery))
	if a.inlineMode == "/" {
		var commands []*Command
		if query == "" {
			commands = a.cmdRegistry.All()
		} else {
			commands = a.cmdRegistry.Filter(query)
		}
		out := make([]inlineEntry, 0, len(commands))
		for _, cmd := range commands {
			out = append(out, inlineEntry{
				Display:     "/" + cmd.Name,
				Insert:      "/" + cmd.Name,
				Description: cmd.Description,
				Execute:     cmd.Execute,
			})
		}
		return out
	}

	if a.inlineMode == "@" {
		out := make([]inlineEntry, 0, 10)
		for _, path := range a.fileCache {
			p := strings.ToLower(path)
			if query != "" && !strings.Contains(p, query) {
				continue
			}
			out = append(out, inlineEntry{
				Display: path,
				Insert:  "@" + path + " ",
			})
			if len(out) >= 10 {
				break
			}
		}
		return out
	}

	return nil
}

func (a *App) updateInlineSuggestions() tea.Cmd {
	if a.state != StateSession || a.streaming || a.commandOpen || a.modelDialogOpen {
		*a = a.closeInlineSuggestions()
		return nil
	}

	text := a.input.Value()
	runes := []rune(text)
	cursor := a.input.CursorOffset()
	if cursor < 0 {
		cursor = 0
	}
	if cursor > len(runes) {
		cursor = len(runes)
	}
	if len(runes) == 0 {
		*a = a.closeInlineSuggestions()
		return nil
	}

	prefix := string(runes[:cursor])
	if strings.HasPrefix(prefix, "/") && !containsWhitespace(prefix) {
		a.inlineOpen = true
		a.inlineMode = "/"
		a.inlineQuery = strings.TrimPrefix(prefix, "/")
		a.inlineStart = 0
		a.inlineEnd = cursor
		a.inlineSelect = 0
		return nil
	}

	// Scan rune slice backwards to find '@' — gives a rune index
	// directly, avoiding byte/rune offset mismatch from strings.LastIndex.
	atIdx := -1
	for i := cursor - 1; i >= 0; i-- {
		if runes[i] == '@' {
			atIdx = i
			break
		}
	}
	if atIdx != -1 {
		beforeOK := atIdx == 0 || unicode.IsSpace(runes[atIdx-1])
		between := string(runes[atIdx+1 : cursor])
		if beforeOK && !containsWhitespace(between) {
			a.inlineOpen = true
			a.inlineMode = "@"
			a.inlineQuery = between
			a.inlineStart = atIdx
			a.inlineEnd = cursor
			a.inlineSelect = 0
			return a.requestFileCache()
		}
	}

	*a = a.closeInlineSuggestions()
	return nil
}

func (a App) handleInlineKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case msg.Code == tea.KeyEscape:
		return a.closeInlineSuggestions(), nil

	case msg.Code == tea.KeyUp || (msg.Mod == tea.ModCtrl && msg.Code == 'p'):
		entries := a.inlineEntries()
		if len(entries) == 0 {
			return a, nil
		}
		a.inlineSelect--
		if a.inlineSelect < 0 {
			a.inlineSelect = len(entries) - 1
		}
		return a, nil

	case msg.Code == tea.KeyDown || (msg.Mod == tea.ModCtrl && msg.Code == 'n'):
		entries := a.inlineEntries()
		if len(entries) == 0 {
			return a, nil
		}
		a.inlineSelect++
		if a.inlineSelect >= len(entries) {
			a.inlineSelect = 0
		}
		return a, nil

	case msg.Code == tea.KeyEnter || msg.Code == tea.KeyTab:
		entries := a.inlineEntries()
		if len(entries) == 0 {
			return a.closeInlineSuggestions(), nil
		}
		if a.inlineSelect < 0 {
			a.inlineSelect = 0
		}
		if a.inlineSelect >= len(entries) {
			a.inlineSelect = len(entries) - 1
		}

		choice := entries[a.inlineSelect]
		if a.inlineMode == "/" && choice.Execute {
			a.input.Reset()
			a = a.closeInlineSuggestions()
			return a.handleSubmit(choice.Insert)
		}

		a.input.ReplaceRange(a.inlineStart, a.inlineEnd, choice.Insert)
		a = a.closeInlineSuggestions()
		cacheCmd := a.updateInlineSuggestions()
		return a, cacheCmd

	default:
		cmd := a.input.Update(msg)
		cacheCmd := a.updateInlineSuggestions()
		return a, tea.Batch(cmd, cacheCmd)
	}
}

func (a App) renderInlineSuggestions() string {
	entries := a.inlineEntries()
	query := strings.TrimSpace(a.inlineQuery)
	if query == "" {
		query = "all"
	}
	header := "Autocomplete " + a.inlineMode + " (Enter select, Esc close) - filter: " + query

	items := make([]pickerItem, len(entries))
	for i, e := range entries {
		items[i] = pickerItem{Label: e.Display, Description: e.Description}
	}
	return renderPickerList(header, items, a.inlineSelect, 8)
}
