package tui

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
)

func dummyHandler(a *App) tea.Cmd { return nil }

func TestRegistry_Register_Command(t *testing.T) {
	r := NewCommandRegistry()
	r.Register(Command{Name: "help", Title: "Help", Handler: dummyHandler})
	assert.NotNil(t, r.Get("help"))
}

func TestRegistry_Get_ByName(t *testing.T) {
	r := NewCommandRegistry()
	r.Register(Command{Name: "help", Title: "Help", Handler: dummyHandler})
	cmd := r.Get("help")
	assert.Equal(t, "help", cmd.Name)
	assert.Equal(t, "Help", cmd.Title)
}

func TestRegistry_Get_ByAlias(t *testing.T) {
	r := NewCommandRegistry()
	r.Register(Command{Name: "help", Aliases: []string{"commands"}, Handler: dummyHandler})
	cmd := r.Get("commands")
	assert.NotNil(t, cmd)
	assert.Equal(t, "help", cmd.Name)
}

func TestRegistry_All_ReturnsAllVisible(t *testing.T) {
	r := NewCommandRegistry()
	r.Register(Command{Name: "a", Title: "A", Handler: dummyHandler})
	r.Register(Command{Name: "b", Title: "B", Handler: dummyHandler})
	r.Register(Command{Name: "c", Title: "C", Handler: dummyHandler})
	all := r.All()
	assert.Len(t, all, 3)
	assert.Equal(t, "a", all[0].Name)
	assert.Equal(t, "b", all[1].Name)
	assert.Equal(t, "c", all[2].Name)
}

func TestRegistry_All_ExcludesHidden(t *testing.T) {
	r := NewCommandRegistry()
	r.Register(Command{Name: "visible", Handler: dummyHandler})
	r.Register(Command{Name: "hidden", Hidden: true, Handler: dummyHandler})
	all := r.All()
	assert.Len(t, all, 1)
	assert.Equal(t, "visible", all[0].Name)
}

func TestRegistry_Suggested_ReturnsSuggestedOnly(t *testing.T) {
	r := NewCommandRegistry()
	r.Register(Command{Name: "help", Suggested: true, Handler: dummyHandler})
	r.Register(Command{Name: "exit", Handler: dummyHandler})
	r.Register(Command{Name: "models", Suggested: true, Handler: dummyHandler})
	suggested := r.Suggested()
	assert.Len(t, suggested, 2)
	assert.Equal(t, "help", suggested[0].Name)
	assert.Equal(t, "models", suggested[1].Name)
}

func TestRegistry_ByCategory_GroupsCorrectly(t *testing.T) {
	r := NewCommandRegistry()
	r.Register(Command{Name: "new", Category: CategorySession, Handler: dummyHandler})
	r.Register(Command{Name: "sidebar", Category: CategoryView, Handler: dummyHandler})
	r.Register(Command{Name: "exit", Category: CategorySystem, Handler: dummyHandler})
	groups := r.ByCategory()
	assert.Len(t, groups[CategorySession], 1)
	assert.Len(t, groups[CategoryView], 1)
	assert.Len(t, groups[CategorySystem], 1)
}

func TestRegistry_Filter_MatchesTitleAndDescription(t *testing.T) {
	r := NewCommandRegistry()
	r.Register(Command{Name: "help", Title: "Help", Description: "Show commands", Handler: dummyHandler})
	r.Register(Command{Name: "exit", Title: "Exit", Description: "Quit ripcode", Handler: dummyHandler})
	r.Register(Command{Name: "models", Title: "Models", Description: "Search models", Handler: dummyHandler})

	matches := r.Filter("model")
	assert.Len(t, matches, 1)
	assert.Equal(t, "models", matches[0].Name)

	matches = r.Filter("quit")
	assert.Len(t, matches, 1)
	assert.Equal(t, "exit", matches[0].Name)
}

func TestRegistry_Disabled_ExcludedFromAll(t *testing.T) {
	r := NewCommandRegistry()
	r.Register(Command{Name: "active", Handler: dummyHandler})
	r.Register(Command{Name: "disabled", Disabled: func() bool { return true }, Handler: dummyHandler})
	all := r.All()
	assert.Len(t, all, 1)
	assert.Equal(t, "active", all[0].Name)
}

func TestRegistry_Register_DuplicateName_Panics(t *testing.T) {
	r := NewCommandRegistry()
	r.Register(Command{Name: "help", Handler: dummyHandler})
	assert.Panics(t, func() {
		r.Register(Command{Name: "help", Handler: dummyHandler})
	})
}
