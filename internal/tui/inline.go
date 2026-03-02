package tui

import (
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"

	tea "charm.land/bubbletea/v2"

	"github.com/stephenbrandon/ripcode/internal/store"
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
		walkErr := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
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
		if walkErr != nil {
			store.LogError("file cache walk", walkErr)
		}

		sort.Strings(files)
		return FileCacheLoadedMsg{Files: files}
	}
}

func (a App) closeInlineSuggestions() App {
	a.inline.open = false
	a.inline.mode = ""
	a.inline.query = ""
	a.inline.selected = 0
	a.inline.start = 0
	a.inline.end = 0
	return a
}

func (a App) inlineEntries() []inlineEntry {
	if !a.inline.open {
		return nil
	}

	query := strings.ToLower(strings.TrimSpace(a.inline.query))
	if a.inline.mode == inlineModeCommand {
		var commands []*Command
		if query == "" {
			commands = a.cmdRegistry.All()
		} else {
			commands = a.cmdRegistry.Filter(query)
		}
		out := make([]inlineEntry, 0, len(commands))
		for _, cmd := range commands {
			desc := cmd.Description
			if cmd.Keybind != "" {
				desc += "  [" + cmd.Keybind + "]"
			}
			if len(cmd.Aliases) > 0 {
				desc += "  (also: /" + strings.Join(cmd.Aliases, ", /") + ")"
			}
			out = append(out, inlineEntry{
				Display:     "/" + cmd.Name,
				Insert:      "/" + cmd.Name,
				Description: desc,
				Execute:     cmd.Execute,
			})
		}
		return out
	}

	if a.inline.mode == inlineModeFile {
		// Parse query: split on ":" to get filename and optional line range
		fileQuery := query
		lineRange := ""
		if idx := strings.IndexByte(query, ':'); idx >= 0 {
			fileQuery = query[:idx]
			lineRange = query[idx:] // includes the colon
		}
		lowerFileQuery := strings.ToLower(fileQuery)

		// Directory expansion: when query ends with "/", match files with that prefix path
		dirExpand := strings.HasSuffix(fileQuery, "/")

		// Collect matching paths
		var matched []string
		for _, path := range a.fileCache {
			p := strings.ToLower(path)
			if dirExpand {
				if strings.HasPrefix(p, lowerFileQuery) {
					matched = append(matched, path)
				}
			} else if lowerFileQuery == "" || strings.Contains(p, lowerFileQuery) {
				matched = append(matched, path)
			}
		}

		// Rank by frecency (frequently used files first)
		if a.frecency != nil && len(matched) > 0 {
			matched = a.frecency.Rank(matched)
		}

		// Build entries with line range preserved in insertion
		out := make([]inlineEntry, 0, 10)
		for _, path := range matched {
			insert := "@" + path + lineRange + " "
			display := path
			if lineRange != "" {
				display = path + lineRange
			}
			out = append(out, inlineEntry{
				Display: display,
				Insert:  insert,
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
	if a.state != StateSession || a.streaming || a.commandPalette.open || a.modelDialog.open {
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
		a.inline.open = true
		a.inline.mode = inlineModeCommand
		a.inline.query = strings.TrimPrefix(prefix, "/")
		a.inline.start = 0
		a.inline.end = cursor
		a.inline.selected = 0
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
			a.inline.open = true
			a.inline.mode = inlineModeFile
			a.inline.query = between
			a.inline.start = atIdx
			a.inline.end = cursor
			a.inline.selected = 0
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
		a.inline.selected--
		if a.inline.selected < 0 {
			a.inline.selected = len(entries) - 1
		}
		return a, nil

	case msg.Code == tea.KeyDown || (msg.Mod == tea.ModCtrl && msg.Code == 'n'):
		entries := a.inlineEntries()
		if len(entries) == 0 {
			return a, nil
		}
		a.inline.selected++
		if a.inline.selected >= len(entries) {
			a.inline.selected = 0
		}
		return a, nil

	case msg.Code == tea.KeyEnter || msg.Code == tea.KeyTab:
		entries := a.inlineEntries()
		if len(entries) == 0 {
			return a.closeInlineSuggestions(), nil
		}
		if a.inline.selected < 0 {
			a.inline.selected = 0
		}
		if a.inline.selected >= len(entries) {
			a.inline.selected = len(entries) - 1
		}

		choice := entries[a.inline.selected]
		if a.inline.mode == inlineModeCommand && choice.Execute {
			a.input.Reset()
			a = a.closeInlineSuggestions()
			return a.handleSubmit(choice.Insert)
		}

		// Record frecency for file selections
		if a.inline.mode == inlineModeFile && a.frecency != nil {
			// Extract path from Insert (strip leading @ and trailing space/line range)
			inserted := strings.TrimPrefix(choice.Insert, "@")
			inserted = strings.TrimRight(inserted, " ")
			if idx := strings.IndexByte(inserted, ':'); idx >= 0 {
				inserted = inserted[:idx]
			}
			a.frecency.Record(inserted)
			a.warnOnErr(a.frecency.Save(), "frecency")
		}
		a.input.ReplaceRange(a.inline.start, a.inline.end, choice.Insert)
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
	query := strings.TrimSpace(a.inline.query)
	if query == "" {
		query = "all"
	}
	header := "Autocomplete " + a.inline.mode + " (Enter select, Esc close) - filter: " + query

	items := make([]pickerItem, len(entries))
	for i, e := range entries {
		items[i] = pickerItem{Label: e.Display, Description: e.Description}
	}
	return renderPickerList(header, items, a.inline.selected, 8)
}
