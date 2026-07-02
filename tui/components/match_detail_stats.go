package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"tifo/internal/domain"
)

var momentumBlocks = []rune{'▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}

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
	if md.Details == nil {
		return mdInfoStyle.Render("cargando estadísticas...")
	}
	if len(md.Details.StatsByPeriod) == 0 {
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

	// Fixed header: momentum + period subtabs (not affected by scroll)
	var header []string
	if len(md.Details.Momentum) > 0 {
		header = append(header, md.renderMomentum(width, homeColor, awayColor))
		header = append(header, "")
	}
	periods := md.availablePeriods()
	if len(periods) > 1 {
		header = append(header, md.renderPeriodSubtabs(periods, width))
		header = append(header, "")
	}

	headerStr := lipgloss.JoinVertical(lipgloss.Left, header...)
	headerH := strings.Count(headerStr, "\n") + 1

	availH := height - headerH
	if availH < 3 {
		availH = 3
	}

	// Scrollable body: stats table only
	stats := md.Details.StatsByPeriod[md.Details.SelectedPeriod]
	var body string
	if len(stats) == 0 {
		body = mdInfoStyle.Render("sin datos para este periodo")
	} else {
		body = md.renderStatsTable(stats, width, homeColor, awayColor)
	}
	centered := lipgloss.NewStyle().Width(width).Align(lipgloss.Center).Render(body)
	scrolled := md.applyScroll(centered, width, availH)

	return lipgloss.JoinVertical(lipgloss.Left,
		lipgloss.NewStyle().Width(width).Align(lipgloss.Center).Render(headerStr),
		scrolled,
	)
}

func (md *MatchDetail) renderMomentum(width int, homeColor, awayColor lipgloss.Color) string {
	points := md.Details.Momentum
	if len(points) == 0 {
		return ""
	}

	chartW := width - 4
	if chartW < 20 {
		chartW = 20
	}

	const rowsPerSide = 3

	// Downsample: map each column to a momentum value (max abs in bucket).
	cols := make([]int, chartW)
	for x := 0; x < chartW; x++ {
		start := x * len(points) / chartW
		end := (x + 1) * len(points) / chartW
		if end <= start {
			end = start + 1
		}
		if end > len(points) {
			end = len(points)
		}
		best := 0
		for i := start; i < end; i++ {
			if absInt(points[i].Value) > absInt(best) {
				best = points[i].Value
			}
		}
		cols[x] = best
	}

	maxMinute := points[len(points)-1].Minute
	if maxMinute < 1 {
		maxMinute = 1
	}

	dimAxis := lipgloss.Color("238")

	var lines []string

	// Home area (above axis), render top to bottom
	for ri := rowsPerSide - 1; ri >= 0; ri-- {
		lines = append(lines, momentumRow(cols, ri, rowsPerSide, true, homeColor))
	}

	// Axis line with tick marks
	lines = append(lines, momentumAxis(chartW, maxMinute, dimAxis))

	// Away area (below axis), render top to bottom
	for ri := 0; ri < rowsPerSide; ri++ {
		lines = append(lines, momentumRow(cols, ri, rowsPerSide, false, awayColor))
	}

	// Time labels
	lines = append(lines, momentumTimeLabels(chartW, maxMinute, dimAxis))

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

func momentumRow(cols []int, ri, rowsPerSide int, home bool, color lipgloss.Color) string {
	var sb strings.Builder
	for _, v := range cols {
		var val float64
		if home && v > 0 {
			val = float64(v) / 100.0 * float64(rowsPerSide)
		} else if !home && v < 0 {
			val = float64(-v) / 100.0 * float64(rowsPerSide)
		}
		switch {
		case val >= float64(ri+1):
			sb.WriteRune('█')
		case val > float64(ri):
			frac := val - float64(ri)
			idx := int(frac * 8)
			if idx > 7 {
				idx = 7
			}
			sb.WriteRune(momentumBlocks[idx])
		default:
			sb.WriteRune(' ')
		}
	}
	return lipgloss.NewStyle().Foreground(color).Render(sb.String())
}

func momentumAxis(chartW, maxMinute int, color lipgloss.Color) string {
	buf := make([]rune, chartW)
	for x := 0; x < chartW; x++ {
		buf[x] = '─'
	}
	step := momentumLabelStep(chartW, maxMinute)
	for m := 0; m <= maxMinute; m += step {
		x := m * chartW / maxMinute
		if x >= chartW {
			x = chartW - 1
		}
		buf[x] = '┬'
	}
	return lipgloss.NewStyle().Foreground(color).Render(string(buf))
}

func momentumTimeLabels(chartW, maxMinute int, color lipgloss.Color) string {
	buf := make([]rune, chartW)
	for i := range buf {
		buf[i] = ' '
	}
	step := momentumLabelStep(chartW, maxMinute)
	for m := 0; m <= maxMinute; m += step {
		x := m * chartW / maxMinute
		if x >= chartW {
			x = chartW - 1
		}
		label := fmt.Sprintf("%d'", m)
		for j, ch := range label {
			pos := x + j
			if pos < chartW {
				buf[pos] = ch
			}
		}
	}
	return lipgloss.NewStyle().Foreground(color).Render(string(buf))
}

// momentumLabelStep picks a minute interval that keeps labels at least ~10 chars apart.
func momentumLabelStep(chartW, maxMinute int) int {
	step := 15
	for step < maxMinute {
		spacing := step * chartW / maxMinute
		if spacing >= 10 {
			break
		}
		step += 15
	}
	return step
}

func absInt(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

func (md *MatchDetail) availablePeriods() []int {
	order := []int{domain.PeriodAll, domain.PeriodFirstHalf, domain.PeriodSecondHalf, domain.PeriodETFirstHalf, domain.PeriodETSecondHalf}
	seen := make(map[int]bool)
	for p := range md.Details.StatsByPeriod {
		seen[p] = true
	}
	for _, ev := range md.Details.Events.Items {
		if ev.Period > 0 {
			seen[ev.Period] = true
		}
	}
	var out []int
	for _, p := range order {
		if p == domain.PeriodAll || seen[p] {
			out = append(out, p)
		}
	}
	return out
}

func (md *MatchDetail) renderPeriodSubtabs(periods []int, width int) string {
	labels := map[int]string{
		domain.PeriodAll:         "Todo",
		domain.PeriodFirstHalf:   "1T",
		domain.PeriodSecondHalf:  "2T",
		domain.PeriodETFirstHalf: "TE1",
		domain.PeriodETSecondHalf: "TE2",
	}
	var parts []string
	for i, p := range periods {
		if i > 0 {
			parts = append(parts, lipgloss.NewStyle().Foreground(lipgloss.Color("236")).Render("│"))
		}
		label := labels[p]
		if p == md.Details.SelectedPeriod {
			parts = append(parts, lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("15")).Background(lipgloss.Color("39")).Padding(0, 2).Render(label))
		} else {
			parts = append(parts, lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Padding(0, 2).Render(label))
		}
	}
	row := lipgloss.JoinHorizontal(lipgloss.Top, parts...)
	hint := lipgloss.NewStyle().Foreground(lipgloss.Color("238")).Render("  a/s cambia periodo")
	return lipgloss.JoinHorizontal(lipgloss.Top, row, hint)
}

func (md *MatchDetail) renderStatsTable(stats []StatCategory, width int, homeColor, awayColor lipgloss.Color) string {
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

	for _, cat := range stats {
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

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}
