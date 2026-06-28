package components

import (
	"tifo/internal/domain"

	"github.com/charmbracelet/lipgloss"
)

type LeagueItem struct {
	TIFOID       domain.TIFOID
	Name         string
	OriginalName string
	Country      string
}

type LeagueList struct {
	leagues     []LeagueItem
	cursor      int
	itemStyle   lipgloss.Style
	cursorStyle lipgloss.Style
}

func NewLeagueList(leagues []LeagueItem) LeagueList {
	return LeagueList{
		leagues: leagues,
		itemStyle: lipgloss.NewStyle().
			PaddingLeft(2).
			Foreground(lipgloss.Color("250")),
		cursorStyle: lipgloss.NewStyle().
			PaddingLeft(1).
			Foreground(lipgloss.Color("39")).
			Bold(true),
	}
}

func (l *LeagueList) SetLeagues(leagues []LeagueItem) {
	l.leagues = leagues
	if l.cursor >= len(leagues) {
		l.cursor = 0
	}
}

func (l *LeagueList) Cursor() int {
	return l.cursor
}

func (l *LeagueList) Selected() *LeagueItem {
	if len(l.leagues) == 0 {
		return nil
	}
	return &l.leagues[l.cursor]
}

func (l *LeagueList) Up() {
	if l.cursor > 0 {
		l.cursor--
	}
}

func (l *LeagueList) Down() {
	if l.cursor < len(l.leagues)-1 {
		l.cursor++
	}
}

func (l LeagueList) Render(width, height int) string {
	if width < 3 || height < 3 {
		return ""
	}

	var items []string
	for i, lg := range l.leagues {
		label := lg.Name
		if i == l.cursor {
			label = l.cursorStyle.Render("▸ " + label)
		} else {
			label = l.itemStyle.Render("  " + label)
		}
		items = append(items, label)
	}

	body := lipgloss.JoinVertical(lipgloss.Top, items...)

	remaining := height - lipgloss.Height(body)
	if remaining > 0 {
		body += "\n" + lipgloss.NewStyle().Width(width).Height(remaining).Render("")
	}

	return body
}
