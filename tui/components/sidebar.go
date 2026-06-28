package components

import (
	"github.com/charmbracelet/lipgloss"
)

type Sidebar struct {
	title string
	items []string
}

func NewSidebar(title string, items []string) Sidebar {
	return Sidebar{title: title, items: items}
}

func (s Sidebar) Render(width, height int) string {
	if width < 2 || height < 2 {
		return ""
	}

	header := lipgloss.NewStyle().
		Width(width).
		Align(lipgloss.Center).
		Bold(true).
		Foreground(lipgloss.Color("63")).
		Render(s.title)

	itemStr := ""
	for i, item := range s.items {
		if i > 0 {
			itemStr += "\n"
		}
		itemStr += lipgloss.NewStyle().
			Width(width).
			PaddingLeft(2).
			Render(item)
	}

	body := lipgloss.JoinVertical(lipgloss.Top, header, "", itemStr)

	remaining := height - lipgloss.Height(body)
	if remaining > 0 {
		body += "\n" + lipgloss.NewStyle().Width(width).Height(remaining).Render("")
	}

	return body
}
