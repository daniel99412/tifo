package components

import (
	"github.com/charmbracelet/lipgloss"
)

type Center struct {
	content string
	style   lipgloss.Style
}

func NewCenter(content string) Center {
	return Center{
		content: content,
		style: lipgloss.NewStyle().
			Align(lipgloss.Center, lipgloss.Center),
	}
}

func (c Center) Render(width, height int) string {
	if width < 2 || height < 2 {
		return ""
	}

	return c.style.Width(width).Height(height).
		Render(c.content)
}
