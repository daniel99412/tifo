package components

import (
	"tifo/fotmob"

	"github.com/charmbracelet/lipgloss"
)

type LeagueList struct {
	leagues     []fotmob.GroupedLeague
	cursor      int
	itemStyle   lipgloss.Style
	cursorStyle lipgloss.Style
}

func NewLeagueList(leagues []fotmob.GroupedLeague) LeagueList {
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

func (l *LeagueList) SetLeagues(leagues []fotmob.GroupedLeague) {
	l.leagues = leagues
	if l.cursor >= len(leagues) {
		l.cursor = 0
	}
}

func (l *LeagueList) Cursor() int {
	return l.cursor
}

func (l *LeagueList) Selected() *fotmob.GroupedLeague {
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
