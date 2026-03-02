package tui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
)

// CommandCategory groups commands in the palette.
type CommandCategory string

const (
	CategorySession CommandCategory = "Session"
	CategoryView    CommandCategory = "View"
	CategoryAgent   CommandCategory = "Agent"
	CategorySystem  CommandCategory = "System"
)

// categoryOrder defines display order in the palette.
var categoryOrder = []CommandCategory{
	CategorySession, CategoryView, CategoryAgent, CategorySystem,
}

// Command is a registered slash command.
type Command struct {
	Name        string
	Aliases     []string
	Category    CommandCategory
	Title       string
	Description string
	Keybind     string
	Suggested   bool
	Hidden      bool
	Execute     bool
	Disabled    func() bool
	Handler     func(a *App) tea.Cmd
	searchText  string // pre-computed lowercase haystack for Filter()
}

func (c *Command) isDisabled() bool {
	return c.Disabled != nil && c.Disabled()
}

// CommandRegistry holds all commands indexed by name and alias.
type CommandRegistry struct {
	commands []*Command
	byName   map[string]*Command
}

// NewCommandRegistry creates an empty registry.
func NewCommandRegistry() *CommandRegistry {
	return &CommandRegistry{byName: make(map[string]*Command)}
}

// Register adds a command. Panics on duplicate name or alias — this catches
// programming errors at init time (similar to regexp.MustCompile).
func (r *CommandRegistry) Register(cmd Command) {
	if _, ok := r.byName[cmd.Name]; ok {
		panic(fmt.Sprintf("duplicate command name: %s", cmd.Name))
	}
	c := cmd // copy
	c.searchText = strings.ToLower(c.Name + " " + strings.Join(c.Aliases, " ") + " " + c.Title + " " + c.Description)
	r.byName[c.Name] = &c
	for _, alias := range c.Aliases {
		if _, ok := r.byName[alias]; ok {
			panic(fmt.Sprintf("duplicate command alias: %s", alias))
		}
		r.byName[alias] = &c
	}
	r.commands = append(r.commands, &c)
}

// Get looks up a command by name or alias.
func (r *CommandRegistry) Get(name string) *Command {
	return r.byName[strings.ToLower(name)]
}

// All returns all visible, enabled commands in insertion order.
func (r *CommandRegistry) All() []*Command {
	out := make([]*Command, 0, len(r.commands))
	for _, c := range r.commands {
		if c.Hidden || c.isDisabled() {
			continue
		}
		out = append(out, c)
	}
	return out
}

// Suggested returns only suggested commands.
func (r *CommandRegistry) Suggested() []*Command {
	out := make([]*Command, 0)
	for _, c := range r.commands {
		if c.Suggested && !c.Hidden && !c.isDisabled() {
			out = append(out, c)
		}
	}
	return out
}

// ByCategory groups visible commands by category.
func (r *CommandRegistry) ByCategory() map[CommandCategory][]*Command {
	out := make(map[CommandCategory][]*Command)
	for _, c := range r.commands {
		if c.Hidden || c.isDisabled() {
			continue
		}
		out[c.Category] = append(out[c.Category], c)
	}
	return out
}

// Filter returns commands matching the query against name, aliases, title, and description.
func (r *CommandRegistry) Filter(query string) []*Command {
	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		return r.All()
	}
	out := make([]*Command, 0)
	for _, c := range r.commands {
		if c.Hidden || c.isDisabled() {
			continue
		}
		if strings.Contains(c.searchText, q) {
			out = append(out, c)
		}
	}
	return out
}
