package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	capStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("51"))
	subOutStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	subInStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("46"))
	yellowStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("220"))
	redStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
)

func indicatorVisualWidth(p PlayerLineup) int {
	parts := 0
	runes := 0
	if p.IsCaptain {
		parts++
		runes++
	}
	if p.SubOut {
		parts++
		runes++
	}
	if p.SubIn {
		parts++
		runes++
	}
	switch p.CardType {
	case "Yellow":
		parts++
		runes++
	case "Red":
		parts++
		runes++
	case "SecondYellow":
		parts++
		runes += 3
	}
	if parts <= 1 {
		return runes
	}
	return runes + parts - 1
}

func (md MatchDetail) playerIndicator(p PlayerLineup) string {
	var parts []string
	if p.IsCaptain {
		parts = append(parts, capStyle.Render("C"))
	}
	if p.SubOut {
		parts = append(parts, subOutStyle.Render("↓"))
	}
	if p.SubIn {
		parts = append(parts, subInStyle.Render("↑"))
	}
	switch p.CardType {
	case "Yellow":
		parts = append(parts, yellowStyle.Render("!"))
	case "Red":
		parts = append(parts, redStyle.Render("R"))
	case "SecondYellow":
		parts = append(parts, yellowStyle.Render("!")+yellowStyle.Render("!")+redStyle.Render("R"))
	}
	if len(parts) == 0 {
		return ""
	}
	return strings.Join(parts, " ")
}

func (md *MatchDetail) renderLineup(width, height int) string {
	if md.Details == nil {
		return mdInfoStyle.Render("sin alineaciones")
	}

	lu := md.Details.Lineup

	homeW := width * 30 / 100
	centerW := width * 40 / 100
	awayW := width * 30 / 100
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

	actionW := 0
	for _, p := range starters {
		if w := indicatorVisualWidth(p); w > actionW {
			actionW = w
		}
	}
	for _, p := range subs {
		if w := indicatorVisualWidth(p); w > actionW {
			actionW = w
		}
	}

	nameW := colW - 8 - posW
	if actionW > 0 {
		nameW = colW - 10 - posW - actionW
	}
	if nameW < 3 {
		nameW = 3
	}

	lines = append(lines, lipgloss.NewStyle().Width(colW).Bold(true).
		Foreground(lipgloss.Color("255")).Render("Titulares"))
	lines = append(lines, "")

	for _, p := range starters {
		line := fmt.Sprintf("  %2s  %-*s  %-*s", p.Number, nameW, p.Name, posW, p.PosName)
		if actionW > 0 {
			indicator := md.playerIndicator(p)
			if vw := indicatorVisualWidth(p); vw < actionW {
				indicator += strings.Repeat(" ", actionW-vw)
			}
			line += "  " + indicator
		}
		lines = append(lines, lipgloss.NewStyle().Width(colW).Foreground(lipgloss.Color("255")).Render(line))
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
		line := fmt.Sprintf("  %2s  %-*s  %-*s", p.Number, nameW, p.Name, posW, p.PosName)
		if actionW > 0 {
			indicator := md.playerIndicator(p)
			if vw := indicatorVisualWidth(p); vw < actionW {
				indicator += strings.Repeat(" ", actionW-vw)
			}
			line += "  " + indicator
		}
		lines = append(lines, lipgloss.NewStyle().Width(colW).Foreground(lipgloss.Color("240")).Render(line))
	}

	return lines
}
