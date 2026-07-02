package components

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (md *MatchDetail) renderH2H(width, height int) string {
	if md.Details == nil {
		return mdInfoStyle.Render("sin datos H2H")
	}

	h2h := md.Details.H2H
	var lines []string

	availW := width - 4
	if availW < 30 {
		availW = 30
	}
	green := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("46")).Render
	drawLabel := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("220")).Render
	colW := (availW - 10) / 2
	if colW < 8 {
		colW = 8
	}
	homeLabel := lipgloss.NewStyle().Width(colW).Align(lipgloss.Right).Bold(true).Foreground(lipgloss.Color("255")).Render(md.HomeName)
	awayLabel := lipgloss.NewStyle().Width(colW).Align(lipgloss.Left).Bold(true).Foreground(lipgloss.Color("255")).Render(md.AwayName)
	lines = append(lines, fmt.Sprintf("  %s   %s   %s",
		homeLabel, drawLabel("EMPATES"), awayLabel))
	homeVal := lipgloss.NewStyle().Width(colW).Align(lipgloss.Right).Render(fmt.Sprintf("%d", h2h.HomeWins))
	drawVal := lipgloss.NewStyle().Width(8).Align(lipgloss.Center).Render(fmt.Sprintf("%d", h2h.Draws))
	awayVal := lipgloss.NewStyle().Width(colW).Align(lipgloss.Left).Render(fmt.Sprintf("%d", h2h.AwayWins))
	lines = append(lines, fmt.Sprintf("  %s   %s   %s", green(homeVal), drawVal, awayVal))
	lines = append(lines, "")

	if len(h2h.Matches) > 0 {
		lines = append(lines, mdSectionHeader.Render("ENFRENTAMIENTOS DIRECTOS"))

		dateW := 11
		scoreW := 7
		teamW := 14
		if teamW < 6 {
			teamW = 6
		}
		compW := availW - dateW - scoreW - teamW*2 - 6
		if compW < 14 {
			compW = 14
		}

		winnerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("46")).Render
		loserStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render
		drawStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("220")).Render

		for _, m := range h2h.Matches {
			score := m.Score
			parts := strings.SplitN(m.Score, " : ", 2)
			if len(parts) == 2 {
				hs, _ := strconv.Atoi(strings.TrimSpace(parts[0]))
				as, _ := strconv.Atoi(strings.TrimSpace(parts[1]))
				if hs > as {
					score = winnerStyle(fmt.Sprintf("%-*s", scoreW, m.Score))
				} else if as > hs {
					score = loserStyle(fmt.Sprintf("%-*s", scoreW, m.Score))
				} else {
					score = drawStyle(fmt.Sprintf("%-*s", scoreW, m.Score))
				}
			} else {
				score = fmt.Sprintf("%-*s", scoreW, m.Score)
			}
			dateStr := fmt.Sprintf("%-*s", dateW, m.Date)
			hTeam := fmt.Sprintf("%*s", teamW, truncate(m.HomeTeam, teamW))
			aTeam := fmt.Sprintf("%-*s", teamW, truncate(m.AwayTeam, teamW))
			comp := truncate(m.Competition, compW)
			matchLine := fmt.Sprintf("%s  %s  %s  %s  %s",
				dateStr, hTeam, score, aTeam, comp)
			lines = append(lines, matchLine)
		}
		lines = append(lines, "")
	}

	if len(h2h.HomeForm) > 0 || len(h2h.AwayForm) > 0 {
		lines = append(lines, mdSectionHeader.Render("ÚLTIMOS PARTIDOS"))

		halfW := (availW - 3) / 2
		if halfW < 20 {
			halfW = 20
		}
		oppW := halfW - 14
		if oppW < 8 {
			oppW = 8
		}
		scoreW := 7
		halfCol := lipgloss.NewStyle().Width(halfW)

		homeHdr := lipgloss.NewStyle().Width(halfW).Align(lipgloss.Center).Bold(true).Foreground(lipgloss.Color("255")).Render(md.HomeName)
		awayHdr := lipgloss.NewStyle().Width(halfW).Align(lipgloss.Center).Bold(true).Foreground(lipgloss.Color("255")).Render(md.AwayName)
		lines = append(lines, fmt.Sprintf("%s │ %s", homeHdr, awayHdr))
		lines = append(lines, fmt.Sprintf("%s┼%s", strings.Repeat("─", halfW+1), strings.Repeat("─", halfW+1)))

		winStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("46"))
		lossStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196"))

		maxRows := len(h2h.HomeForm)
		if len(h2h.AwayForm) > maxRows {
			maxRows = len(h2h.AwayForm)
		}

		for i := 0; i < maxRows; i++ {
			left := strings.Repeat(" ", halfW)
			if i < len(h2h.HomeForm) {
				fe := h2h.HomeForm[i]
				var rowStyle lipgloss.Style
				switch fe.Result {
				case "W":
					rowStyle = winStyle
				case "L":
					rowStyle = lossStyle
				}
				opp := rowStyle.Render(fmt.Sprintf("%-*s", oppW, truncate(fe.Opponent, oppW)))
				sc := rowStyle.Render(fmt.Sprintf("%-*s", scoreW, fe.Score))
				left = halfCol.Render(fmt.Sprintf("%s  %s", opp, sc))
			}
			right := strings.Repeat(" ", halfW)
			if i < len(h2h.AwayForm) {
				fe := h2h.AwayForm[i]
				var rowStyle lipgloss.Style
				switch fe.Result {
				case "W":
					rowStyle = winStyle
				case "L":
					rowStyle = lossStyle
				}
				opp := rowStyle.Render(fmt.Sprintf("%-*s", oppW, truncate(fe.Opponent, oppW)))
				sc := rowStyle.Render(fmt.Sprintf("%-*s", scoreW, fe.Score))
				right = halfCol.Render(fmt.Sprintf("%s  %s", opp, sc))
			}
			lines = append(lines, fmt.Sprintf("%s │ %s", left, right))
		}
	}

	body := lipgloss.JoinVertical(lipgloss.Top, lines...)
	return md.applyScroll(body, width, height)
}

func truncate(s string, maxW int) string {
	if maxW <= 0 {
		return ""
	}
	if len(s) <= maxW {
		return s
	}
	if maxW <= 1 {
		return "…"
	}
	return s[:maxW-1] + "…"
}
