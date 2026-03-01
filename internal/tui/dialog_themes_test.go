package tui

import (
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/stephenbrandon/ripcode/internal/tui/components"
	"github.com/stretchr/testify/assert"
)

func TestApp_ThemesCommand_OpensDialog(t *testing.T) {
	a := makeSessionApp(t)
	model, _ := a.Update(components.InputSubmitMsg{Value: "/themes"})
	a = model.(App)
	assert.True(t, a.themesDialogOpen)
}

func TestApp_ThemesDialog_EscCloses(t *testing.T) {
	a := makeSessionApp(t)
	model, _ := a.Update(components.InputSubmitMsg{Value: "/themes"})
	a = model.(App)
	model, _ = a.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	a = model.(App)
	assert.False(t, a.themesDialogOpen)
}
