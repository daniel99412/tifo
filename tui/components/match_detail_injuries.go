package components

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

func (md *MatchDetail) renderInjuries(width, height int) string {
	if md.Details == nil {
		return mdInfoStyle.Render("sin datos de lesiones")
	}

	inf := md.Details.Injuries
	colW := (width - 4) / 2
	if colW < 15 {
		colW = 15
	}

	homeLines := md.injuryColumn(md.HomeName, inf.Home, colW)
	awayLines := md.injuryColumn(md.AwayName, inf.Away, colW)

	if len(homeLines) == 0 && len(awayLines) == 0 {
		return mdInfoStyle.Render("sin jugadores lesionados")
	}

	maxLen := len(homeLines)
	if len(awayLines) > maxLen {
		maxLen = len(awayLines)
	}

	var rendered []string
	for i := 0; i < maxLen; i++ {
		h := ""
		a := ""
		if i < len(homeLines) {
			h = homeLines[i]
		}
		if i < len(awayLines) {
			a = awayLines[i]
		}
		row := lipgloss.JoinHorizontal(lipgloss.Top, h, lipgloss.NewStyle().Width(4).Render(""), a)
		rendered = append(rendered, row)
	}

	body := lipgloss.JoinVertical(lipgloss.Top, rendered...)
	return md.applyScroll(body, width, height)
}

func (md MatchDetail) injuryColumn(teamName string, players []InjuryPlayer, colW int) []string {
	if len(players) == 0 {
		return nil
	}

	var lines []string
	lines = append(lines, lipgloss.NewStyle().Width(colW).Bold(true).
		Foreground(lipgloss.Color("255")).Render(teamName))
	lines = append(lines, "")

	for _, p := range players {
		tag := p.Type
		if tag == "" {
			tag = "lesión"
		}
		ret := p.Return
		if ret == "" {
			ret = "?"
		}
		line := lipgloss.NewStyle().Width(colW).
			Foreground(lipgloss.Color("196")).Render(fmt.Sprintf("  %s (%s) — %s", p.Name, tag, ret))
		lines = append(lines, line)
	}

	return lines
}
