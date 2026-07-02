package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (md *MatchDetail) renderLineup(width, height int) string {
	if md.Details == nil {
		return mdInfoStyle.Render("sin alineaciones")
	}

	lu := md.Details.Lineup

	homeW := width * 20 / 100
	centerW := width * 60 / 100
	awayW := width * 20 / 100
	if homeW < 10 {
		homeW = 10
	}
	if awayW < 10 {
		awayW = 10
	}

	homeLines := md.teamColumn(lu.HomeFormation, lu.HomeCoach, lu.HomeStarters, lu.HomeSubs, homeW)
	awayLines := md.teamColumn(lu.AwayFormation, lu.AwayCoach, lu.AwayStarters, lu.AwaySubs, awayW)

	maxLen := len(homeLines)
	if len(awayLines) > maxLen {
		maxLen = len(awayLines)
	}

	var rendered []string
	for i := 0; i < maxLen; i++ {
		h := lipgloss.NewStyle().Width(homeW).Render("")
		a := lipgloss.NewStyle().Width(awayW).Render("")
		if i < len(homeLines) {
			h = homeLines[i]
		}
		if i < len(awayLines) {
			a = awayLines[i]
		}
		row := lipgloss.JoinHorizontal(lipgloss.Top,
			h,
			lipgloss.NewStyle().Width(centerW).Render(""),
			a,
		)
		rendered = append(rendered, row)
	}

	body := lipgloss.JoinVertical(lipgloss.Top, rendered...)
	return md.applyScroll(body, width, height)
}

func (md MatchDetail) teamColumn(formation, coach string, starters, subs []PlayerLineup, colW int) []string {
	var lines []string

	lines = append(lines, lipgloss.NewStyle().Width(colW).Align(lipgloss.Center).
		Bold(true).Foreground(lipgloss.Color("39")).Render(fmt.Sprintf("[%s]", formation)))
	lines = append(lines, "")

	lines = append(lines, lipgloss.NewStyle().Width(colW).
		Foreground(lipgloss.Color("240")).Render(fmt.Sprintf("DT: %s", coach)))
	lines = append(lines, "")

	posW := 3
	for _, p := range starters {
		if len(p.PosName) > posW {
			posW = len(p.PosName)
		}
	}
	for _, p := range subs {
		if len(p.PosName) > posW {
			posW = len(p.PosName)
		}
	}
	nameW := colW - 8 - posW
	if nameW < 3 {
		nameW = 3
	}

	lines = append(lines, lipgloss.NewStyle().Width(colW).Bold(true).
		Foreground(lipgloss.Color("255")).Render("Titulares"))
	lines = append(lines, "")

	for _, p := range starters {
		line := lipgloss.NewStyle().Width(colW).
			Foreground(lipgloss.Color("255")).Render(fmt.Sprintf("  %2s  %-*s  %-*s", p.Number, nameW, p.Name, posW, p.PosName))
		lines = append(lines, line)
	}

	lines = append(lines, "")
	sep := strings.Repeat("─", colW)
	lines = append(lines, lipgloss.NewStyle().Width(colW).
		Foreground(lipgloss.Color("236")).Render(sep))
	lines = append(lines, "")

	lines = append(lines, lipgloss.NewStyle().Width(colW).Bold(true).
		Foreground(lipgloss.Color("240")).Render("Suplentes"))
	lines = append(lines, "")

	for _, p := range subs {
		line := lipgloss.NewStyle().Width(colW).
			Foreground(lipgloss.Color("240")).Render(fmt.Sprintf("  %2s  %-*s  %-*s", p.Number, nameW, p.Name, posW, p.PosName))
		lines = append(lines, line)
	}

	return lines
}
