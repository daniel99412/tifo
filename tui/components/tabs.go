package components

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	tabSep = lipgloss.NewStyle().
		Foreground(lipgloss.Color("236")).
		Render("│")

	tabActiveStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("15")).
			Background(lipgloss.Color("39")).
			Padding(0, 2)

	tabInactiveStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("240")).
				Padding(0, 2)
)

type Tabs struct {
	items  []string
	active int
}

func NewTabs(items []string) Tabs {
	return Tabs{items: items, active: 0}
}

func (t *Tabs) Left() {
	t.active--
	if t.active < 0 {
		t.active = len(t.items) - 1
	}
}

func (t *Tabs) Right() {
	t.active++
	if t.active >= len(t.items) {
		t.active = 0
	}
}

func (t Tabs) Active() int {
	return t.active
}

func (t Tabs) ActiveName() string {
	if t.active < len(t.items) {
		return t.items[t.active]
	}
	return ""
}

func (t Tabs) Render(width int) string {
	var parts []string
	for i, name := range t.items {
		if i > 0 {
			parts = append(parts, tabSep)
		}
		if i == t.active {
			parts = append(parts, tabActiveStyle.Render(name))
		} else {
			parts = append(parts, tabInactiveStyle.Render(name))
		}
	}
	row := lipgloss.JoinHorizontal(lipgloss.Top, parts...)
	rowW := lipgloss.Width(row)

	// fill gap with bottom border line
	fill := ""
	if rowW < width {
		fill = lipgloss.NewStyle().
			Foreground(lipgloss.Color("236")).
			Render(strings.Repeat("─", width-rowW))
	}

	return lipgloss.JoinHorizontal(lipgloss.Top, row, fill)
}
