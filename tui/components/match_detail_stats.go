package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func parseFloatVal(s string) (float64, bool) {
	if s == "" || s == "<nil>" || s == "-" {
		return 0, false
	}
	var v float64
	if _, err := fmt.Sscanf(s, "%f", &v); err == nil {
		return v, true
	}
	return 0, false
}

func statBarSplit(homeVal, awayVal float64, homeOK, awayOK bool, halfW int, homeColor, awayColor lipgloss.Color) string {
	if halfW < 4 {
		halfW = 4
	}
	dimColor := lipgloss.Color("238")
	sepColor := lipgloss.Color("240")

	if !homeOK && !awayOK {
		empty := lipgloss.NewStyle().Foreground(dimColor).Render(strings.Repeat("░", halfW))
		sep := lipgloss.NewStyle().Foreground(sepColor).Render("│")
		return empty + sep + empty
	}

	total := homeVal + awayVal
	if total == 0 {
		total = 1
	}

	homeFill := int((homeVal / total) * float64(halfW))
	awayFill := int((awayVal / total) * float64(halfW))
	if homeVal > 0 && homeFill == 0 {
		homeFill = 1
	}
	if awayVal > 0 && awayFill == 0 {
		awayFill = 1
	}
	if homeFill > halfW {
		homeFill = halfW
	}
	if awayFill > halfW {
		awayFill = halfW
	}

	homeEmpty := halfW - homeFill
	awayEmpty := halfW - awayFill

	homeBar := lipgloss.NewStyle().Foreground(dimColor).Render(strings.Repeat("░", homeEmpty)) +
		lipgloss.NewStyle().Foreground(homeColor).Render(strings.Repeat("█", homeFill))

	awayBar := lipgloss.NewStyle().Foreground(awayColor).Render(strings.Repeat("█", awayFill)) +
		lipgloss.NewStyle().Foreground(dimColor).Render(strings.Repeat("░", awayEmpty))

	sep := lipgloss.NewStyle().Foreground(sepColor).Render("│")
	return homeBar + sep + awayBar
}

func displayVal(s string) string {
	if s == "" || s == "<nil>" {
		return "—"
	}
	return s
}

func (md *MatchDetail) renderStats(width, height int) string {
	if md.Details == nil || len(md.Details.Stats) == 0 {
		return mdInfoStyle.Render("sin estadísticas")
	}

	homeColor := lipgloss.Color("51")
	awayColor := lipgloss.Color("196")
	if md.Details.Events.ExtraInfo != nil {
		if md.Details.Events.ExtraInfo.HomeColor != "" {
			c := md.Details.Events.ExtraInfo.HomeColor
			if !strings.HasPrefix(c, "#") {
				c = "#" + c
			}
			homeColor = lipgloss.Color(c)
		}
		if md.Details.Events.ExtraInfo.AwayColor != "" {
			c := md.Details.Events.ExtraInfo.AwayColor
			if !strings.HasPrefix(c, "#") {
				c = "#" + c
			}
			awayColor = lipgloss.Color(c)
		}
	}

	labelW := 14
	valW := 5
	available := width - labelW - valW*2 - 4
	halfW := available / 2
	if halfW < 6 {
		halfW = 6
	}
	if halfW > 20 {
		halfW = 20
	}

	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	homeWinStyle := lipgloss.NewStyle().Width(valW).Align(lipgloss.Right).Bold(true).Foreground(homeColor)
	awayWinStyle := lipgloss.NewStyle().Width(valW).Align(lipgloss.Left).Bold(true).Foreground(awayColor)
	homeLoseStyle := lipgloss.NewStyle().Width(valW).Align(lipgloss.Right).Foreground(lipgloss.Color("240"))
	awayLoseStyle := lipgloss.NewStyle().Width(valW).Align(lipgloss.Left).Foreground(lipgloss.Color("240"))
	homeTieStyle := lipgloss.NewStyle().Width(valW).Align(lipgloss.Right).Foreground(lipgloss.Color("255"))
	awayTieStyle := lipgloss.NewStyle().Width(valW).Align(lipgloss.Left).Foreground(lipgloss.Color("255"))
	labelStyle := lipgloss.NewStyle().Width(labelW).Align(lipgloss.Left).Foreground(lipgloss.Color("240"))

	homeName := md.HomeName
	awayName := md.AwayName
	if len(homeName) > 14 {
		homeName = homeName[:13] + "…"
	}
	if len(awayName) > 14 {
		awayName = awayName[:13] + "…"
	}

	var lines []string

	for _, cat := range md.Details.Stats {
		lines = append(lines, mdSectionHeader.Render(cat.Title))
		for _, stat := range cat.Stats {
			hv, hOK := parseFloatVal(stat.Home)
			av, aOK := parseFloatVal(stat.Away)

			homeValStr := displayVal(stat.Home)
			awayValStr := displayVal(stat.Away)

			var homeStyled, awayStyled string
			if hOK && aOK {
				if hv > av {
					homeStyled = homeWinStyle.Render(homeValStr)
					awayStyled = awayLoseStyle.Render(awayValStr)
				} else if av > hv {
					homeStyled = homeLoseStyle.Render(homeValStr)
					awayStyled = awayWinStyle.Render(awayValStr)
				} else {
					homeStyled = homeTieStyle.Render(homeValStr)
					awayStyled = awayTieStyle.Render(awayValStr)
				}
			} else if hOK {
				homeStyled = homeWinStyle.Render(homeValStr)
				awayStyled = dimStyle.Width(valW).Align(lipgloss.Left).Render(awayValStr)
			} else if aOK {
				homeStyled = dimStyle.Width(valW).Align(lipgloss.Right).Render(homeValStr)
				awayStyled = awayWinStyle.Render(awayValStr)
			} else {
				homeStyled = dimStyle.Width(valW).Align(lipgloss.Right).Render(homeValStr)
				awayStyled = dimStyle.Width(valW).Align(lipgloss.Left).Render(awayValStr)
			}

			bar := statBarSplit(hv, av, hOK, aOK, halfW, homeColor, awayColor)

			row := lipgloss.JoinHorizontal(lipgloss.Top,
				labelStyle.Render(stat.Label),
				" ",
				homeStyled,
				" ",
				bar,
				" ",
				awayStyled,
			)
			lines = append(lines, row)
		}
		lines = append(lines, "")
	}

	body := lipgloss.JoinVertical(lipgloss.Left, lines...)
	centered := lipgloss.NewStyle().Width(width).Align(lipgloss.Center).Render(body)
	return md.applyScroll(centered, width, height)
}
