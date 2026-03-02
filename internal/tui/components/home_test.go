package components

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHome_RendersLogo(t *testing.T) {
	h := NewHome()
	h.SetSize(80, 30)
	view := h.View()
	assert.Contains(t, view, "██████╗", "home should render ASCII logo")
}

func TestHome_RendersCodeSubtitle(t *testing.T) {
	h := NewHome()
	h.SetSize(80, 30)
	view := h.View()
	assert.Contains(t, view, "code", "home should render 'code' subtitle")
}

func TestHome_RendersInput(t *testing.T) {
	h := NewHome()
	h.SetSize(80, 30)
	view := h.View()
	assert.Contains(t, view, "┃", "home should render input accent border")
	assert.Contains(t, view, "What do you want to do?", "home should render input placeholder")
}

func TestHome_RendersFooter(t *testing.T) {
	h := NewHome()
	h.SetSize(80, 30)
	h.SetWorkDir("/tmp/project")
	h.SetVersion("v0.1.0")
	view := h.View()
	assert.Contains(t, view, "/tmp/project", "home should render workdir in footer")
	assert.Contains(t, view, "v0.1.0", "home should render version in footer")
}

func TestHome_RendersTip(t *testing.T) {
	h := NewHome()
	h.SetSize(80, 30)
	view := h.View()
	assert.Contains(t, view, "●", "home should render tip bullet")
}

func TestHome_Input(t *testing.T) {
	h := NewHome()
	h.SetSize(80, 30)
	// Should have accessible Input
	assert.NotNil(t, h.Input())
}
