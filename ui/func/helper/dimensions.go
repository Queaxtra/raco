package helper

import (
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
)

type DimensionInputs struct {
	URLInput         *textinput.Model
	BodyInput        *textarea.Model
	ResponseViewport *viewport.Model
}

func Dimensions(width, height int, inputs DimensionInputs) {
	sidebarWidth := width / 4
	if sidebarWidth < 30 {
		sidebarWidth = 30
	}
	mainWidth := width - sidebarWidth

	inputs.URLInput.Width = mainWidth - 30
	inputs.BodyInput.SetWidth(mainWidth - 8)
	inputs.BodyInput.SetHeight((height / 3) - 4)
	inputs.ResponseViewport.Width = mainWidth - 8
	inputs.ResponseViewport.Height = height - 12
}
